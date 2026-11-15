---
schema_version: 1
id: "iss-2611150344428650"
slug: "a-fork-whose-session-start-could-not"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of the session-name hold fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Give the prompt hook the fact it lacks: have the fork's first prompt carry or look up that the session is a fork (the host's hook input names the source of a session start; where the prompt input does not, record every session id seen at a prompt and treat an id first seen at a prompt, with no start recorded, whose title a live entry on another host accounts for, as an unrecorded fork only when the host's transcript for that id names a parent session). Then record the title as given to the fork so the claim step renames it once with the notice. Proven by a register test: a fork with no start record prompts while its parent runs and ends with its own name and one notice; a resumed session with the person's own title is left alone."
---

A fork whose session start could not take the name register's lock, or whose start hook never ran, shares its parent's name while both run. No record of the inherited name is written for the fork's session id, so at its prompt the fork's title is accounted for by the parent's live entry on another host process and is recorded as the person's own. At the prompt this cannot be told apart from a resumed session carrying a title the person chose, which must be kept. Found while fixing iss-2611141504101439, which heals the other lost holds.
