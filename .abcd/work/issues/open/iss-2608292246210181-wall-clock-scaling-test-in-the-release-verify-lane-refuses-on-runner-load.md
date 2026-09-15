---
schema_version: 1
id: "iss-2608292246210181"
slug: "wall-clock-scaling-test-in-the-release-verify-lane-refuses-on-runner-load"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.6.9-release"
found_at: "internal/adapter/scanner/adjacency_test.go"
---

TestGallopingProbeCostIsLinearInLineLength in internal/adapter/scanner is a wall-clock scaling test (quadruple the line, require under 8x) that sits in release.yml's verify job, so a loaded shared runner can refuse a tag on timing alone: the v0.6.9 auto-release measured 8.4x (2.16s to 18.1s) after the same tree passed the merge-queue check leg, while the release machine measures 4.0x across three runs; the tag was already minted, so the refusal consumed nothing but needed a manual rerun. The scaling property should be asserted on operation counts or a deterministic probe budget rather than elapsed time, or the test excluded from the verify lane.
