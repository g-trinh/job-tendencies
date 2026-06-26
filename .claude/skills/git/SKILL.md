---
name: git
description: >
  Git commit convention shared by all software skills. Load this skill
  whenever you need to commit work.
---

# Git Skill

## Commit Convention

```
(feat|fix|infra|archi)(<slug>): <message>
```

| Type | Use when |
|---|---|
| `feat` | Adding a new feature or capability |
| `fix` | Fixing a bug or incorrect behaviour |
| `infra` | Infrastructure, tooling, CI/CD, configuration changes |
| `archi` | Architecture documents, ADRs, diagrams |

- **slug**: the ticket ID (`PROJ-123`) or short hyphenated description of the change (`order-splitting-support`).
- **message**: one sentence, imperative mood, lowercase after the colon, no trailing period.
- Never add a `Co-Authored-By` line or any co-author trailer.

## Examples

```
feat(order-splitting): add split payment use case
fix(auth-middleware): return 401 on expired token instead of 500
infra(ci): add govulncheck step to pipeline
archi(replace-legacy-auth): record ADR-003 for session token storage
```

## Commit Granularity

**One commit per task.** A task is either a single explicit developer request
from the user outside the workflow framework, or one task ID from a tech
breakdown (`docs/feature/<slug>/tech-breakdown-*.md`). Each task ID (e.g.
S2-BE-1, S7-BE-4) gets its own commit. Never bundle multiple tasks into one
commit, even if they belong to the same story. This makes review, revert, and
bisect possible at the task level.

## How to Commit

Stage only the files produced or modified by the current task — never use `git add -A` or `git add .`:

```bash
git add <file1> <file2> ...
git commit -m "(type)(slug): message"
```

Verify with `git status` after committing to confirm a clean state.
