package sandbox

import (
	"fmt"
	"strings"
	"time"
)

// Result contains the outcome of a sandbox execution.
type Result struct {
	Engine          string
	Executed        bool
	Duration        time.Duration
	TimedOut        bool
	NetworkAttempts []string
	EnvReads          []string
	FileReads         []string
	FileWrites        []string
	DynamicExec       []string
	TimerCreations    []string
	Warnings          []string
	SuspiciousScore   int
	Verdict           string
}

// RunOptions configures the sandbox execution.
type RunOptions struct {
	Timeout       time.Duration
	Strict        bool
	Offline       bool
	EntryStrategy string // "auto", "cjs", "esm"
}

// signal definitions for behavior detection.
const (
	signalNetwork    = "network"
	signalEnvRead    = "env_read"
	signalFileRead   = "file_read"
	signalFileWrite  = "file_write"
	signalDynamicExec = "dynamic_exec"
	signalTimer       = "timer"
	signalDecodeChain = "decode_chain"
	signalFingerprint = "fingerprint"
)

// scoreWeights defines the suspiciousness weight per signal type.
var scoreWeights = map[string]int{
	signalNetwork:     30,
	signalEnvRead:     15,
	signalFileRead:    5,
	signalFileWrite:   10,
	signalDynamicExec: 20,
	signalTimer:       10,
	signalDecodeChain: 25,
	signalFingerprint: 35,
}

// addSignal records a suspicious behavior and updates the score.
func (r *Result) addSignal(signalType, detail string) {
	r.Warnings = append(r.Warnings, fmt.Sprintf("%s: %s", signalType, detail))
	r.SuspiciousScore += scoreWeights[signalType]
	if r.SuspiciousScore > 100 {
		r.SuspiciousScore = 100
	}
}

// computeVerdict determines the final verdict based on the score.
func (r *Result) computeVerdict() {
	switch {
	case r.SuspiciousScore >= 70:
		r.Verdict = "high_risk"
	case r.SuspiciousScore >= 40:
		r.Verdict = "medium_risk"
	case r.SuspiciousScore >= 15:
		r.Verdict = "low_risk"
	default:
		r.Verdict = "clean"
	}
}

// isKnownBenign checks if the package name is a well-known legitimate package
// to reduce false positive severity (not score).
func isKnownBenign(name string) bool {
	benign := []string{
		"react", "react-dom", "lodash", "express", "axios", "typescript",
		"chalk", "debug", "async", "commander", "moment", "uuid",
		"webpack", "babel", "eslint", "jest", "mocha", "underscore",
		"jquery", "vue", "angular", "next", "tailwindcss", "dotenv",
		"cors", "body-parser", "morgan", "fs-extra", "glob", "semver",
	}
	lower := strings.ToLower(name)
	for _, b := range benign {
		if lower == b {
			return true
		}
	}
	return false
}
