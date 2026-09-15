---
schema_version: 1
id: "iss-2608301747003705"
slug: "issue-grounds-changed-from-a-string-to-a-list-in-the-json-en"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: "internal/surface/cli"
---

Issue.Grounds changed from a string to a list in the json envelope on an unreleased branch

Reported by the round-5 builder. `Issue.Grounds` moved from `string` to
`[]string` in the JSON envelope when grounds became append-only.

No in-tree consumer, and the field shipped on this same unreleased branch, so no
released surface moves and this is not a break. Recorded only so that the shape
change is findable if anything external bound to it early, and so the next
release note does not have to rediscover it.

Also noted by the same review and folded here rather than given an id: `capture`
importing `intent` for `CreateDraft`/`Buckets` remains, which is pre-existing
and unrelated to this round.
