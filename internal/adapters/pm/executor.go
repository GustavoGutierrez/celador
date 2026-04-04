package pm

import (
	"context"
	"io"
	"os/exec"

	installcore "github.com/GustavoGutierrez/celador/internal/core/install"
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
	binary, cmdArgs, err := installcore.CommandForManager(workspace.PackageManager, args)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	cmd.Dir = workspace.Root
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	return cmd.Run()
}
