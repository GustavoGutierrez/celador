package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GustavoGutierrez/celador/internal/app"
)

func main() {
	ctx := context.Background()
	bootstrap, err := app.NewBootstrap(ctx, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := bootstrap.Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if exitErr, ok := err.(app.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
