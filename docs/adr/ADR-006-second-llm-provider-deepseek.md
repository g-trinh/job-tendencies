---
ai_context:
  decision: "Add DeepSeek as a second, env-selected LLM provider serving every LLM task alongside Claude; Claude remains the default"
  chosen: "global LLM_PROVIDER startup switch (claude default | deepseek), resolved by a factory in internal/infra/llm returning a Provider interface that satisfies all three ports; DeepSeek client co-located with claude.go to reuse the schemas, prompts and parsers verbatim; hand-rolled net/http OpenAI-compatible chat/completions with forced function-calling; pure-Go PDF-to-text for the text-only identity path"
  rejected: ["per-task provider routing or fallback", "branching the provider choice in each cmd/main", "a second copy of the extraction/identity/adapter schemas and parsers for DeepSeek", "an OpenAI SDK dependency (go-openai) for the DeepSeek transport", "forcing Claude for identity import while DeepSeek serves the rest"]
  must:
    - "One provider serves all three ports (AdapterGenerator, ListingExtractor, IdentityExtractor); no per-task routing and no fallback"
    - "Provider is chosen once at startup from LLM_PROVIDER; default is claude; config.Load validates it fail-fast"
    - "Selection lives in a single factory infra/llm.NewProvider; cmd/main binaries call the factory and never branch on provider"
    - "The DeepSeek client reuses the exact schemas (extractionProperties/Required, adapterSpecSchema, identityProperties/Required), system prompts, and parsers (parseExtractedListing, parseExtractedIdentity) from claude.go"
    - "Model id is env-configurable per provider (LLM_MODEL_ID for Claude, DEEPSEEK_MODEL_ID for DeepSeek)"
  must_not:
    - "route different LLM tasks to different providers, or fall back between providers"
    - "duplicate the extraction/identity/adapter JSON schemas or response parsers for DeepSeek"
    - "add an OpenAI SDK dependency for the DeepSeek transport"
    - "claim Claude-quality PDF fidelity on the DeepSeek identity path"
  parent: "docs/architecture/overview.md"
  extends: "ADR-004 (LLM port and model selection)"
---

# ADR-006 — Second LLM provider (DeepSeek) via a global env switch

## Status
Accepted — 2026-07-03

## Context
ADR-004 defined one `llm` port with three use cases — adapter generation, listing
extraction, and LinkedIn-PDF identity import — implemented once in `internal/infra/llm`
with the Anthropic Go SDK, and noted that "swapping models or providers is an infra
change". We now want a second provider, DeepSeek, as a cost/availability lever. The
product decision is fixed: **one provider serves every LLM task**, selected globally at
startup, with **Claude remaining the default**. There is no per-task routing and no
fallback. The DeepSeek "V4" model id is not yet known, so it must be parameterised.

DeepSeek's chat API is OpenAI-compatible and **text-only** — it cannot accept a PDF
natively the way Claude's document content block can. Since the same provider must serve
identity import, the DeepSeek path has to turn a LinkedIn PDF into text before the model
sees it.

## Decision

1. **Global env switch, resolved by a factory.** A new `LLM_PROVIDER` variable
   (`claude` default | `deepseek`) selects the provider once at startup.
   `config.Load()` validates it fail-fast (unknown value ⇒ error; `deepseek` ⇒
   `DEEPSEEK_API_KEY` and `DEEPSEEK_MODEL_ID` required). Selection lives in a single
   factory `internal/infra/llm.NewProvider(cfg, logger) (Provider, error)`. `Provider`
   is an interface in `infra/llm` embedding the three method sets; the returned value is
   assigned to the existing consumer ports (`domain/llm.AdapterGenerator`,
   `domain/llm.ListingExtractor`, `app/profiles.IdentityExtractor`), which it satisfies
   structurally. Both `cmd/api` and `cmd/extract-worker` drop their
   `infrallm.New(...)` line and call `NewProvider`; **neither main branches on the
   provider**.

2. **Behavioural parity by reuse, not reimplementation.** The DeepSeek client lives in
   the **same package** `internal/infra/llm` as `claude.go`, so it reuses the existing
   unexported schemas (`extractionProperties`/`extractionRequired`, `adapterSpecSchema()`,
   `identityProperties`/`identityRequired`), the system prompts, and the response parsers
   (`parseExtractedListing`, `parseExtractedIdentity`) verbatim. Only the transport
   differs. Identical schema + identical parser ⇒ identical `ExtractedListing`,
   `AdapterSpec`, and `ExtractedIdentity` regardless of provider.

