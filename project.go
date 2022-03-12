package project

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/buildkite/interpolate"
	"github.com/flynn/json5"
	"github.com/google/go-github/v43/github"
	"github.com/imdario/mergo"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type PortAttribute struct {
	Label         string `json:"label"`
	OnAutoForward string `json:"onAutoForward"`
}

type DevContainer struct {
	DirPath string
	Name    string `json:"name"`
	Build   struct {
		Dockerfile string            `json:"dockerfile"`
		Context    string            `json:"context"`
		Args       map[string]string `json:"args"`
	} `json:"build"`
	RunArgs           []string                 `json:"runArgs"`
	WorkspaceMount    string                   `json:"workspaceMount"`
	WorkspaceFolder   string                   `json:"workspaceFolder"`
	Settings          map[string]interface{}   `json:"settings"`
	Extensions        []string                 `json:"extensions"`
	ForwardPorts      []string                 `json:"forwardPorts"`
	PortsAttributes   map[string]PortAttribute `json:"portsAttributes"`
	PostCreateCommand string                   `json:"postCreateCommand"`
	RemoteUser        string                   `json:"remoteUser"`
}

type ServiceURL struct {
	Host            string
	Port            int
	WorkspaceFolder string
}

func (s *ServiceURL) String() string {
	return fmt.Sprintf("http://%s:%d/?folder=%s", s.Host, s.Port, s.WorkspaceFolder)
}

type ContainerContext struct {
	cmd  *exec.Cmd
	name string
}

func (c *ContainerContext) Run() error {
	if err := c.cmd.Start(); err != nil {
		return err
	}
	defer c.cmd.Wait()

	c.waitForSignal()
	return c.stop()
}

func (c *ContainerContext) stop() error {
	cmd := exec.Command("docker", "kill", c.name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *ContainerContext) waitForSignal() {
	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-s
}

type KeyBinding struct {
	Key     string `json:"key"`
	Command string `json:"command"`
	When    string `json:"when"`
}

func getImageTag(devcontainer DevContainer) string {
	name := strings.ToLower(devcontainer.Name)
	name = strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s_code_coder_server", name)
}

func getBuildContext(devcontainer DevContainer) string {
	if filepath.IsAbs(devcontainer.Build.Context) {
		return devcontainer.Build.Context
	} else {
		return filepath.Join(devcontainer.DirPath, devcontainer.Build.Context)
	}
}

func BuildImage(devcontainer DevContainer) (string, error) {
	dockerfileContent, err := wrapDockerFile(devcontainer)
	if err != nil {
		return "", err
	}

	tag := getImageTag(devcontainer)
	context := getBuildContext(devcontainer)

	args := []string{"build", "-t", tag, "-f", "-"}
	for k, v := range devcontainer.Build.Args {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, context)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = strings.NewReader(dockerfileContent)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return tag, nil
}

func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

func getIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("No IP address found, and no localhost found")
}

func GetServiceURL(devcontainer DevContainer) (ServiceURL, error) {
	var host string
	var err error
	host, err = getHostname()
	if err != nil {
		host, err = getIPAddress()
		if err != nil {
			return ServiceURL{}, err
		}
	}

	port, err := getAvailablePort()
	if err != nil {
		return ServiceURL{}, err
	}

	workspaceFolder, err := getWorkspaceFolder(devcontainer)
	if err != nil {
		return ServiceURL{}, err
	}

	return ServiceURL{
		Host:            host,
		Port:            port,
		WorkspaceFolder: workspaceFolder,
	}, nil
}

func getMapEnv(devcontainer DevContainer) interpolate.Env {
	localWorkspaceFolder := filepath.Dir(devcontainer.DirPath)
	localWorkspaceFolderBasename := filepath.Base(localWorkspaceFolder)
	env := map[string]string{
		"localWorkspaceFolder":         localWorkspaceFolder,
		"localWorkspaceFolderBasename": localWorkspaceFolderBasename,
	}
	return interpolate.NewMapEnv(env)
}

func getSettingsSyncGistId() (string, error) {
	settingsSyncGistId := os.Getenv("SETTINGS_SYNC_GIST_ID")
	if settingsSyncGistId == "" {
		return "", fmt.Errorf("SETTINGS_SYNC_GIST_ID is not set")
	}
	return settingsSyncGistId, nil
}

