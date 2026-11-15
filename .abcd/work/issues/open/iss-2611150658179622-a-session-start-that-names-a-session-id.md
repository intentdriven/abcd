---
schema_version: 1
id: "iss-2611150658179622"
slug: "a-session-start-that-names-a-session-id"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname"
remedy: "Re-key an entry only when its host is the caller's host or is no longer alive; otherwise reserve a new name and add a note."
---

A session start that names a session id already in the name register takes that entry for the calling host without checking that the entry's host is the caller's or gone.
