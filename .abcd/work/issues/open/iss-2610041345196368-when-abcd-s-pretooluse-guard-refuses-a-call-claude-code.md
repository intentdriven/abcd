---
schema_version: 1
id: "iss-2610041345196368"
slug: "when-abcd-s-pretooluse-guard-refuses-a-call-claude-code"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "a downstream project's session on v0.12.0, 2026-10-04, reported by the person"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Move each hook's shell into one script file in the plugin payload (hooks/<event>.sh, sharing the binary-resolution rung) so hooks.json carries a short command such as \"$CLAUDE_PLUGIN_ROOT/hooks/pretooluse.sh\"; the host's error line then shows a short path followed by the guard's message. Keep the owned-only PATH rung and the bootstrap behaviour byte-for-byte, with a test that the scripts and the old inline commands agree."
---

When abcd's PreToolUse guard refuses a call, Claude Code prints the error as 'PreToolUse:<tool> hook error: [<the hook's whole command>]: <stderr>', and abcd's hook commands are inline shell of about 2,500 to 2,800 characters each (hooks/hooks.json: UserPromptSubmit 2,505, PreToolUse 2,774, PreCompact 2,408, SessionEnd 2,169, SubagentStop 2,173, SessionStart 959). The person sees a wall of mangled shell, wrapped and cut at the terminal width, and the guard's actual message (here the asking guard's 'the mode reads managed … say whom … then ask again') sits unreadable at the end. Seen 2026-10-04 in a downstream project on v0.12.0, when its agent asked the owner a coordination question and the asking guard refused it.
