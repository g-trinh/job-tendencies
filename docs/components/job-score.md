# Component: Job Score

Fit score for a (job, profile) pair computed by the scoring pipeline. Produced after extraction; consumed by the job-browser and dashboard.

## Properties

| Property | Type | Description |
|----------|------|-------------|
| JobID | string | Scored job identifier |
| ProfileID | string | Profile the score was computed against |
| PassesDealbreakers | bool | False when the job fails at least one hard filter |
| WeightedScore | float64 (0–1) | Weighted preference aggregate |
| Breakdown | ComponentBreakdown | Per-component raw scores (0–1) |
| ScoredAt | time.Time | When this score was last computed |

## Breakdown Components

| Component | Default Weight | Score Logic |
|-----------|---------------|-------------|
| PreferredSkills | 40% | Fraction of preferred skills present (case-insensitive) |
| Salary | 25% | 1.0 when job.salary_min ≥ profile min; proportional otherwise; 0 if unknown |
| Location | 15% | 1.0 on case-insensitive substring match; 0 on mismatch |
| OfficeDays | 10% | 1.0 when job ≤ max; inverse-proportional when exceeded |
| WorkingDays | 10% | 1.0 on exact match; 0 on mismatch |

## Dealbreaker Gate

Applied before the weighted score. A job fails the gate when any of the following conditions set on the profile do not match:

- Contract type (exact match)
- Remote policy (exact match)
- Min salary — uses `job.salary_max`; nil salary fails the gate
- Required skills — all must be present (case-insensitive)

Jobs with `passes_dealbreakers = false` are excluded from the dashboard top-matches view.

## Notes

- Stored per `(job_id, profile_id)` in `job_score` (PK). Upsert semantics — re-scoring overwrites the prior result.
- `weighted_score` is always computed, even when `passes_dealbreakers` is false (allows offline analysis).
- Consumed by [dashboard](../feature/dashboard/feature.md) and the job-browser fit-score column.
