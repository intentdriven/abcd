---
schema_version: 1
id: "iss-2610262033096678"
slug: "the-launch-chapter-says-the-repo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 366 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:489"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-b53564d053da2e42
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030345408. Same End A (04-launch.md line 489); End B is itd-24's own statement that phases are retired."
duplicates: [iss-2610262030345408]
---

The launch chapter says the repo organises work by phase; itd-24 records that phases are retired and the release is the unit

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:489` — "the repo organises work by **phase**"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

The launch chapter's versioning section says versions are an output and that the repo organises work by phase (adr-9). Its ownership note also places the release cut in Phase 1. itd-24 rules, citing adr-2609212115255771, that phases are retired, so its `<phase-id>` reads as the release tag and the release is the unit of work. One document treats 'phase' as a live organising unit and the other treats it as a retired word.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 366.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030345408. Same End A (04-launch.md line 489); End B is itd-24's own statement that phases are retired.
