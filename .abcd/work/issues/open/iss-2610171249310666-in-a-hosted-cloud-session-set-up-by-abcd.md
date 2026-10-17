---
schema_version: 1
id: "iss-2610171249310666"
slug: "in-a-hosted-cloud-session-set-up-by-abcd"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "product thinker's live hosted-session announcement, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-push"
remedy: "Prove the shape without the history beneath the shallow boundary: the commits a push adds are those reachable from the local tip and not from any ref the remote already has (git rev-list <local> --not --remotes, or the remote tip on an update), and every one of them must carry the empty tree; refuse only when that set cannot be computed. Keep refusing a range whose walk crosses a shallow line into unread commits. Pin it with a pre-push test from a shallow clone that passes an empty-tree status commit and still refuses one that changes a file."
---

In a hosted cloud session set up by abcd's setup script, abcd announce and abcd reserve cannot push: the pre-push hook refuses every push from a shallow clone, and hosted sessions start from a shallow clone. reservation_shape returns failure as soon as git rev-parse --is-shallow-repository is true, so an empty-tree status or reservation commit is asked for a make preflight receipt that a hosted session never has. Seen live on 2026-10-17: a claude.ai/code session ran abcd announce asking #1 and was refused until it ran git fetch --unshallow by hand (.git grew from 23 MB to 63 MB); the announcement then reached origin as an empty-tree commit on abcd-status/reppl. commands/announce.md says announce works from hosted sessions, and acceptance criterion 13 of the announcements intent needs it to.
