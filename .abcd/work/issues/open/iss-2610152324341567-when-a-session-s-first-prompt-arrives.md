---
schema_version: 1
id: "iss-2610152324341567"
slug: "when-a-session-s-first-prompt-arrives"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "fix round 4 of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Record given[session] = name in that case too, since the title is the reserved name either way; test resume after a rename to the reserved name."
---

When a session's first prompt arrives with a title already equal to the name abcd reserved for it (the person renamed it to exactly that name), the prompt hook marks the entry sent but records nothing in given, so a later resume reads the title as the person's and does not hold it.
