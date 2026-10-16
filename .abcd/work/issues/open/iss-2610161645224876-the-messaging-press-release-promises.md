---
schema_version: 1
id: "iss-2610161645224876"
slug: "the-messaging-press-release-promises"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "product thinker interview 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Make the announcement reach origin before the push it announces: a session that pushes work for an item with a live announcement (or one it is required to make) publishes the announcement first, and the push waits for it. Cover the plain git push path through the pre-push hook as well as abcd land, keep a failed announcement from silently dropping the push's notice, and test that a colleague's status read shows the announcement before the pushed branch exists on origin."
---

The messaging press release promises that a colleague's session sees an announcement such as 'merging the login fix' before the session that made it pushes. The build guarantees it only before the work merges into the main line, at the land step. The product thinker chose on 2026-10-16 to make the build keep the promise rather than reword it.
