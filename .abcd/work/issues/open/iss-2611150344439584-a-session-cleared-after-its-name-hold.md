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
remedy: "At a resume of a session id that has no record of its own in the register (no given-names record at all, which is a session from before naming existed or one whose start could not record anything), when the title equals a name the register has given out to another session id and no live entry holds it for this session, make the hold through the claim step, taking the next free name with the notice when a live session holds it. A session id that has a record keeps what the record says, and a title the register never gave out stays the person's own, so the rule that only a fork inherits a name and that the person's own title wins is unchanged. Proven by a register test: clear with a failed hold, process ends before a prompt, resume the new id, start another session: two names, one notice; and a resumed session whose record says the title is the person's own is left alone."
---

A session cleared after its name hold failed, whose host process ends before its first prompt, can later share its name. When the new session id is resumed, the only register entry accounting for its title belongs to the ended process, so the title is read as the person's own; once that entry is dropped and the name is given out again, two running sessions carry it and nothing corrects it. Found while fixing iss-2611141504101439, which heals a clear whose process is still running at its prompt.
