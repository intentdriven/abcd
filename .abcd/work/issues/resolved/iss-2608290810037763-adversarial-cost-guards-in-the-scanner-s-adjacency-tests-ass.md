---
schema_version: 1
id: "iss-2608290810037763"
slug: "adversarial-cost-guards-in-the-scanner-s-adjacency-tests-ass"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: "internal/adapter/scanner/adjacency_test.go"
resolution: "The four remaining wall-clock cost guards in internal/adapter/scanner/adjacency_test.go (TestAdjacencyProbeStaysLinearOnLongLines, TestAdjacencyProbeWindowIsBounded, TestJunctionBacktrackIsBounded, TestGallopingProbeStaysBoundedOnLongLines) now assert the growth of the bytes the scan hands its probes when the input quadruples, a deterministic count, rather than elapsed seconds against a 15s ceiling. There is no stopwatch left in the file. Each guard was watched failing on a scratch archive with its bound removed (window 16.04x, backtrack 9.45x, long-line 3.85x against a flat 1.5x bar, galloping 16.17x) and passing on the real code. OVERLAP: the unmerged parked commit 25ce3984 on test/wall-clock-principle converts the same guards and also resolves this record. Whoever lands that parked sweep drops its adjacency_test.go, scanner.go and percent.go changes and this record's move as already delivered here."
impact: internal
resolved_by:
  commit: "3637c4cf"
---

Adversarial cost guards in the scanner's adjacency tests assert wall-clock seconds, which is a property of the machine rather than of the code, so they pass locally and fail on slower CI hardware under the race detector. One failed on both CI runners at 22.5 seconds against a 15 second bar while passing locally at 11.0 seconds on the same commit; the branch that introduced the probe and the merged tree measured identically, so it was not a regression. It was fixed for the costliest shape by shrinking that case's input, which is the lever the test file's own helper documents, but three sibling bars carry the same fragility and the same thin headroom. The machine-independent cost-CLASS guard beside them is the better model: it doubles the input and asserts the growth ratio, which no hardware difference can flip. Worth converting the remaining wall-clock ceilings to ratio assertions, or at least measuring a per-machine baseline and asserting a multiple of it.

## Grounds

- pursued: the guards pass or fail identically on an idle and a loaded machine, and each still refuses the regression it names. A guard that fails on the real code under load, or passes with its bound removed, would show this wrong.
