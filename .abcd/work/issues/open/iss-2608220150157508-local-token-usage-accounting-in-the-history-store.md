---
schema_version: 1
id: "iss-2608220150157508"
slug: "local-token-usage-accounting-in-the-history-store"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcdev-site decision interview 2026-08-22"
found_at: "~/.abcd/history (user-level transcript store)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan local token-usage accounting in the history store?"
---

Local-only token-usage accounting in the history store: at history capture time, extract per-message usage (input, output, cache-read, cache-write) and the model id from the transcript's own usage fields, store the aggregate with a timestamp in the stored transcript's metadata, and offer a report summing per session and per repo. Pricing is a user-level table (~/.abcd), recorded on explicit ask only: when a report meets a model with no recorded pricing it offers an explicit refresh verb in the docs-cite-refresh mould — never an implicit fetch (adr-38). Estimates are labelled API-equivalent (subscription seats do not bill per token; cache tiers price differently). Local-only by construction: never committed, never rendered on any site