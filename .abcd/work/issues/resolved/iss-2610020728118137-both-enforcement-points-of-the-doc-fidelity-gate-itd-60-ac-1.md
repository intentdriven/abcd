---
schema_version: 1
id: "iss-2610020728118137"
slug: "both-enforcement-points-of-the-doc-fidelity-gate-itd-60-ac-1"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-60"
origin: researcher-authored
production_mode: hand-written
remedy: "Run layer 1 at both enforcement points whatever the population (judge coverage with an empty population; keep layer 2's receipt requirement gated on a shipped intent, as the launch page already states), and add a release-cut test with no intent shipped and one uncovered surface that asserts the refusal. Grounds: spc-2609020903498198 'Layer 1 ... cheap enough to run at every enforcement point and in CI' and itd-60 decision 'every verb, sub-verb and agent the binary ships must have a brief chapter naming it'."
resolution: "docfidelity.Enforce runs layer 1 (the coverage floor) at both enforcement points whatever the population, a --remainder spec close and a release cut that ships no intent included; layer 2's saved docs review stays required only where an intent ships. The brief, spec close help and launch page say so."
impact: fix
resolved_by:
  commit: "c3a7d23fdf1e8b3f1218b5924af55eeedff1d8c5"
---

Both enforcement points of the doc-fidelity gate (itd-60, ac-1 and ac-4) skip the whole gate when the population is empty: enforceDocFidelity returns nil on len(population)==0 (internal/surface/cli/docfidelity.go:55) and the release emit runs docfidelity.Gate only inside 'if len(shipped) > 0' (internal/core/release/emit.go:272). So a spec close that ships no intent and, more consequentially, a release cut carrying only issue resolutions (a patch release) run neither layer, layer 1 included: a sub-verb or agent added by an issue fix in such a cut meets no coverage floor at the cut, while the spec says the deterministic floor 'is cheap enough to run at every enforcement point and in CI'. The bare 'abcd docs fidelity' with no --intent does run layer 1, so only the enforcement points have the gap.

## Grounds

- pursued: a patch cut, or a spec close shipping no intent, with a surface no brief chapter names now refuses doc-fidelity without asking for a docs review (TestEmitWithNoIntentShippedStillRefusesAnUncoveredSurface, TestSpecCloseWithARemainderRefusesAnUndocumentedSurface); shown wrong if such a cut or close proceeds, or if a fully covered one with no intent shipped demands a docs review (TestEmitWithNoIntentShippedJudgesNoPopulation, TestSpecCloseWithARemainderNeedsNoDocsReview).
