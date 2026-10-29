---
schema_version: 1
id: "iss-2610262043562914"
slug: "itd-200-puts-the-status-line-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1172 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200, itd-77]
finding_ref: cfk-f3b8b2fdeefe123e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The two names are one folder before and after the rename."
---

itd-200 puts the status line's user-level defaults under `~/.abcd/`; itd-77 puts the user-level home at `~/.abcd.noindex/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override, and"
- End B: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "abcd's user-level home — `~/.abcd.noindex/` by default, the same dotfile convention"

itd-200 says the status line's defaults move into the user-level home under `~/.abcd/` at install time. itd-77 says the user-level home is `~/.abcd.noindex/` by default. The two records place the same user-level home in different directories.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1172.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The two names are one folder before and after the rename.
