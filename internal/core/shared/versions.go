package shared

import (
	"strings"

	"golang.org/x/mod/semver"
)

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
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	if !semver.IsValid(value) {
		return ""
	}
	return value
}
