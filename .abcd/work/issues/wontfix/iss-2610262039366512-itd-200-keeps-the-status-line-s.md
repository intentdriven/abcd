---
schema_version: 1
id: "iss-2610262039366512"
slug: "itd-200-keeps-the-status-line-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 872 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-9bf2be8f8bed0e49
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decisions 2 and 6 of the rename intent introduce the stop; itd-200 cannot have named them."
---

itd-200 keeps the status line's user-level defaults under `~/.abcd/`; the configuration chapter says that while anything stands at `~/.abcd` the status line, like every command, writes nothing

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override, and"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:704` — "While any entry stands there, a folder, a file or a link, every command, every hook and the status line write nothing and say the stop"

itd-200 says the status line's defaults move into the user-level home under `~/.abcd/` at install time, and it scopes 'the user-level defaults under `~/.abcd/`'. The configuration chapter says the user scope is `~/.abcd.noindex/` (holding `statusline.json`). It says any entry at the old `~/.abcd` stops every command, every hook and the status line. The two documents cannot both be right about where the status line's settings live.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 872.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decisions 2 and 6 of the rename intent introduce the stop; itd-200 cannot have named them.
