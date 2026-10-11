---
schema_version: 1
id: "iss-2610041345196368"
slug: "when-abcd-s-pretooluse-guard-refuses-a"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "a downstream project's session on v0.12.0, 2026-10-04, reported by the person"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Move each hook's shell into one script file in the plugin payload (hooks/<event>.sh, sharing the binary-resolution rung) so hooks.json carries a short command such as \"$CLAUDE_PLUGIN_ROOT/hooks/pretooluse.sh\"; the host's error line then shows a short path followed by the guard's message. Keep the owned-only PATH rung and the bootstrap behaviour byte-for-byte, with a test that the scripts and the old inline commands agree."
resolution: "Each hook's shell moved verbatim into hooks/<event>.sh, run from hooks.json by a one-line sh command of about fifty characters (47 to 54), so the host's hook-error line shows a short path and then the hook's own message. The install check and the release gate read through the command into the script."
impact: fix
resolved_by:
  commit: "dcc682538"
---

When abcd's PreToolUse guard refuses a call, Claude Code prints the error as 'PreToolUse:<tool> hook error: [<the hook's whole command>]: <stderr>', and abcd's hook commands are inline shell of about 2,500 to 2,800 characters each (hooks/hooks.json: UserPromptSubmit 2,505, PreToolUse 2,774, PreCompact 2,408, SessionEnd 2,169, SubagentStop 2,173, SessionStart 959). The person sees a wall of mangled shell, wrapped and cut at the terminal width, and the guard's actual message (here the asking guard's 'the mode reads managed … say whom … then ask again') sits unreadable at the end. Seen 2026-10-04 in a downstream project on v0.12.0, when its agent asked the owner a coordination question and the asking guard refused it.

## Evidence 2026-10-07

Seen again in abcd's own repository on v0.13.1 (main at 2413a9843): the asking guard refused three questions in one session (one for the register and the row count, two for the row count alone), and each refusal printed the full PreToolUse command before the guard's message. The person asked for these messages to go, believing them already fixed. Nothing on any branch fixes it yet. The commands have grown since the record was filed: UserPromptSubmit 3,807 characters, SessionStart 2,344, PreToolUse 4,264, PreCompact 3,716, SessionEnd 3,469, SubagentStop 3,475 (`jq` over `hooks/hooks.json`), so the wall is now about half as long again as the one first reported.

## Grounds

- pursued: we expect a guard refusal to read as one short line followed by its message in every repository on the next plugin update, because the host prints only the command it ran; shown wrong if a refusal still shows the shim's source
