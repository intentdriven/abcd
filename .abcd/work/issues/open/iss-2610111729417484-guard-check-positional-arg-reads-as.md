---
schema_version: 1
id: "iss-2610111729417484"
slug: "guard-check-positional-arg-reads-as"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "security triage and fix run, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard.go"
remedy: "Accept one positional argument as the command line (the same as --command), or refuse a positional argument with a message naming --command and stdin and an exit code distinct from the unparsable-command exit 2; prove both with CLI tests, watched failing first"
---

`abcd guard check` given the command line as a positional argument (`abcd guard check 'git status'`) refuses with cobra's "unknown command … for abcd guard check" and exit 2, which the verb documents as 'a guard that cannot be evaluated'. A caller reading the exit code is told the command is unparsable, and the message does not mention --command or stdin, the two forms the verb takes. Hit on 2026-10-11 while reproducing guard advisories: every case returned exit 2 until the --command flag was found in the help.
