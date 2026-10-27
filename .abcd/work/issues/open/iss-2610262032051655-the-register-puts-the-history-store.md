---
schema_version: 1
id: "iss-2610262032051655"
slug: "the-register-puts-the-history-store"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 219 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:227"
remedy: "none (filed automatically)"
related_intents: [itd-40]
finding_ref: cfk-bb972efacb3acda2
---

The register puts the history-store registry at `~/.abcd.noindex/history/index.json`; itd-40 puts it at `~/.abcd/history/index.json`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:227` — "History-store registry at `~/.abcd.noindex/history/index.json` recording each repo's identity + lineage"
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:35` — "`~/.abcd/history/index.json` is the sole user-scope registry"

Both documents describe the one user-scope registry keyed on the root commit, but they give it different homes: the brief under `~/.abcd.noindex/`, itd-40 under `~/.abcd/`. Folder classification built from itd-40 would read and bootstrap a path the brief does not use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 219.
