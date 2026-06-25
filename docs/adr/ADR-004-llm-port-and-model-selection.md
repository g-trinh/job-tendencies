---
ai_context:
  decision: "One llm port with two use cases; declarative adapters; Claude model selection with structured-output confidence/understanding"
  chosen: "domain llm port (AdapterGenerator + ListingExtractor); declarative AdapterSpec (selectors/JSONPath, never code); claude-opus-4-8 default, configurable to Sonnet 4.6 / Haiku 4.5; structured outputs carry per-field confidence + understanding"
  rejected: ["code-generation adapters (executable scraper code)", "free-text LLM output parsed heuristically", "hardcoding a single model"]
  must:
    - "LLM access goes through domain interfaces; SDK only in infra/llm"
    - "Adapters are declarative specs validated against a schema before approval; never executed code"
    - "Extraction uses structured output: each field {value, confidence:0..100} + top-level understanding:0..100"
    - "Model id is configurable; default claude-opus-4-8"
  must_not:
    - "exec/compile LLM-generated code"
    - "translate raw scraped data"
  parent: "docs/architecture/overview.md"
---

# ADR-004 — LLM port, declarative adapters, model selection

## Status
Accepted — 2026-06-25

## Context
The LLM is used twice: (a) generating a board scraping adapter from a user-supplied example
page, and (b) extracting structured fields from raw listings with confidence/understanding
scores. Default provider is Claude (Opus 4.8 / Sonnet 4.6 / Haiku 4.5). Extraction is
high-volume; adapter generation is rare. Executing LLM-generated scraper code would be a
security and maintenance hazard.

## Decision
Define one **`llm` port** in `internal/domain/llm` with two interfaces —
`AdapterGenerator.GenerateAdapter(boardURL, exampleResponse) → AdapterSpec` and
`ListingExtractor.Extract(raw) → ExtractedListing` — implemented once in
`internal/infra/llm` with the Anthropic Go SDK.

- **Adapters are declarative.** `GenerateAdapter` returns an `AdapterSpec` (fetch mode,
  URL template, `param_map`, pagination, JSONPath/CSS result fields, incremental config) —
  data, not code. It is validated against a schema and human-approved before going live; a
  generic crawler evaluates it. We never execute generated code.
- **Extraction uses structured outputs.** A JSON schema where each field is
  `{value, confidence:0..100}` plus a top-level `understanding:0..100`. These scores are
  produced by the model and persisted on the job.
- **Model selection.** Default `claude-opus-4-8` for both uses; the extraction model id is
  configurable to `claude-sonnet-4-6` / `claude-haiku-4-5` as a cost lever. Use prompt
  caching for the stable system prompt + schema; consider the Batch API for scheduled bulk
  extraction (latency not user-facing).

## Alternatives considered
- **Code-generation adapters** (LLM emits executable scraper code) — rejected: arbitrary
  code execution risk, requires recompiles, not meaningfully reviewable.
- **Free-text LLM output parsed heuristically** — rejected: brittle; structured outputs
  give typed fields and let confidence/understanding be first-class.
- **Hardcoding one model** — rejected: extraction volume makes cost/quality a tunable; keep
  the model id configurable.

## Consequences
- (+) Domain is isolated from the SDK; swapping models or providers is an infra change.
- (+) Declarative adapters are safe to run and reviewable at approval time.
- (+) Confidence/understanding are structured, enabling badges and threshold filters.
- (−) Self-reported confidence is a heuristic, not calibrated (logged in tech_debt.md).
- (−) Structured-output schemas must evolve carefully as extracted fields change.

## Implementation constraints
- DO route all LLM calls through the `llm` domain interfaces; keep the SDK in `infra/llm`.
- DO validate `AdapterSpec` against a schema before approval; keep it declarative.
- DO request structured output with per-field confidence + per-listing understanding.
- DO keep the extraction model id configurable (default `claude-opus-4-8`).
- DO NOT `exec` or compile LLM-generated code.
- DO NOT translate raw scraped data; structured fields are displayed in French by the
  frontend.
