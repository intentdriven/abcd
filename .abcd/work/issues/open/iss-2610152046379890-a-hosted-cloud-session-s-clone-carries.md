---
schema_version: 1
id: "iss-2610152046379890"
slug: "a-hosted-cloud-session-s-clone-carries"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "cloud probe and build of itd-2609151838312703, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "hosted cloud sessions' git identity"
remedy: "Give the cloud-sessions intent's setup step (its criteria C1 and C11) a session-start action that sets the clone's local git user.name and user.email from the environment token's forge login and id, as login and id+login@users.noreply.github.com, before any commit, with a test that the vendor identity appears in no commit, ref, trailer or output; record it on that intent as a decision when its branch is next worked."
---

A hosted cloud session's clone carries the host's default vendor identity as its git author, not the person's: the cloud probe of 2026-10-15 found it in the clone's git config, while the forge credited the push to the person's own account through the token. The product thinker ruled the same day that this vendor identity must not be shown anywhere, in no commit, ref, trailer or output, and that no person should have to set it by hand ('Who has to remember what? No manual steps.'). Session announcements (itd-2609151838312703) already fall back to the token's login for their own commits; every other commit a cloud session makes still carries the vendor identity until its setup sets the person's.
