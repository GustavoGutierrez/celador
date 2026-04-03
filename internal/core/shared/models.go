package shared

import "time"

type PackageManager string

const (
	PackageManagerUnknown PackageManager = "unknown"
	PackageManagerNPM     PackageManager = "npm"
	PackageManagerPNPM    PackageManager = "pnpm"
	PackageManagerBun     PackageManager = "bun"
	PackageManagerDeno    PackageManager = "deno"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type FindingSource string

const (
	FindingSourceOSV       FindingSource = "osv"
	FindingSourceRule      FindingSource = "rule"
	FindingSourceHeuristic FindingSource = "heuristic"
)

type Workspace struct {
	Root           string
	PackageManager PackageManager
	Frameworks     []string
	Lockfiles      []string
	ManifestPath   string
	ConfigPath     string
	TTY            bool
	CI             bool
}

type Dependency struct {
	Name       string
	Version    string
	Ecosystem  string
	Direct     bool
	Manifest   string
	Transitive bool
}

type FindingLocation struct {
	Path string
	Line int
}

type Finding struct {
	ID          string
	Source      FindingSource
	Severity    Severity
	Target      string
	Summary     string
	PackageName string
	RuleID      string
	Locations   []FindingLocation
	FixVersion  string
	Fixable     bool
	Ignored     bool
}

type IgnoreRule struct {
	Selector  string
	Reason    string
	ExpiresAt *time.Time
}

type ScanResult struct {
	Workspace       Workspace
	Dependencies    []Dependency
	Findings        []Finding
	IgnoredCount    int
	FromCache       bool
	OfflineFallback bool
	Fingerprint     string
	RuleVersion     string
	GeneratedAt     time.Time
}

type FixOperation struct {
	File            string
	PackageName     string
	CurrentVersion  string
	ProposedVersion string
	Strategy        string
	Diff            string
	BlastRadius     []string
	RequiresInstall bool
}

type FixPlan struct {
	Operations []FixOperation
	Summary    string
	DryRunDiff string
}

type RulePack struct {
	Version string       `yaml:"version"`
	Rules   []RuleConfig `yaml:"rules"`
}

type RuleConfig struct {
	ID          string   `yaml:"id"`
	Frameworks  []string `yaml:"frameworks"`
	File        string   `yaml:"file"`
	Glob        string   `yaml:"glob"`
	MustContain string   `yaml:"mustContain"`
	MustNotFind string   `yaml:"mustNotFind"`
	Severity    Severity `yaml:"severity"`
	Summary     string   `yaml:"summary"`
	Remediation string   `yaml:"remediation"`
}

type InstallAssessment struct {
	Package       string
	Risk          Severity
	Reasons       []string
	Unknown       bool
	TarballURL    string
	Manager       PackageManager
	ShouldPrompt  bool
	SuggestedArgs []string
}
