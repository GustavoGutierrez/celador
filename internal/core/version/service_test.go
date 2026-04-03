package version

import (
	"context"
	"errors"
	"testing"
)

func TestReportMarksNewerReleaseAndHomebrewInstall(t *testing.T) {
	t.Parallel()

	svc := NewService("v1.2.3", stubReleaseSource{latest: "v1.3.0"}, "/opt/homebrew/Cellar/celador/1.2.3/bin/celador")
	report := svc.Report(context.Background())

	if !report.UpdateAvailable {
		t.Fatalf("expected newer release to be reported")
	}
	if report.Latest != "v1.3.0" {
		t.Fatalf("expected latest version to be preserved, got %q", report.Latest)
	}
	if !report.InstalledViaHomebrew {
		t.Fatalf("expected Homebrew install detection")
	}
}

func TestReportFallsBackToCurrentVersionWhenCheckFails(t *testing.T) {
	t.Parallel()

	svc := NewService("v1.2.3", stubReleaseSource{err: errors.New("boom")}, "/usr/local/bin/celador")
	report := svc.Report(context.Background())

	if report.Current != "v1.2.3" {
		t.Fatalf("expected current version in report, got %q", report.Current)
	}
	if report.Latest != "" {
		t.Fatalf("expected no latest version on failure, got %q", report.Latest)
	}
	if report.UpdateAvailable {
		t.Fatalf("did not expect update availability when check fails")
	}
}

type stubReleaseSource struct {
	latest string
	err    error
}

func (s stubReleaseSource) Latest(context.Context) (string, error) {
	return s.latest, s.err
}
