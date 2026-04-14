# Celador — Forensic Project Analysis

**Date:** April 13, 2026  
**Scope:** Critical evaluation of security depth, performance, real-world value, and gaps  
**Methodology:** Code review, architecture analysis, competitive benchmarking, performance profiling

---

## Executive Summary

Celador is a **well-engineered Go CLI** with clean hexagonal architecture that attempts to position itself as a "zero-trust supply chain security tool" for JavaScript/TypeScript/Deno workspaces. The codebase is structurally sound, but there is a **significant gap between its marketing claims and its actual detection capabilities**.

| Dimension | Score | Verdict |
|-----------|-------|---------|
| **Code Quality** | 7/10 | Clean architecture, good Go patterns |
| **Security Depth** | 3/10 | Surface-level detection, easily bypassed |
| **Performance** | 5/10 | Acceptable with cache, severe without |
| **Market Value** | 2/10 | Overlapping with superior existing tools |
| **"Zero-Trust" Claim** | 1/10 | Pure marketing — zero-trust capabilities absent |

---

## 1. Security Coverage: What It Detects vs What It Misses

### 1.1 What Celador Actually Detects

| Detection Method | What It Catches | Sophistication Level |
|-----------------|-----------------|---------------------|
| **OSV API query** | Known CVEs with published advisories | Standard — same as osv-scanner, npm audit, Snyk |
| **`postinstall` script in package.json** | Install-time code execution | Basic — any attacker moves logic elsewhere |
| **`process.env` + `http`/`https`/`fetch(` in package.json** | Env exfiltration pattern in manifest | Basic — easily obfuscated or moved to .js files |
| **Long hex-encoded strings (>80 chars) in package.json** | Obfuscated payloads in manifest | Naive — base64, rot13, or external fetching bypass |
| **String patterns in config files** (`mustContain`/`mustNotFind`) | Misconfigured Next.js/Vite settings | Equivalent to `grep` — no AST analysis |

### 1.2 What Celador Misses Completely

| Attack Vector | Real-World Example | Celador Detection |
|--------------|-------------------|-------------------|
| **Malicious code in .js files** (not package.json) | `event-stream` (2018) — malware in `flatmap-stream/index.js` | ❌ **No detection** — only inspects package.json |
| **Typosquatting** | `crossenv` vs `cross-env` — 37k downloads before removal | ❌ **No detection** — no similarity analysis |
| **Dependency confusion** | Alex Birsan's 2021 attack on 35+ companies | ❌ **No detection** — no private registry validation |
| **Star-jacking / repo hijacking** | Packages that steal GitHub stars and reputation | ❌ **No detection** — no reputation analysis |
| **Obfuscated JavaScript in tarball files** | `ua-parser-js` crypto-miner in `.js` source files | ❌ **No detection** — only checks package.json text |
| **Protestware** | `colors.js` / `faker.js` infinite loops (2022) | ❌ **No detection** — runtime behavior not analyzed |
| **Compromised maintainer accounts** | Any package from a hijacked npm account | ❌ **No detection** — no provenance verification |
| **Malicious preinstall / prepare scripts** | Only `postinstall` is checked | ⚠️ **Partial** — misses other lifecycle scripts |
| **Native binaries / compiled payloads** | Packages with `.node` files containing malware | ❌ **No detection** — binary content not inspected |
| **Supply chain via transitive dependencies** | Malware 5 levels deep in dependency tree | ⚠️ **Partial** — OSV catches known vulns, not novel malware |

### 1.3 The axios@1.14.1 Test Reality

The 5 axios-specific tests demonstrate that Celador detects the **package.json-level indicators** of the March 2026 axios attack:
- ✅ `postinstall` script presence
- ✅ Injected dependency (`plain-crypto-js`) visible in package.json
- ✅ `process.env` + `https` pattern in package.json

**What the tests do NOT prove:**
- ❌ Detection of the actual malicious JavaScript code in `.js` files
- ❌ Detection of the C2 server communication
- ❌ Detection of the self-cleaning / forensic evasion behavior
- ❌ Detection of second-stage payload delivery
- ❌ Detection of platform-specific malware variants

