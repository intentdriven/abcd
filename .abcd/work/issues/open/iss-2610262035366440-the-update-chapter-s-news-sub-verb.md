---
schema_version: 1
id: "iss-2610262035366440"
slug: "the-update-chapter-s-news-sub-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 584 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md:87"
remedy: "none (filed automatically)"
finding_ref: cfk-2179252c4bdaeba8
---

The update chapter's news sub-verb writes under `~/.abcd.noindex/after-update/`; the configuration chapter's inventory of the user scope has no after-update store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/21-update.md:87` — "under `~/.abcd.noindex/after-update/`, never in the repository: one records"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:621` — "**User scope, `~/.abcd.noindex/`** — one per machine, machine-local shared state only: the"

21-update says two forms of the news sub-verb write, and only under `~/.abcd.noindex/after-update/`, and that session start reads that store. 03-configuration's enumeration of the user scope, which it calls the one list, names no `after-update/` store. The verb writes a store the inventory says is not there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 584.
