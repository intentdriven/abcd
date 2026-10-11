---
schema_version: 1
id: "iss-2608220150157508"
slug: "local-token-usage-accounting-in-the"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcdev-site decision interview 2026-08-22"
found_at: "~/.abcd/history (user-level transcript store)"
related_intents: [itd-2609292107351737]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J4 of 2026-09-29: session token accounting in the history store joins the metering intent of ruling J3. Filed as draft itd-2609292107351737. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

Local-only token-usage accounting in the history store: at history capture time, extract per-message usage (input, output, cache-read, cache-write) and the model id from the transcript's own usage fields, store the aggregate with a timestamp in the stored transcript's metadata, and offer a report summing per session and per repo. Pricing is a user-level table (~/.abcd), recorded on explicit ask only: when a report meets a model with no recorded pricing it offers an explicit refresh verb in the docs-cite-refresh mould — never an implicit fetch (adr-38). Estimates are labelled API-equivalent (subscription seats do not bill per token; cache tiers price differently). Local-only by construction: never committed, never rendered on any site