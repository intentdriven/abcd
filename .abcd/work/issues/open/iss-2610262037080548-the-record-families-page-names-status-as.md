---
schema_version: 1
id: "iss-2610262037080548"
slug: "the-record-families-page-names-status-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 701 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:28"
remedy: "none (filed automatically)"
finding_ref: cfk-2d26a5b9c039d8bf
---

The record-families page names status as a computed Now / Next / Later view; the announce chapter names a stored per-person branch of announcements status.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:28` — "Now / Next / Later, rendered from the shelves, the gate and the build's state"
- End B: `.abcd/development/brief/04-surfaces/38-announce.md:39` — "The announcement is one empty-tree commit on the sender's own status branch"

record-families.md defines the status family as Now / Next / Later, rendered from the shelves, the gate and the build's state: a view that is computed and never stored. 38-announce.md calls each person's branch of announcement commits on origin their status branch, read by `abcd status`. One name covers a computed view of the record and a stored message channel.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 701.
