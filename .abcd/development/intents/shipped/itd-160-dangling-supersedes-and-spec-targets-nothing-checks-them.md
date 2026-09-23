---
id: itd-160
shipped_in: v0.6.8
slug: dangling-supersedes-and-spec-targets-nothing-checks-them
spec_id: spc-52
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608220150157498
impact: additive
---

# Eight typed cross-references point at targets absent from the tree — adr-22 supersedes adr-14, adr-15 and adr-17; adr-25 supersedes adr-8; adr-27 supersedes adr-16; adr-28 supersedes adr-18; adr-35 supersedes adr-4 (all retired under retire-the-name); itd-3 names spec_id spc-1 which has no file — and nothing checks supersedes targets today. The 2026-08-21 site investigation counted six; the in-session grep found eight. The planned site build arms the detector via the .abcd/site-baseline.json ratchet, seeded with what the build finds, and the tombstones-or-stubs question (itd-136/itd-137) decides how they render

## Press Release

> _Seeded by promotion from iss-2608220150157498. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608220150157498`: Eight typed cross-references point at targets absent from the tree — adr-22 supersedes adr-14, adr-15 and adr-17; adr-25 supersedes adr-8; adr-27 supersedes adr-16; adr-28 supersedes adr-18; adr-35 supersedes adr-4 (all retired under retire-the-name); itd-3 names spec_id spc-1 which has no file — and nothing checks supersedes targets today. The 2026-08-21 site investigation counted six; the in-session grep found eight. The planned site build arms the detector via the .abcd/site-baseline.json ratchet, seeded with what the build finds, and the tombstones-or-stubs question (itd-136/itd-137) decides how they render. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a record that introduces a new `supersedes` reference naming a record absent from the tree, **when** the site-baseline reference detector runs, **then** it fails as a red gate.
- **Given** a record whose `spec_id` names a `spc-N` that has no file, **when** the reference detector runs, **then** it fails as a red gate.
- **Given** the existing backlog of dangling supersedes and spec-target references, **when** it is seeded into `.abcd/site-baseline.json`, **then** those references are baselined and do not newly fail the gate.
- **Given** a dangling reference that has been baselined, **when** its target is later added to the tree, **then** the detector still passes and the reference no longer counts as dangling.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-80f5e908a6e3 -->
Fidelity review — receipt rcp-80f5e908a6e3 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:151a233b1a269fb92a1dfffcc83626d1a1a4863ae9fe7d9338089d204d8042fa
Input attestations: diff:328a6755^1..328a6755 (PR #555, commit 34429a7f) -- internal/core/site check/recordjson/tests and .abcd/site-baseline.json, judged against the tree at 0ab2ad02@sha256:044fdbb79bee531c54777ef22799255016bfa2c069ea04bcf1d08472efe0054a;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: measureHealth counts a supersedes dangle even when its target reads as pruned, checkBaseline fails any live unresolved reference outside the committed baseline, and TestCheckRefusesANewSupersedesDangle asserts a CheckBaseline finding for a new supersedes naming an absent adr while the admitted backlog entry does not fail; site check is the site-render gate in make preflight
  evidence: internal/core/site/recordjson.go:429 — "if e.Field != "supersedes" && retired[e.To] {"
  evidence: internal/core/site/check.go:1401 — "c.fail(CheckBaseline, where, "","
  evidence: internal/core/site/check_test.go:660 — "func TestCheckRefusesANewSupersedesDangle"
  evidence: internal/core/site/check_test.go:708 — "func TestSupersedesToAPrunedRecordStillCounts"
- ac-2 — MET: a spec_id graph-field dangle reaches Health.Unresolved through the same walk and fails through the same c.fail; TestCheckRefusesADanglingSpecTarget writes an intent whose spec_id names spc-404 and asserts the CheckBaseline finding
  evidence: internal/core/site/check_test.go:674 — "func TestCheckRefusesADanglingSpecTarget"
  evidence: internal/core/site/recordjson.go:425 — "for _, e := range graph.Dangling {"
  evidence: internal/core/site/check.go:1401 — "c.fail(CheckBaseline, where, "","
- ac-3 — MET: the committed .abcd/site-baseline.json carries the eight backlog entries (adr-22 to adr-14/15/17, adr-25 to adr-8, adr-27 to adr-16, adr-28 to adr-18, adr-35 to adr-4, itd-3 to spc-1); checkBaseline skips a live reference the baseline admits, and TestCheckPassesABaselinedBacklog asserts a clean CheckBaseline over both field kinds when admitted
  evidence: .abcd/site-baseline.json:3 — ""unresolved_references": ["
  evidence: .abcd/site-baseline.json:11 — "{"from": "itd-3", "to": "spc-1"}"
  evidence: internal/core/site/check.go:1391 — "if admitted[key] {"
  evidence: internal/core/site/check_test.go:686 — "func TestCheckPassesABaselinedBacklog"
- ac-4 — MET: a baseline entry whose reference now resolves is a c.note inviting the baseline to shrink, never a failure; TestCheckPassesWhenABaselinedTargetArrives writes the missing adr-2 back and asserts the gate passes
  evidence: internal/core/site/check.go:1415 — "c.note(CheckBaseline, where,"
  evidence: internal/core/site/check_test.go:773 — "func TestCheckPassesWhenABaselinedTargetArrives"
  evidence: internal/core/site/check_test.go:570 — "func TestCheckInvitesAShrinkingBaseline"

Gap audit:
- honoured:
  - a dangling supersedes or spec target fails the site build as a red gate, ratcheted against a committed baseline
    evidence: internal/core/site/check.go:1401 — "c.fail(CheckBaseline, where, "","
    evidence: internal/core/site/check_test.go:660 — "func TestCheckRefusesANewSupersedesDangle"
  - the existing backlog is seeded into the baseline and does not newly fail
    evidence: .abcd/site-baseline.json:3 — ""unresolved_references": ["
    evidence: internal/core/site/check_test.go:686 — "func TestCheckPassesABaselinedBacklog"
  - the detector is armed by tests for all four ratchet behaviours; the detector itself and the seeded baseline predate PR #555 (baseline landed in 2f8fd254), so the delivery is the arming, not a new mechanism
    evidence: internal/core/site/check_test.go:708 — "func TestSupersedesToAPrunedRecordStillCounts"
    evidence: internal/core/site/check_test.go:773 — "func TestCheckPassesWhenABaselinedTargetArrives"
- diverged: (none)
- missing: (none)