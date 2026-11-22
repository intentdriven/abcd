---
schema_version: 1
id: "iss-2611212039216889"
slug: "four-narrow-shapes-still-get-past-the"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Count SHELLOPTS exported with allexport as turning it on for the shells after it, carry allexport into a script's read when its shell was started with it, treat a set with an unfixed option as turning allexport on, and read an env assignment whose name is not fixed as naming every config variable"
resolution: "The export walk follows SHELLOPTS exported before a shell, carries the environment and allexport into a script the line runs, reads a set whose option words the line does not fix as turning allexport on, and reads an assignment whose name or value the line does not fix as setting the variable"
impact: fix
---

Four narrow shapes still get past the guard's agent-launch export checks: export SHELLOPTS=allexport before a shell; a script run by a shell started with -a, whose own read starts with allexport off; a set whose option name the line does not fix; and an env assignment whose name is joined to a command substitution, such as env "$(true)"NAME=value, which neither the claude nor the opencode config entry catches.
