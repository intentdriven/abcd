---
schema_version: 1
id: "iss-2610262042030813"
slug: "the-memory-chapter-places-abcd-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1043 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-40]
finding_ref: cfk-80dc61e872dd5c53
---

The memory chapter places abcd's user-level home at ~/.abcd.noindex/; itd-40 names ~/.abcd/history/index.json as the sole user-scope registry.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:14` — "~/.abcd.noindex/memory/"
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:35` — "`~/.abcd/history/index.json` is the sole user-scope registry"

07-memory, like the rest of the brief, puts abcd's user-level stores under ~/.abcd.noindex/, the home the folder rename moved to. itd-40 still names the sole user-scope registry at ~/.abcd/history/index.json and bootstraps ~/.abcd/history/. The renamed home makes abcd stop until any old ~/.abcd is renamed, so the two documents give two names for the one user-level home, and only one of them can exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1043.
