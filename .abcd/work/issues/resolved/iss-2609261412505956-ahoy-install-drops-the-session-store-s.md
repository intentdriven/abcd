---
schema_version: 1
id: "iss-2609261412505956"
slug: "ahoy-install-drops-the-session-store-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "stepHistory notes a failed home lookup, a failed index and a failed transcript store, each naming the store and the reason"
impact: fix
resolved_by:
  commit: "bde87810"
---

ahoy install drops the session store's failures silently: stepHistory (internal/core/ahoy/apply.go) ignores bootstrapHistory's error and historyRoot's, so a session store abcd could not create (a file where ~/.abcd belongs, or no HOME) leaves no note, against the ahoy brief chapter's rule that an install write it could not make is a note naming the file and the reason, never a silent omission (review2-ahoy item 1)

## Grounds

- pursued: an install whose session store cannot be created says so in its notes; shown wrong if TestSessionStoreFailureIsNoted's three cases ever pass with no note
