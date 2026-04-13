# Test Coverage Report

**Last updated:** April 13, 2026  
**Overall coverage:** 71.3% of statements  
**Total test functions:** 95+  
**Test files:** 20

---

## Coverage by Package

| Package | Coverage | Status |
|---------|----------|--------|
| `internal/adapters/pm` | **100.0%** | ✅ Complete |
| `internal/adapters/rules` | **100.0%** | ✅ Complete |
| `internal/adapters/system` | **100.0%** | ✅ Complete |
| `internal/core/fix` | **89.1%** | ✅ Strong |
| `internal/adapters/releases` | **93.8%** | ✅ Strong |
| `internal/core/audit` | **87.2%** | ✅ Strong |
| `internal/adapters/fs` | **85.5%** | ✅ Strong |
| `internal/adapters/osv` | **78.6%** | ⚠️ Good |
| `internal/core/shared` | **76.7%** | ⚠️ Good |
| `internal/core/version` | **75.6%** | ⚠️ Good |
| `internal/core/workspace` | **73.6%** | ⚠️ Good |
| `internal/adapters/tui` | **69.2%** | ⚠️ Moderate |
| `internal/adapters/cache` | **55.9%** | ⚠️ Moderate |
| `internal/core/install` | **55.6%** | ⚠️ Moderate |
| `internal/app` | **57.9%** | ⚠️ Moderate |
| `cmd/celador` | **0.0%** | — Entrypoint only |

---

## Security-Critical Packages

These packages handle vulnerability detection and install-time risk assessment:

### OSV Client — 78.6%

| Function | Coverage |
|----------|----------|
| `Query` | ✅ Tested (empty deps, no vulns, multiple vulns, hydration, errors) |
| `parseOSVSeverity` | ✅ Tested (critical, high, medium, low, empty, non-CVSS) |
| `fetchAdvisory` | ✅ Tested (success, HTTP error, hydration failure) |
| `fixedVersionForPackage` | ✅ Tested (93.8%) |
| `fixedVersionForRange` | ⚠️ 71.4% |
| `summarizeVulnerability` | ✅ Tested (100%) |
| `advisoryNeedsHydration` | ✅ Tested (100%) |

### Registry Inspector — ~62%

| Function | Coverage |
|----------|----------|
| `InspectPackage` | ⚠️ 60% (happy paths + errors tested) |
| `inspectTarball` | ⚠️ 64.1% (package.json parsing + pattern detection tested) |
| `isHexLike` | ✅ Tested (100%) |
| `maxSeverity` | ✅ Tested (100%) |

### Patch Writer — ~90%

| Function | Coverage |
|----------|----------|
| `Apply` | ✅ 90.3% (bump, override, missing manifest, invalid JSON) |
| `Preview` | ✅ 100% |
| `RenderPlanDiff` | ✅ 91.7% |

---

## Test Files

### New test files added this session

| File | Tests | Purpose |
|------|-------|---------|
| `internal/adapters/osv/registry_inspector_test.go` | 8 | Postinstall, 404, network failure, hex, severity |
| `internal/adapters/osv/registry_inspector_axios_test.go` | 5 | **axios@1.14.1 malware detection**, clean false-positive, C2 payload, injected dep |
| `internal/adapters/osv/client_test.go` | 22 | CVSS parsing, HTTP errors, hydration, summary, fix versions |
| `internal/adapters/fs/patch_writer_test.go` | 8 | Bump deps, overrides, invalid JSON, missing manifest, diff |
| `internal/adapters/fs/osfs_test.go` | 6 | Read/write, stat, mkdir, glob, walk, exec root |
| `internal/adapters/fs/ignore_store_test.go` | 4 | Rule parsing, missing file, comments, invalid dates |
| `internal/adapters/fs/templates_test.go` | 3 | Read error propagation, create when missing, replace block |
| `internal/adapters/pm/executor_test.go` | 8 | NPM, PNPM, Bun, unknown manager, binary missing, context cancel, stderr |
| `internal/adapters/rules/loader_test.go` | 6 | Valid files, empty dir, invalid YAML, glob error, read error, multi-file |
| `internal/adapters/releases/github_test.go` | 8 | Release found, 404, network, invalid JSON, HTTP 500, defaults, User-Agent |
| `internal/adapters/system/node_version_test.go` | 5 | Valid output, no node, unexpected format, cancel, regex |
| `internal/core/audit/rules_test.go` | 7 | Empty rules, SQL injection, Tailwind dynamic, framework mismatch |
| `internal/core/audit/parsers_test.go` | 6 | Supports(), FileSystem(), read error, invalid JSON/YAML |
| `internal/core/audit/service_test.go` | 14 | Ignore, cache TTL, offline fallback, OSV query, cache expiry, parse deps, sort |
| `internal/core/workspace/service_errors_test.go` | 5 | Read error propagation for ignore files, npmrc, bunfig, deno.json |

### Existing test files (maintained)

| File | Tests |
|------|-------|
| `internal/adapters/cache/file_cache_test.go` | Existing |
| `internal/adapters/tui/terminal_test.go` | Existing |
| `internal/app/commands_test.go` | Existing |
| `internal/core/fix/service_test.go` | Existing |
| `internal/core/install/service_test.go` | Existing |
| `internal/core/shared/scan_findings_test.go` | Existing |
| `internal/core/version/service_test.go` | Existing |
| `internal/core/workspace/service_test.go` | Existing |
| `test/e2e/celador_e2e_test.go` | Existing |

---

## Coverage History

| Date | Overall Coverage | Change |
|------|-----------------|--------|
| Before analysis | **58.3%** | Baseline |
| After Phase 1 (critical fixes) | **67.1%** | +8.8 pts |
| After Phase 2 (remaining adapters) | **68.1%** | +1.0 pts |
| After Phase 3 (releases, rules, audit) | **71.3%** | +3.2 pts |
| **Total improvement** | **+13.0 pts** | 58.3% → 71.3% |

---

## Packages at 100% Coverage

- ✅ `internal/adapters/pm` — Package manager executor (NPM, PNPM, Bun)
- ✅ `internal/adapters/rules` — YAML rule loader
- ✅ `internal/adapters/system` — Node version detector

---

## Packages Below Target (<80%)

These packages have room for improvement but are not blocking release:

| Package | Coverage | Next steps |
|---------|----------|------------|
| `internal/adapters/tui` | 69.2% | TUI components are hard to unit test; consider integration tests |
| `internal/adapters/cache` | 55.9% | `PutOSV` and error paths in `GetOSV` need tests |
| `internal/core/install` | 55.6% | `CommandForManager` and `Execute` error paths |
| `internal/app` | 57.9% | Bootstrap and command wiring tests |
| `cmd/celador` | 0.0% | Entry point — acceptable for CLI apps |

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Run specific package
go test ./internal/adapters/osv -v -cover
```

## CI Coverage Gate (Recommended)

Add to `.github/workflows/release.yml`:

```yaml
- name: Enforce minimum test coverage
  run: |
    go test ./... -coverprofile=coverage.out
    TOTAL=$(go tool cover -func=coverage.out | grep "total:" | grep -oP '\d+\.\d+')
    echo "Coverage: ${TOTAL}%"
```
