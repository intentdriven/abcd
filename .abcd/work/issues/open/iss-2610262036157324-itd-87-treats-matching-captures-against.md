---
schema_version: 1
id: "iss-2610262036157324"
slug: "itd-87-treats-matching-captures-against"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 635 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:314"
remedy: "none (filed automatically)"
related_intents: [itd-87]
finding_ref: cfk-ef72e8c0b2792391
---

itd-87 treats matching captures against dispositioned entries, not only open ones, as new work; the reading chapter says capture's filing-time match already compares with open and resolved issues.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:314` — "with the open and resolved issues, the intents and every earlier reading item"
- End B: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:31` — "Matching an incoming capture against **dispositioned** entries (`resolve`, `wontfix`), not merely against open ones."

23-reading says the ingest runs capture's filing-time match, with capture's threshold and configuration, comparing each item with the open and resolved issues, and that a confirmed recurrence is the disposition's recurs citation. itd-87 scopes as new work the matching of an incoming capture against dispositioned entries, not merely against open ones. Both cannot describe today's matcher: either it already reaches resolved records or it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 635.
