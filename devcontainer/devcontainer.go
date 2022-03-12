package devcontainer

import (
	"github.com/flynn/json5"
	"io/ioutil"
	"path/filepath"
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

func ParseJson(path string) (DevContainer, error) {
	var devcontainer DevContainer
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return devcontainer, err
	}
	if err := json5.Unmarshal(raw, &devcontainer); err != nil {
		return devcontainer, err
	}
	absDirPath, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return devcontainer, err
	}
	devcontainer.DirPath = absDirPath
	return devcontainer, nil
}
