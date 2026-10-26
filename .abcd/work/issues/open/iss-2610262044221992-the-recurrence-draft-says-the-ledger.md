---
schema_version: 1
id: "iss-2610262044221992"
slug: "the-recurrence-draft-says-the-ledger"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1202 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-87, itd-2609212137116617]
finding_ref: cfk-043addd1e460ec69
---

The recurrence draft says the ledger folds a match into the closed entry, but the shipped matcher always writes a new record and links it as a duplicate

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17` — "the ledger no longer quietly folds it into the closed entry"
- End B: `.abcd/development/intents/shipped/itd-2609212137116617-a-new-capture-or-draft-is-matched.md:40` — "a likely match is written onto the new record as `duplicates:` (near-identical) or `refines:` (narrower), naming the candidate"

itd-87's premise is that today a finding matching a resolved or wontfixed entry is folded into the closed entry and its evidence destroyed. itd-2609212137116617 compares new text with open and resolved issues and writes a new record carrying a `duplicates:` or `refines:` link, never dropping anything. The draft describes current behaviour wrongly, and its recurrence-versus-duplicate distinction falls on the link vocabulary the shipped record defines.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1202.
