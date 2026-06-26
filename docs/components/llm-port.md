# Component: LLM Port

Domain port defining the two LLM use cases: board adapter generation and listing field extraction. No SDK import; the Anthropic implementation lives in `infra/llm`.

## Interfaces

### AdapterGenerator

```go
GenerateAdapter(ctx context.Context, boardURL string, exampleResponse string) (*AdapterSpec, error)
```

Produces a declarative `AdapterSpec` from a board URL + example page. The spec is data only — never executable code. Human approval required before the scraper evaluates it.

### ListingExtractor

```go
Extract(ctx context.Context, raw string) (*ExtractedListing, error)
```

Extracts structured fields from raw HTML/JSON. Each field carries a per-field `Confidence`; the listing carries an overall `Understanding` score.

## Data Types

### AdapterSpec

Declarative scraping configuration (fetch mode, URL template, param map, pagination, JSONPath/CSS selectors, incremental config). See `docs/architecture/pipeline.md §1`.

### ExtractedListing

Structured fields + confidence scores returned by extraction:

| Field | Type | Notes |
|-------|------|-------|
| `Skills` | `ExtractedField[[]string]` | |
| `RemotePolicy` | `ExtractedField[kernel.RemotePolicy]` | |
| `OfficeDays` | `ExtractedField[int]` | Days/week on-site |
| `ContractType` | `ExtractedField[kernel.ContractType]` | |
| `WorkingDays` | `ExtractedField[kernel.WorkingDays]` | |
| `SalaryMin/Max` | `ExtractedField[*int64]` | Whole euros; nil = absent |
| `Seniority` | `ExtractedField[kernel.Seniority]` | |
| `Recruiter` | `ExtractedField[*Recruiter]` | nil = not extractable |
| `Understanding` | `kernel.Understanding` | Top-level 0–100 score |

`ExtractedField[T]` is a generic wrapper `{Value T; Confidence kernel.Confidence}`.

## Notes

- Package path: `internal/domain/llm`
- Imports only `kernel` — no SDK dependency.
- Implementation: `internal/infra/llm.Client` (Anthropic Go SDK, prompt caching).
