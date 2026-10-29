---
schema_version: 1
id: "iss-2610262032031758"
slug: "the-naming-register-retires-abcd-logbook"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 214 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:83"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-b224c9d3a5934614
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and a later brief ruling retired the name with a build-failing test; a shipped intent is history and the live drafts carrying the name are the ones to edit."
---

The naming register retires `.abcd/logbook/`; itd-34 puts the shape-classification findings under `.abcd/logbook/audit/`.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "Findings live at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`."

The reserved-vocabulary page says `.abcd/logbook/` is a retired maritime name that must not be re-minted, with run output going to the local ephemeral tier. itd-34 still names `.abcd/logbook/audit/shape-<ts>/` as the findings location. Following the intent would re-mint a name the register forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 214.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and a later brief ruling retired the name with a build-failing test; a shipped intent is history and the live drafts carrying the name are the ones to edit.
