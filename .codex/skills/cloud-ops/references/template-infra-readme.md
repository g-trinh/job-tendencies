# Infrastructure

## Cloud Provider

**Provider**: <AWS | GCP | Azure | ...>
**Primary region(s)**: <region(s)>
**Why**: <one-line rationale — existing org account, team familiarity, data residency, specific managed service needed, etc.>

## State Backend

**dev**: <backend type + location/key, e.g. S3 bucket `<name>`, key `dev/terraform.tfstate`>
**prod**: <backend type + location/key — must be separate from dev>

## Security Tier

**Current tier**: Tier <0-3> (set <date>)
**Basis**: MAU ~<value>, MRR ~<value>

Controls currently implemented: <list, or "Tier <N> baseline — see cloud-ops skill">

### Deferred controls

Controls from a higher tier that are relevant but not yet implemented, with the
trigger to revisit:

| Control | Tier | Revisit when |
|---|---|---|
| <e.g. DDoS protection on ingress> | <2> | <MAU > 10k> |

## Environments

| Environment | Purpose | Notes |
|---|---|---|
| dev | Development/testing | <sizing notes, e.g. smaller instances> |
| prod | Production | <sizing notes> |

## Modules

<Running list of modules in `infra/modules/`, one line each — name + what it provisions. Update as modules are added.>
