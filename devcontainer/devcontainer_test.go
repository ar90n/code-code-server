package devcontainer

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestDevcontainer(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "devcontainer.json")
	defer os.Remove(tmpFile.Name())

	contents := `// For format details, see https://aka.ms/devcontainer.json. For config options, see the README at:
	// https://github.com/microsoft/vscode-dev-containers/tree/v0.224.2/containers/go
	{
		"name": "Go",
		"build": {
		"dockerfile": "Dockerfile",
			"args": {
			// Update the VARIANT arg to pick a version of Go: 1, 1.16, 1.17
			// Append -bullseye or -buster to pin to an OS version.
 			// Use -bullseye variants on local arm64/Apple Silicon.
			"VARIANT": "1-bullseye",
			// Options
			"NODE_VERSION": "lts/*"
		}
	},
	"runArgs": [ "--cap-add=SYS_PTRACE", "--security-opt", "seccomp=unconfined" ],

	// Set *default* container specific settings.json values on container create.
	"settings": {
	"go.useLanguageServer": true,
	"go.gopath": "/go",
	"go.goroot": "/usr/local/go"
	},

	// Add the IDs of extensions you want installed when the container is created.
	"extensions": [
	"golang.Go"
	],

	// Use 'forwardPorts' to make a list of ports inside the container available locally.
	"portsAttributes": {
    "8000": {
      "label": "Web",
      "onAutoForward": "openBrowser"
    }},

	// Use 'postCreateCommand' to run commands after the container is created.
	"postCreateCommand": "go version",

	// Comment out to connect as root instead. More info: https://aka.ms/vscode-remote/containers/non-root.
	"remoteUser": "vscode"
	}`
	tmpFile.WriteString(contents)

	devcontainer, err := ParseJson(tmpFile.Name())
	if err != nil {
		t.Errorf("Error parsing devcontainer.json: %v", err)
	}
	if devcontainer.Name != "Go" {
		t.Errorf("Expected devcontainer.json name to be 'Go', got %s", devcontainer.Name)
	}
	if devcontainer.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Expected devcontainer.json build.dockerfile to be 'Dockerfile', got %s", devcontainer.Build.Dockerfile)
	}
	if devcontainer.Build.Args["VARIANT"] != "1-bullseye" {
		t.Errorf("Expected devcontainer.json build.args.VARIANT to be '1-bullseye', got %s", devcontainer.Build.Args["VARIANT"])
	}
	if devcontainer.Build.Args["NODE_VERSION"] != "lts/*" {
		t.Errorf("Expected devcontainer.json build.args.NODE_VERSION to be 'lts/*', got %s", devcontainer.Build.Args["NODE_VERSION"])
	}
	if devcontainer.RunArgs[0] != "--cap-add=SYS_PTRACE" {
		t.Errorf("Expected devcontainer.json runArgs[0] to be '--cap-add=SYS_PTRACE', got %s", devcontainer.RunArgs[0])
	}
	if devcontainer.RunArgs[1] != "--security-opt" {
		t.Errorf("Expected devcontainer.json runArgs[1] to be '--security-opt', got %s", devcontainer.RunArgs[1])
	}
	if devcontainer.Settings["go.useLanguageServer"] != true {
		t.Errorf("Expected devcontainer.json settings.go.useLanguageServer to be true, got %v", devcontainer.Settings["go.useLanguageServer"])
	}
	if devcontainer.Settings["go.gopath"] != "/go" {
		t.Errorf("Expected devcontainer.json settings.go.gopath to be '/go', got %s", devcontainer.Settings["go.gopath"])
	}
	if devcontainer.Extensions[0] != "golang.Go" {
		t.Errorf("Expected devcontainer.json extensions[0] to be 'golang.Go', got %s", devcontainer.Extensions[0])
	}
	if devcontainer.PortsAttributes["8000"].Label != "Web" {
		t.Errorf("Expected devcontainer.json portsAttributes[8000].label to be 'Web', got %s", devcontainer.PortsAttributes["8000"].Label)
	}
	if devcontainer.PortsAttributes["8000"].OnAutoForward != "openBrowser" {
		t.Errorf("Expected devcontainer.json portsAttributes[8000].onAutoForward to be 'openBrowser', got %s", devcontainer.PortsAttributes["8000"].OnAutoForward)
	}
	if devcontainer.PostCreateCommand != "go version" {
		t.Errorf("Expected devcontainer.json postCreateCommand to be 'go version', got %s", devcontainer.PostCreateCommand)
	}
	if devcontainer.RemoteUser != "vscode" {
		t.Errorf("Expected devcontainer.json remoteUser to be 'vscode', got %s", devcontainer.RemoteUser)
	}
}