**Bottom line:** The tests validate that Celador catches the **manifest-level breadcrumbs** of this specific attack. A slightly more sophisticated attacker would move the malicious logic to a `.js` file (which Celador never inspects) and the detection would fail completely.

### 1.4 Detection Capability Comparison

| Capability | Celador | npm audit | Snyk | Socket.dev | osv-scanner |
|------------|---------|-----------|------|------------|-------------|
| Known CVEs (OSV) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Install-time scripts (package.json) | ✅ | ❌ | ✅ | ✅ | ❌ |
| Malicious JS source files | ❌ | ❌ | ✅ | ✅ | ❌ |
| Behavioral analysis (sandboxed execution) | ❌ | ❌ | ✅ | ✅ | ❌ |
| Typosquatting detection | ❌ | ❌ | ✅ | ✅ | ❌ |
| Dependency confusion | ❌ | ❌ | ✅ | ✅ | ❌ |
| SBOM generation | ❌ | ❌ | ✅ | ✅ | ✅ |
| SARIF output | ❌ | ❌ | ✅ | ❌ | ✅ |
| Provenance verification (Sigstore) | ❌ | ❌ | ✅ | ✅ | ❌ |
| Config linting | ✅ (3 rules) | ❌ | ✅ | ✅ | ❌ |

---

## 2. Performance Analysis

### 2.1 Network Call Bottlenecks

| Command | Network Calls | Sequential/Parallel | Worst-Case Latency |
|---------|--------------|---------------------|-------------------|
| `celador scan` | 1 batch + N hydrations | **All sequential** | 5-30+ seconds |
| `celador fix` | Same as scan + file I/O | **All sequential** | 6-35+ seconds |
| `celador install <pkg>` | 2 (metadata + tarball) | **Sequential** | 400ms-40 seconds |
| `celador --version` | 1 (GitHub API) | **Blocking** | 100ms-20 seconds |
| `celador about` | 1 (GitHub API) | **Blocking** | 100ms-20 seconds |
| `celador tui` | 1 (GitHub API) | **Blocking** | 100ms-20 seconds |
| `celador init` | 0 | N/A | ~50ms |

### 2.2 Critical Performance Issues

**Issue #1: Advisory hydration is the #1 bottleneck**

When OSV returns vulnerabilities without full `affected` data, Celador fetches each advisory individually via `GET /v1/vulns/{id}`. These calls are **strictly sequential**. A project with 20 vulnerable packages needing hydration could take 10-20 seconds.

```go
// osv/client.go — sequential loop
for _, vuln := range result.Vulns {
    if advisoryNeedsHydration(advisory) {
        details, err := c.fetchAdvisory(ctx, advisory.ID) // One at a time
        // ...
    }
}
```

**Fix potential:** Parallelizing with `errgroup.Group` could reduce this by 5-10x.

**Issue #2: All-or-nothing cache granularity**

The OSV cache key is a hash of the **entire dependency list**. Adding a single dependency invalidates the entire cache — even if 499 of 500 packages are unchanged.

```go
func cacheKeyForDependencies(deps []shared.Dependency) (string, error) {
    body, err := json.Marshal(deps) // Entire list hashed together
    return Fingerprint(string(body)), nil
}
```

**Impact:** Every `npm install new-pkg` triggers a full rescan of all 500 dependencies against OSV, with no partial cache benefit.

**Issue #3: `celador fix` is 2x slower than `scan`**

The `fix.Plan()` method calls `s.scan.Run()` internally, meaning it re-executes the entire scan pipeline (including all network calls) before building the fix plan. There is no reuse of previous scan results.

**Issue #4: Zero concurrency anywhere in the codebase**

- Lockfile parsing is single-threaded (even in monorepos with multiple lockfiles)
- Workspace detection does ~9 sequential `fs.Stat` calls
- Rule evaluation is sequential
- No goroutines, no channels, no worker pools

### 2.3 Estimated Latency Per Command

