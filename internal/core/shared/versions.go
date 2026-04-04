package shared

import (
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var semverTokenPattern = regexp.MustCompile(`v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?`)

func CompareVersions(left string, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	leftSemver := NormalizeVersion(left)
	rightSemver := NormalizeVersion(right)
	if leftSemver != "" && rightSemver != "" {
		return semver.Compare(leftSemver, rightSemver)
	}
	return strings.Compare(left, right)
}

func NormalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if normalized := ensureVersionPrefix(value); semver.IsValid(normalized) {
		return normalized
	}
	match := semverTokenPattern.FindString(value)
	if match == "" {
		return ""
	}
	normalized := ensureVersionPrefix(match)
	if !semver.IsValid(normalized) {
		return ""
	}
	return normalized
}

func IsPrereleaseVersion(value string) bool {
	normalized := NormalizeVersion(value)
	if normalized == "" {
		return false
	}
	return semver.Prerelease(normalized) != ""
}

func VersionMajor(value string) string {
	normalized := NormalizeVersion(value)
	if normalized == "" {
		return ""
	}
	return semver.Major(normalized)
}

func ensureVersionPrefix(value string) string {
	if strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}
