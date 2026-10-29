---
schema_version: 1
id: "iss-2610262032283069"
slug: "ahoy-s-inventory-of-abcd-noindex-has-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 274 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:420"
remedy: "Add the after-update/ store (the person's answer to the setup offer per repository and release, and the news-read record) to the user-scope tree in 01-ahoy.md and to the mirrored inventory prose in 05-internals/03-configuration.md."
finding_ref: cfk-25f061260a8102c9
---

ahoy's inventory of ~/.abcd.noindex/ has no after-update store; update's news sub-verb writes under ~/.abcd.noindex/after-update/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:420` — "USER SCOPE — one per machine (machine-local state only)"
- End B: `.abcd/development/brief/04-surfaces/21-update.md:86` — "Two forms write, and only under `~/.abcd.noindex/after-update/`, never in the repository"

01-ahoy.md's user-scope tree, presented as the inventory of ~/.abcd.noindex/, lists no after-update/ directory. 21-update.md says two forms of the news sub-verb write under ~/.abcd.noindex/after-update/ (the person's answer to the setup offer for each repository and release, and the record that the news was read), and session start reads that store. Either the inventory is incomplete, or the store lives elsewhere.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 274.
