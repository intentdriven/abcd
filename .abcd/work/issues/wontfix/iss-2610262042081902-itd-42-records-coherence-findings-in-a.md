---
schema_version: 1
id: "iss-2610262042081902"
slug: "itd-42-records-coherence-findings-in-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1048 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:61"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-c0181c2575ddcb9f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped pre-pass writes a planning brief only (prepass.go), the record's banner re-scopes it, and its Gap audit lists the grill-report claim as diverged."
---

itd-42 records coherence findings in a grill report and a PRD behind the GR002 gate; the verification matrix says no grill report, no PRD and no GR* code exist.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:61` — "Coherence questions and grounded conflicts are recorded in the Phase 1 `grill-report.{json,md}` alongside the existing question stream"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:78` — "`intent` registers no `grill` sub-verb, no PRD or grill report is written anywhere"

itd-42 writes coherence questions and grounded conflicts into the Phase 1 grill-report.{json,md}, has the Phase 2 PRD reflect them, and makes the full grill the default at 'the existing GR002 no-PRD gate'. The matrix's Grill row says intent registers no grill sub-verb, no PRD or grill report is written anywhere, and no GR* code exists in any tree. The artefacts itd-42 delivers into, and the gate it relies on, are ones the matrix says are absent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1048.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped pre-pass writes a planning brief only (prepass.go), the record's banner re-scopes it, and its Gap audit lists the grill-report claim as diverged.