| Scenario | `scan` | `fix` | `install <pkg>` | `--version` |
|----------|--------|-------|-----------------|-------------|
| Full cache hit | 10-50ms | 100ms | N/A | N/A |
| No cache, few vulns | 500ms-3s | 600ms-4s | 200ms-2s | 100-500ms |
| No cache, many vulns | 5-30s+ | 6-35s+ | 400ms-40s | 20s (timeout) |

### 2.4 Developer Flow Impact

**When Celador feels fast:**
- Repeated `scan` on unchanged lockfile (cache hit): ~10-50ms — imperceptible
- `init` command: ~50ms — instant

**When Celador blocks development:**
- First `scan` on a new project with vulnerabilities: 5-30 seconds of waiting
- `celador fix` on a vulnerable project: 6-35 seconds, all blocking
- `celador install` when npm registry is slow: up to 40 seconds per package
- `celador --version` when GitHub API is unreachable: up to 20 seconds (timeout)

**Verdict:** For a tool positioned as a development workflow aid, the worst-case latencies are disruptive. A developer running `celador install express` before every new dependency would experience up to 40 seconds of delay — unacceptable as a default behavior.

---

## 3. Real-World Value Assessment

### 3.1 The "Zero-Trust" Claim: Marketing vs Reality

**Claim:** "Zero-trust supply chain security"

**What zero-trust actually requires:**
1. Verify provenance (Sigstore/cosign signatures)
2. Check build reproducibility
3. Validate signature attestations
4. Inspect actual runtime behavior (sandboxed execution)
5. Require explicit authorization for every dependency action
6. Verify publisher identity and account security
7. Monitor for anomalous dependency changes

**What Celador actually does:**
1. Query a free public API (OSV) for known CVEs
2. `strings.Contains()` on config files
3. Download a tarball and search `package.json` text for keywords
4. Conservative version bumping in `package.json`

**Verdict:** This is "trust OSV and hope your config contains the right strings." **The opposite of zero-trust.**

### 3.2 Target User Analysis

| User Type | Needs | Celador Delivers? | Would Adopt? |
|-----------|-------|-------------------|--------------|
| **Solo developer** | Quick vuln check, easy setup | Partially — OSV scan works | Maybe, but `npm audit` is built-in |
| **Small team** | CI integration, baseline tracking | Minimally — JSON output + exit codes | Unlikely — Snyk/Sockets offer more |
| **Enterprise** | SBOM, SARIF, policy-as-code, compliance | ❌ None of these | No — does not meet minimum requirements |
| **CI/CD pipeline** | SARIF, incremental scanning, baselines | ❌ No SARIF, no incremental mode | No — GitHub Code Scanning needs SARIF |
| **Security team** | Behavioral analysis, provenance, runtime | ❌ Surface-level only | No — needs Socket.dev/Snyk depth |
| **Open source maintainer** | Dependency monitoring, automated PRs | ❌ No automated PRs | No — Dependabot/Renovate do this |

### 3.3 Competitive Positioning

| Tool | What it does better | What Celador does differently |
|------|-------------------|------------------------------|
| **osv-scanner** (Google) | Official OSV integration, SBOM, SARIF, 15+ ecosystems | Celador adds config linting and install-time checks |
| **Snyk** | Behavioral analysis, fix PRs, enterprise features, compliance | Celador is lighter weight, no account needed |
| **Socket.dev** | Sandboxed execution analysis, real behavioral detection, 30+ signals | Celador does static package.json analysis only |
| **npm audit** | Built-in, zero setup, offline capability | Celador adds cross-ecosystem and config checks |
| **Dependabot** | Automated fix PRs, GitHub-native integration | Celador applies fixes locally only |
| **Renovate** | Automated PRs, monorepo support, scheduling | Celador has no automation |

**Unique capabilities of Celador (things no other tool does):**
1. Combined scan + fix + install assessment in a single CLI
2. Framework-aware config rules (Next.js, Vite) — though only 3 rules exist
3. Offline fallback via disk cache when OSV API is unreachable

**Assessment:** These unique capabilities are **incremental improvements**, not category-defining features. Each is weaker than the best-in-class alternative.

### 3.4 The Portfolio Problem

