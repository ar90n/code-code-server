package dockerfile

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	. "github.com/ar90n/code-code-server/devcontainer"
	. "github.com/ar90n/code-code-server/settings"
	"github.com/flynn/json5"
	"github.com/imdario/mergo"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

type KeyBinding struct {
	Key     string `json:"key"`
	Command string `json:"command"`
	When    string `json:"when"`
}

const (
	CodeServerInstall = `RUN curl -fsSL https://code-server.dev/install.sh | sh`
	Entrypoint        = `ENTRYPOINT ["/opt/code-server/entrypoint.sh"]`
)

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

func createSettingJson(ctx context.Context, devcontainer DevContainer, repository Repository) (string, error) {
	settings := devcontainer.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}

	if contentsFromSync, err := repository.Get(ctx, "settings.json"); err == nil {
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

func createKeybindingsJson(ctx context.Context, devcontainer DevContainer, repository Repository) (string, error) {
	keybindingsJsonFilenames := [...]string{
		"keybindings.json",
		"keybindingsMac.json",
	}

	for _, filename := range keybindingsJsonFilenames {
		if contentsFromSync, err := repository.Get(ctx, filename); err == nil {
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

func WrapDockerFile(devcontainer DevContainer, repository Repository) (string, error) {
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

	settingJsonCreation, err := createSettingJson(ctx, devcontainer, repository)
	if err != nil {
		log.Print(err)
		settingJsonCreation = ""
	}

	keybindingsJsonCreation, err := createKeybindingsJson(ctx, devcontainer, repository)
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
