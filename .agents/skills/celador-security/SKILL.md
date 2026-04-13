---
name: celador-security
description: Security-first Go development patterns for Celador. Enforces error handling, input validation, path traversal prevention, severity propagation, and secure external API integration. Use when implementing security-critical features, handling user input, writing file operations, or reviewing code for vulnerabilities.
---

# Celador Security Patterns

Security-focused development guidelines for Celador that address identified vulnerabilities and
enforce secure coding practices.

## When to Activate

- Implementing new CLI commands or arguments
- Writing file read/write operations
- Integrating external APIs (OSV, npm registry, GitHub)
- Handling user input or configuration
- Implementing vulnerability scanning or severity logic
- Reviewing code for security vulnerabilities
- Adding new adapters or core services

## Critical Rules

### 1. Error Handling: NEVER discard silently

All errors must be handled, wrapped with context, or explicitly documented if intentionally ignored.

**FORBIDDEN:**

```go
// NEVER do this
content, _ := s.fs.ReadFile(ctx, path)
executablePath, _ := os.Executable()
```

**REQUIRED:**

```go
content, err := s.fs.ReadFile(ctx, path)
if err != nil {
    if os.IsNotExist(err) {
        content = []byte{} // Safe: file doesn't exist, will create
    } else {
        return fmt.Errorf("read %s: %w", path, err)
    }
}
```

**When reading files in loops or optional configs:**

```go
// Good: Distinguish between "missing" and "broken"
func readFileOrEmpty(ctx context.Context, fs ports.FileSystem, path string) ([]byte, error) {
    content, err := fs.ReadFile(ctx, path)
    if err != nil {
        if os.IsNotExist(err) {
            return []byte{}, nil
        }
        return nil, fmt.Errorf("read %s: %w", path, err)
    }
    return content, nil
}
```

**Why this matters:** The analysis found multiple locations where discarded read errors could cause
data loss. If a file exists but is unreadable (permissions, corruption), treating it as empty and
proceeding to write will **destroy user content**.

---

### 2. Input Validation: Validate ALL CLI arguments

CLI arguments must be validated before use. Never assume user input is well-formed.

**FORBIDDEN:**

```go
// Accepts empty strings or whitespace-only values
if len(args) == 0 {
    return NewExitError(2, "install requires at least one package argument")
}
// args could be ["", "   "] and pass this check
```

**REQUIRED:**

```go
if len(args) == 0 {
    return NewExitError(2, "install requires at least one package argument")
}
for _, arg := range args {
    if strings.TrimSpace(arg) == "" {
        return NewExitError(2, "package name cannot be empty or whitespace")
    }
    // Additional validation as needed
    if strings.Contains(arg, "..") {
        return NewExitError(2, "package name contains invalid characters")
    }
}
```

**Validation checklist for CLI commands:**

- [ ] Empty string check
- [ ] Whitespace-only check
- [ ] Path traversal patterns (`..`, `/`, `\`)
- [ ] Special characters that could cause injection
- [ ] Length limits where applicable

**Why this matters:** Empty or malicious package names cause confusing downstream errors, SSRF
attacks against registries, or unexpected behavior.

---

### 3. Path Traversal Prevention

When accepting file paths from user config or CLI, validate they remain within the workspace root.

**FORBIDDEN:**

```go
// Writes to user-controlled path without validation
err := fs.WriteFile(ctx, config.TargetPath, content)
```

**REQUIRED:**

```go
func validateWorkspacePath(path string, wsRoot string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("resolve path: %w", err)
    }
    absRoot, err := filepath.Abs(wsRoot)
    if err != nil {
        return fmt.Errorf("resolve root: %w", err)
    }
    if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
        return fmt.Errorf("path %q is outside workspace root %q", path, wsRoot)
    }
    return nil
}
```

**Why this matters:** Malicious config files could specify paths like `../../../etc/passwd` to
write outside the intended workspace.

---

### 4. Preserve Severity from External APIs

Never hard-code severity levels when the upstream API provides them. Parse and propagate actual
severity data.

**FORBIDDEN:**

```go
// All vulnerabilities reported as HIGH regardless of actual severity
finding := shared.Finding{
    Severity: shared.SeverityHigh, // WRONG
    // ...
}
```

**REQUIRED:**

```go
finding := shared.Finding{
    Severity: parseSeverityFromOSV(advisory),
    // ...
}

