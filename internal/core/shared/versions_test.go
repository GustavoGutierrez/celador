package shared

import "testing"

func TestNormalizeVersionExtractsSemverFromRanges(t *testing.T) {
	t.Parallel()

	if got := NormalizeVersion("^7.0.6"); got != "v7.0.6" {
		t.Fatalf("expected ranged version to normalize to v7.0.6, got %q", got)
	}
}

func TestIsPrereleaseVersionDetectsPreReleaseTargets(t *testing.T) {
	t.Parallel()

	if !IsPrereleaseVersion("15.6.0-canary.61") {
		t.Fatalf("expected canary target to be treated as prerelease")
	}
}

func TestVersionMajorNormalizesManifestRanges(t *testing.T) {
	t.Parallel()

	if got := VersionMajor("^3.0.3"); got != "v3" {
		t.Fatalf("expected major v3, got %q", got)
	}
}