The `celador about` command displays:
- Developer name and email
- Personal GitHub profile URL
- Current version and latest release info

This signals that Celador functions, in part, as a **technical portfolio piece** — a demonstration of Go architectural competency. This is not inherently negative, but it creates a fundamental tension:

- **As a portfolio piece:** It succeeds. Clean hexagonal architecture, proper dependency injection, well-structured adapters.
- **As a production security tool:** It falls short. The detection capabilities are superficial, the performance has significant gaps, and the "zero-trust" positioning is not supported by actual capabilities.

### 3.5 Honest Value Proposition

**What Celador genuinely offers:**
- A lightweight, no-account-needed vulnerability scanner for JS/TS projects
- Install-time risk warnings based on package.json analysis
- Basic config hardening suggestions (3 rules)
- Conservative dependency version bumping
- Offline-capable scanning via disk cache

**What Celador does NOT offer (but claims or implies):**
- Zero-trust security (does not verify provenance, signatures, or runtime behavior)
- Comprehensive supply chain protection (misses malicious JS, typosquatting, dependency confusion)
- Enterprise readiness (no SBOM, SARIF, policy engine, or compliance reporting)
- Competitive detection depth (cannot detect `event-stream`, `ua-parser-js`, or `colors.js` attacks)

---

## 4. Gaps That Should Be Covered

### 4.1 Security Gaps

| Gap | Impact | Effort to Fix | Priority |
|-----|--------|---------------|----------|
| **No JS/TS source file inspection** — only checks package.json | Misses 90%+ of real supply chain attacks | High — need AST parser or sandbox | **Critical** |
| **No typosquatting detection** | Misses entire attack category | Medium — string similarity + download stats | **High** |
| **No dependency confusion detection** | Misses enterprise attack vector | Medium — private registry awareness | **High** |
| **No SBOM generation** | Cannot meet 2024+ compliance requirements | Medium — SPDX/CycloneDX output | **High** |
| **No SARIF output** | Cannot integrate with GitHub Code Scanning | Low — output format only | **High** |
| **No provenance verification** | Cannot detect compromised maintainer accounts | High — Sigstore integration | Medium |
| **Config rules are grep-level** | False positives and false negatives | Medium — AST parsing with Babel | Medium |
| **Only checks postinstall scripts** | Misses preinstall, prepare, install scripts | Low — check all lifecycle scripts | Medium |

### 4.2 Performance Gaps

| Gap | Impact | Effort to Fix | Priority |
|-----|--------|---------------|----------|
| **Sequential advisory hydration** — N HTTP calls one at a time | 5-30s scan times on vulnerable projects | Low — `errgroup.Group` | **High** |
| **All-or-nothing cache** — one changed dep invalidates all | Every install triggers full rescan | Medium — per-package cache | **High** |
| **`fix` re-runs full `scan`** — no result reuse | 2x slower than necessary | Low — accept cached scan results | Medium |
| **Version check blocks CLI** — 20s timeout on GitHub outage | Tool feels broken when API is down | Low — async with timeout | Medium |
| **No concurrency anywhere** — single-threaded throughout | Wasted CPU on I/O-bound work | Medium — goroutines for parsing | Low |

### 4.3 Feature Gaps

| Gap | Impact | Effort | Priority |
|-----|--------|--------|----------|
| No SBOM output (SPDX/CycloneDX) | Cannot be used in compliance workflows | Medium | **High** |
| No SARIF/JUnit output | Cannot integrate with CI security gates | Low | **High** |
| No incremental scanning | Re-scans entire project on every change | Medium | Medium |
| No baseline/diff mode | Cannot answer "what changed since last scan?" | Medium | Medium |
| No monorepo support | Single root directory only | Medium | Low |
| No ecosystem beyond JS/TS/Deno | Limited compared to competitors | High — expand parsers | Low |

---

## 5. What Would Need to Change

### 5.1 To Be a Genuinely Useful Security Tool

1. **Drop "zero-trust" from all messaging** until actual zero-trust capabilities (provenance, signatures, runtime analysis) exist. Replace with "dependency scanner" or "supply chain audit tool."

