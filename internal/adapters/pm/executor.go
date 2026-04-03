package pm

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type Executor struct {
	stdout io.Writer
	stderr io.Writer
}

func NewExecutor(stdout io.Writer, stderr io.Writer) *Executor {
	return &Executor{stdout: stdout, stderr: stderr}
}

func (e *Executor) Install(ctx context.Context, workspace shared.Workspace, args []string) error {
	var binary string
	var cmdArgs []string
	switch workspace.PackageManager {
	case shared.PackageManagerNPM:
		binary = "npm"
		cmdArgs = append([]string{"install"}, args...)
	case shared.PackageManagerPNPM:
		binary = "pnpm"
		cmdArgs = append([]string{"install"}, args...)
	case shared.PackageManagerBun:
		binary = "bun"
		cmdArgs = append([]string{"add"}, args...)
	default:
		return fmt.Errorf("install is unsupported for %s", workspace.PackageManager)
	}
	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	cmd.Dir = workspace.Root
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	return cmd.Run()
}
