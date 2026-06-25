---
name: go-vet
description: >
  Go static analysis quality gate using go vet. Use as part of the pre-PR
  quality gate in a Go project. Catches real bugs and suspicious constructs
  that the compiler does not reject. Triggered automatically by
  software-development at step 10 (quality gate), or on demand when the user
  asks to check for bugs, run static analysis, or verify their code is correct.
---

# go-vet

`go vet` ships with the Go toolchain — no installation required. It runs a
suite of analysers that catch real bugs: things the compiler accepts but that
are almost certainly wrong. It is faster and more reliable than any third-party
linter for what it covers. Always run it before `golangci-lint`.

---

## What it checks

| Analyser | What it catches |
|---|---|
| `printf` | Mismatched `Printf`/`Sprintf` format verbs and argument counts |
| `shadow` | Variables that shadow outer-scope declarations unintentionally |
| `structtag` | Malformed struct field tags (e.g. broken JSON tags) |
| `unreachable` | Code after `return`, `panic`, or `goto` that can never execute |
| `unusedresult` | Calls to functions whose return values must not be discarded |
| `copylock` | `sync.Mutex` or similar passed by value instead of pointer |
| `assign` | Useless assignments (result immediately overwritten) |
| `atomic` | Incorrect use of `sync/atomic` (e.g. non-64-bit-aligned access) |
| `composites` | Composite literals that omit field names for exported types |
| `httpresponse` | HTTP response body not closed after use |
| `lostcancel` | `context.CancelFunc` not called, causing context leak |

---

## Running

**Full module** (standard — always use this):
```bash
go vet ./...
```

**Single package:**
```bash
go vet ./internal/domain/order/...
```

**With a specific analyser disabled** (rare — document why):
```bash
go vet -printf=false ./...
```

---

## Interpreting output

`go vet` prints the file, line, and a description for each issue. Every reported
issue must be fixed — there are no false positives worth ignoring. If a finding
looks wrong, read it carefully; `go vet` is almost always right.

Example output:
```
./internal/handler/order.go:42:3: Printf call has arguments but no formatting directives
./internal/domain/order.go:17:6: assignment to entry in nil map
```

---

## Common findings and fixes

**`lostcancel` — context leak:**
```go
// Bad
ctx, _ := context.WithTimeout(parent, 5*time.Second)

// Good
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()
```

**`copylock` — mutex copied by value:**
```go
// Bad — copies the mutex
func process(mu sync.Mutex) { mu.Lock() }

// Good — pass by pointer
func process(mu *sync.Mutex) { mu.Lock() }
```

**`printf` — mismatched format verb:**
```go
// Bad
fmt.Printf("order id: %d", order.ID) // ID is a string

// Good
fmt.Printf("order id: %s", order.ID)
```

---

## CI integration

Run immediately after the format check, before the full linter suite:

```yaml
- name: Vet
  run: go vet ./...
```

---

## Quality gate result

| Outcome | Meaning | Action |
|---|---|---|
| No output, exit 0 | No issues found | Proceed to `go-golangci-lint` |
| Issues listed, exit 1 | Real bugs or suspicious code found | Fix all findings before proceeding |
