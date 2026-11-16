---
schema_version: 1
id: "iss-2611160240149245"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous-run heartbeat"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "In landed(), when the merged tree differs from the default branch, read the branch as superseded when every record id it adds or changes is already on the default branch, in any status folder, and every other path it changes has also changed on the default branch since the merge base, and name it for the person to delete, not as owed a push."
---

While the person is away, the stop hook names a local branch as owed a push when another branch's pull request has already landed the same records. On 16 Nov it named records-run-1104b. Each of the 44 record ids it adds is on the default branch, landed through #1393 from records-run-1104c and since edited, moved or resolved there. The branch also changes three files that main has moved on from.

The owed reader's `landed` test merges the branch into the default branch with `git merge-tree` and counts it landed only when the merged tree equals the default branch's tree, falling back to `git cherry` on a conflict. A superseded records branch fails the first test, because its older copies of the records change the merged tree, and its commits carry other patch ids should the fallback run. So it reads as unlanded work.

A session must not push or delete a branch it did not create, so the item cannot move. Once the owed set has not moved over two wakes, no wake releases the stop. iss-2611150851542274 has the same stop-loop symptom from another cause, a branch a live drive still owns, and iss-2610050532499080 asks for spent branches to be cleaned up after a cut.
