---
schema_version: 1
id: "iss-2610262037542283"
slug: "the-load-check-intent-defines-a-stray-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 758 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:240"
remedy: "none (filed automatically)"
related_intents: [itd-2609231434459890]
finding_ref: cfk-23ef38553afbb1b8
---

The load-check intent defines a stray as a process outside abcd's lanes at near full CPU; the implement chapter defines it by a share of the process's fair share, a 0.4-core loop included, with nothing exempt by name.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:240` — "a lifetime CPU share of at least 0.9 of it makes a stray: forty busy loops on 16 cores, each at 0.4 of a core, are all strays"
- End B: `.abcd/development/intents/shipped/itd-2609231434459890-abcd-s-own-test-lanes-check-the-machine.md:70` — "a process outside abcd's own lanes at near full CPU for longer than the stray limit"

itd-2609231434459890 defines the stray trigger as a process outside abcd's own lanes at near full CPU for longer than the stray limit. 27-implement.md defines a stray as a process whose lifetime CPU share reaches 0.9 of its fair share of the machine as loaded, so forty loops at 0.4 of a core each on 16 cores are all strays, and it exempts nothing by name: abcd's own lanes are exempt only by time. The word stray names a different set of processes in each document.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 758.
