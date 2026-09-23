---
id: itd-158
shipped_in: v0.6.8
slug: provenance-lacks-pass-b-exemption-marker
spec_id: spc-51
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-136]
impact: additive
---

# itd-88's fidelity gap audit found a missing press-release claim: Pass B is promised to ship as a declared exemption in _provenance.json, never a silent gap, but no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct — a promise with no implementing code, recorded in itd-88's Audit Notes (receipt rcp-4d07032fc6ab)

## Press Release

> _Seeded by promotion from iss-136. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-136`: itd-88's fidelity gap audit found a missing press-release claim: Pass B is promised to ship as a declared exemption in _provenance.json, never a silent gap, but no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct — a promise with no implementing code, recorded in itd-88's Audit Notes (receipt rcp-4d07032fc6ab). Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a lifeboat package in which Pass B is exempt, **when** its `_provenance.json` is written, **then** the Provenance record carries an explicit Pass-B exemption marker rather than a silent gap.
- **Given** a Provenance record that carries the exemption marker, **when** the consumer reads it, **then** it recognises the record as exempt rather than treating it as an unmarked gap.
- **Given** a Provenance record that carries no exemption marker, **when** the consumer reads it, **then** it is treated exactly as before, as an unexempt record.
- **Given** the Provenance struct in the lifeboat package, **when** it is marshalled, **then** the exemption field is present, closing the promised-but-unimplemented claim recorded in itd-88's Audit Notes.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-932e48c90725 -->
Fidelity review — receipt rcp-932e48c90725 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:72b196d1a941beb8669fa4e773b65cd82e89e4e8292dcfcfe9ff8c95bb3da0b9
Input attestations: diff:328a6755^1..328a6755 (PR #555) -- internal/core/lifeboat, judged against the tree at 0ab2ad02@sha256:5ac0608bc98792e9ccfeaaf2093d33bddd1bf9f629f0f9f054c5a6463c0608b5;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the provenance assembly sets PassBExemption from passBExemption(cov.TiersPresent, transcriptTiers), which declares the exemption with a reason for every package no transcript tier grounded; TestPlanDeclaresThePassBExemption reads _provenance.json back and asserts the field and a non-empty reason
  evidence: internal/core/lifeboat/plan.go:305 — "PassBExemption: passBExemption(cov.TiersPresent, transcriptTiers),"
  evidence: internal/core/lifeboat/plan.go:119 — "func passBExemption(present []Tier, transcript map[Tier]bool) *PassBExemption {"
  evidence: internal/core/lifeboat/plan_test.go:777 — "func TestPlanDeclaresThePassBExemption"
- ac-2 — MET_WITH_CONCERNS: readProvenance's Provenance flows into readCoverageHandoff, which copies the sanitised marker onto CoverageHandoff, and the render prints 'pass B (transcripts): declared exempt — < reason>' above the blanks; TestEmbarkReportsADeclaredPassBExemption asserts it; concern: recognition is record-level only — the blanks Pass B would have grounded are still listed for the human, where spc-51 said the section would not be enumerated among them (iss-2609231012180437)
  evidence: internal/core/lifeboat/embark.go:294 — "coverage: readCoverageHandoff(lifeboatAbs, prov.PassBExemption),"
  evidence: internal/core/lifeboat/embark.go:522 — "h.PassBExemption = &PassBExemption{Reason: sanitize(exempt.Reason)}"
  evidence: internal/core/lifeboat/embark_render.go:85 — "if ex := cov.PassBExemption; ex != nil {"
  evidence: internal/core/lifeboat/embark_test.go:1146 — "func TestEmbarkReportsADeclaredPassBExemption"
  evidence: internal/core/lifeboat/embark_test.go:1167 — "func TestEmbarkUnmarkedProvenanceReadsAsBefore"
- ac-3 — MET: the field is an omitempty pointer, so an unmarked record unmarshals to nil and the consumer prints no declaration; TestProvenanceWithoutTheExemptionIsUnchanged asserts nil plus byte-identical round-trip, and TestEmbarkUnmarkedProvenanceReadsAsBefore strips the marker from a packed lifeboat and asserts the report differs by the declaration line alone
  evidence: internal/core/lifeboat/plan.go:84 — "PassBExemption *PassBExemption `json:"pass_b_exemption,omitempty"`"
  evidence: internal/core/lifeboat/plan_test.go:799 — "func TestProvenanceWithoutTheExemptionIsUnchanged"
  evidence: internal/core/lifeboat/embark_test.go:1183 — "if plan.Coverage != nil && plan.Coverage.PassBExemption != nil {"
- ac-4 — MET: the Provenance struct carries the typed PassBExemption field and json.MarshalIndent at the plan site writes it into _provenance.json; the round-trip is asserted in both directions, and TestPassBExemptionStopsWhenATranscriptTierGrounds pins that the declaration stops being written once a transcript tier grounds a pack
  evidence: internal/core/lifeboat/plan.go:97 — "type PassBExemption struct {"
  evidence: internal/core/lifeboat/plan.go:306 — "pj, err := json.MarshalIndent(prov, "", " ")"
  evidence: internal/core/lifeboat/plan_test.go:832 — "func TestPassBExemptionStopsWhenATranscriptTierGrounds"

Gap audit:
- honoured:
  - Pass B ships as a declared exemption in _provenance.json, with a reason, never a silent gap
    evidence: internal/core/lifeboat/plan.go:305 — "PassBExemption: passBExemption(cov.TiersPresent, transcriptTiers),"
    evidence: internal/core/lifeboat/plan_test.go:777 — "func TestPlanDeclaresThePassBExemption"
  - an unmarked record round-trips byte-identically and reads as before
    evidence: internal/core/lifeboat/plan_test.go:799 — "func TestProvenanceWithoutTheExemptionIsUnchanged"
  - the untrusted reason is sanitised before it reaches a terminal
    evidence: internal/core/lifeboat/embark.go:522 — "h.PassBExemption = &PassBExemption{Reason: sanitize(exempt.Reason)}"
    evidence: internal/core/lifeboat/embark_test.go:1216 — "func TestEmbarkSanitisesTheExemptionReason"
- diverged:
  - the consumer reclassifies the Pass-B section as exempt rather than listing it among the coverage blanks — delivered as one record-level declaration line; the blanks listing is unchanged
    evidence: internal/core/lifeboat/embark_render.go:85 — "if ex := cov.PassBExemption; ex != nil {"
    evidence: internal/core/lifeboat/embark_test.go:1167 — "func TestEmbarkUnmarkedProvenanceReadsAsBefore"
- missing: (none)