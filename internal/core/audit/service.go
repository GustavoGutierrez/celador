package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type clock interface{ Now() time.Time }

type Service struct {
	detector ports.WorkspaceDetector
	ignore   ports.IgnoreStore
	loader   ports.RuleLoader
	eval     ports.RuleEvaluator
	osv      ports.VulnerabilitySource
	cache    ports.ScanCache
	clock    clock
	osvTTL   time.Duration
	parsers  []ports.LockfileParser
}

func NewService(detector ports.WorkspaceDetector, ignore ports.IgnoreStore, loader ports.RuleLoader, eval ports.RuleEvaluator, osv ports.VulnerabilitySource, cache ports.ScanCache, clock clock, osvTTL time.Duration, parsers []ports.LockfileParser) *Service {
	return &Service{detector: detector, ignore: ignore, loader: loader, eval: eval, osv: osv, cache: cache, clock: clock, osvTTL: osvTTL, parsers: parsers}
}

func (s *Service) Run(ctx context.Context, root string, tty bool, ci bool) (shared.ScanResult, error) {
	ws, err := s.detector.Detect(ctx, root, tty, ci)
	if err != nil {
		return shared.ScanResult{}, err
	}
	if len(ws.Lockfiles) == 0 {
		return shared.ScanResult{}, fmt.Errorf("no supported lockfile found (package-lock.json, pnpm-lock.yaml, bun.lock, deno.lock)")
	}
	lockText, err := readLockfileBody(ctx, parserFS(s.parsers), ws.Lockfiles)
	if err != nil {
		return shared.ScanResult{}, err
	}
	ignores, err := s.ignore.Load(ctx, root)
	if err != nil {
		return shared.ScanResult{}, err
	}
	rules, ruleVersion, err := s.loader.Load(ctx, root)
	if err != nil {
		return shared.ScanResult{}, err
	}
	fingerprint := Fingerprint(lockText, fmt.Sprint(ignores), ruleVersion)
	scanCached, scanCacheHit, err := s.cache.GetScan(ctx, fingerprint)
	if err != nil {
		return shared.ScanResult{}, err
	}
	deps, err := s.parseDependencies(ctx, ws)
	if err != nil {
		return shared.ScanResult{}, err
	}
	osvKey, err := cacheKeyForDependencies(deps)
	if err != nil {
		return shared.ScanResult{}, err
	}
	findings, cacheExpired, fromOSVCache, err := s.loadOSVFindings(ctx, osvKey)
	if err != nil {
		return shared.ScanResult{}, err
	}
	if !fromOSVCache {
		findings, err = s.osv.Query(ctx, deps)
		if err != nil {
			if scanCacheHit {
				scanCached.FromCache = true
				scanCached.OfflineFallback = true
				return scanCached, nil
			}
			if cacheExpired {
				fromOSVCache = true
			} else {
				return shared.ScanResult{}, fmt.Errorf("query osv: %w", err)
			}
		} else if err := s.cache.PutOSV(ctx, osvKey, findings, s.osvTTL); err != nil {
			return shared.ScanResult{}, err
		}
	}
	ruleFindings, err := s.eval.Evaluate(ctx, ws, rules)
	if err != nil {
		return shared.ScanResult{}, err
	}
	findings = append(findings, ruleFindings...)
	filtered, ignoredCount := applyIgnores(findings, ignores, s.clock.Now())
	sortFindings(filtered)
	result := shared.ScanResult{Workspace: ws, Dependencies: deps, Findings: filtered, IgnoredCount: ignoredCount, Fingerprint: fingerprint, RuleVersion: ruleVersion, GeneratedAt: s.clock.Now()}
	if fromOSVCache {
		result.FromCache = true
	}
	if cacheExpired {
		result.OfflineFallback = true
	}
	if err := s.cache.PutScan(ctx, fingerprint, result); err != nil {
		return shared.ScanResult{}, err
	}
	return result, nil
}

func (s *Service) loadOSVFindings(ctx context.Context, key string) ([]shared.Finding, bool, bool, error) {
	findings, ok, expiresAt, err := s.cache.GetOSV(ctx, key)
	if err != nil {
		return nil, false, false, err
	}
	if !ok {
		return nil, false, false, nil
	}
	if expiresAt.After(s.clock.Now()) {
		return findings, false, true, nil
	}
	return findings, true, false, nil
}

func cacheKeyForDependencies(deps []shared.Dependency) (string, error) {
	body, err := json.Marshal(deps)
	if err != nil {
		return "", fmt.Errorf("marshal dependency cache key: %w", err)
	}
	return Fingerprint(string(body)), nil
}

func (s *Service) parseDependencies(ctx context.Context, ws shared.Workspace) ([]shared.Dependency, error) {
	var deps []shared.Dependency
	for _, path := range ws.Lockfiles {
		parsed := false
		for _, parser := range s.parsers {
			if !parser.Supports(path) {
				continue
			}
			items, err := parser.Parse(ctx, ws, path)
			if err != nil {
				return nil, err
			}
			deps = append(deps, items...)
			parsed = true
			break
		}
		if !parsed {
			return nil, fmt.Errorf("no parser for %s", path)
		}
	}
	sortDeps(deps)
	return deps, nil
}

func applyIgnores(findings []shared.Finding, rules []shared.IgnoreRule, now time.Time) ([]shared.Finding, int) {
	selectors := map[string]shared.IgnoreRule{}
	for _, rule := range rules {
		selectors[rule.Selector] = rule
	}
	filtered := make([]shared.Finding, 0, len(findings))
	ignoredCount := 0
	for _, finding := range findings {
		rule, ok := selectors[finding.ID]
		if !ok {
			rule, ok = selectors[finding.Target]
		}
		if ok && (rule.ExpiresAt == nil || rule.ExpiresAt.After(now)) {
			ignoredCount++
			continue
		}
		filtered = append(filtered, finding)
	}
	return filtered, ignoredCount
}

func sortFindings(findings []shared.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		left := strings.Join([]string{string(findings[i].Source), string(findings[i].Severity), findings[i].ID, findings[i].Target}, "|")
		right := strings.Join([]string{string(findings[j].Source), string(findings[j].Severity), findings[j].ID, findings[j].Target}, "|")
		return left < right
	})
}

type fsProvider interface{ FS() ports.FileSystem }

func parserFS(parsers []ports.LockfileParser) ports.FileSystem {
	for _, parser := range parsers {
		if provider, ok := parser.(interface{ FileSystem() ports.FileSystem }); ok {
			return provider.FileSystem()
		}
	}
	return nil
}
