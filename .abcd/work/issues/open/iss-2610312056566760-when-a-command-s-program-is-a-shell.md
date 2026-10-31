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
remedy: "Keep failing closed for a program word the guard cannot resolve. Resolve a variable whose value a preceding segment of the same line assigns as a literal (A=/literal && $A ...), so the command is judged exactly as the spelled form; and when a refusal comes from a guessed reading of an unfixed program, say that the program name is unfixed and give the existing advice to spell it, rather than naming a script the command runs. Do not treat quoted words as data for an unknown program: that opens the guessed eval and sh -c readings."
---

When a command's program is a shell variable assigned a literal earlier on the same line (A=/path/to/scripts/abcd.sh && $A capture remedy iss-N "... (internal/core/guard/payload.go). Either ..."), the guard does not resolve the variable, so it treats the program as possibly eval and parses the quoted remedy text as shell: the parenthesised path becomes a subshell that runs that Go file, '. Either' becomes a source of a missing file, and the command is refused as script-runs-hazard on an 'rm -rf' string the file holds. For eval that reading is right, since quoted text passed to eval is code, so failing closed on an unknown program is correct; the defect is that a literal the same line assigns is not resolved, and the refusal names a script 'this command runs' rather than saying the program name is unfixed. Spelling the program (scripts/abcd.sh capture remedy ...) is allowed (reproduced with abcd guard check on 2026-10-31).
