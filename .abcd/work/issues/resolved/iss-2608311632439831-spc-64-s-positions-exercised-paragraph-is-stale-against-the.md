---
schema_version: 1
id: "iss-2608311632439831"
slug: "spc-64-s-positions-exercised-paragraph-is-stale-against-the"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "reviewing the evals audit fix round"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/closed/spc-64-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md"
resolution: "spc-64's Positions exercised decision now states what the eval asserts: all three assertions over every sentinel class at widening, entailment and detection, and comparative outside that set because it does not assemble at all — its object is the widening reading's pre-admission output, which no channel supplies, so the verb refuses (itd-199) and TestComparativeRefusesToAssemble holds the refusal and the absence of any artefact. The ac-4 mapping row carried the same phrasing and is corrected with it. Cites evals/coldreading_fixture_test.go (assemblingPositions, fullyAsserted) and evals/coldreading_test.go by repo-relative path. The eval is untouched."
impact: internal
resolved_by:
  commit: "4ac29c46"
---

spc-64's Positions exercised paragraph is stale against the eval it specifies. It says widening, entailment and detection are asserted in full and that at the comparative position the eval asserts only the read-block over the instrument's own stored prior outputs, making no claim about the in-cycle candidate set. The delivered eval asserts every sentinel class at every position, which the first review round required after finding that carving comparative out made an assembler leak at that one position invisible. The spec therefore understates its own delivery, and understating is not harmless here: a reader deciding what the eval covers would conclude that a comparative-only leak goes unnoticed, which is exactly the defect the round closed.

## Grounds

- pursued: a reader sizing up the eval's coverage from spc-64 must reach the same answer the eval's own position tables give; falsified if the spec's paragraph and the position sets in evals/coldreading_fixture_test.go disagree again, or if a reader concludes from it that some position carries less than the full assertion set for a reason other than refusing to assemble.
