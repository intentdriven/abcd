---
schema_version: 1
id: "iss-2609240203462704"
slug: "the-scanner-s-count-based-cost-guards-measure"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/adjacency_test.go"
deferred_after: "v0.9.0"
deferral_reason: "Counting at the scanLine level needs a production seam inside scanText that this test-only lane did not add; the uncovered stages were not guarded by any bound before either, only timed, so the seam is taken as its own lane next cycle."
---

The scanner's count-based cost guards measure scanAllPatterns alone, so the rest of the per-line scan is outside every cost guard. probeWork (internal/adapter/scanner/adjacency_test.go) tallies only the bytes handed to the adjacency probes and the junction probe inside scanAllPatterns; the wall-clock guards it replaced timed scanLine, which is the whole of scanText. Uncovered stages: (1) each pattern's Skip and SkipAt callbacks, run per match against the matched text and the whole line; (2) the identity matchers (matchers.findings), run over the raw line and again over the decoded copy; (3) the percent-decode pre-pass (decodedLineFindings: percentDecodeBounded, the second scanAllPatterns run over the decoded copy, mapDecodedSpan); (4) the top-level cp.Re.FindAllStringIndex pass per pattern, which is RE2-linear but untallied. A regression that makes any of these superlinear passes every guard in the package. Counting them at the scanLine level needs a seam in scanText, which builds its probes internally and constructs the identity matchers from a fixed type with no hook for a counting stand-in, so it is a production change rather than a test-only one; hence captured, not added, in the gallop lane.
