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
remedy: "Following dcn-2611012113432054: when a runner fails to start a role, pause the lane and say why, rather than handing the role to the host. Have each later implement step retry the paused role once on the runner, and let --route, or --restart while a route is in force, retry it at once. The run record names each retry. Record the runner's first stderr line in the pause's reason, so 'Not logged in' or a limit reads as the cause rather than 'exited with status 1'. A restart today never consults a runner, whatever the route."
---

Once a role falls back from a runner to the host, nothing can send it back to the runner, so a lane on the API key cannot recover from one failed start. On 2026-11-01 the claude runner exited with status 1 on lane 2 of run-2610312053481114: it had no API key in its environment, and claude --bare answered 'Not logged in', which the loop recorded only as 'exited with status 1'. The loop handed the implementer to the host session. After that, --route on the lane was refused, because the step hands no role out: a re-told await starts nothing, and --route is refused with --restart. The lane cannot be discarded, because it is not held. Lane 2 of run-2611010452297194, handed to the host before the person routed agent work to the API key, is stuck the same way. The same applies to a runner that hits a rate or spending limit mid-step: switching to a second key helps only the next step, not the lane that failed. Both lanes are parked with no agent.

The machine's configured route does not rescue such a lane either. Later the same day the person set roles.implementer.runner to claude in ~/.abcd.noindex/config.json. A peer session then ran implement step --restart on a lane that had fallen back (run-2611012037155265 lane 1), with the API key in the command's environment. The restart reset the lane from its last commit and handed the implementer to the host again, without asking the runner. So the restart path follows the recorded fallback, not the route in force, and run-2611012037155265 is parked as well.
