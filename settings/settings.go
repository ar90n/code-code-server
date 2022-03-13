package settings

import (
	"context"
)

type Repository interface {
	Get(ctx context.Context, filename string) (string, error)
}
