---
schema_version: 1
id: "iss-2611211733085834"
slug: "an-ended-away-record-tells-the-sessions"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "lab-261121170045-71e07e7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_away.go"
remedy: "While a record has ended unclosed, give the session one line as well as the person: the away window ended at its time, questions and turn ends are no longer checked, and the mode is set again only on the person's word. Have the line the person is shown say that consequence too, not only what is unread. Grounds: dcn-2610181003474765 (the line to the person) and loud-staging."
---

An away record that has ended tells the sessions nothing. When a record ends at its time or at its seven-day cap with steps or stops unread, the prompt hook gives the person one line on every prompt and gives the session no block, so a session cannot tell that its questions and turn ends are no longer checked. The line the person is shown says the window ended and what is unread; it does not say that sessions have gone back to asking and stopping. On this machine the record set on 2026-11-07 ended at its cap on 2026-11-14 with 15 steps queued and was never closed; an autonomous run was declared the next day and extended on 2026-11-21, and no record held during it until a session set one that evening. Read at commit 71e07e7ea in lab-261121170045-71e07e7 (the prompt hook's away view and the record's lapsed line).
