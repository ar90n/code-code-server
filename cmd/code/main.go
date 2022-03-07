package main

import (
	"codecodeserver"
	"fmt"
	"github.com/flynn/json5"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func parseDevcontainerJson(path string) (project.DevContainer, error) {
	var devcontainer project.DevContainer
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

func prettyUrlPrint(url project.ServiceURL) {
	log.Printf("==============================================================================================")
	log.Printf("Code Server running at %s", url.String())
	log.Printf("==============================================================================================")
}

func main() {
	app := &cli.App{
		Name:    "code",
		Version: "0.0.1",
		Usage:   "code",
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				return fmt.Errorf("Please provide a project directory")
			}

			projectDirPath := c.Args().Get(0)
			if _, err := os.Stat(projectDirPath); os.IsNotExist(err) {
				return fmt.Errorf("Project directory does not exist")
			}

			devcontainerDirPath := filepath.Join(projectDirPath, ".devcontainer")
			if _, err := os.Stat(devcontainerDirPath); os.IsNotExist(err) {
				return fmt.Errorf("Project directory does not contain a .devcontainer directory")
			}

			devcontainerJsonPath := filepath.Join(devcontainerDirPath, "devcontainer.json")
			devcontainerObj, err := parseDevcontainerJson(devcontainerJsonPath)
			if err != nil {
				return err
			}

			tag, err := project.BuildImage(devcontainerObj)
			if err != nil {
				return err
			}

			url, err := project.GetServiceURL(devcontainerObj)
			if err != nil {
				return err
			}

			cmd, err := project.CreateRunCmd(tag, devcontainerObj, url)
			if err != nil {
				return err
			}

			prettyUrlPrint(url)
			cmd.Run()

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
