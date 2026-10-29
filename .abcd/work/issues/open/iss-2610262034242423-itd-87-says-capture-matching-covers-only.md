---
schema_version: 1
id: "iss-2610262034242423"
slug: "itd-87-says-capture-matching-covers-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 493 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:31"
remedy: "Rewrite itd-87's premise and scope against the shipped matcher (itd-2609212137116617): capture already compares open and resolved issue bodies and writes duplicates: or refines: links, never folding or dropping a capture. State the remaining delta only: wontfix records (not candidates today, per internal/core/capture/match.go) and a recurrence marker on the link."
related_intents: [itd-87]
finding_ref: cfk-518c8d5b111d4cbb
---

itd-87 says capture matching covers only open entries and folds recurrences away; the capture chapter says the match already covers resolved issues and never drops a capture.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:31` — "Matching an incoming capture against **dispositioned** entries (`resolve`, `wontfix`), not merely against open ones."
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:106` — "every open and resolved issue's body and every"

itd-87 sets out to start matching incoming captures against resolved and wontfixed entries, not only open ones. Its premise is that deduplication today folds a recurrence into the closed entry. 06-capture.md says the filing-time match already compares every open and resolved issue's body, writes at most a `duplicates:` or `refines:` link, and never refuses or drops a capture. The two documents describe the ledger's current behaviour differently.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 493.