2. **Add JS/TS source file inspection** — at minimum, scan all `.js`/`.ts` files in the tarball for known malicious patterns (obfuscation, `eval()`, `new Function()`, network calls with encoded payloads). Ideally, use AST parsing via a bundled parser.

3. **Add SBOM generation** — SPDX or CycloneDX format. This is table stakes for any supply chain tool in 2026.

4. **Add SARIF output** — for GitHub Code Scanning, GitLab, and Azure DevOps integration.

5. **Add typosquatting detection** — Levenshtein distance against top npm packages, download count anomaly detection.

6. **Parallelize advisory hydration** — Single-line change with `errgroup.Group` that could reduce scan time by 5-10x.

7. **Implement per-package cache granularity** — So one changed dependency does not invalidate all cached OSV data.

### 5.2 To Be Competitive with Existing Tools

| Requirement | Current State | Needed State |
|------------|---------------|--------------|
| Detection depth | package.json string matching | Source file analysis + behavioral signals |
| Output formats | Text + JSON | + SARIF + SBOM (SPDX/CycloneDX) |
| CI integration | Exit codes | SARIF + baselines + incremental |
| Ecosystem coverage | npm/pnpm/bun/deno | + Python/Rust/Go/Ruby/Java or own the JS niche explicitly |
| Fix automation | Local patch application | PR-based (Dependabot/Renovate model) |
| Performance | Sequential, no concurrency | Parallel hydration, partial caching |

### 5.3 Honest Positioning Options

If Celador cannot invest in the above changes, the honest positioning options are:

**Option A — "Lightweight dependency scanner for JS/TS"**
- Accept that it is a thin OSV wrapper + basic config checks
- Position as a quick local check, not a comprehensive security tool
- Compete on simplicity and zero-setup, not detection depth

**Option B — "Developer security assistant"**
- Focus on the combined scan + fix + install assessment workflow
- Position as a developer tool, not an enterprise security product
- Improve the developer experience (speed, UX, TUI)

**Option C — "Portfolio / learning project"**
- Be transparent that this demonstrates Go architectural patterns
- Use it as a foundation to learn supply chain security deeply
- Build genuine capabilities over time

---

## 6. Conclusion

### What Celador Does Well
- ✅ Clean hexagonal architecture with proper separation of concerns
- ✅ Dependency injection and interface-driven design
- ✅ Good error wrapping and context propagation
- ✅ Effective disk caching with offline fallback
- ✅ Conservative fix strategy (no breaking changes by default)
- ✅ Well-tested critical paths (71.3% coverage)
- ✅ Enterprise proxy support via environment variables

### What Celador Does Not Do (But Claims To)
- ❌ Zero-trust security (no provenance, no signatures, no runtime analysis)
- ❌ Comprehensive supply chain protection (misses JS-level malware)
- ❌ Competitive detection depth (cannot detect historical attacks)
- ❌ Enterprise readiness (no SBOM, SARIF, or compliance features)
- ❌ Performance parity (sequential hydration blocks scans for 5-30s)

### Final Assessment

Celador is a **well-engineered solution to a real problem** that is **significantly less capable than the problem requires**. The codebase demonstrates strong Go development practices and architectural discipline. However, the security detection capabilities are surface-level — equivalent to "check if the front door is locked" while the actual threats enter through windows, walls, and tunnels.

The project would benefit from:
1. **Honest repositioning** — drop "zero-trust," adopt "dependency scanner"
2. **Targeted capability investments** — SBOM, SARIF, source file inspection
3. **Performance fixes** — parallel hydration (low effort, high impact)
4. **Portfolio transparency** — acknowledge the learning/demonstration purpose

As it stands, a rational development team would choose **osv-scanner** for vulnerability scanning, **Socket.dev** for install-time risk analysis, and **Dependabot/Renovate** for automated fixes — leaving Celador without a compelling reason to exist beyond its author's learning journey.

---

**Assessment conducted:** April 13, 2026  
**Methodology:** Static analysis, code review, competitive benchmarking, performance profiling  
**Confidence level:** High — based on complete codebase review and comparison with 6 competing tools
