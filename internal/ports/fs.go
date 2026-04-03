package ports

import "context"

type FileSystem interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte) error
	Stat(ctx context.Context, path string) (bool, error)
	MkdirAll(ctx context.Context, path string) error
	Glob(ctx context.Context, root string, patterns ...string) ([]string, error)
	WalkFiles(ctx context.Context, root string, exts []string) ([]string, error)
	ExecRoot() string
}
