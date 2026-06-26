# Component: Kernel (Shared Kernel)

Cross-cutting value objects, typed IDs, enums, domain errors, and pagination DTOs shared by all bounded contexts. No outward dependencies.

## Typed IDs

| Type | Aggregate |
|------|-----------|
| `ProfileID` | Search profile |
| `JobID` | Deduplicated job listing |
| `BoardID` | Job board source |
| `AdapterID` | Board scraping adapter spec |
| `RawListingID` | Captured raw listing |
| `ContactID` | Recruiter contact |
| `ScrapeRunID` | Pipeline execution |

## Value Objects

| Type | Description |
|------|-------------|
| `Money` | Immutable monetary amount (amountCents int64 + Currency ISO 4217). Default currency EUR. Constructed via `NewMoney` or `ParseMoney("60000 EUR")`. |
| `Confidence` | LLM per-field extraction confidence, uint8 in [0, 100]. |
| `Understanding` | LLM per-listing overall parse quality, uint8 in [0, 100]. |

## Enums

| Type | Values |
|------|--------|
| `ContractType` | `cdi`, `cdd`, `freelance`, `interim` |
| `RemotePolicy` | `on_site`, `hybrid`, `full_remote` |
| `WorkingDays` | `full_time`, `part_time`, `four_day` |
| `Seniority` | `entry`, `mid`, `senior`, `lead`, `exec` |
| `ApplicationStatus` | `saved`, `applied`, `interview`, `offer`, `rejected` |

All enums expose `Parse<Type>(s string)` returning an error for unrecognised values, and `.IsValid() bool`.

## Domain Errors

| Sentinel | HTTP mapping | Typed variant |
|----------|-------------|---------------|
| `ErrNotFound` | 404 | `*NotFoundError{Kind, ID}` |
| `ErrInvalidInput` | 400 | `*ValidationError{Field, Message}` |
| `ErrConflict` | 409 | — |
| `ErrUnauthorized` | 401 | — |

Use `errors.Is` / `errors.As` at handler boundaries. `ValidationError.Is` matches `ErrInvalidInput`; `NotFoundError.Is` matches `ErrNotFound`.

## Pagination

`PageFilter{Page int, Limit int}` with `Validate()`, `Offset()`, and `DefaultPageFilter()`. `SortOrder` with `ParseSortOrder("asc"|"desc")`.

## Notes

- Package path: `internal/domain/kernel`
- Zero external imports. Safe to import from any bounded context.
