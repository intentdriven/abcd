---
schema_version: 1
id: "iss-2610262034543078"
slug: "itd-28-says-no-abcd-verb-files-a-dated"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 532 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:117"
remedy: "none (filed automatically)"
related_intents: [itd-28]
finding_ref: cfk-2e4f6b8b4abb2c4d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/consistency.go files the dated consistency folder under the reviews shelf with the pin, after itd-28's decision, which anticipated exactly such a writer."
---

itd-28 says no abcd verb files a dated review folder; the intent surface page has the consistency pass write a dated report under .abcd/work/reviews/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:117` — "No abcd verb files a dated review folder: the semantic-gate receipts are abcd's only writer under `.abcd/work/reviews/`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:622` — "The report lands at `.abcd/work/reviews/<date>-consistency/00-summary.md` for the corpus"

itd-28's decision 3 states that the semantic-gate receipts are abcd's only writer under .abcd/work/reviews/ and that no abcd verb files a dated review folder, so the dated folders are commissioned by hand. 05-intent.md says the consistency pass's report lands at .abcd/work/reviews/< date>-consistency/00-summary.md, written by its ingest with a review_of_commit pin. Either the consistency ingest is an abcd writer of dated review folders or no such writer exists; the shipped record's account of who writes the reviews shelf no longer matches the surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 532.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/consistency.go files the dated consistency folder under the reviews shelf with the pin, after itd-28's decision, which anticipated exactly such a writer.
