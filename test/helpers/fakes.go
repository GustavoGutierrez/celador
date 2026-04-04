package helpers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type StubClock struct{ Value time.Time }

func (c StubClock) Now() time.Time { return c.Value }

type StubUI struct {
	ConfirmResult bool
	ConfirmCalls  int
	ScanCalls     int
	OverviewCalls int
	LastOverview  shared.Overview
	LastScan      shared.ScanResult
	LastScanOpts  shared.ScanRenderOptions
	Interactive   bool
	Output        strings.Builder
}

func (ui *StubUI) Confirm(context.Context, string) (bool, error) {
	ui.ConfirmCalls++
	return ui.ConfirmResult, nil
}
func (ui *StubUI) RenderScan(_ context.Context, result shared.ScanResult, options shared.ScanRenderOptions) error {
	ui.ScanCalls++
	ui.LastScan = result
	ui.LastScanOpts = options
	ui.Output.WriteString(result.Fingerprint)
	return nil
}
func (ui *StubUI) RenderFixPlan(_ context.Context, plan shared.FixPlan) error {
	ui.Output.WriteString(plan.DryRunDiff)
	return nil
}
func (ui *StubUI) RenderInstallAssessment(_ context.Context, assessment shared.InstallAssessment) error {
	ui.Output.WriteString(assessment.Package)
	return nil
}
func (ui *StubUI) RenderOverview(_ context.Context, overview shared.Overview, interactive bool) error {
	ui.OverviewCalls++
	ui.LastOverview = overview
	ui.Interactive = interactive
	ui.Output.WriteString(overview.Developer)
	ui.Output.WriteString("\n")
	ui.Output.WriteString(overview.GitHubProfile)
	ui.Output.WriteString("\n")
	ui.Output.WriteString(overview.CurrentVersion)
	if overview.LatestVersion != "" {
		ui.Output.WriteString("\n")
		ui.Output.WriteString(overview.LatestVersion)
	}
	return nil
}
func (ui *StubUI) Printf(format string, args ...any) {
	_, _ = ui.Output.WriteString(fmt.Sprintf(format, args...))
}

type StubRuleLoader struct {
	Rules   []shared.RuleConfig
	Version string
}

func (s StubRuleLoader) Load(context.Context, string) ([]shared.RuleConfig, string, error) {
	return s.Rules, s.Version, nil
}

type StubRuleEvaluator struct{ Findings []shared.Finding }

func (s StubRuleEvaluator) Evaluate(context.Context, shared.Workspace, []shared.RuleConfig) ([]shared.Finding, error) {
	return s.Findings, nil
}

type StubOSV struct {
	Findings []shared.Finding
	Err      error
	Calls    int
}

func (s *StubOSV) Query(context.Context, []shared.Dependency) ([]shared.Finding, error) {
	s.Calls++
	return s.Findings, s.Err
}

type StubMetadata struct {
	Assessment  shared.InstallAssessment
	LastManager shared.PackageManager
}

func (s *StubMetadata) InspectPackage(_ context.Context, manager shared.PackageManager, _ string) (shared.InstallAssessment, error) {
	s.LastManager = manager
	return s.Assessment, nil
}

type StubPM struct {
	Calls      [][]string
	Workspaces []shared.Workspace
}

func (pm *StubPM) Install(_ context.Context, workspace shared.Workspace, args []string) error {
	pm.Calls = append(pm.Calls, args)
	pm.Workspaces = append(pm.Workspaces, workspace)
	return nil
}

type StubDetector struct{ Workspace shared.Workspace }

func (d StubDetector) Detect(context.Context, string, bool, bool) (shared.Workspace, error) {
	return d.Workspace, nil
}

type StubNodeVersionDetector struct {
	Version string
	OK      bool
	Calls   int
}

func (d *StubNodeVersionDetector) Detect(context.Context, string) (string, bool) {
	d.Calls++
	return d.Version, d.OK
}

type StubIgnore struct{ Rules []shared.IgnoreRule }

func (s StubIgnore) Load(context.Context, string) ([]shared.IgnoreRule, error) { return s.Rules, nil }

type StubCache struct {
	Scan        map[string]shared.ScanResult
	OSV         map[string][]shared.Finding
	OSVExpiry   map[string]time.Time
	PutOSVCalls int
}

func (c *StubCache) GetScan(_ context.Context, key string) (shared.ScanResult, bool, error) {
	if c.Scan == nil {
		return shared.ScanResult{}, false, nil
	}
	result, ok := c.Scan[key]
	return result, ok, nil
}
func (c *StubCache) PutScan(_ context.Context, key string, result shared.ScanResult) error {
	if c.Scan == nil {
		c.Scan = map[string]shared.ScanResult{}
	}
	c.Scan[key] = result
	return nil
}
func (c *StubCache) getOSV(key string) ([]shared.Finding, bool, time.Time, error) {
	if c.OSV == nil {
		return nil, false, time.Time{}, nil
	}
	findings, ok := c.OSV[key]
	if !ok {
		return nil, false, time.Time{}, nil
	}
	return findings, true, c.OSVExpiry[key], nil
}

func (c *StubCache) GetOSV(_ context.Context, key string) ([]shared.Finding, bool, time.Time, error) {
	return c.getOSV(key)
}

func (c *StubCache) PutOSV(_ context.Context, key string, findings []shared.Finding, ttl time.Duration) error {
	if c.OSV == nil {
		c.OSV = map[string][]shared.Finding{}
	}
	if c.OSVExpiry == nil {
		c.OSVExpiry = map[string]time.Time{}
	}
	c.OSV[key] = findings
	c.OSVExpiry[key] = time.Now().Add(ttl)
	c.PutOSVCalls++
	return nil
}

type StubPatchWriter struct{ Applied shared.FixPlan }

func (p *StubPatchWriter) Preview(context.Context, shared.Workspace, shared.FixPlan) (string, error) {
	return "", nil
}
func (p *StubPatchWriter) Apply(_ context.Context, _ shared.Workspace, plan shared.FixPlan) error {
	p.Applied = plan
	return nil
}

var _ ports.PromptUI = (*StubUI)(nil)
var _ ports.NodeVersionDetector = (*StubNodeVersionDetector)(nil)
