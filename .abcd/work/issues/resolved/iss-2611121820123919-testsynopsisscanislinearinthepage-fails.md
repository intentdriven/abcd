---
schema_version: 1
id: "iss-2611121820123919"
slug: "testsynopsisscanislinearinthepage-fails"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "merge queue run of 12 Nov"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage_test.go"
remedy: "Measure the scan's growth by comparing two input sizes in the same run instead of an absolute time limit, so a slow runner changes both sides alike."
resolution: "The synopsis scan test now compares two page sizes in one run: it reads a page whose scan takes at least 10ms of CPU and one four times larger, five times each, and fails when the larger takes more than eight times the smaller, so a slow runner slows both alike."
impact: internal
resolved_by:
  commit: "1b3e533744e2bb85fce70b6f5fa7d37cfe93064e"
---

TestSynopsisScanIsLinearInThePage fails now and then on a slow merge-queue runner: it holds a scan to a CPU-time limit the runner missed on 12 Nov, which dropped an unrelated pull request from the merge queue with no notice; it passed on the second try.
