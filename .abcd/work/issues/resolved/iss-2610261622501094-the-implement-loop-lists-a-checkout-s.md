---
schema_version: 1
id: "iss-2610261622501094"
slug: "the-implement-loop-lists-a-checkout-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "Have readRuns order the runs by their created_at, falling back to the id only on a tie, and correct runIDs' comment that name order is mint order; prove it with a test that writes two runs whose ids sort against their start order."
resolution: "readRuns orders the runs by created_at, the id breaking only a tie, so two runs started in one second list in start order."
impact: fix
---

The implement loop lists a checkout's runs oldest first by sorting their ids, but a run id is the second it was minted in plus four random digits, so two runs started within one second come back in random order. superseded() then reads a handed-back run as the newer of the pair, both count as in progress, and a bare record read or step refuses with '2 runs are in progress'. TestARecordReadWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue failed this way in CI on 2026-10-26 on a records-only pull request (#1136).
