---
schema_version: 1
id: "iss-2608311459220516"
slug: "the-read-block-eval-s-coverage-matrix-does-not-name-six-of-t"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-186 fidelity audit rcp-e3f354adbbee"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_coverage_test.go"
resolution: "The coverage matrix names the six absent rules, and the corpus falsifies all six: a match-grain plant at two grains, a subsection under an excluded heading, a block scalar under an excluded key, a differently-cased denied path component, and a refusal corpus that reaches the exclusion floor's fail-closed half."
impact: internal
---

The read-block eval's coverage matrix does not name six of the assembler's rules, and the omission is worse than the limit the matrix discloses: the rules are absent rather than rewritten, so the declared-gap discipline never engaged on them. Two matter. The file-grain of positive inclusion, the per-row Match list, is named by no row and declared no gap; forcing the match to always succeed leaves the fixture manifests byte-identical and the lane green, and on a corpus holding one unmatched-extension file inside an admitted source it admits two extra items and leaks both tokens. Worse, the redaction verifier -- around 130 lines and six refusal paths, the fail-closed half of the key-and-heading floor -- can have its call DELETED OUTRIGHT with the lane still green. It is falsifiable in principle but not against this corpus: with a setext-underlined excluded heading in a record that travels whole, the shipped binary exits 2 naming the heading and the same binary without the verifier exits 0 with the token in the bundle. Four more mechanisms are green under mutation and unnamed: the section-span same-level rule, the block-scalar continuation drop, the case-insensitivity of the deny segments, and the same-rendering check. The fix is additive, six rows with at least two of them gaps.

## Grounds

- pursued: a rule enforced by refusing needs a shape to refuse, so the corpus grows one rather than the matrix growing a gap; wrong if a mutation of any of the six leaves the lane green
