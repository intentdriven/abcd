---
schema_version: 1
id: "iss-2608292246210181"
slug: "wall-clock-scaling-test-in-the-release-verify-lane-refuses-on-runner-load"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.6.9-release"
found_at: "internal/adapter/scanner/adjacency_test.go"
resolution: "TestGallopingProbeCostIsLinearInLineLength counts the bytes the scan hands every probe and the junction search (a tallyMatcher through the existing matcher seam) instead of timing two scans, so machine load cannot move it: 4.12x at 320 units on the real code against a 6x bar, 10.72x with gallopingFind's budget removed."
impact: internal
resolved_by:
  commit: "0ab1cab13f885031da481066b114fd7a64c0ab9e"
---

TestGallopingProbeCostIsLinearInLineLength in internal/adapter/scanner is a wall-clock scaling test (quadruple the line, require under 8x) that sits in release.yml's verify job, so a loaded shared runner can refuse a tag on timing alone: the v0.6.9 auto-release measured 8.4x (2.16s to 18.1s) after the same tree passed the merge-queue check leg, while the release machine measures 4.0x across three runs; the tag was already minted, so the refusal consumed nothing but needed a manual rerun. The scaling property should be asserted on operation counts or a deterministic probe budget rather than elapsed time, or the test excluded from the verify lane.

## Grounds

- pursued: a byte count through the probe seam measures the quadratic class load-independently; it would be shown wrong by a regression that squares the scan without handing the probes more bytes, or by the count passing where the unbudgeted gallop is reintroduced