func getWorkspaceBinding(devcontainer DevContainer) (string, error) {
	workspaceMount := devcontainer.WorkspaceMount
	if workspaceMount == "" {
		workspaceMount = "source=${localWorkspaceFolder},target=/workspace/${localWorkspaceFolderBasename},type=bind"
	}

	mapEnv := getMapEnv(devcontainer)
	return interpolate.Interpolate(mapEnv, workspaceMount)
}

func getWorkspaceFolder(devcontainer DevContainer) (string, error) {
	workspaceFolder := devcontainer.WorkspaceFolder
	if workspaceFolder == "" {
		workspaceFolder = "/workspace/${localWorkspaceFolderBasename}"
	}

	mapEnv := getMapEnv(devcontainer)
	return interpolate.Interpolate(mapEnv, workspaceFolder)
}

func createEntryScriptCommands(ctx context.Context, devcontainer DevContainer) ([]string, error) {
	scriptCommands := []string{`#!/bin/bash`, `set -e`, `set -x`, devcontainer.PostCreateCommand}
	scriptCommands = append(scriptCommands, `code-server --user-data-dir /opt/code-server/.vscode --config /opt/code-server/config.yml --bind-addr 0.0.0.0:8080`)
	return scriptCommands, nil
}

func createEntryScript(ctx context.Context, devcontainer DevContainer) (string, error) {
	entryScriptCommands, err := createEntryScriptCommands(ctx, devcontainer)
	if err != nil {
		return "", err
	}
	entryScriptContents := strings.Join(entryScriptCommands, "\n")
	b64EntryScriptContents := b64.StdEncoding.EncodeToString([]byte(entryScriptContents))

	dockerfileCommands := []string{
		`RUN mkdir -p /opt/code-server`,
		`RUN echo '` + b64EntryScriptContents + `' | base64 -d > /opt/code-server/entrypoint.sh`,
		`RUN chmod +x /opt/code-server/entrypoint.sh`,
	}
	result := strings.Join(dockerfileCommands, "\n")
	return result, nil
}

func fetchContentsFromGist(ctx context.Context, filename string) (string, error) {
	gistId, err := getSettingsSyncGistId()
	if err != nil {
		return "", err
	}

	client := github.NewClient(nil)
	gist, _, err := client.Gists.Get(ctx, gistId)
	if err != nil {
		return "", err
	}

	gistFile, ok := gist.GetFiles()[github.GistFilename(filename)]
	if !ok {
		return "", fmt.Errorf("%s not found in gist", filename)
	}

	return gistFile.GetContent(), nil
}

func dumpAsJson(obj interface{}) (string, error) {
	data := new(bytes.Buffer)
	encoder := json.NewEncoder(data)
	encoder.SetEscapeHTML(false)
	encoder.Encode(obj)

	var out bytes.Buffer
	err := json.Indent(&out, data.Bytes(), "", "  ")
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func createSettingJson(ctx context.Context, devcontainer DevContainer) (string, error) {
	settings := devcontainer.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}

	if contentsFromSync, err := fetchContentsFromGist(ctx, "settings.json"); err == nil {
		var obj map[string]interface{}
		if err := json5.Unmarshal([]byte(contentsFromSync), &obj); err == nil {
			mergo.Merge(&settings, obj)
		}
	}

	settingsJsonContents, err := dumpAsJson(settings)
	if err != nil {
		return "", err
	}

	b64SettingsJsonContents := b64.StdEncoding.EncodeToString([]byte(settingsJsonContents))
	dockerfileCommands := []string{
		`RUN mkdir -p /opt/code-server/.vscode/User`,
		`RUN echo '` + b64SettingsJsonContents + `' | base64 -d > /opt/code-server/.vscode/User/settings.json`,
	}
	result := strings.Join(dockerfileCommands, "\n")
	return result, nil
}

