---
schema_version: 1
id: "iss-2610262037469144"
slug: "itd-200-places-the-user-level-defaults"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 748 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-5e5ef172c5de7e5f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped rename intent says which holds and the code resolves the home through the one resolver."
---

itd-200 places the user-level defaults under ~/.abcd/; 34-build reads the machine layer from ~/.abcd.noindex/config.json

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:227` — "the same keys in `~/.abcd.noindex/config.json`"

Shipped itd-200 says its status-line defaults and off switch live in the user-level home under ~/.abcd/. 34-build names the machine layer of the same layered configuration as ~/.abcd.noindex/config.json. Two different paths are given for the one user-level home, so it is unclear where the user-level setting itd-200 promises is read from.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 748.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped rename intent says which holds and the code resolves the home through the one resolver.
