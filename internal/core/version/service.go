package version

import (
	"context"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/ports"
)

type Report struct {
	Current              string
	Latest               string
	UpdateAvailable      bool
	InstalledViaHomebrew bool
}

type Service struct {
	current        string
	checker        ports.ReleaseVersionSource
	executablePath string
}

func NewService(current string, checker ports.ReleaseVersionSource, executablePath string) *Service {
	return &Service{current: normalizeVersion(current), checker: checker, executablePath: executablePath}
}

func (s *Service) Current() string {
	return s.current
}

func (s *Service) Report(ctx context.Context) Report {
	report := Report{
		Current:              s.current,
		InstalledViaHomebrew: strings.Contains(s.executablePath, "/Cellar/celador/"),
	}
	if s.checker == nil {
		return report
	}
	latest, err := s.checker.Latest(ctx)
	if err != nil {
		return report
	}
	latest = normalizeVersion(latest)
	if latest == "" {
		return report
	}
	report.Latest = latest
	report.UpdateAvailable = compareVersions(latest, report.Current) > 0
	return report
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "(devel)" {
		return "dev"
	}
	if strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}

func compareVersions(left string, right string) int {
	leftParts, leftOK := parseVersion(left)
	rightParts, rightOK := parseVersion(right)
	if !leftOK || !rightOK {
		return 0
	}
	for i := range leftParts {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func parseVersion(value string) ([3]int, bool) {
	var parts [3]int
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "v")
	segments := strings.Split(trimmed, ".")
	if len(segments) != 3 {
		return parts, false
	}
	for i, segment := range segments {
		for _, ch := range segment {
			if ch < '0' || ch > '9' {
				return parts, false
			}
			parts[i] = parts[i]*10 + int(ch-'0')
		}
	}
	return parts, true
}
