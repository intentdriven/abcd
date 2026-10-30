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
resolution: "The pre-push hook no longer refuses every push from a shallow clone. Its shape check for reservations and announcements now lists the commits the push adds, as before, and refuses only when one of them is a commit the shallow clone cut off from its parents, since only then can commits carrying files hide beneath. So an empty-tree announcement or reservation pushed from a hosted session's shallow clone passes without a receipt, and nothing is fetched or deepened. A push that changes a file, a rewrite, and a range reaching past the shallow line are still refused."
impact: fix
---

In a hosted cloud session set up by abcd's setup script, abcd announce and abcd reserve cannot push: the pre-push hook refuses every push from a shallow clone, and hosted sessions start from a shallow clone. reservation_shape returns failure as soon as git rev-parse --is-shallow-repository is true, so an empty-tree status or reservation commit is asked for a make preflight receipt that a hosted session never has. Seen live on 2026-10-17: a claude.ai/code session ran abcd announce asking #1 and was refused until it ran git fetch --unshallow by hand (.git grew from 23 MB to 63 MB); the announcement then reached origin as an empty-tree commit on abcd-status/<login>. commands/announce.md says announce works from hosted sessions, and acceptance criterion 13 of the announcements intent needs it to.

## Grounds

- pursued: expected a hosted session's shallow clone to push an announcement and a reservation with no receipt while every refusal from a full clone still holds; new pre-push tests from a real depth-one clone pass both kinds and still refuse a file change, a rewrite and a range past the shallow line; a test failing, or a hosted session still asked for a receipt, would show it wrong
