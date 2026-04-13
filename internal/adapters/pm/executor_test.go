package pm

import (
	"context"
	"os"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestExecutor_Install_UnknownManager(t *testing.T) {
	t.Parallel()
	executor := NewExecutor(os.Stdout, os.Stderr)
	workspace := shared.Workspace{
		Root:           t.TempDir(),
		PackageManager: shared.PackageManagerUnknown,
	}

	err := executor.Install(context.Background(), workspace, []string{"pkg"})
	if err == nil {
		t.Fatal("expected error for unknown package manager, got nil")
	}
}
