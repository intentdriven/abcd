---
schema_version: 1
id: "iss-2610290809526033"
slug: "reservation-and-announcement-commits"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "a privacy review of what abcd pushes, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/message.go"
remedy: "Push an opaque random per-account holder id, minted once into the person's own abcd home, in place of the login in both trailers and the announcement branch name; tell your own reservations from the ids your home declares as yours, keep reading the old login trailers and branches, and push no new login-named one."
resolution: "Reservation and announcement commits now name an opaque holder id, minted once per machine account and kept in the abcd home, instead of the forge login, in their trailers and in the status branch name. Holds made before the change are still read and count as the person's own. A copy on an older release reads the new holds as held until it updates, and a person running two versions renews only from the new one. No trailer and no ref name carries the login any more; the commit author and committer fields still carry the person's git identity, by ruling, and holds already pushed under a login are not rewritten. A hosted or ephemeral session keeps one id by setting ABCD_HOLDER_ID to it, and a fresh mint is told once on stderr."
impact: fix
---

Reservation and announcement commits that abcd pushes to origin name the holder by their forge login: the Abcd-Holder and Abcd-Sender trailers carry it, and the announcement branch is named after it, so every reservation publishes which account made it in text and in the branch list.
