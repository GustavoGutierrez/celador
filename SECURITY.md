# Security Policy

## Supported Versions

Only the latest stable release receives security fixes. Older versions are not backported.

| Version | Supported |
|---------|-----------|
| Latest (`v0.6.x`) | Yes |
| `v0.5.x` and older | No |

If you are on an older version, upgrade to the latest release before reporting:

```bash
brew update && brew upgrade GustavoGutierrez/celador/celador
```

---

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately using one of these channels:

- **GitHub Private Disclosure:** Use [GitHub Security Advisories](https://github.com/GustavoGutierrez/celador/security/advisories/new) to report confidentially.
- **Email:** Contact the maintainer directly at the address listed on the [GitHub profile](https://github.com/GustavoGutierrez).

### What to include in your report

A useful report provides enough context for the maintainer to reproduce and assess the issue:

- **Description** — what the vulnerability is and what it allows an attacker to do
- **Affected component** — which command, adapter, or code path is involved
- **Steps to reproduce** — the exact input, flags, or environment needed to trigger it
- **Impact** — what an attacker gains (e.g. arbitrary command execution, path traversal, data exfiltration)
- **Suggested fix** — optional, but welcome if you have a specific patch in mind
- **Celador version** — output of `celador --version`
- **Operating system and architecture**

### What happens after you report

| Timeline | Action |
|---|---|
| Within 3 days | Maintainer acknowledges the report |
| Within 14 days | Initial assessment and severity classification |
| Within 60 days | Fix released (critical issues prioritized) |
| After fix ships | CVE filed if applicable; reporter credited in release notes |

If you do not receive an acknowledgment within 3 days, send a follow-up ping via email.

---

## Scope

The following are **in scope** for security reports:

- Arbitrary command execution or code injection via CLI input
- Path traversal in workspace file operations
- OSV or provenance data tampering via MITM or cache poisoning
- Secrets or credentials leaked in output, cache files, or logs
- Denial of service via crafted lockfiles or package metadata
- Logic errors in the sandbox or provenance adapter that produce false negatives on known-malicious inputs

The following are **out of scope**:

- Vulnerabilities in third-party dependencies (report those directly to the upstream project via OSV / GitHub Advisories)
- Issues that require physical access to the developer's machine
- Social engineering attacks
- Findings from automated scanners without a demonstrated proof of concept

---

## Security Design Principles

Celador follows these security principles in its own codebase. If you find a violation, that is a valid report:

- All user-supplied CLI input is validated before use
- File paths are checked to remain within the workspace root (path traversal prevention)
- External API endpoints are configurable and never hard-coded without a safe default
- Errors are never silently discarded — all failures are surfaced or explicitly documented
- Cache entries are schema-versioned to prevent stale or tampered data from being reused
- The behavioral sandbox runs offline by default with no real network or filesystem access
