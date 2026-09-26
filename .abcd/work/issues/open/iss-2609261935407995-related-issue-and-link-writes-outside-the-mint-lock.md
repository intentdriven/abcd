---
schema_version: 1
id: "iss-2609261935407995"
slug: "related-issue-and-link-writes-outside-the-mint-lock"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainA1 sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

intent.AddRelatedIssue (the intent half of capture promote --intent, any bucket including shipped/) and intent.Link (abcd intent link, planned/) rewrite the intent's frontmatter as a read-modify-write outside the intent mint lock (internal/core/intent/lifecycle.go): a hold, condition disposition, verdict ingest or review emit landing on the same record between the read and the write is erased, and both verbs exit 0. Every other intent writer holds withIntentMintLock.
