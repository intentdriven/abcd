---
schema_version: 1
id: "iss-2611101319479162"
slug: "no-command-runs-a-lane-s-new-tests"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_narrow.go"
remedy: "Add a red-check mode to the narrow gate that runs the named new tests on a copy of the base tree with only the test files applied, and passes only if they fail there; a test runs it on a test that already passes at the base and finds it refused."
---

No command runs a lane's new tests against the code of the head before the change, so watching a new test fail first is done by hand every round. On 9 Nov at 07:05 a fix-round agent saved its fixed scripts aside and restored the committed versions with git show; on 8 Nov at 00:40 watching a test fail at the judged head meant making a copy with git archive; on 9 Nov lane agents removed their change, ran the new tests and put it back.
