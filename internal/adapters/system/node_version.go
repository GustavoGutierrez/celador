package system

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

var nodeVersionPattern = regexp.MustCompile(`^v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)$`)

type NodeVersionDetector struct{}

func NewNodeVersionDetector() *NodeVersionDetector {
	return &NodeVersionDetector{}
}

func (d *NodeVersionDetector) Detect(ctx context.Context, root string) (string, bool) {
	cmd := exec.CommandContext(ctx, "node", "--version")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	match := nodeVersionPattern.FindStringSubmatch(strings.TrimSpace(string(output)))
	if len(match) != 2 {
		return "", false
	}
	return match[1], true
}