func createKeybindingsJson(ctx context.Context, devcontainer DevContainer) (string, error) {
	keybindingsJsonFilenames := [...]string{
		"keybindings.json",
		"keybindingsMac.json",
	}

	for _, filename := range keybindingsJsonFilenames {
		if contentsFromSync, err := fetchContentsFromGist(ctx, filename); err == nil {
			if len(contentsFromSync) == 0 {
				continue
			}

			var obj []KeyBinding
			err := json5.Unmarshal([]byte(contentsFromSync), &obj)
			if err != nil {
				continue
			}

			keybindingsJsonContents, err := dumpAsJson(obj)
			if err != nil {
				continue
			}

			b64KeybindingsJsonContents := b64.StdEncoding.EncodeToString([]byte(keybindingsJsonContents))
			dockerfileCommands := []string{
				`RUN mkdir -p /opt/code-server/.vscode/User`,
				`RUN echo '` + b64KeybindingsJsonContents + `' | base64 -d > /opt/code-server/.vscode/User/keybindings.json`,
			}
			result := strings.Join(dockerfileCommands, "\n")
			return result, nil
		}
	}

	return "", nil
}

func modifyCodeServerDirPermissions(ctx context.Context, devcontainer DevContainer) (string, error) {
	return `RUN chmod -R o+wr /opt/code-server/`, nil
}

func installExtensions(ctx context.Context, devcontainer DevContainer) (string, error) {
	commands := []string{}
	for _, v := range devcontainer.Extensions {
		commands = append(commands, fmt.Sprintf("RUN code-server --install-extension %s --extensions-dir /opt/code-server/.vscode/extensions/", v))
	}

	result := strings.Join(commands, "\n")
	return result, nil
}

func createConfigYaml(ctx context.Context, container DevContainer) (string, error) {
	return `RUN echo "auth: none" > /opt/code-server/config.yml`, nil
}

const (
	CodeServerInstall = `RUN curl -fsSL https://code-server.dev/install.sh | sh`
	Entrypoint        = `ENTRYPOINT ["/opt/code-server/entrypoint.sh"]`
)

func wrapDockerFile(devcontainer DevContainer) (string, error) {
	ctx := context.Background()

	dockerfilePath := filepath.Join(devcontainer.DirPath, devcontainer.Build.Dockerfile)
	dockerfile, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return "", err
	}

	entryScriptCreation, err := createEntryScript(ctx, devcontainer)
	if err != nil {
		return "", err
	}

	extensionsInstallation, err := installExtensions(ctx, devcontainer)
	if err != nil {
		log.Print(err)
		extensionsInstallation = ""
	}

	codeServerDirPermissionModification, err := modifyCodeServerDirPermissions(ctx, devcontainer)
	if err != nil {
		log.Print(err)
		codeServerDirPermissionModification = ""
	}

	configYamlCreation, err := createConfigYaml(ctx, devcontainer)
	if err != nil {
		log.Print(err)
		configYamlCreation = ""
	}

	settingJsonCreation, err := createSettingJson(ctx, devcontainer)
	if err != nil {
		log.Print(err)
		settingJsonCreation = ""
	}

	keybindingsJsonCreation, err := createKeybindingsJson(ctx, devcontainer)
	if err != nil {
		log.Print(err)
		keybindingsJsonCreation = ""
	}

	dockerfileContent := string(dockerfile)
	dockerfileContent = strings.Join([]string{
		dockerfileContent,
		CodeServerInstall,
		settingJsonCreation,
		keybindingsJsonCreation,
		entryScriptCreation,
		extensionsInstallation,
		configYamlCreation,
		codeServerDirPermissionModification,
		Entrypoint}, "\n")

	return dockerfileContent, nil
}

func makeRandomString() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func NewContainerContext(tag string, devcontainer DevContainer, serviceURL ServiceURL) (ContainerContext, error) {
	name := makeRandomString()
	portBinding := fmt.Sprintf("0.0.0.0:%d:8080", serviceURL.Port)
	args := []string{"run", "--rm", "-p", portBinding, "--name", name}

	workspaceBinding, err := getWorkspaceBinding(devcontainer)
	if err != nil {
		return ContainerContext{}, err
	}
	args = append(args, "--mount", workspaceBinding)

	args = append(args, "-w", serviceURL.WorkspaceFolder)

	for _, v := range devcontainer.RunArgs {
		args = append(args, v)
	}
	for _, v := range devcontainer.ForwardPorts {
		args = append(args, "-p", v)
	}
	if devcontainer.RemoteUser != "" {
		args = append(args, "-u", devcontainer.RemoteUser)
	}
	args = append(args, tag)
	args = append(args)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ctx := ContainerContext{
		cmd:  cmd,
		name: name,
	}
	return ctx, nil
}