func parseSeverityFromOSV(advisory osvAdvisory) string {
    // Check for CVSS or custom severity
    if len(advisory.Severity) > 0 {
        for _, sev := range advisory.Severity {
            if sev.Type == "CVSS_V3" || sev.Type == "CVSS_V2" {
                return cvssToCeladorSeverity(sev.Score)
            }
        }
    }
    // Default to medium, not high
    return shared.SeverityMedium
}
```

**Why this matters:** Users cannot prioritize remediation when all vulnerabilities appear equally
critical. Alert fatigue undermines trust in the tool.

---

### 5. Guard Against Empty External API Calls

Skip network calls when there is nothing to query. Avoid unnecessary latency and bandwidth usage.

**FORBIDDEN:**

```go
// Sends POST with {"queries": []} to OSV API
queries := make([]map[string]any, 0, len(deps))
// ... build queries ...
resp, err := c.httpClient.Do(req) // Wasteful if deps is empty
```

**REQUIRED:**

```go
if len(deps) == 0 {
    return []shared.Finding{}, nil
}

// Proceed with API call only when there's data to query
```

**Why this matters:** Empty queries waste CI/CD pipeline time, add latency, and consume API quotas.

---

### 6. Use Public Port Interfaces Consistently

Always use public port interfaces. Never redefine private duplicates.

**FORBIDDEN:**

```go
// Private interface duplicates ports.Clock
type clock interface{ Now() time.Time }

type Service struct {
    clock clock // Should use ports.Clock
}
```

**REQUIRED:**

```go
type Service struct {
    clock ports.Clock
}

func NewService(clock ports.Clock) *Service {
    return &Service{clock: clock}
}
```

**Why this matters:** Duplicate interfaces create confusion, inconsistency, and make testing harder.

---

### 7. External Endpoints Must Be Configurable

External API endpoints should support environment variable overrides for enterprise proxies,
air-gapped environments, and testing.

**FORBIDDEN:**

```go
// Hard-coded, non-configurable endpoint
endpoint: "https://api.osv.dev/v1/querybatch",
```

**REQUIRED:**

```go
func NewClient(ttl time.Duration) *Client {
    endpoint := os.Getenv("CELADOR_OSV_ENDPOINT")
    if endpoint == "" {
        endpoint = "https://api.osv.dev/v1/querybatch"
    }

    vulnAPI := os.Getenv("CELADOR_OSV_VULN_API")
    if vulnAPI == "" {
        vulnAPI = "https://api.osv.dev/v1/vulns"
    }

    return &Client{
        httpClient: &http.Client{Timeout: 20 * time.Second},
        ttl:        ttl,
        endpoint:   endpoint,
        vulnAPI:    vulnAPI,
    }
}
```

**Environment variables to support:**

- `CELADOR_OSV_ENDPOINT` - OSV batch query endpoint
- `CELADOR_OSV_VULN_API` - OSV individual vulnerability endpoint
- `CELADOR_NPM_REGISTRY` - npm registry URL
- `CELADOR_GITHUB_API` - GitHub API endpoint for version checks

**Why this matters:** Enterprise users need proxy support. Testability requires mockable endpoints.

---

### 8. Handle Binary Formats Correctly

When claiming support for binary file formats, implement proper parsing or explicitly decline
support with a user warning.

**FORBIDDEN:**

```go
func (p *BunParser) Supports(path string) bool {
    // Claims support but only parses as text
    return filepath.Base(path) == "bun.lock" || filepath.Base(path) == "bun.lockb"
}

func (p *BunParser) Parse(path string) ([]Dependency, error) {
    // Parses binary file as text - returns garbage
    lines := strings.Split(string(body), "\n")
    // ...
}
```

**REQUIRED:**

```go
func (p *BunParser) Supports(path string) bool {
    base := filepath.Base(path)
    if base == "bun.lockb" {
        // Binary format not yet supported
        return false
    }
    return base == "bun.lock"
}

