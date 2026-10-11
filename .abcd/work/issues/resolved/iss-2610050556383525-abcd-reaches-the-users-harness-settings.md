---
schema_version: 1
id: "iss-2610050556383525"
slug: "abcd-reaches-the-users-harness-settings"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "/abcd board session, after the stale SubagentStop hook in ~/.claude/settings.json recreated ~/.abcd"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ahoy.md"
remedy: "Record the ruling as a decision, then hold ahoy install's status-line write to it: write only after an explicit yes, show the one entry's before and after, keep a backup, replace the file atomically leaving every other key untouched, verify the JSON after, wire only the binary path-entry records under the same ownership and writability checks the hooks apply, keep abcd statusline read-only and time-bounded under a test, and say how to restore the previous line before an uninstall."
resolution: "recorded as adr-2610050711287503; the status-line write happens only after a yes over the shown before and after, keeps a copy (newest 10), is read back and restored on a mismatch, wires only the recorded trusted PATH install, never records abcd's own verb as previous, and the status verb writes nothing and is time-bounded under tests; uninstall restores the previous command"
impact: fix
resolved_by:
  commit: "e73d514546cbab6c87fe33a2234aba300a04467a"
---

Product thinker ruling, 2026-10-05: abcd does not touch a person's ~/.claude/ folder, and nothing in it calls abcd, with one exception the person consents to: the status line, which is essential for product thinkers and which a plugin cannot set (a plugin's settings may set only agent and subagentStatusLine; the hooks abcd needs can all live in the plugin's hooks/hooks.json). Today ahoy install offers and writes the statusLine entry, and the person's settings had also gained a SubagentStop entry that called a stale abcd build (the defect that recreated ~/.abcd the same day). Because the status line runs on every refresh in every session, the risks the ruling accepts are: a stale or missing binary at the wired path; a status-line verb that writes state; a line left behind after an uninstall; the person's previous status command recorded once and gone stale; a concurrent write corrupting settings.json so the harness ignores the whole file; a wired path a project or another user can replace; and settings synced to a machine without that path.
