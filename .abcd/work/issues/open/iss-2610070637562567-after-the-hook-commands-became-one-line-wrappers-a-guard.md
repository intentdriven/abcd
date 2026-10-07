---
schema_version: 1
id: "iss-2610070637562567"
slug: "after-the-hook-commands-became-one-line-wrappers-a-guard"
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
