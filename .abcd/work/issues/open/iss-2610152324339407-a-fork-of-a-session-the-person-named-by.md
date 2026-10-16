---
schema_version: 1
id: "iss-2610152324339407"
slug: "a-fork-of-a-session-the-person-named-by"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "fix round 4 of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Record the person's own title by name as well as by id (a set of titles the person chose), and treat a fork whose title is in that set as the person's; or carry the parent id when the host provides it. Test a fork of a hand-named Bob while abcd's Bob runs elsewhere."
---

A fork of a session the person named by hand is renamed when abcd has given that same persona name to another running session: the fork's own id has no given record, its title matches a name given to a different id, so it is read as an inherited abcd name. abcd cannot link a fork to its parent's empty (person's own) record. Sessions named before the given record existed have no record at all.
