---
schema_version: 1
id: "iss-2610262043406581"
slug: "itd-1-calls-the-fidelity-review-sub-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1154 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-80]
finding_ref: cfk-b83a774f90f64cfa
---

itd-1 calls the fidelity-review sub-verb /abcd:intent audit; itd-80 ships it as abcd intent review ingest

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "a shipped intent's `## Audit Notes` is populated only when `/abcd:intent audit` is run by hand"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict"

Both documents describe the intent sub-verb that puts a fidelity verdict into a shipped intent's Audit Notes. itd-1 calls it /abcd:intent audit, while itd-80 ships abcd intent review ingest. That is two verb names for one act, so a command page or a builder that follows either record can name a verb that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1154.
