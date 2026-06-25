---
name: go-goimports
description: >
  Go formatting quality gate using goimports. Use as part of the pre-PR quality
  gate in a Go project. Ensures all code is consistently formatted and import
  groups are correctly organised before any review happens. Triggered
  automatically by software-development at step 10 (quality gate), or on demand
  when the user asks to format code, check formatting, or fix imports.
---

# go-goimports

`goimports` is a strict superset of `gofmt`. It formats code identically to
`gofmt` and additionally organises import groups automatically. There is no
debate about formatting in Go — the tool decides.

---

## What it checks

- Consistent indentation, spacing, and brace placement (via `gofmt` rules).
- Import grouping: stdlib imports first, then external, then internal — each
  group separated by a blank line.
- Removal of unused imports.
- Addition of missing imports that can be inferred from usage.

---

## Installation

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

Verify: `goimports -version`

---

## Running

**Check only** (CI / quality gate — exits non-zero if any file needs formatting):
```bash
goimports -l ./.. | grep . && echo "formatting issues found" && exit 1 || exit 0
```

**Fix in place** (local development):
```bash
goimports -w ./..
```

**Single file:**
```bash
goimports -w path/to/file.go
```

---

## Editor integration

Configure your editor to run `goimports -w` on save. This should be the
default for any Go project — never commit unformatted code manually.

- **VS Code**: set `"editor.formatOnSave": true` and `"go.formatTool": "goimports"` in settings.
- **GoLand / IntelliJ**: enable *Reformat code on save* and set goimports as the formatter.
- **Neovim**: configure via `null-ls` or `conform.nvim` with the `goimports` source.

---

## CI integration

Add as the first step in the lint job — it is the fastest check and fails
immediately if code was committed without formatting:

```yaml
- name: Format check
  run: |
    goimports -l ./.. | grep . && echo "❌ Unformatted files found. Run: goimports -w ./..." && exit 1 || echo "✅ Formatting OK"
```

---

## Quality gate result

| Outcome | Meaning | Action |
|---|---|---|
| No output | All files are correctly formatted | Proceed to `go-vet` |
| File paths listed | One or more files need formatting | Run `goimports -w ./..`, re-check |
