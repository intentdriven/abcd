---
schema_version: 1
id: "iss-2610262041342697"
slug: "itd-200-puts-the-status-line-defaults-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1010 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-1968af4def8b87a6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped rename intent says which holds and the code resolves the home through the one resolver."
---

itd-200 puts the status-line defaults in the user-level home under `~/.abcd/`; the memory chapter puts the user scope under `~/.abcd.noindex/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override"
- End B: `.abcd/development/brief/05-internals/07-memory.md:14` — "| **user** | `~/.abcd.noindex/memory/` |"

itd-200's decisions and scope move the status-line defaults, and the off switch, into the user-level home under `~/.abcd/`. The memory chapter locates the user scope under `~/.abcd.noindex/`. The two documents name different directories for the one user-level home, so the defaults and the switch an implementer or a user reaches for are in different places in each.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1010.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped rename intent says which holds and the code resolves the home through the one resolver.
