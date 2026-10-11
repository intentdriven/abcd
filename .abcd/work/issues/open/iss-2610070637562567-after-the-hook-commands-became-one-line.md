---
schema_version: 1
id: "iss-2610070637562567"
slug: "after-the-hook-commands-became-one-line"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the hook-message fix of 2026-10-07 (PR #837)"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Measure first: in a lab, run one PreToolUse hook that denies through permissionDecision and one that exits 2, on the current Claude Code, and record what the person and the model see; if the structured deny shows the reason without the error framing, move abcd guard hook's block to it (keeping exit 2 for hosts that do not read the JSON), with the exit-code fence tests updated."
---

After the hook commands became one-line wrappers, a guard refusal still reaches the person as 'PreToolUse:<tool> hook error: [sh "${CLAUDE_PLUGIN_ROOT}/hooks/pre-tool-use.sh"]: <message>', framed as an error, because the guard blocks by exiting 2 with its message on stderr. Claude Code's hooks reference documents a structured alternative for PreToolUse: exit 0 with hookSpecificOutput.permissionDecision "deny" and permissionDecisionReason on stdout, which blocks the call (a question included) and hands the model only the reason; the command echo in front of an exit-2 message is not documented at all. Unverified here: the guide that reported it worked from summarised documentation.

Measured 2026-10-07 on Claude Code 2.1.292, in live interactive sessions in a scratch repository whose only hook was a project-level PreToolUse hook, for the shell tool and the question tool alike:

- Exit 2 with the reason on stderr shows the person `Error: PreToolUse:<tool> hook error: [<hook command>]: <reason>`.
- Exit 0 with `permissionDecision` "deny" shows `Error: PreToolUse:<tool> hook error: <reason>`: the command echo goes, the error framing stays. The legacy `decision` "block", a deny with `suppressOutput`, and a deny with `systemMessage` render the same framing (the last adds a second line).
- Exit 1 shows the person a `hook error` line and the model never receives the message.
- Exit 0 with `additionalContext` and no decision lets the call run with no error line, and the model receives the note after the tool returns.

So the host frames every refusal as an error, and the framing itself is outside abcd's reach. The guard now refuses through the JSON deny (the shorter line), and a question whose only fault is its length is shown with a note to the agent rather than refused, which removes the most frequent refusal the person saw. A refusal of a shell hazard, of another layout rule, or of a question asked while the mode reads managed still shows the shorter error line. The before and after captures are kept in the local tier of the checkout that measured them.
