---
name: go-golangci-lint
description: >
  Go linting quality gate using golangci-lint. Use as part of the pre-PR quality
  gate in a Go project. Runs a curated set of linters that enforce the rules
  defined in go-guidelines: error wrapping, context propagation, cognitive
  complexity, code smells, and more. Triggered automatically by
  software-development at step 10 (quality gate), or on demand when the user
  asks to lint their code, check for code smells, or run the full linter suite.
---

# go-golangci-lint

`golangci-lint` is a linter runner — it orchestrates multiple linters under a
single config file and runs them in parallel. It is the standard linting tool
for Go projects.

---

## Installation

Install via the official script with a pinned version — do **not** use
`go install` (it can pick up incorrect versions):

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.57.2
```

Pin the version in CI. Never use `@latest` in automated pipelines.

Verify: `golangci-lint version`

---

## Configuration

Place `.golangci.yml` at the module root. This baseline is calibrated to enforce
the rules in `go-guidelines`:

```yaml
linters:
  enable:
    - errcheck        # never ignore errors
    - staticcheck     # deep static analysis + deprecated API detection
    - revive          # style and idiom rules (replaces golint)
    - wrapcheck       # errors must be wrapped at package boundaries
    - contextcheck    # context.Context must be propagated correctly
    - gocritic        # code smells and anti-patterns
    - exhaustive      # switch on enums must cover all cases
    - gocognit        # cognitive complexity limit per function

linters-settings:
  gocognit:
    min-complexity: 10        # functions harder to reason about than this are flagged

  wrapcheck:
    ignoreSigs:
      - .Errorf(              # allow fmt.Errorf wrapping
      - errors.New(
      - errors.Unwrap(
      - errors.Join(

  revive:
    rules:
      - name: exported        # all exported symbols must have doc comments
      - name: var-naming      # enforce Go naming conventions
      - name: unused-parameter

  exhaustive:
    default-signifies-exhaustive: true  # a default: case satisfies exhaustiveness

issues:
  exclude-rules:
    - path: _test\.go         # relax rules in test files
      linters:
        - wrapcheck           # test helpers don't need to wrap errors
        - gocritic            # test code style is more permissive
    - path: cmd/              # relax in main — it's a wiring file, not domain code
      linters:
        - wrapcheck
```

---

## Running

**Full module** (standard):
```bash
golangci-lint run ./...
```

**Single package:**
```bash
golangci-lint run ./internal/domain/order/...
```

**Show which linters are enabled:**
```bash
golangci-lint linters
```

**Auto-fix where supported** (formatting, some style issues):
```bash
golangci-lint run --fix ./...
```

---

## Per-rule tool mapping

Every linter maps to a specific rule from `go-guidelines`:

| Linter | `go-guidelines` rule enforced |
|---|---|
| `errcheck` | §7 — never ignore errors |
| `wrapcheck` | §7 — always wrap errors with context at package boundaries |
| `contextcheck` | §5 — propagate `context.Context` correctly |
| `gocognit` | §1 (via `base-guidelines` §4) — functions stay focused |
| `revive` (exported) | §1 — all exported symbols are documented |
| `revive` (var-naming) | `base-guidelines` §4 — names reveal intent |
| `staticcheck` | Broad — deprecated APIs, unreachable code, incorrect usage |
| `gocritic` | §8 — Go-specific code smells |
| `exhaustive` | §3 — all enum cases are handled explicitly |

---

## Interpreting output

Each finding includes: file path, line number, linter name, and message.

```
internal/app/order/service.go:54:9: error returned from external package is unwrapped: ... (wrapcheck)
internal/domain/order/order.go:23:1: exported function NewOrder should have comment (revive)
internal/handler/http.go:87:5: cognitive complexity 14 of function HandleCreate is high (gocognit)
```

**For each finding:**
1. Understand which `go-guidelines` rule it maps to (see table above).
2. Fix it — do not add `//nolint` without a written justification in a comment.
3. If a finding is genuinely a false positive, add `//nolint:lintername // reason` on the line.
   The reason is mandatory — a bare `//nolint` is not acceptable.

---

## CI integration

Run after `go vet`, before `go-govulncheck`:

```yaml
- name: Lint
  run: golangci-lint run ./...
```

Cache the lint cache in CI to keep runs fast:

```yaml
- name: Cache golangci-lint
  uses: actions/cache@v3
  with:
    path: ~/.cache/golangci-lint
    key: golangci-lint-${{ hashFiles('.golangci.yml') }}
```

---

## Quality gate result

| Outcome | Meaning | Action |
|---|---|---|
| No output, exit 0 | All linters passed | Proceed to `go-govulncheck` |
| Findings listed, exit 1 | Rule violations found | Fix all findings or add justified `//nolint` |
