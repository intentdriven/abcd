---
schema_version: 1
id: "iss-2609261106287078"
slug: "inbox-readers-follow-symlinked-ancestor"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cutfix item 2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
---

The inbox readers follow a symlinked ancestor the inbox writers refuse: peekInbox (internal/core/report/inbox.go) checks the inbox leaf with fsutil.IsRealDir, which lstats the leaf only, so with ~/.abcd a symlink abcd inbox, abcd inbox show and the session-start count read through the link while abcd report and abcd inbox promote refuse it through fsutil.EnsureRealDirAll. The comment on the refusal claims every verb refuses a symlinked level, which is true only of a symlink at the inbox leaf. Readers and writers should refuse the same paths, as the rules loader refuses ~/.abcd/rules.json behind a symlinked ~/.abcd.
