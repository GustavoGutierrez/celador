package output

import (
	"encoding/json"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// SARIFReport is the top-level SARIF v2.1.0 structure.
type SARIFReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single run of the tool.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool describes the tool that produced the results.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver contains driver metadata.
type SARIFDriver struct {
	Name           string `json:"name"`
	Version        string `json:"version,omitempty"`
	InformationURI string `json:"informationUri"`
}

// SARIFResult is a single finding.
type SARIFResult struct {
	RuleID    string         `json:"ruleId"`
	Level     string         `json:"level"`
	Message   SARIFMessage   `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

// SARIFMessage wraps the result text.
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFLocation describes where the finding applies.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation points to the artifact.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

// SARIFArtifactLocation holds the file URI.
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// severityToSARIFLevel maps Celador severity to SARIF level.
func severityToSARIFLevel(severity shared.Severity) string {
	switch severity {
	case shared.SeverityCritical, shared.SeverityHigh:
		return "error"
	case shared.SeverityMedium:
		return "warning"
	case shared.SeverityLow:
		return "note"
	default:
		return "warning"
	}
}

// ToSARIF converts scan findings and rules into a SARIF v2.1.0 report.
func ToSARIF(findings []shared.Finding, version string) []byte {
	report := SARIFReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{{
			Tool: SARIFTool{
				Driver: SARIFDriver{
					Name:           "Celador",
					Version:        version,
					InformationURI: "https://github.com/GustavoGutierrez/celador",
				},
			},
			Results: make([]SARIFResult, 0, len(findings)),
		}},
	}

	for _, f := range findings {
		result := SARIFResult{
			RuleID:  f.ID,
			Level:   severityToSARIFLevel(f.Severity),
			Message: SARIFMessage{Text: f.Summary},
			Locations: []SARIFLocation{{
				PhysicalLocation: SARIFPhysicalLocation{
					ArtifactLocation: SARIFArtifactLocation{
						URI: f.Target,
					},
				},
			}},
		}
		report.Runs[0].Results = append(report.Runs[0].Results, result)
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	return out
}
