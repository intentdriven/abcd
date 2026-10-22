---
schema_version: 1
id: "iss-2610222031486962"
slug: "a-guard-warning-reaches-the-person-as-a"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "PT saw it in a session during the 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_hook_output.go"
remedy: "deliver a guard warn from the PreToolUse hook as exit 0 with the host's JSON output (hookSpecificOutput carrying the warning as the reason shown to the person and as context the agent reads), leading with the one-line fix before the explanation so a truncated render still says what to do; check the field names against the host's current hook documentation before building, and keep blocks on the existing blocking path"
---

A guard warning reaches the person as a host error and is cut short: a warn (for example unrecognised-launcher) leaves the PreToolUse hook on stderr with exit 1, which the host renders as 'PreToolUse:Bash hook error / Failed with non-blocking status code: Warning by the abcd guard (unrecognised-launcher): This command line starts with a program the guard does not recognise, and what follows it could not be read: Th…'. The person sees an error for what is only a warning, the text is truncated before it says what to do, and a non-blocking stderr message may not reach the agent that ran the command, so the warning changes nothing. Seen by the product thinker on 2026-10-22.
