---
schema_version: 1
id: "iss-2610262036341317"
slug: "itd-200-puts-the-status-line-defaults-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 658 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:67"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-80b006ef2c5ea834
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 11-history.md describes the state after the rename."
---

itd-200 puts the status-line defaults in the user-level home under ~/.abcd/; the history chapter calls ~/.abcd a legacy location

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:67` — "The defaults ship with abcd and move into the user-level home under `~/.abcd/` at install time"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:216` — "or a legacy `~/.abcd` standing with no plugin binary in the plugin root"

itd-200 says the status-line defaults move into the user-level home under ~/.abcd/ at install time. 11-history treats ~/.abcd as a legacy home the hooks report on, and the user-level home the brief chapters name is ~/.abcd.noindex/. The user-level home goes by two names, one of which the brief marks as no longer current.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 658.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 11-history.md describes the state after the rename.
