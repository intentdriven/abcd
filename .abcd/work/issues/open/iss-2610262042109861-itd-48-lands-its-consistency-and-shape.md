---
schema_version: 1
id: "iss-2610262042109861"
slug: "itd-48-lands-its-consistency-and-shape"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1051 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-0efead12ebbe0746
---

itd-48 lands its consistency and shape reports under `.abcd/logbook/audit/`; the build sequence says `.abcd/logbook/` is retired and guarded by a detector.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:77` — "Reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:176` — "`.abcd/logbook/` is a retired location, held retired by an armed detector"

itd-48 sends Role 2 reports to .abcd/logbook/audit/consistency-< ts>/ and Role 3 reports to .abcd/logbook/audit/shape-< ts>/. The build sequence says .abcd/logbook/ is a retired location, held retired by an armed detector that fails the build if any non-test Go source under internal/ names it, and that runtime output goes to the gitignored local tier (iss-73). Building the report path itd-48 states would trip that detector.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1051.
