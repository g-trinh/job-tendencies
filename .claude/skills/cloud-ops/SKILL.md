---
name: cloud-ops
description: >
  TRIGGER — load this skill BEFORE any cloud infrastructure work: requests to
  provision, change, or destroy cloud resources, write or edit OpenTofu/Terraform
  (.tf, .tfvars) files, design an `infra/` directory, add a dev or prod
  environment, create a shared infra module, or pick a cloud provider for a
  project. Trigger on phrases like: "set up infra for X", "create a bucket/
  database/queue on AWS|GCP|Azure", "write the terraform/opentofu for...",
  "add a staging/prod environment", "provision...", "what cloud provider should
  we use". Also load this skill after `software-development` and/or
  `frontend-development` finish a story's implementation tasks, to check
  whether `infra/` needs new or updated OpenTofu resources before the story is
  considered done. Covers: OpenTofu module design, dev/prod environment
  structure, state isolation, and a security posture that scales with MAU/MRR.
  Does NOT cover application code or architecture — those are
  `software-architecture` and `software-development`.
---

# Cloud Ops Skill

## Scope of This Skill

| In scope | Out of scope |
|---|---|
| OpenTofu module & environment design | Application code |
| Cloud provider selection (per project) | Application architecture (`software-architecture`) |
| Security posture scaled to MAU/MRR | CI/CD pipeline implementation (unless it's the IaC for it) |
| `infra/` repo layout, state isolation | Cost optimization unrelated to security |

---

## 0. Story-Completion Check (Workflow Trigger)

In the software bundle's task execution loop, this skill runs **after**
`software-development` and `frontend-development` have finished a story's
implementation tasks, and **before** the story is marked done. Goal: confirm
the cloud infra can actually serve what was just built.

1. Read the story's `docs/feature/<slug>/tech-breakdown-<story-id>.md` and the
   files changed for it. Look for new infra-relevant references introduced by
   the implementation: new env vars pointing at external services, new
   ports/listeners, new database tables/queues/topics/buckets, new third-party
   API calls that need credentials or egress rules.
2. Compare against `infra/modules/` and `infra/environments/{dev,prod}/` — does
   something already provide each of these?
3. **Nothing missing** — report "infra covers this story's needs, no changes."
   Done; story can be marked complete.
4. **Something missing** — follow the normal workflow (Sections 1–6 below) to
   add/extend the module(s) and wire them into both `dev` and `prod`. Then stop
   per Section 5 (present the plan, no `apply`) before the story is marked done.

---

## Input Resolution

Before doing anything else:

1. **Check for `infra/README.md`** at the repo root.
   - If it exists: read it. It records the chosen cloud provider, regions, state
     backend, and the project's current security tier (see Section 4). Treat
     these as binding decisions — don't re-litigate them without reason.
   - If it does not exist: run the **Provider Decision Process** (Section 1)
     before writing any `.tf` file.
2. **Check for `infra/modules/` and `infra/environments/`** — if they exist,
   follow the conventions already established there (naming, variable patterns,
   backend config) even if they diverge slightly from Section 2's defaults.
   Consistency with existing code wins over the default layout.

---

## 1. Provider Decision Process

The cloud provider is decided **once per project** and recorded in
`infra/README.md`. If that file doesn't exist yet:

1. Ask the user (one exchange, not a long form):
   - Existing cloud accounts / org agreements (AWS, GCP, Azure, OVH, Hetzner...)?
   - Team familiarity — is there a strong preference to avoid a learning curve?
   - Data residency / compliance constraints (region restrictions)?
   - Any service this project clearly needs that's much better on one provider
     (e.g. heavy use of a specific managed service)?
2. Recommend **one** provider with a one-line rationale. Don't present a long
   comparison matrix — pick the one that fits, state the trade-off, ask for
   confirmation.
3. Once confirmed, create `infra/README.md` using
   `references/template-infra-readme.md`. It must record:
   - Chosen provider, primary region(s), and why.
   - State backend choice (Section 3).
   - Current security tier per Section 4, based on the project's current MAU/MRR
     (ask if unknown — assume Tier 0 if pre-launch).
4. Commit `infra/README.md` on its own using the `git` skill before writing any
   module (`infra(<slug>): record cloud provider decision`).

> **Never pick a provider silently.** A provider switch later is expensive —
> this is a one-time decision the user must make explicitly.

---

## 2. Repository Layout & Module Conventions

```
infra/
├── README.md                 # provider decision, regions, state backend, security tier
├── modules/                   # shared, provider-specific building blocks
│   ├── network/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   ├── versions.tf
│   │   └── README.md          # inputs, outputs, what it provisions
│   ├── database/
│   └── ...
└── environments/
    ├── dev/
    │   ├── main.tf             # calls modules with dev-specific inputs
    │   ├── variables.tf
    │   ├── terraform.tfvars
    │   ├── backend.tf           # dev state backend (own state file)
    │   └── versions.tf
    └── prod/
        ├── main.tf
        ├── variables.tf
        ├── terraform.tfvars
        ├── backend.tf           # separate prod state backend
        └── versions.tf
```

### Module rules
- A module does **one thing** (network, database, compute, queue, CDN...). If a
  module's `main.tf` mixes unrelated resource families, split it.
- Every module has `variables.tf` (typed, with `description`), `outputs.tf`
  (only what consumers need), `versions.tf` (pinned provider version), and a
  `README.md` describing inputs/outputs in one table each.
- No environment-specific values (account IDs, CIDRs, instance sizes) hardcoded
  inside a module — they're inputs.

### Environment rules
- `environments/dev` and `environments/prod` are **thin** — they call shared
  modules and supply environment-specific `.tfvars`. They contain no resource
  logic that isn't also in a module, except environment-only glue (e.g. a dev
  seed bucket).
