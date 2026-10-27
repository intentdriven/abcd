---
schema_version: 1
id: "iss-2610262036146367"
slug: "itd-86-defines-the-cold-reading-as-one"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 634 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:193"
remedy: "Supersede itd-86 by the shipped itd-183 (the four-position cold reading) or rewrite its press release and scope to what shipped, dropping the single-document premise and the landing of detections in the issue tier."
related_intents: [itd-86]
finding_ref: cfk-5606991aae838f69
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-86 defines the cold reading as one target document read against its own constraints; the reading chapter's detection position reads the shipped tree against the claim record, assembled at a commit from an include table.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:193` — "Where is the shipped tree in tension with the claim record?"
- End B: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29` — "A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone."

In 23-reading, the detection agent answers where the shipped tree is in tension with the claim record. Assembly takes only a position and a commit, and the reader gets what the committed include table admits. itd-86 instead scopes the cold reading as a surface that takes exactly one target document and detects only against constraints visible in that document. It also lands detections in the issue tier, while the chapter keeps reading records in the readings store, separate from the issue ledger. The two give incompatible objects for the same reading.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 634.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
