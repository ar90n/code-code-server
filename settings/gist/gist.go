package gist

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v43/github"
)

type GistRepository struct {
	gistId string
}

func (r *GistRepository) Get(ctx context.Context, filename string) (string, error) {
	client := github.NewClient(nil)
	gist, _, err := client.Gists.Get(ctx, r.gistId)
	if err != nil {
		return "", err
	}

	gistFile, ok := gist.GetFiles()[github.GistFilename(filename)]
	if !ok {
		return "", fmt.Errorf("%s not found in gist", filename)
	}

	return gistFile.GetContent(), nil
}

func New() (GistRepository, error) {
	gistId := os.Getenv("SETTINGS_SYNC_GIST_ID")
	if gistId == "" {
		return GistRepository{}, fmt.Errorf("SETTINGS_SYNC_GIST_ID is not set")
	}

	return NewWithGistID(gistId)
}

func NewWithGistID(gistId string) (GistRepository, error) {
	repository := GistRepository{
		gistId: gistId,
	}
	return repository, nil
}
