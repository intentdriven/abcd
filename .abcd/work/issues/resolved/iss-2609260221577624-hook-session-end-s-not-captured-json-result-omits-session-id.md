---
schema_version: 1
id: "iss-2609260221577624"
slug: "hook-session-end-s-not-captured-json-result-omits-session-id"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "hook session-end's not_captured --json result carries the parsed session_id, masked, as subagent-stop's does; an unparsable payload names none."
impact: fix
resolved_by:
  commit: "95fb6dd5"
---

hook session-end's not_captured --json result omits session_id, where hook subagent-stop's not_captured result includes it, so a caller cannot tell which session a session-end failure lost from the result line alone.

## Grounds

- pursued: a session-end payload with no transcript_path yields a not_captured result naming its session_id (TestSessionEndNotCapturedNamesTheSession); a not_captured line from a parsed payload with an empty session_id would show it wrong
