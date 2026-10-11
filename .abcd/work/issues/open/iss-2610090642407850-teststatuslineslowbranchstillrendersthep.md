---
schema_version: 1
id: "iss-2610090642407850"
slug: "teststatuslineslowbranchstillrendersthep"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_deadline_test.go"
remedy: "Reproduce TestStatuslineSlowBranchStillRendersThePartialRow under load with -count and -race, and give it the same bound the sibling fix used: prove the partial row rendered and the slow git was stopped, with a generous ceiling instead of the production deadline."
---

TestStatuslineSlowBranchStillRendersThePartialRow failed once at 0.34s in a local landing preflight under a load average above 50; a retry passed. It is the sibling of TestStatuslineSlowPreviousCommandIsStopped, whose wall-clock budget was fixed in iss-2610080243367006, but that fix did not reach this test, so the same family can still fail under a loaded host.
