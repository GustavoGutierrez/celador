# Celador — Test Coverage Analysis & Improvement Plan

**Date:** April 13, 2026  
**Current Coverage:** 58.3%  
**Target Coverage:** 80%+ (industry standard for security-focused tools)

---

## Executive Summary

| Metric | Current | Target | Gap |
|--------|---------|--------|-----|
| **Statement Coverage** | 58.3% | 80%+ | **−21.7%** |
| **Uncovered Functions** | 84 | ~20 | **64+ functions need tests** |
| **Estimated Tests Needed** | — | — | **~45 new test functions** |
| **Packages with 0% coverage** | 7 | 0 | **7 packages** |
| **Packages below 60%** | 4 | 0 | **4 packages** |

---

## Current Coverage by Package

| Package | Coverage | Status | Risk Level |
|---------|----------|--------|------------|
| `internal/core/fix` | **89.1%** | ✅ Good | Low |
| `internal/core/shared` | **76.7%** | ⚠️ Near target | Low |
| `internal/core/version` | **75.6%** | ⚠️ Near target | Low |
| `internal/core/workspace` | **73.6%** | ⚠️ Near target | Medium |
| `internal/core/audit` | **63.8%** | ⚠️ Below target | Medium |
| `internal/adapters/tui` | **69.2%** | ⚠️ Below target | Low |
| `internal/app` | **57.9%** | ❌ Below target | Medium |
| `internal/adapters/cache` | **55.9%** | ❌ Below target | Medium |
| `internal/core/install` | **55.6%** | ❌ Below target | Medium |
| `internal/adapters/osv` | **44.8%** | ❌ Critical gap | **High** |
| `internal/adapters/fs` | **15.2%** | ❌ Critical gap | **High** |
| `internal/adapters/pm` | **0.0%** | ❌ No tests | **High** |
| `internal/adapters/releases` | **0.0%** | ❌ No tests | Medium |
| `internal/adapters/rules` | **0.0%** | ❌ No tests | Medium |
| `internal/adapters/system` | **0.0%** | ❌ No tests | Low |
| `cmd/celador` | **0.0%** | ❌ No tests | Low |

---

## Priority 1: Critical Gaps (Security-Impact)

### 1.1 OSV Registry Inspector — 0% coverage
**File:** `internal/adapters/osv/registry_inspector.go`  
**Functions uncovered:** `NewRegistryInspector`, `InspectPackage`, `inspectTarball`, `isHexLike`, `maxSeverity`  
**Estimated tests needed:** **8-10**  
**Why critical:** This is the install-time risk assessment engine. Untested = users may get false security reports.

| Test | What it verifies |
|------|-----------------|
| `TestInspectPackage_HighRiskPostinstall` | Detects risky postinstall scripts |
| `TestInspectPackage_ProcessEnvWithNetwork` | Detects env+network exfiltration pattern |
| `TestInspectPackage_HexEncodedStrings` | Detects obfuscated strings |
| `TestInspectPackage_CleanPackage` | Reports low risk for clean packages |
| `TestInspectPackage_DenoV1Unsupported` | Handles Deno v1 gracefully |
| `TestInspectPackage_NetworkFailure` | Handles registry timeouts |
| `TestInspectPackage_InvalidTarball` | Handles corrupted tarball downloads |
| `TestInspectPackage_404NotFound` | Handles missing packages |
| `TestIsHexLike_DetectsLongHexStrings` | Hex detection threshold |
| `TestMaxSeverity_ReturnsHighest` | Severity aggregation logic |

---

### 1.2 OSV Client — 44.8% coverage
**File:** `internal/adapters/osv/client.go`  
**Functions uncovered/partially covered:** `Query` (error paths), `fetchAdvisory`, `parseOSVSeverity`  
**Estimated tests needed:** **6-8**

| Test | What it verifies |
|------|-----------------|
| `TestClientQuery_NetworkError` | Handles OSV API downtime |
| `TestClientQuery_HTTPError` | Handles 4xx/5xx responses |
| `TestClientQuery_EmptyResults` | No vulnerabilities found |
| `TestClientQuery_MultipleVulns` | Multiple CVEs per package |
| `TestParseOSVSeverity_CVSSv3_Critical` | Score ≥9.0 → critical |
| `TestParseOSVSeverity_CVSSv3_High` | Score ≥7.0 → high |
| `TestParseOSVSeverity_DefaultMedium` | No severity data → medium |
| `TestClientQuery_EmptyDepsSkipped` | No network call for empty deps |

---

### 1.3 Patch Writer — 0% coverage
**File:** `internal/adapters/fs/patch_writer.go`  
**Functions uncovered:** `NewPatchWriter`, `Preview`, `Apply`, `ensureMap`, `RenderPlanDiff`  
**Estimated tests needed:** **6-7**  
**Why critical:** Applies fixes to `package.json`. Bugs here can corrupt user manifests.

