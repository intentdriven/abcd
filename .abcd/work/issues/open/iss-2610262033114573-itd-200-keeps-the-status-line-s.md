---
schema_version: 1
id: "iss-2610262033114573"
slug: "itd-200-keeps-the-status-line-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 372 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:497"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-c7f9180f627b6f77
---

itd-200 keeps the status line's user-level defaults under `~/.abcd/`; the ahoy chapter keeps `statusline.json` in `~/.abcd.noindex/`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:497` — "statusline.json the status line's settings: the elements switched"
- End B: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override, and"

itd-200 says the status line's defaults move into the user-level home under `~/.abcd/` at install time, and lists the user-level defaults under `~/.abcd/` in its scope. The ahoy chapter's user scope is `~/.abcd.noindex/`, where `statusline.json` holds the status line's settings: the elements switched on and the command the harness ran before abcd took the row. One home has two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 372.
