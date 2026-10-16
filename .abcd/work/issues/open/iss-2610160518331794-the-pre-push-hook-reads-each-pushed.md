---
schema_version: 1
id: "iss-2610160518331794"
slug: "the-pre-push-hook-reads-each-pushed"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "messaging step 2 review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-push"
remedy: "Export GIT_NO_REPLACE_OBJECTS=1 at the top of .githooks/pre-push, so every read in the hook sees the objects as stored."
---

The pre-push hook reads each pushed commit's tree through replace refs, so a local replace ref can make a commit that carries files read as the empty tree in the reservation and status rows, while the push sends the commit's stored objects.
