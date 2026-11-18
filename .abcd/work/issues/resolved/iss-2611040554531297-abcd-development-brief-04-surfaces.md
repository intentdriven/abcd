---
schema_version: 1
id: "iss-2611040554531297"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Name the five events, or refer to the internals section."
resolution: "21-update.md now names the five events whose bootstrap salvage runs with --unseen (UserPromptSubmit, PreToolUse, PostToolUse, PermissionRequest, PreCompact) and points to the internals page that counts them; a test holds the page to hooks/hooks.json."
impact: internal
resolved_by:
  commit: "9bbae3b3e0c0a6a18d5ce3fc60b0eff6e183d3c0"
---

.abcd/development/brief/04-surfaces/21-update.md:189-192 says the salvage that discards its output is run by the per-prompt, per-command and pre-compaction hooks. hooks/post-tool-use.sh, matched to the question tool, and hooks/permission-request.sh run the same `bootstrap.sh --unseen` (hooks/hooks.json). .abcd/development/brief/05-internals/03-configuration.md:1833-1836 counts five events.

## Grounds

- pursued: the update page names every event whose hook passes --unseen in hooks/hooks.json; TestUpdatePageNamesEveryUnseenSalvageEvent failing, or a hook passing --unseen that the page does not name, would show it wrong
