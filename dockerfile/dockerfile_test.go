package dockerfile

import (
	"context"
	"fmt"
	. "github.com/ar90n/code-code-server/devcontainer"
	"io/ioutil"
	"os"
	"testing"
)

type MemoryRepository struct {
	data map[string]string
}

func (r *MemoryRepository) Get(ctx context.Context, filename string) (string, error) {
	if value, ok := r.data[filename]; ok {
		return value, nil
	}
	return "", fmt.Errorf("Filename %s not found", filename)
}

func TestDockerfile(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "Dockerfile")
	defer os.Remove(tmpFile.Name())

	dockerfileContents := `FROM golang:1.12.5`
	tmpFile.WriteString(dockerfileContents)

	devcontainer := DevContainer{}
	devcontainer.Name = "test"
	devcontainer.Build.Dockerfile = tmpFile.Name()
	devcontainer.Build.Context = "."

	repository := MemoryRepository{data: map[string]string{}}
	contents, err := WrapDockerFile(devcontainer, &repository)

	if err != nil {
		t.Errorf("Error wrapping Dockerfile: %s", err)
	}

	expectDockerfileContents := `FROM golang:1.12.5
RUN curl -fsSL https://code-server.dev/install.sh | sh
RUN mkdir -p /opt/code-server/.vscode/User
RUN echo 'e30K' | base64 -d > /opt/code-server/.vscode/User/settings.json

RUN mkdir -p /opt/code-server
RUN echo 'IyEvYmluL2Jhc2gKc2V0IC1lCnNldCAteAoKY29kZS1zZXJ2ZXIgLS11c2VyLWRhdGEtZGlyIC9vcHQvY29kZS1zZXJ2ZXIvLnZzY29kZSAtLWNvbmZpZyAvb3B0L2NvZGUtc2VydmVyL2NvbmZpZy55bWwgLS1iaW5kLWFkZHIgMC4wLjAuMDo4MDgw' | base64 -d > /opt/code-server/entrypoint.sh
RUN chmod +x /opt/code-server/entrypoint.sh

RUN echo "auth: none" > /opt/code-server/config.yml
RUN chmod -R o+wr /opt/code-server/
ENTRYPOINT ["/opt/code-server/entrypoint.sh"]`
	if contents != expectDockerfileContents {
		t.Errorf("Expected Dockerfile contents to be %s, got %s", expectDockerfileContents, contents)
	}
}
