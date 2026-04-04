package app

import (
	"context"
	"fmt"

	installcore "github.com/GustavoGutierrez/celador/internal/core/install"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/spf13/cobra"
)

func commandInteractivity(cmd *cobra.Command, rt *Runtime) (tty bool, ci bool, noInteractive bool, err error) {
	noInteractive, err = cmd.Flags().GetBool("no-interactive")
	if err != nil {
		return false, false, false, err
	}
	return rt.TTY && !noInteractive, rt.CI || noInteractive, noInteractive, nil
}

func newRootCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "celador",
		Short:         "Zero-trust dependency security for JS/TS workspaces",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			showVersion, err := cmd.Flags().GetBool("version")
			if err != nil {
				return err
			}
			if showVersion {
				return runVersionCommand(cmd, rt)
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().Bool("no-interactive", false, "Disable prompts and TTY flows")
	cmd.Flags().Bool("version", false, "Print version information and update status")
	cmd.AddCommand(newInitCommand(rt), newScanCommand(rt), newFixCommand(rt), newInstallCommand(rt), newAboutCommand(rt), newTUICommand(rt))
	return cmd
}

func runVersionCommand(cmd *cobra.Command, rt *Runtime) error {
	report := rt.VersionSv.Report(cmd.Context())
	rt.UI.Printf("celador %s\n", report.Current)
	if !report.UpdateAvailable {
		return nil
	}
	rt.UI.Printf("Update available: %s\n", report.Latest)
	if report.InstalledViaHomebrew {
		rt.UI.Printf("Update with Homebrew: brew update && brew upgrade GustavoGutierrez/celador/celador\n")
		return nil
	}
	rt.UI.Printf("If installed with Homebrew, run: brew update && brew upgrade GustavoGutierrez/celador/celador\n")
	return nil
}

func renderBrandingHeader(ctx context.Context, rt *Runtime) error {
	if rt == nil || rt.UI == nil {
		return nil
	}
	return rt.UI.RenderBrandingHeader(ctx, runtimeVersion(rt))
}

func runtimeVersion(rt *Runtime) string {
	if rt != nil && rt.VersionSv != nil {
		return rt.VersionSv.Current()
	}
	return currentVersion()
}

func newInitCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap Celador hardening for a workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewExitError(2, "init does not accept positional arguments")
			}
			tty, ci, _, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			allowHooks, _ := cmd.Flags().GetBool("install-hook")
			res, err := rt.InitSvc.Run(cmd.Context(), rt.Root, tty, ci, allowHooks)
			if err != nil {
				return err
			}
			if err := renderBrandingHeader(cmd.Context(), rt); err != nil {
				return err
			}
			return rt.UI.RenderInit(cmd.Context(), res.Report)
		},
	}
	cmd.Flags().Bool("install-hook", false, "Install a pre-commit hook without prompting")
	return cmd
}

func newScanCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Audit lockfiles and framework configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewExitError(2, "scan does not accept positional arguments")
			}
			jsonOutput, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}
			tty, ci, _, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			result, err := rt.ScanSvc.Run(cmd.Context(), rt.Root, tty, ci)
			if err != nil {
				return err
			}
			renderOptions := shared.ScanRenderOptions{Format: shared.ScanRenderFormatText, Verbose: verbose}
			if jsonOutput {
				renderOptions.Format = shared.ScanRenderFormatJSON
			}
			if !jsonOutput {
				if err := renderBrandingHeader(cmd.Context(), rt); err != nil {
					return err
				}
			}
			if err := rt.UI.RenderScan(cmd.Context(), result, renderOptions); err != nil {
				return err
			}
			renderedFindingCount := shared.RenderedFindingCount(result.Findings)
			if renderedFindingCount > 0 {
				return NewExitError(3, "%d findings detected", renderedFindingCount)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Render structured JSON scan output")
	cmd.Flags().Bool("verbose", false, "Render additional scan context in text output")
	return cmd
}

func newFixCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Plan or apply conservative remediation",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewExitError(2, "fix does not accept positional arguments")
			}
			tty, ci, noInteractive, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			diffOnly, _ := cmd.Flags().GetBool("diff")
			plan, err := rt.FixSvc.Plan(cmd.Context(), rt.Root, tty, ci)
			if err != nil {
				return err
			}
			if err := renderBrandingHeader(cmd.Context(), rt); err != nil {
				return err
			}
			if err := rt.UI.RenderFixPlan(cmd.Context(), plan); err != nil {
				return err
			}
			if len(plan.Operations) == 0 {
				return NewExitError(4, "no safe remediation available")
			}
			if diffOnly {
				return nil
			}
			if !yes {
				if !tty || ci || noInteractive {
					return NewExitError(2, "fix requires confirmation; rerun with --yes in CI or use a TTY")
				}
				confirmed, err := rt.UI.Confirm(cmd.Context(), "Apply remediation plan?")
				if err != nil {
					return err
				}
				if !confirmed {
					return NewExitError(2, "remediation cancelled")
				}
			}
			return rt.FixSvc.Apply(cmd.Context(), rt.Root, tty, ci, plan)
		},
	}
	cmd.Flags().Bool("yes", false, "Apply fixes without prompting")
	cmd.Flags().Bool("diff", false, "Only render the planned diff")
	return cmd
}

func newInstallCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [packages...]",
		Short: "Run package installation with preflight checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return NewExitError(2, "install requires at least one package argument")
			}
			tty, ci, noInteractive, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			assessment, err := rt.InstallSv.Assess(cmd.Context(), rt.Root, tty, ci, args)
			if err != nil {
				return err
			}
			if err := renderBrandingHeader(cmd.Context(), rt); err != nil {
				return err
			}
			if err := rt.UI.RenderInstallAssessment(cmd.Context(), assessment); err != nil {
				return err
			}
			approval := shared.InstallApprovalNotNeeded
			if assessment.ShouldPrompt && !yes {
				if !tty || ci || noInteractive {
					return NewExitError(2, "install risk requires approval; rerun with --yes or in a TTY")
				}
				confirmed, err := rt.UI.Confirm(cmd.Context(), fmt.Sprintf("Proceed with installing %s after reviewing the preflight warnings?", assessment.Package))
				if err != nil {
					return err
				}
				if !confirmed {
					return NewExitError(2, "installation cancelled")
				}
				approval = shared.InstallApprovalPromptApproved
			} else if assessment.ShouldPrompt && yes {
				approval = shared.InstallApprovalAutoApproved
			}
			command, err := buildInstallTimelineCommand(assessment.Manager, args)
			if err != nil {
				return err
			}
			timeline := shared.InstallTimeline{
				Assessment:     assessment,
				RequestedArgs:  append([]string(nil), args...),
				Command:        command,
				Approval:       approval,
				ExecutionState: shared.InstallExecutionRunning,
			}
			if err := rt.UI.RenderInstallTimeline(cmd.Context(), timeline); err != nil {
				return err
			}
			if err := rt.InstallSv.Execute(cmd.Context(), rt.Root, tty, ci, args); err != nil {
				timeline.ExecutionState = shared.InstallExecutionFailed
				timeline.Failure = err.Error()
				if renderErr := rt.UI.RenderInstallTimeline(cmd.Context(), timeline); renderErr != nil {
					return renderErr
				}
				return err
			}
			timeline.ExecutionState = shared.InstallExecutionSucceeded
			return rt.UI.RenderInstallTimeline(cmd.Context(), timeline)
		},
	}
	cmd.Flags().Bool("yes", false, "Continue on risky findings without prompting")
	return cmd
}

func buildInstallTimelineCommand(manager shared.PackageManager, args []string) ([]string, error) {
	binary, cmdArgs, err := installcore.CommandForManager(manager, args)
	if err != nil {
		return nil, err
	}
	return append([]string{binary}, cmdArgs...), nil
}

func newAboutCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "about",
		Short: "Show project, version, and developer information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewExitError(2, "about does not accept positional arguments")
			}
			return rt.UI.RenderOverview(cmd.Context(), buildOverview(cmd.Context(), rt), false)
		},
	}
}

func newTUICommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive Celador overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewExitError(2, "tui does not accept positional arguments")
			}
			tty, ci, noInteractive, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			interactive := tty && !ci && !noInteractive
			return rt.UI.RenderOverview(cmd.Context(), buildOverview(cmd.Context(), rt), interactive)
		},
	}
}

func buildOverview(ctx context.Context, rt *Runtime) shared.Overview {
	overview := shared.Overview{
		Title:         "Celador CLI",
		Subtitle:      "Zero-trust dependency security for JavaScript, TypeScript, and Deno workspaces.",
		Developer:     "Gustavo Gutierrez",
		GitHubProfile: "https://github.com/GustavoGutierrez",
		Commands: []shared.OverviewCommand{
			{Name: "celador init", Summary: "Bootstrap workspace hardening and managed guidance.", Example: "celador init"},
			{Name: "celador scan", Summary: "Audit supported lockfiles and framework rules.", Example: "celador scan"},
			{Name: "celador fix", Summary: "Plan or apply conservative dependency remediation.", Example: "celador fix --diff"},
			{Name: "celador install", Summary: "Assess package risk before installation.", Example: "celador install express"},
			{Name: "celador --version", Summary: "Print the current version and update status.", Example: "celador --version"},
			{Name: "celador tui", Summary: "Open this interactive command center.", Example: "celador tui", Interactive: true},
		},
		QuickStart: []string{
			"Run `celador init` once per workspace to install configuration and guidance.",
			"Use `celador scan` in CI to detect dependency and framework findings deterministically.",
			"Review `celador fix --diff` before applying `celador fix --yes` in trusted environments.",
		},
		Documentation: []string{"README.md", "docs/commands.md", "docs/configuration.md", "docs/security-rules.md"},
	}
	if rt == nil || rt.VersionSv == nil {
		overview.CurrentVersion = currentVersion()
		return overview
	}
	report := rt.VersionSv.Report(ctx)
	overview.CurrentVersion = report.Current
	overview.LatestVersion = report.Latest
	overview.UpdateAvailable = report.UpdateAvailable
	overview.InstalledViaHomebrew = report.InstalledViaHomebrew
	if report.UpdateAvailable && report.InstalledViaHomebrew {
		overview.UpgradeCommand = "brew update && brew upgrade GustavoGutierrez/celador/celador"
	} else if report.UpdateAvailable {
		overview.UpgradeCommand = "brew update && brew upgrade GustavoGutierrez/celador/celador"
	}
	return overview
}
