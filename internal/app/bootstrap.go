package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	cacheadapter "github.com/GustavoGutierrez/celador/internal/adapters/cache"
	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	osvadapter "github.com/GustavoGutierrez/celador/internal/adapters/osv"
	pmadapter "github.com/GustavoGutierrez/celador/internal/adapters/pm"
	rulesadapter "github.com/GustavoGutierrez/celador/internal/adapters/rules"
	tuiadapter "github.com/GustavoGutierrez/celador/internal/adapters/tui"
	"github.com/GustavoGutierrez/celador/internal/core/audit"
	"github.com/GustavoGutierrez/celador/internal/core/fix"
	"github.com/GustavoGutierrez/celador/internal/core/install"
	"github.com/GustavoGutierrez/celador/internal/core/workspace"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ExitCoder interface {
	error
	ExitCode() int
}

type exitError struct {
	err  error
	code int
}

func (e exitError) Error() string { return e.err.Error() }
func (e exitError) Unwrap() error { return e.err }
func (e exitError) ExitCode() int { return e.code }

func NewExitError(code int, format string, args ...any) error {
	return exitError{err: fmt.Errorf(format, args...), code: code}
}

type Runtime struct {
	Root      string
	TTY       bool
	CI        bool
	FS        ports.FileSystem
	Cache     ports.ScanCache
	UI        ports.PromptUI
	Detector  ports.WorkspaceDetector
	Ignore    ports.IgnoreStore
	Rules     ports.RuleLoader
	Eval      ports.RuleEvaluator
	OSV       ports.VulnerabilitySource
	Metadata  ports.PackageMetadataSource
	PM        ports.PackageManager
	Patches   ports.PatchWriter
	Parsers   []ports.LockfileParser
	Clock     ports.Clock
	Config    *viper.Viper
	ScanSvc   *audit.Service
	InitSvc   *workspace.Service
	FixSvc    *fix.Service
	InstallSv *install.Service
	RootCmd   *cobra.Command
}

type Bootstrap struct {
	runtime *Runtime
}

func NewBootstrap(ctx context.Context, args []string) (*Bootstrap, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	fs := fsadapter.NewOSFileSystem(root)
	config := viper.New()
	config.SetConfigName(".celador")
	config.SetConfigType("yaml")
	config.AddConfigPath(root)
	config.SetDefault("cache.ttl", "24h")
	config.SetDefault("rules.version", "v1")
	if err := config.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	tty := tuiadapter.IsTTY(os.Stdin.Fd(), os.Stdout.Fd())
	ci := os.Getenv("CI") != ""
	clock := systemClock{}

	cacheDir := filepath.Join(root, ".celador", "cache")
	cache := cacheadapter.NewFileCache(fs, cacheDir, clock)
	ui := tuiadapter.NewTerminalUI(os.Stdin, os.Stdout, tty, ci)
	detector := workspace.NewDetector(fs)
	ignore := fsadapter.NewIgnoreStore(fs)
	loader := rulesadapter.NewYAMLLoader(fs)
	evaluator := audit.NewRuleEvaluator(fs)
	osv := osvadapter.NewClient(config.GetDuration("cache.ttl"))
	metadata := osvadapter.NewRegistryInspector()
	pm := pmadapter.NewExecutor(os.Stdout, os.Stderr)
	patches := fsadapter.NewPatchWriter(fs)
	parsers := []ports.LockfileParser{
		audit.NewNPMParser(fs),
		audit.NewPNPMParser(fs),
		audit.NewBunParser(fs),
		audit.NewDenoParser(fs),
	}

	rt := &Runtime{
		Root:     root,
		TTY:      tty,
		CI:       ci,
		FS:       fs,
		Cache:    cache,
		UI:       ui,
		Detector: detector,
		Ignore:   ignore,
		Rules:    loader,
		Eval:     evaluator,
		OSV:      osv,
		Metadata: metadata,
		PM:       pm,
		Patches:  patches,
		Parsers:  parsers,
		Clock:    clock,
		Config:   config,
	}

	rt.ScanSvc = audit.NewService(rt.Detector, rt.Ignore, rt.Rules, rt.Eval, rt.OSV, rt.Cache, rt.Clock, config.GetDuration("cache.ttl"), rt.Parsers)
	rt.InitSvc = workspace.NewService(fs, detector, ignore, ui)
	rt.FixSvc = fix.NewService(rt.ScanSvc, rt.Patches, fs, ui)
	rt.InstallSv = install.NewService(detector, metadata, pm, ui)
	rt.RootCmd = newRootCommand(rt)
	rt.RootCmd.SetArgs(args)

	_ = ctx
	return &Bootstrap{runtime: rt}, nil
}

func (b *Bootstrap) Execute(ctx context.Context) error {
	return b.runtime.RootCmd.ExecuteContext(ctx)
}

func (b *Bootstrap) OverrideOutput(out io.Writer) {
	b.runtime.UI = tuiadapter.NewTerminalUI(strings.NewReader("y\n"), out, false, false)
	b.runtime.RootCmd.SetOut(out)
	b.runtime.RootCmd.SetErr(out)
}

func (b *Bootstrap) OverrideInteractivity(tty bool, ci bool) {
	b.runtime.TTY = tty
	b.runtime.CI = ci
}

func (b *Bootstrap) OverrideUI(ui ports.PromptUI) {
	b.runtime.UI = ui
}

func (b *Bootstrap) OverridePackageManager(pm ports.PackageManager) {
	b.runtime.PM = pm
	b.runtime.InstallSv = install.NewService(b.runtime.Detector, b.runtime.Metadata, pm, b.runtime.UI)
}

func (b *Bootstrap) OverridePackageMetadata(meta ports.PackageMetadataSource) {
	b.runtime.Metadata = meta
	b.runtime.InstallSv = install.NewService(b.runtime.Detector, meta, b.runtime.PM, b.runtime.UI)
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