// Or: implement proper binary parsing
func (p *BunParser) Parse(path string) ([]Dependency, error) {
    if strings.HasSuffix(path, ".lockb") {
        return parseBunLockb(path) // Proper binary parser
    }
    return parseBunLock(path)     // Text parser
}
```

**Why this matters:** Parsing binary files as text silently returns garbage data, giving users a
false sense of security when vulnerabilities are missed.

---

## Secure HTTP Client Patterns

### Response Body Handling

**ALWAYS:**

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    return nil, err
}
defer resp.Body.Close() // MUST be after error check

if resp.StatusCode >= 300 {
    body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
    return nil, fmt.Errorf("API returned %s: %s", resp.Status, string(body))
}
```

**NEVER:**

```go
resp, err := c.httpClient.Do(req)
defer resp.Body.Close() // WRONG: resp might be nil if err != nil
if err != nil {
    return nil, err
}
```

---

## Secure File Write Patterns

### Managed File Writes

When writing to managed files (e.g., `.gitignore`, templates):

```go
func (t *TemplateWriter) WriteManagedSection(ctx context.Context, path string, content string) error {
    // 1. Read existing file (handle errors correctly)
    existing, err := t.fs.ReadFile(ctx, path)
    var existingContent string
    if err != nil {
        if os.IsNotExist(err) {
            existingContent = ""
        } else {
            return fmt.Errorf("read %s: %w", path, err)
        }
    } else {
        existingContent = string(existing)
    }

    // 2. Check if managed section exists
    if strings.Contains(existingContent, managedStart) {
        // Replace existing section
        return t.replaceManagedSection(path, existingContent, content)
    }

    // 3. Append new section
    return t.appendManagedSection(path, existingContent, content)
}
```

---

## Checklist for New Features

Before committing code that touches security-critical paths:

### Error Handling
- [ ] No `value, _ := function()` calls without justification
- [ ] All errors wrapped with context using `%w`
- [ ] File read errors distinguish "not exists" from "read failed"
- [ ] HTTP errors include status code and body snippet

### Input Validation
- [ ] CLI arguments validated for empty/whitespace
- [ ] File paths validated against workspace root
- [ ] User-provided strings sanitized before use in URLs or shell commands
- [ ] Config values validated before use

### External APIs
- [ ] Empty data checks skip network calls
- [ ] Severity data parsed and propagated (not hard-coded)
- [ ] Timeouts reasonable and configurable
- [ ] Endpoints support environment variable overrides

### File Operations
- [ ] Read errors handled correctly (not discarded)
- [ ] Write paths validated against workspace root
- [ ] Managed files preserve non-managed content
- [ ] Binary formats parsed correctly or declined

### Interfaces
- [ ] Using public port interfaces (`ports.Clock`, `ports.Logger`)
- [ ] No private interface duplicates of public ports
- [ ] Constructor accepts interface, stores concrete type if needed

### Testing
- [ ] Error paths tested (not just happy path)
- [ ] Empty input cases tested
- [ ] File-not-found cases tested
- [ ] API error responses tested
- [ ] Coverage meets 80% target for critical paths

---

## Common Vulnerability Patterns to Avoid

### 1. Silent Data Loss

```go
// BAD: Overwrites file if read fails for ANY reason
content, _ := fs.ReadFile(ctx, path) // Error discarded
fs.WriteFile(ctx, path, newContent)  // Could destroy content
```

### 2. False Security Reports

```go
// BAD: Reports "no vulnerabilities" when scan failed
deps, err := parser.Parse(path)
if err != nil {
    fmt.Println("No vulnerabilities found") // WRONG: should report error
}
```

### 3. Timing Attacks

```go
// BAD: String comparison vulnerable to timing attacks
if input == expectedSecret { ... }

// GOOD: Use constant-time comparison
if subtle.ConstantTimeCompare([]byte(input), []byte(expectedSecret)) == 1 { ... }
```

### 4. Insecure Random

```go
// BAD: Predictable random (math/rand)
token := fmt.Sprintf("%d", rand.Int())

// GOOD: Cryptographic random (crypto/rand)
token := make([]byte, 32)
_, err := rand.Read(token)
```

---

## When in Doubt

1. **Ask:** "What happens if this file doesn't exist? What if it's unreadable?"
2. **Ask:** "What if the user passes an empty string? A path with `..`?"
3. **Ask:** "Am I losing information silently?"
4. **Ask:** "Would a malicious config file cause harm here?"

If uncertain, invoke this skill and review all patterns against the checklist above.
