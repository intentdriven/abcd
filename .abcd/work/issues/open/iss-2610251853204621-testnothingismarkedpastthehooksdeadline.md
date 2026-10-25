---
schema_version: 1
id: "iss-2610251853204621"
slug: "testnothingismarkedpastthehooksdeadline"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "joint preflight of fix/unit-stage-budget, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_deadline_test.go"
remedy: "Read under a deadline wide enough for any machine, check that the notes carry the hook's deadline less promptWriteReserve, then move the notes' deadline into the past and mark. The past-deadline branch of MarkShown is then tested whatever the read costs, with no sleep."
---

TestNothingIsMarkedPastTheHooksDeadline (internal/core/claim) fails under -race on a loaded machine. It gives the whole status read a real 400ms deadline, but it tests only that a block marked after the hook's deadline keeps nothing. The read itself (git through the status state, then render) outran 400ms in the joint preflight's race stage on 2026-10-25, so PromptNotes returned 'status unavailable' and the test failed before it reached the marking. It is the same class as iss-2610241911001655: a load-dependent budget standing in front of the behaviour under test.
