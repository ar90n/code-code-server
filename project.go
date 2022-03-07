package project

import (
	"fmt"
	"github.com/buildkite/interpolate"
	"io/ioutil"
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
		Args       map[string]string `json:"args"`
	} `json:"build"`
	RunArgs           []string                 `json:"runArgs"`
	WorkspaceMount    string                   `json:"workspaceMount"`
	WorkspaceFolder   string                   `json:"workspaceFolder"`
	Settings          map[string]string        `json:"settings"`
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

func getImageTag(devcontainer DevContainer) string {
	name := strings.ToLower(devcontainer.Name)
	name = strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s_code_coder_server", name)
}

func BuildImage(devcontainer DevContainer) (string, error) {
	dockerfileContent, err := wrapDockerFile(devcontainer)
	if err != nil {
		return "", err
	}

	tag := getImageTag(devcontainer)
	context := devcontainer.DirPath

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
		// check the address type and if it is not a loopback the display it
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
	for k, v := range devcontainer.Settings {
		env[k] = v
	}
	return interpolate.NewMapEnv(env)
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

func createEntryScriptCommands(devcontainer DevContainer) ([]string, error) {
	scriptCommands := []string{`#!/bin/bash`, `set -e`, `set -x`, devcontainer.PostCreateCommand}
	for _, v := range devcontainer.Extensions {
		scriptCommands = append(scriptCommands, fmt.Sprintf("code-server --install-extension %s", v))
	}
	scriptCommands = append(scriptCommands, `echo "auth: none" > /tmp/config.yml`)
	scriptCommands = append(scriptCommands, `code-server --config /tmp/config.yml --bind-addr 0.0.0.0:8080`)
	return scriptCommands, nil
}

func createEntryScript(devcontainer DevContainer) (string, error) {
	entryScriptCommands, err := createEntryScriptCommands(devcontainer)
	if err != nil {
		return "", err
	}

	dockerfileCommands := []string{`RUN mkdir -p /opt/code-server`, `RUN { \`}
	for _, v := range entryScriptCommands {
		dockerfileCommands = append(dockerfileCommands, fmt.Sprintf(`echo '%s'; \`, v))
	}
	dockerfileCommands = append(dockerfileCommands, `} > /opt/code-server/entrypoint.sh`)
	dockerfileCommands = append(dockerfileCommands, `RUN chmod +x /opt/code-server/entrypoint.sh`)

	result := strings.Join(dockerfileCommands, "\n")
	return result, nil
}

const (
	CodeServerInstall = `RUN curl -fsSL https://code-server.dev/install.sh | sh`
	Entrypoint        = `ENTRYPOINT ["/opt/code-server/entrypoint.sh"]`
)

func wrapDockerFile(devcontainer DevContainer) (string, error) {
	dockerfilePath := filepath.Join(devcontainer.DirPath, devcontainer.Build.Dockerfile)
	dockerfile, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return "", err
	}
	entryScriptCreation, err := createEntryScript(devcontainer)
	if err != nil {
		return "", err
	}

	dockerfileContent := string(dockerfile)
	dockerfileContent = strings.Join([]string{dockerfileContent, CodeServerInstall, entryScriptCreation, Entrypoint}, "\n")

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
