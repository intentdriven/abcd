---
schema_version: 1
id: "iss-2610262041529246"
slug: "the-build-sequence-says-abcd-logbook-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1030 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:176"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-fdd1ddd2ae8d54c2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and a later record (the relocation noted in 04-universal-patterns.md) says which holds; no code writes to the old path."
---

The build sequence says .abcd/logbook/ is retired, with a detector that fails the build on it; itd-34 writes shape findings under .abcd/logbook/audit/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:176` — "`.abcd/logbook/` is a retired location, held retired by an armed detector"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "Findings live at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`."

01-build-sequence says .abcd/logbook/ is a retired location, and that an armed detector fails the build if non-test Go source names it. Run output belongs in the gitignored local tier. The shipped itd-34 places the shape-classification findings at .abcd/logbook/audit/shape-< ts>/report.{json,md}. Code built to that scope would trip the detector.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1030.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and a later record (the relocation noted in 04-universal-patterns.md) says which holds; no code writes to the old path.