| Test | What it verifies |
|------|-----------------|
| `TestPatchWriter_Appply_BumpDependency` | Version bump in dependencies |
| `TestPatchWriter_Appply_BumpDevDependency` | Version bump in devDependencies |
| `TestPatchWriter_Appply_AddOverride` | Adds overrides section |
| `TestPatchWriter_Appply_MissingManifestPath` | Returns clear error |
| `TestPatchWriter_Apply_InvalidJSON` | Handles malformed package.json |
| `TestPatchWriter_Preview_ReturnsDiff` | Diff matches plan |
| `TestRenderPlanDiff_EmptyPlan` | Friendly message for no ops |

---

### 1.4 File System Adapter (osfs.go) — 0% coverage
**File:** `internal/adapters/fs/osfs.go`  
**Functions uncovered:** `NewOSFileSystem`, `ReadFile`, `WriteFile`, `Stat`, `MkdirAll`, `Glob`, `WalkFiles`, `ExecRoot`  
**Estimated tests needed:** **5-6**

| Test | What it verifies |
|------|-----------------|
| `TestOSFileSystem_ReadWriteRoundTrip` | Write then read same data |
| `TestOSFileSystem_Stat_ExistsAndNotExists` | Both true and false cases |
| `TestOSFileSystem_MkdirAll_CreatesNestedDirs` | Nested directory creation |
| `TestOSFileSystem_Glob_MatchesPattern` | Glob pattern matching |
| `TestOSFileSystem_WalkFiles_FiltersByExtension` | Extension filtering |
| `TestOSFileSystem_ExecRoot_ReturnsCwd` | Returns current working directory |

---

## Priority 2: Medium Gaps (Core Functionality)

### 2.1 Workspace Service — Partially covered (73.6%)
**File:** `internal/core/workspace/service.go`  
**Functions uncovered:** `installHook`  
**Estimated tests needed:** **3**

| Test | What it verifies |
|------|-----------------|
| `TestInstallHook_CreatesPreCommitScript` | Writes correct hook content |
| `TestInstallHook_SkipsIfHookExists` | Doesn't overwrite existing hooks |
| `TestInstallHook_WriteError` | Handles filesystem write failures |

---

### 2.2 Audit Rules Evaluator — 0% coverage
**File:** `internal/core/audit/rules.go`  
**Functions uncovered:** `NewRuleEvaluator`, `Evaluate`, `scanTailwindArbitraryValues`, `matchesFramework`  
**Estimated tests needed:** **4-5**

| Test | What it verifies |
|------|-----------------|
| `TestRuleEvaluator_Evaluate_NoRules` | Empty rule set returns clean |
| `TestRuleEvaluator_Evaluate_TailwindArbitrary` | Detects arbitrary Tailwind classes |
| `TestRuleEvaluator_Evaluate_SQLInjection` | Detects raw SQL interpolation |
| `TestRuleEvaluator_Evaluate_FrameworkMismatch` | Wrong framework rules ignored |
| `TestScanTailwindArbitraryValues_DynamicClasses` | Only flags truly dynamic patterns |

---

### 2.3 Package Manager Executor — 0% coverage
**File:** `internal/adapters/pm/executor.go`  
**Functions uncovered:** `NewExecutor`, `Install`  
**Estimated tests needed:** **3-4**

| Test | What it verifies |
|------|-----------------|
| `TestExecutor_Install_NPM` | Runs `npm install` correctly |
| `TestExecutor_Install_PNPM` | Runs `pnpm install` correctly |
| `TestExecutor_Install_Bun` | Runs `bun install` correctly |
| `TestExecutor_Install_CommandFails` | Propagates exit errors |

---

### 2.4 YAML Rule Loader — 0% coverage
**File:** `internal/adapters/rules/loader.go`  
**Functions uncovered:** `NewYAMLLoader`, `Load`  
**Estimated tests needed:** **3**

| Test | What it verifies |
|------|-----------------|
| `TestYAMLLoader_LoadsRuleFiles` | Loads all YAML files from configs/rules |
| `TestYAMLLoader_EmptyDirectory` | Returns empty rules, no error |
| `TestYAMLLoader_InvalidYAML` | Returns parse error for bad YAML |

---

### 2.5 Ignore Store — 0% coverage
**File:** `internal/adapters/fs/ignore_store.go`  
**Functions uncovered:** `NewIgnoreStore`, `Load`  
**Estimated tests needed:** **3**

| Test | What it verifies |
|------|-----------------|
| `TestIgnoreStore_Load_WithRules` | Parses pipe-delimited format |
| `TestIgnoreStore_Load_MissingFile` | Returns empty rules, no error |
| `TestIgnoreStore_Load_ExpiredRules` | Loads rules regardless of expiry |

---

## Priority 3: Low Gaps (Cosmetic/Edge Cases)

### 3.1 GitHub Release Source — 0% coverage
**File:** `internal/adapters/releases/github.go`  
**Functions uncovered:** `NewGitHubLatestReleaseSource`, `Latest`  
**Estimated tests needed:** **2-3**

| Test | What it verifies |
|------|-----------------|
| `TestGitHubLatest_LatestRelease` | Returns tag_name from API |
| `TestGitHubLatest_NetworkError` | Handles API downtime |
| `TestGitHubLatest_404NoReleases` | Handles repo with no releases |