3. **Hand-rolled OpenAI-compatible transport with forced function-calling.** The DeepSeek
   client issues a `net/http` POST to `/chat/completions`, forcing the tool via
   `tool_choice: {type:"function", function:{name:...}}`. The OpenAI `function.parameters`
   object is `{type:"object", properties, required}` — the existing property/required
   maps are dropped in unchanged. The tool-call arguments string is read as
   `json.RawMessage` and fed to the existing parsers. No SDK is added: the surface is one
   request/response type pair plus a shared `doChatCompletion` helper.

4. **Pure-Go PDF-to-text for the identity path.** Because DeepSeek is text-only, its
   `ExtractIdentity` extracts text from the PDF with a pure-Go extractor
   (`github.com/ledongthuc/pdf`, no cgo) and sends it as a text message using the same
   `identitySystemPrompt` and identity schema. This is the one place providers are not
   byte-identical: LinkedIn's multi-column PDF layout extracts with jumbled reading order,
   so identity quality is lower than Claude's native document block. Recorded in
   `tech_debt.md`.

5. **Claude-only features degrade gracefully.** Anthropic prompt caching (`cache_control`)
   and the Batch API cost lever (`LLM_BATCH_ENABLED`, ADR-004) are Claude-specific. The
   DeepSeek path simply omits them; DeepSeek performs context caching server-side.
   `LLM_BATCH_ENABLED` remains a no-op under DeepSeek.

## Alternatives considered
- **Per-task routing / fallback between providers** — rejected by product decision: one
  provider serves everything, chosen globally; routing and fallback add cost, config
  surface, and non-determinism for no current need (YAGNI).
- **Branch the provider choice in each `cmd/main`** — rejected: duplicates the switch
  across two binaries and leaks provider knowledge into the composition root. A single
  factory keeps one decision point.
- **A second copy of the schemas/parsers for DeepSeek** — rejected: two authoritative
  copies of the extraction/identity contract would drift; co-locating the client with
  `claude.go` reuses them (DRY on the domain contract).
- **An OpenAI SDK dependency (`go-openai`)** — rejected: the needed surface is one forced
  tool call; a hand-rolled `net/http` client avoids a dependency (and the `go get`
  network cost) for marginal convenience.
- **Force Claude for identity import, DeepSeek for the rest** — rejected: violates the
  "one provider serves every task" decision and reintroduces per-task routing.

## Consequences
- (+) Flipping providers is a single env var; the default stays Claude.
- (+) Extraction/adapter/identity behaviour is identical across providers by construction
  (shared schema + parser); only transport and PDF handling differ.
- (+) No new dependency for the DeepSeek transport; blast radius is one package plus two
  one-line `cmd` edits.
- (−) The identity path loses PDF fidelity under DeepSeek (jumbled multi-column text) —
  accepted and logged in `tech_debt.md`.
- (−) DeepSeek forgoes Anthropic prompt caching and the Batch lever; cost characteristics
  differ from the Claude path.
- (−) `ledongthuc/pdf` must be added via `go get` (network); a fully offline build
  sandbox blocks that one task (flagged for the implementer).
- (−) DeepSeek's forced named `tool_choice` may be less reliable than Anthropic's forced
  tool; a `tool_choice:"auto"` or `response_format:{type:"json_object"}` fallback is the
  contingency, kept out of the default path.

## Implementation constraints
- DO select the provider once at startup from `LLM_PROVIDER` (default `claude`); validate
  it fail-fast in `config.Load()`.
- DO put the selection in `infra/llm.NewProvider`; keep `cmd/api` and `cmd/extract-worker`
  free of any provider branch.
- DO place the DeepSeek client in package `internal/infra/llm` and reuse the existing
  schemas, prompts, and parsers unchanged.
- DO keep the model id env-configurable per provider (`LLM_MODEL_ID` for Claude,
  `DEEPSEEK_MODEL_ID` for DeepSeek, no default for the latter).
- DO add compile-time assertions that both `*Client` and the DeepSeek client satisfy
  `Provider`.
- DO NOT route different tasks to different providers, and DO NOT fall back between them.
- DO NOT duplicate the extraction/identity/adapter schemas or the response parsers.
- DO NOT add an OpenAI SDK for the DeepSeek transport.
- DO NOT present the DeepSeek identity path as equivalent in quality to Claude's native
  PDF ingestion.
