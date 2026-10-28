---
schema_version: 1
id: "iss-2610262041346676"
slug: "itd-117-puts-the-user-scope-rules-file"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1009 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-1af3f7da08fde08a
---

itd-117 puts the user-scope rules file at `~/.abcd/rules.json`; the memory chapter puts the user scope under `~/.abcd.noindex/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json`"
- End B: `.abcd/development/brief/05-internals/07-memory.md:14` — "| **user** | `~/.abcd.noindex/memory/` |"

itd-117 says machine conventions now live once in `~/.abcd/rules.json`, the user scope that already exists for machine-local shared state. The memory chapter, describing the same two `.abcd/` scopes, locates the user scope under `~/.abcd.noindex/` (`~/.abcd.noindex/memory/`). The user-level home cannot be both directories, so a user following the shipped intent would write their rules where the loader does not look.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1009.
