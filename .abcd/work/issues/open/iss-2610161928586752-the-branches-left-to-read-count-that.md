---
schema_version: 1
id: "iss-2610161928586752"
slug: "the-branches-left-to-read-count-that"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "messaging step 3 fix round 4 (feat/announce-step3, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Add a CLI test that runs abcd status --json over a world whose intake cap is spent and checks branches_left_to_read and the not-read-yet line."
refines: [iss-2610150526192138]
---

The branches_left_to_read count that abcd status reports when an intake leaves branches unread has a test in the claim package but none at the command line, so its JSON field and text line are not held at the surface.
