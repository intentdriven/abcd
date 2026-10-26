---
schema_version: 1
id: "iss-2610261617079475"
slug: "testastepwithnorunnamedaddressesthenewer"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lowcpu run 2026-10-25: CI on PR 1134"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_repace_test.go"
remedy: "First add a test that mints both runs in one second with the newer run's suffix the lower, and watch it fail. Then decide which run supersedes from the run record's own hand-back order or injected-clock start, not from id order."
---

TestAStepWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue failed once in CI's check (ubuntu-latest) job on PR 1134 and passed when the same tree was run again. Resolve reported two runs in progress, run-2610261029093535 and run-2610261029099737, so the handed-back run was not treated as superseded by the newer one. Both run ids were minted in the same wall-clock second, although the test's injected clock moves a minute between the two runs. So the ids come from the real clock, and the newer run is likely told apart by id order, which within one second is the random suffix.
