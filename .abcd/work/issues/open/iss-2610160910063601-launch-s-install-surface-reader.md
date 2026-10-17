---
schema_version: 1
id: "iss-2610160910063601"
slug: "launch-s-install-surface-reader"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR1 fix round, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/installsurface.go"
remedy: "Teach pluginRootRefs the ${CLAUDE_PLUGIN_ROOT:-} spelling and a one-level alias assigned from it in the same script (r=\"${CLAUDE_PLUGIN_ROOT:-}\"), and test that a 0644 bootstrap.sh run as \"$r/hooks/bootstrap.sh\" is flagged."
---

launch's install-surface reader (pluginRootRefs) does not see the plugin root reached through ${CLAUDE_PLUGIN_ROOT:-} or a shell alias of it (r="${CLAUDE_PLUGIN_ROOT:-}" then "$r/hooks/..."), the form most of abcd's own hook scripts use; post-tool-use.sh, pre-compact.sh, pre-tool-use.sh and user-prompt-submit.sh each execute "$r/hooks/bootstrap.sh" directly, and the gate sees none of those runs. bootstrap.sh is judged today only because session-start.sh also names it by $CLAUDE_PLUGIN_ROOT; were that line to use the alias, a missing or 0644 bootstrap.sh would pass the hook-compliance gate's existence and mode checks without a warning. Pre-existing; found while fixing iss-2610160608456302, 2026-10-16.
