---
name: go-govulncheck
description: >
  Go vulnerability scanning quality gate using govulncheck. Use as the final
  step of the pre-PR quality gate in a Go project. Checks all dependencies
  against the Go vulnerability database and reports only vulnerabilities that
  are reachable from the actual code — not just present in go.sum. Triggered
  automatically by software-development at step 10 (quality gate), or on demand
  when the user asks to check for vulnerabilities, audit dependencies, or scan
  for security issues.
---

# go-govulncheck

`govulncheck` is the official Go vulnerability scanner, maintained by the Go
team. It is smarter than a naive dependency audit: it performs static analysis
to report only vulnerabilities that are **reachable** from your code, reducing
false positives significantly.

---

## What it checks

- All direct and transitive dependencies in `go.mod` against the
  [Go Vulnerability Database](https://vuln.go.dev).
- Only reports vulnerabilities in code paths that are actually called by your
  module — not every CVE in every transitive dependency.
- Also checks the Go standard library version in use.

---

## Installation

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Verify: `govulncheck -version`

---

## Running

**Full module** (standard):
```bash
govulncheck ./...
```

**Verbose output** (shows the call chain that reaches the vulnerability):
```bash
govulncheck -v ./...
```

**Scan a specific binary** (after build):
```bash
govulncheck -mode=binary ./bin/myservice
```

**JSON output** (for tooling integration):
```bash
govulncheck -json ./...
```

---

## Interpreting output

`govulncheck` distinguishes between two finding types:

**Called vulnerability** (must fix):
```
Vulnerability #1: GO-2024-1234
    A vulnerability in package foo/bar allows...
    More info: https://pkg.go.dev/vuln/GO-2024-1234
  Module: github.com/example/foo
    Found in: github.com/example/foo@v1.2.0
    Fixed in: github.com/example/foo@v1.2.1
  Call stacks (1):
    internal/infra/postgres/repo.go:45:12: calls foo.VulnerableFunction
```
→ Your code directly or indirectly calls the vulnerable function. **Must be fixed before merging.**

**Imported but not called** (informational):
```
Vulnerability #2: GO-2024-5678 (informational)
    ...
  Module: github.com/example/bar
    Found in: github.com/example/bar@v2.0.0
    Fixed in: github.com/example/bar@v2.0.1
  No call stacks found.
```
→ The vulnerable package is in your dependency graph but the vulnerable code
is not reachable. Upgrading is still recommended but is not a blocker.

---

## Fixing vulnerabilities

**Step 1 — upgrade the dependency:**
```bash
go get github.com/example/foo@v1.2.1
go mod tidy
```

**Step 2 — verify the fix:**
```bash
govulncheck ./...
```

**If no fixed version exists:**
- Check if the vulnerability is actually reachable (read the call stack carefully).
- If reachable with no fix available, assess the risk and document it as an
  accepted risk with a date for re-evaluation.
- Consider replacing the dependency.

---

## CI integration

Run as the last step of the quality gate, after linting:

```yaml
- name: Vulnerability scan
  run: govulncheck ./...
```

For repositories that need to tolerate known unfixable vulnerabilities, use
`-json` output and a policy script rather than failing blindly.

---

## Quality gate result

| Outcome | Meaning | Action |
|---|---|---|
| No findings, exit 0 | No reachable vulnerabilities | ✅ Quality gate passed — ready for PR |
| Informational only, exit 0 | Vulnerable code present but unreachable | Upgrade when convenient; proceed |
| Called vulnerability, exit 1 | Reachable vulnerability found | Fix before merging |
