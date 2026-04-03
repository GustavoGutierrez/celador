package app

import (
	"fmt"

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
	}

	cmd.PersistentFlags().Bool("no-interactive", false, "Disable prompts and TTY flows")
	cmd.AddCommand(newInitCommand(rt), newScanCommand(rt), newFixCommand(rt), newInstallCommand(rt))
	return cmd
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
			rt.UI.Printf("Initialized %s (%s)\n", res.Workspace.Root, res.Workspace.PackageManager)
			for _, msg := range res.Messages {
				rt.UI.Printf("- %s\n", msg)
			}
			return nil
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
			tty, ci, _, err := commandInteractivity(cmd, rt)
			if err != nil {
				return err
			}
			result, err := rt.ScanSvc.Run(cmd.Context(), rt.Root, tty, ci)
			if err != nil {
				return err
			}
			if err := rt.UI.RenderScan(cmd.Context(), result); err != nil {
				return err
			}
			if len(result.Findings) > 0 {
				return NewExitError(3, "%d findings detected", len(result.Findings))
			}
			return nil
		},
	}
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
			if err := rt.UI.RenderInstallAssessment(cmd.Context(), assessment); err != nil {
				return err
			}
			if assessment.ShouldPrompt && !yes {
				if !tty || ci || noInteractive {
					return NewExitError(2, "install risk requires approval; rerun with --yes or in a TTY")
				}
				confirmed, err := rt.UI.Confirm(cmd.Context(), fmt.Sprintf("Continue installing %s?", assessment.Package))
				if err != nil {
					return err
				}
				if !confirmed {
					return NewExitError(2, "installation cancelled")
				}
			}
			return rt.InstallSv.Execute(cmd.Context(), rt.Root, tty, ci, args)
		},
	}
	cmd.Flags().Bool("yes", false, "Continue on risky findings without prompting")
	return cmd
}
