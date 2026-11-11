---
id: dcn-2611081501502395
slug: in-a-repository-whose-default-branch-the
date: 2026-11-08
---
2026-11-08 — In a repository whose default branch the live forge ruleset guards with a merge queue, the stop check's BEHIND item (iss-2610311339274372) stays owed for a non-draft pull request with auto-merge not armed, and its next move reads 'arm it' rather than 'merge it in'; a draft keeps the item it has today. Grounds: the queue tests every entry against the latest base, so merging the base in is never the move there, but a pull request nobody has armed will not reach the queue, so the session still owes it; a draft is unfinished by its own declaration, and saying it is held for a later event is a separate gap. Reverse by dropping the unarmed case, which then reads as it does today.
