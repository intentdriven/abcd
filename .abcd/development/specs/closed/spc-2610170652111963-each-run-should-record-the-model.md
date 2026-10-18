---
id: spc-2610170652111963
slug: each-run-should-record-the-model
intent: itd-166
origin: researcher-authored
production_mode: dictated-and-formatted
---
# each-run-should-record-the-model


## Summary

Delivers itd-166: every model call records the side abcd asked for and the side that answered, on the receipts that already exist, with every field a route cannot read named with its reason. Decisions 1–10 on the intent are the design record.

**Shape.** Two seams, one per path a call takes, sharing one vocabulary.

- **Shared vocabulary in `internal/core/oracle`** (the runner package already imports oracle; oracle imports nothing from runner): `Usage` (prompt, completion, reasoning, cache-read, cache-write tokens), `Cost` (a pointer, so zero is not confused with unreported, with its unit fixed per route), and `Unpinned` (field name and reason: `not_reported`, `self_reported`, `harness_computed`, `not_configured`) checked against the fixed asked and served lists of decision 10.
- **Provider adapter path** (`internal/adapter/openaiapi`, `internal/core/oracle`): the request adds `stream_options.include_usage` unless the provider's connection turns it off (`stream_usage: false`, default on), and a refusal's error names that setting; it is recorded as adapter-added, not as a setting sent. `Result`, `Usage` and the stream `chunk` gain the generation `id`, a top-level `provider`, `prompt_tokens_details.cached_tokens`, `completion_tokens_details.reasoning_tokens` and `cost`; the parser keeps reading past the finish reason to `[DONE]` for the usage chunk. Every provider-supplied string is cleaned and bounded in `admit`, as the reported model is. `complete()` hashes the rendered messages. `CallRecord` gains prompt sha256, provider, generation id, finish reason, usage, cost and the unpinned list; the harness hand-off marks `ReceiptRoute`'s served model self-reported. No request is added.
- **Runner path** (`internal/core/runner`): `Answer` gains usage, cost, served models and harness version, and its comment stops saying "or the one it was asked for". The claude runner reads the result event's `usage` and `total_cost_usd` (harness-computed), the served models from `modelUsage`'s keys (cross-checked against the init event's model), and `claude_code_version`; reasoning tokens are listed not reported. The opencode runner stops stamping the asked model (resolves iss-2610161709506207): served model empty and not reported, tokens and cost summed across `step_finish` events (cost harness-computed; sub-session events are already filtered out, so the totals are a floor and say so). `RouteRecord` gains the asked model, usage, cost, harness version and unpinned list; `Route.Model` is the canonical served model and `ReceiptRecord.Model` stays as the legacy copy.
- **Persistence**: no new file. Run state moves to schema 14 (a version-13 file reads as not recorded and is written back at 14; a file at 13 or lower carrying the new fields is refused). Built as 13; renumbered when release v0.15.1 took 13 for a drain lane's `refused`. `maskRoute` names every new string and slice. The consistency answer's `route.json` carries the oracle receipt, cleaned at the adapter.

**How each acceptance criterion is met.** (1) adapter both sides: step 1; (2) claude: step 2; (3) opencode: step 2; (4) reasons against fixed lists: steps 1 and 2; (5) host hand-off self-reported: step 1; (6) usage opt-out: step 1; (7) receipts, cleaning, no new file: steps 1 and 3; (8) no added request: step 1.

## Footprint

- packages: internal/adapter/openaiapi, internal/core/oracle, internal/core/runner, internal/core/implement/loop, internal/core/intent, internal/surface/cli
- tests: OpenRouter-shaped streamed and plain responses fill provider, generation id, finish reason, usage details and cost; a usage chunk after the finish reason is read; absent fields are listed with their reason; the request carries `stream_options.include_usage` unless the connection turns it off, and a refusing server's error names the setting; the request count is the same with and without the record; the prompt hash is stable and sensitive to one byte; control characters in provider, generation id and model are cleaned; a claude result event's usage, cost, `modelUsage` keys and version are read from a fixture taken from a real stream; an opencode stream's step-finish tokens and cost are summed and its served model is empty and not reported; the oracle's host hand-off marks the served model self-reported; state.json at 14 round-trips, at 13 reads as not recorded, and an older file carrying the new fields is refused; the mask covers each new field; drive passes the asked model and usage through

## Steps

1. The provider adapter records both sides
   - packages: internal/adapter/openaiapi, internal/core/oracle
   - tests: OpenRouter-shaped streamed and plain responses fill the served side; the usage chunk after the finish reason is read; absent fields listed with reasons; the usage request and its opt-out, with the refusal naming it; no added request; prompt sha256; provider strings cleaned and bounded; the host hand-off's served model self-reported
2. The runners record both sides
   - packages: internal/core/runner
   - tests: claude result usage, harness-computed cost, `modelUsage` served models and version read from a real-stream fixture, reasoning listed not reported; opencode step-finish tokens and cost summed, served model empty and not reported, never the asked model; the asked model kept beside the served one
3. The sides travel with the existing receipts
   - packages: internal/core/implement/loop, internal/core/intent, internal/surface/cli
   - tests: run state schema 14 with the version-13 rule; the lane receipt and the consistency route file round-trip the new fields with no new file; `maskRoute` covers each new string and slice; drive passes the asked model and usage through; the route line render names the served model and cost when present
