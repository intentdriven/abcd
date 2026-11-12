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
---

TestSynopsisScanIsLinearInThePage fails now and then on a slow merge-queue runner: it holds a scan to a CPU-time limit the runner missed on 12 Nov, which dropped an unrelated pull request from the merge queue with no notice; it passed on the second try.
