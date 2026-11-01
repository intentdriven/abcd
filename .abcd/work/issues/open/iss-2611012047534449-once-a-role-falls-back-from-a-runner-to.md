---
schema_version: 1
id: "iss-2611012047534449"
slug: "once-a-role-falls-back-from-a-runner-to"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "routing build agents to the claude runner, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Let implement step retry a role on a runner when its await was handed to the host by a fallback, or by a step before the route existed, and nothing has been received: --route on such a lane re-hands the role to the runner, and the run record names the retry. Record the runner's first stderr line in the fallback's detail, so 'Not logged in' or a limit reads as the cause rather than 'exited with status 1'."
---

Once a role falls back from a runner to the host, nothing can send it back to the runner, so a lane on the API key cannot recover from one failed start. On 2026-11-01 the claude runner exited with status 1 on lane 2 of run-2610312053481114: it had no API key in its environment, and claude --bare answered 'Not logged in', which the loop recorded only as 'exited with status 1'. The loop handed the implementer to the host session. After that, --route on the lane was refused, because the step hands no role out: a re-told await starts nothing, and --route is refused with --restart. The lane cannot be discarded, because it is not held. Lane 2 of run-2611010452297194, handed to the host before the person routed agent work to the API key, is stuck the same way. The same applies to a runner that hits a rate or spending limit mid-step: switching to a second key helps only the next step, not the lane that failed. Both lanes are parked with no agent.
