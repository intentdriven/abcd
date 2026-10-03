---
schema_version: 1
id: "iss-2610031156150387"
slug: "batching-related-judge-items-into-one-request-beats-one"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "sibling-session measurement relayed 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi"
remedy: "When a delegated many-item step is routed to a provider, batch related items into one request up to the model's served context; where a request must be split, carry each piece's context (neighbouring lines, file kind, source); bound reasoning per item; always stream; and measure on the target model before choosing a per-item fan-out. Grounds: the 2026-10-03 measurement above, to be widened by the two further models before any default changes."
---

Batching related judge items into one request beats one request per item on a local OpenAI-compatible model service. A sibling session's measurement on 2026-10-03 (44 labelled lines of banned-term hits and near-misses, one local 8-bit model, streaming, temperature 0, max_tokens from the served context): all 44 in one request scored 41/44 in 181 s and 8,872 completion tokens; one request per item scored 30/44 in 2,487 s (14x) and 116,044 completion tokens (13x). One-by-one lost cross-item context (five false blocks on abcd's own documentation lines, two misses), let reasoning run away (three single items ran 566-589 s and hit max_tokens with no verdict), and paid a fixed overhead per request. abcd's delegated many-item steps (consistency, cold readings) are batch-shaped, so fanning them out per item to a local provider without their context is the losing shape. More data from two further 27B models in both modes is to follow.

## Evidence 2026-10-03 (second relay from the same sibling session)

The same 44-line test, streaming, temperature 0, `max_tokens` from the served context, on a local OpenAI-compatible service (unnamed here), across five models:

| model | one batch request | one request per item |
|---|---|---|
| an 8-bit flash chat model with reasoning | 41/44, 181 s | 30/44, 2,487 s; 3 cut off at the token limit |
| a 2-bit 27B chat model | 31/44, 274 s; 5 items silently skipped (finish reason stop, no error) | 38/44, 1,625 s |
| a 30B chat model with reasoning | did not finish in more than 50 min (stopped) | not run |
| an 8-bit 27B chat model with reasoning | did not finish in 64 min (stopped) | not run (projected hours) |
| a 4-bit decision model, served by its own reference server | 34/44 in 13 s | 31/44 in 25 s |

What it adds to the remedy: whether to batch depends on the model (the stronger model gained from cross-item context, the 2-bit model lost track of a long batch), so a batch is checked for completeness, every item answered, and a missing or failed item is re-sent alone with its context; a batch can end normally with fewer answers than items, and the sibling's own scorer first miscounted until unanswered items counted as misses; reasoning models may not converge on a long batch within an hour, so bulk judging prefers non-reasoning or decision models, or bounds reasoning per item; a decision model is a different cost class (about 14 times faster than the fastest chat judge here, no positives missed, but 10 to 13 false positives, and moving its cut-off traded errors about one for one); streaming is required for long runs (the adapter's streaming fix, captured the same day), and in one case a dropped stream left the server generating for minutes.
