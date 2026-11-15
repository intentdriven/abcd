---
schema_version: 1
id: "iss-2611150344439584"
slug: "a-session-cleared-after-its-name-hold"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of the session-name hold fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "At a resume, when the title equals a name the register has given out (in the given-names record of any session id) and no live entry holds it for this session, treat the title as abcd's and make the hold through the claim step, taking the next free name with the notice when a live session holds it; a title the register never gave out stays the person's own. Proven by a register test: clear with a failed hold, process ends before a prompt, resume the new id, start another session: two names, one notice."
---

A session cleared after its name hold failed, whose host process ends before its first prompt, can later share its name. When the new session id is resumed, the only register entry accounting for its title belongs to the ended process, so the title is read as the person's own; once that entry is dropped and the name is given out again, two running sessions carry it and nothing corrects it. Found while fixing iss-2611141504101439, which heals a clear whose process is still running at its prompt.
