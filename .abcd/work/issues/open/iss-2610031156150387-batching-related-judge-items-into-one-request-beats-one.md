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
