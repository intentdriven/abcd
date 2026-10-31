---
schema_version: 1
id: "iss-2610312056566760"
slug: "when-a-command-s-program-is-a-shell"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "recording review remedies under away mode"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "When the program word cannot be resolved (a shell variable, a command substitution), keep reading only the arguments the shell itself would execute as scripts, and treat a word inside quotes as data unless the guard knows the program runs it; keep refusing when the program is a known interpreter or shell given that path."
---

When a command's program is a shell variable (A=/path/to/scripts/abcd.sh && $A capture remedy iss-N "... (internal/core/guard/payload.go). Either ..."), the guard cannot tell what runs, treats each argument as a script the unknown program may run, finds a repository path inside a double-quoted argument, reads that Go source file as a shell script and refuses the command as script-runs-hazard on an 'rm -rf' string at its line 1034, noting that '. Either' names no file. The same command spelt scripts/abcd.sh capture remedy ... is allowed (reproduced with abcd guard check on 2026-10-31). A quoted argument is data to the program it is passed to, not a script the shell runs, so the refusal is a false positive that blocks recording any remedy which cites a guard source file.