- Every new piece of infrastructure must be wired into **both** `dev` and
  `prod` (even if `prod` starts with smaller/cheaper sizing via `tfvars`) — an
  environment that only exists in one place is a gap, not a shortcut.
- Adding a third environment (staging, etc.) later means adding another
  directory under `environments/` that calls the same modules — never
  duplicating module code.

---

## 3. State Isolation

- **Separate state per environment** (separate backend config / state file or
  bucket key per `dev` and `prod`) — never a single shared state or Terraform
  workspaces for prod vs dev. A `tofu apply` mistake in dev must not be able to
  touch prod state.
- Remote backend always, even for dev (S3+lock table, GCS bucket, Azure storage
  container, etc., per the chosen provider) — no local state files committed.
- Record the backend choice and naming convention in `infra/README.md` so the
  next module follows it without re-deciding.

---

## 4. Security Tiering — Scaling with MAU/MRR

Security is a top priority, but every control has a cost (managed service fees,
ops overhead, latency). Don't build Tier 3 controls for a pre-launch product.
**Always start from the Tier 0 baseline — it's non-negotiable regardless of
scale.** Higher tiers are cumulative additions on top of it.

### Tier 0 — Baseline (always, every project, every environment)
- Encryption at rest and in transit for everything (default-on for most managed
  services — verify, don't assume).
- Least-privilege IAM: no `*` actions/resources, separate roles per
  module/service, no shared root/admin credentials in CI.
- Secrets in a managed secret store (provider's secrets manager / parameter
  store) — never in `.tf`, `.tfvars`, or state-adjacent plaintext.
- No database or storage with public/unauthenticated access by default.
- Audit logging enabled on the account/project (even if nothing reads it yet).

### Tier 1 — Early traction (MAU > ~1k or MRR > $0, paying customers exist)
Add to Tier 0:
- WAF or equivalent on public-facing endpoints.
- Network segmentation: data tier in private subnets, no direct internet route.
- Automated backups with tested restore, retention matching data sensitivity.
- Dependency/vulnerability scanning in CI for IaC and container images.

### Tier 2 — Growth (MAU > ~10k or MRR > ~$10k)
Add to Tier 1:
- DDoS protection on public ingress.
- Secrets rotation (automated) for credentials accessed by services.
- Centralized monitoring/alerting on security-relevant events (failed auth
  spikes, IAM changes, unusual data access).
- Multi-AZ (or equivalent) redundancy for prod data stores.

### Tier 3 — Scale / regulated (MAU > ~100k or MRR > ~$100k, or handling
regulated data — PII at scale, payments, health data)
Add to Tier 2:
- mTLS or equivalent for service-to-service traffic.
- Dedicated key management (HSM-backed / KMS with strict key policies).
- Continuous compliance monitoring (e.g. CIS benchmark scanning) and periodic
  external pen testing.

### Applying this
1. State the project's current MAU/MRR (ask if unknown; default to Tier 0 for
   pre-launch).
2. Record the resulting tier in `infra/README.md` along with the date.
3. When proposing a new resource, name which tier controls apply to it and
   implement those. If a higher-tier control would be relevant but the project
   isn't there yet, **note it** in `infra/README.md` under "deferred controls"
   rather than building it — and rather than silently skipping it.
4. **Revisit the tier** whenever the user reports a meaningful MAU/MRR change —
   update `infra/README.md` and flag any newly-applicable controls as a
   follow-up, don't retrofit them unprompted into an unrelated task.

---

## 5. Workflow

```
1. Input Resolution               → infra/README.md exists? read it / create it (Section 1)
2. Confirm security tier          → from infra/README.md, or ask + record (Section 4)
3. Scope the change                → which module(s) are new/changed?
4. Write/update module(s)          → main.tf, variables.tf, outputs.tf, versions.tf, README.md
5. Wire into environments/dev      → module call + tfvars
6. Wire into environments/prod     → module call + tfvars (sizing may differ)
7. Validate                        → tofu fmt -recursive, tofu validate (both envs)
8. Plan (dev first)                → tofu plan — review output with the user
9. Commit                          → load `git` skill, infra(<slug>): message
10. STOP                           → report what was planned; do NOT run `tofu apply`
```

> **Never run `tofu apply` (or any state-mutating command) without explicit,
> per-action user confirmation.** This provisions real, billable, often
> hard-to-reverse infrastructure. Planning, validating, and presenting the plan
> output is this skill's job — applying is the user's call.

---

## 6. Module Authoring Checklist

Before presenting a module as done:
- [ ] Single responsibility — one resource family per module.
- [ ] All environment-specific values are variables, none hardcoded.
- [ ] `versions.tf` pins the provider version (and OpenTofu `required_version`).
- [ ] Outputs expose only what other modules/environments actually consume.
- [ ] Module `README.md` lists inputs and outputs in tables.
- [ ] Tier 0 baseline controls applied (Section 4) — encryption, least-privilege
      IAM, no public access by default, secrets via secret store.
- [ ] Higher-tier controls applied per the project's recorded tier, or noted as
      deferred in `infra/README.md`.
- [ ] Wired into both `dev` and `prod` environments.
- [ ] `tofu fmt -recursive` and `tofu validate` pass for both environments.

---

## 7. What This Skill Does NOT Cover

| Concern | Covered by |
|---|---|
| Application architecture, bounded contexts | `software-architecture` |
| Application code | `software-development` |
| Commit conventions | `git` |
