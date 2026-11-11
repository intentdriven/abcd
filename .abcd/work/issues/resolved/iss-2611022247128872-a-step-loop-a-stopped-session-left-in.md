---
schema_version: 1
id: "iss-2611022247128872"
slug: "a-step-loop-a-stopped-session-left-in"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "Record the run's live driver (session, pid, start time) and each in-flight launch with its key service in the run state, show them in implement status, and refuse a second driver while the first is alive."
related_intents: [itd-2611041549461012]
resolution: "The run now records its live driver (session, process and start time) and each runner launch with its route, and refuses a second driver while the first is alive. abcd implement resume lists them, and judges each runner agent by its own process from the launch ledger. Showing the same on implement status was not built; iss-2611080058096268 tracks it."
impact: fix
resolved_by:
  intent: "itd-2611041549461012"
  commit: "f175c4db5c64edfd20fd1f509472af560a6b8a1c"
---

A step loop a stopped session left in the background keeps stepping its run and launching agents on a key nobody watches, and a preflight it started is orphaned (its parent gone): on 2026-11-02 a successor found run-2611020739372478 mid-step with no way to learn from the run who was driving it, from which process, or which key its launches spent.
