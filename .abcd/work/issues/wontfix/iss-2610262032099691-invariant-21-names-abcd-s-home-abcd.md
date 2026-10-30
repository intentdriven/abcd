---
schema_version: 1
id: "iss-2610262032099691"
slug: "invariant-21-names-abcd-s-home-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 230 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:179"
remedy: "none (filed automatically)"
related_intents: [itd-2609231434459890]
finding_ref: cfk-6d01656d52bf9b27
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and the home-rename intent and ADR say which holds. The brief and the code use the renamed home."
---

Invariant 21 names abcd's home `~/.abcd.noindex`; itd-2609231434459890 keeps the load-check limits in a settings file under `~/.abcd/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "the `.noindex` suffix the home carries, `~/.abcd.noindex`"
- End B: `.abcd/development/intents/shipped/itd-2609231434459890-abcd-s-own-test-lanes-check-the-machine.md:78` — "Per-machine limits in a settings file under `~/.abcd/`"

The brief places abcd's home at `~/.abcd.noindex`. The load-check intent places the per-machine limits under `~/.abcd/`. A person tuning the limits from the intent would edit a folder the brief does not use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 230.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and the home-rename intent and ADR say which holds. The brief and the code use the renamed home.