---

### 3.2 Node Version Detector — 0% coverage
**File:** `internal/adapters/system/node_version.go`  
**Functions uncovered:** `NewNodeVersionDetector`, `Detect`  
**Estimated tests needed:** **3**

| Test | What it verifies |
|------|-----------------|
| `TestNodeVersionDetector_Detect_ValidOutput` | Parses `v20.11.1` correctly |
| `TestNodeVersionDetector_Detect_NoNodeInstalled` | Returns false when node missing |
| `TestNodeVersionDetector_Detect_UnexpectedFormat` | Handles non-semver output |

---

### 3.3 TUI Overview — 0% coverage
**File:** `internal/adapters/tui/overview.go`  
**Functions uncovered:** `RenderOverview`, `newOverviewModel`, `Init`, `Update`, `View`, `runOverviewProgram`  
**Estimated tests needed:** **2** (TUI components are hard to unit test)

| Test | What it verifies |
|------|-----------------|
| `TestRenderOverview_RendersAllSections` | Developer, GitHub, version info present |
| `TestRunOverviewProgram_ExitsCleanly` | No panic, returns without error |

---

### 3.4 CLI Bootstrap — 0% coverage
**File:** `internal/app/bootstrap.go`  
**Functions uncovered:** `NewBootstrap`, `Execute`, `OverrideOutput`, `OverrideInteractivity`, `OverridePackageManager`, `OverridePackageMetadata`  
**Estimated tests needed:** **3-4**

| Test | What it verifies |
|------|-----------------|
| `TestNewBootstrap_CreatesAllServices` | All services non-nil |
| `TestBootstrap_OverridePackageManager` | Re-creates install service |
| `TestBootstrap_OverrideInteractivity` | Updates TTY/CI flags |
| `TestExitError_ErrorFormatting` | Error message and exit code correct |

---

## Summary: Test Count by Priority

| Priority | Packages | Estimated Tests | Impact |
|----------|----------|----------------|--------|
| **P1 — Critical** | OSV registry, OSV client, Patch writer, OS FS | **25-31** | Security correctness |
| **P2 — Medium** | Workspace, Rules, PM executor, Rule loader, Ignore store | **16-19** | Core functionality |
| **P3 — Low** | GitHub releases, Node detector, TUI overview, Bootstrap | **10-12** | Polish and edge cases |
| **Total** | 15 packages | **~51-62 tests** | Full coverage to 80%+ |

---

## Recommended Implementation Order

### Phase 1: Security-Critical (Week 1)
1. OSV Registry Inspector tests (8-10 tests)
2. OSV Client tests (6-8 tests)
3. Patch Writer tests (6-7 tests)

**Expected coverage after Phase 1:** ~68%

### Phase 2: Core Functionality (Week 2)
4. OS File System tests (5-6 tests)
5. Rule Evaluator tests (4-5 tests)
6. Workspace installHook tests (3 tests)
7. Package Manager Executor tests (3-4 tests)
8. YAML Rule Loader tests (3 tests)
9. Ignore Store tests (3 tests)

**Expected coverage after Phase 2:** ~78%

### Phase 3: Polish (Week 3)
10. GitHub Release Source tests (2-3 tests)
11. Node Version Detector tests (3 tests)
12. TUI Overview tests (2 tests)
13. CLI Bootstrap tests (3-4 tests)

**Expected coverage after Phase 3:** ~82%+

---

## Test Quality Guidelines

All new tests should follow these patterns:

1. **Table-driven tests** for input/output verification
2. **FakeFileSystem** for isolating file I/O
3. **StubHTTPClient** for isolating network calls
4. **Error path coverage** — at least one test per function for failure scenarios
5. **Edge cases** — empty input, malformed data, timeout scenarios
6. **No real network calls** — all HTTP interactions should use stubs or mocks

---

## Tools for Measuring Progress

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser (HTML report)
go tool cover -html=coverage.out

# Check specific package coverage
go test ./internal/adapters/osv -cover

# Minimum coverage gate (add to CI)
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep "total:" | awk '{print $3}'
```

---

## CI Integration Recommendation

Add to `.github/workflows/release.yml`:

```yaml
- name: Enforce minimum test coverage
  run: |
    go test ./... -coverprofile=coverage.out
    TOTAL=$(go tool cover -func=coverage.out | grep "total:" | grep -oP '\d+\.\d+' | tail -1)
    echo "Coverage: ${TOTAL}%"
    if (( $(echo "$TOTAL < 80.0" | bc -l) )); then
      echo "Coverage ${TOTAL}% is below 80% threshold"
      exit 1
    fi
```

---

**Current status:** ⚠️ 58.3% — Acceptable for general software, below standard for security tooling  
**After Phase 1:** ⚠️ ~68% — Better but still gaps in critical paths  
**After Phase 2:** ✅ ~78% — Near target, most critical paths covered  
**After Phase 3:** ✅ ~82%+ — Meets industry standard for security-focused tools
