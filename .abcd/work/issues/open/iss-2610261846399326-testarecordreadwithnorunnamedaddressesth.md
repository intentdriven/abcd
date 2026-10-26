---
schema_version: 1
id: "iss-2610261846399326"
slug: "testarecordreadwithnorunnamedaddressesth"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-26 CI on PR 1137"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_readmit.go"
remedy: "Decide supersession by the run's own start time recorded in its state (or by the drain's lane order), not by the position its id gives it in Runs, and mint the run id from the Options clock so a test's clock orders its runs; keep the three readmission tests and add one that opens both runs in the same second."
---

TestARecordReadWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue (and its two siblings in drain_repace_test.go) fail about half the time: superseded() in drain_readmit.go treats a later entry of Runs as the newer run, and Runs is ordered by run id, but the readmittedRuns fixture opens both runs within one wall-clock second (its clock advances o.Now by minutes, which the run id does not read), so the two ids share their timestamp and order by the random suffix. CI failed PR 1137 on it on 2026-10-26 with run-2610261539014434 (newer) sorting before run-2610261539014869 (handed back): '2 runs are in progress'. In production two runs of one issue in one second are rare, so the defect is the ordering assumption more than the fixture.
