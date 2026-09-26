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
resolution: "The reading verbs walk the inbox's levels through fsutil.ProbeRealDirAll, the read-only counterpart of EnsureRealDirAll, promoted/ included, so List, Show and Count refuse what File and Promote refuse; a symlinked home or ~/.abcd with no inbox behind it reads as no inbox, as the rules loader reads rules.json behind a symlinked ~/.abcd."
impact: fix
resolved_by:
  commit: "7170858f8acd09be35106c551b243fae77600603"
---

The inbox readers follow a symlinked ancestor the inbox writers refuse: peekInbox (internal/core/report/inbox.go) checks the inbox leaf with fsutil.IsRealDir, which lstats the leaf only, so with ~/.abcd a symlink abcd inbox, abcd inbox show and the session-start count read through the link while abcd report and abcd inbox promote refuse it through fsutil.EnsureRealDirAll. The comment on the refusal claims every verb refuses a symlinked level, which is true only of a symlink at the inbox leaf. Readers and writers should refuse the same paths, as the rules loader refuses ~/.abcd/rules.json behind a symlinked ~/.abcd.

## Grounds

- pursued: with ~/.abcd a symlink to a directory holding an inbox, every inbox verb refuses naming ~/.abcd; a reading verb that returns reports through that link, or one that refuses a dotfiles ~/.abcd holding no inbox, would show it wrong
