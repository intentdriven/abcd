---
schema_version: 1
id: "iss-2610301026198794"
slug: "when-git-s-identity-in-a-session-is-not"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "round-4 fixes on the opaque reservation holder (iss-2610290809526033)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/prepare.go"
remedy: "Pick an author and committer identity for reservation and announcement commits made without a person's git identity that passes the attribution gate's human-author rule and names no login, and write it in place of the token account's login and no-reply address."
---

When git's identity in a session is not a person's, abcd authors and commits each reservation and announcement commit it pushes as the account the forge token resolves to, with that account's login and no-reply address, so those commits' author and committer fields carry the login that the holder id keeps out of every trailer and ref name. That is the stated exception for the release bringing the holder id; the follow-up is an author identity for those commits that the human-author attribution gate accepts and that carries no login.
