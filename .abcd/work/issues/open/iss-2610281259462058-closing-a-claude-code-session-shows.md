---
schema_version: 1
id: "iss-2610281259462058"
slug: "closing-a-claude-code-session-shows"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "closing sessions, 28 Oct 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/session-end.sh"
refines: ["iss-2610241519088123"]
remedy: "Land fix/session-end-stage, ship it in the next cut, and have the SessionStart notice say when the installed plugin is older than the newest release so the person updates it; meanwhile recover the cancelled sessions' transcripts with abcd history drain on the next start."
---

Closing a Claude Code session shows: SessionEnd hook [sh "${CLAUDE_PLUGIN_ROOT}/hooks/session-end.sh"] failed: Hook cancelled. Seen by the person on 28 Oct 2026 while closing several sessions at a 1-minute load near 290 on 10 cores, with the plugin still at 0.13.3. Each cancelled hook means that session's transcript was not staged. The fix for the cancellation is on fix/session-end-stage (iss-2610241519088123: the hook runs no git and only stages the raw transcript), but it reaches this person only after a release and a plugin update, because the installed plugin is 0.13.3 while v0.15.1 is released (iss-2610241228359748, which copy runs).
