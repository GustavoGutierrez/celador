package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/GustavoGutierrez/celador/internal/ports"
)

type OSFileSystem struct{ root string }

func NewOSFileSystem(root string) ports.FileSystem { return &OSFileSystem{root: root} }

func (fs *OSFileSystem) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *OSFileSystem) WriteFile(_ context.Context, path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (fs *OSFileSystem) Stat(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (fs *OSFileSystem) MkdirAll(_ context.Context, path string) error {
	return os.MkdirAll(path, 0o755)
}

func (fs *OSFileSystem) Glob(_ context.Context, root string, patterns ...string) ([]string, error) {
	results := []string{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		results = append(results, matches...)
	}
	sort.Strings(results)
	return results, nil
}

func (fs *OSFileSystem) WalkFiles(_ context.Context, root string, exts []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		for _, ext := range exts {
			if filepath.Ext(path) == ext {
				files = append(files, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (fs *OSFileSystem) ExecRoot() string { return fs.root }
