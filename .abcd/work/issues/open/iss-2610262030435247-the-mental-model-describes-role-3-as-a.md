---
schema_version: 1
id: "iss-2610262030435247"
slug: "the-mental-model-describes-role-3-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 35 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:72"
remedy: "Edit 03-mental-model.md (classification-at-plan-time paragraph): the third role on intent-auditor is a design target with no registered verb, on demand only when built; continuous invocation is deferred."
finding_ref: cfk-117460a9a7ad6fbf
---

The mental model describes Role 3 as a continuous audit role; the intent chapter makes it on-demand only and defers continuous invocation

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:72` — "A continuous audit role (the third role on `intent-auditor`, see"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:334` — "Scheduled / continuous invocation is a deferred follow-up."

03-mental-model says a continuous audit role, the third role on intent-auditor, suggests reclassifications as patterns emerge, while 05-intent says Role 3 is a design target with no verb registered, on demand only, with scheduled or continuous invocation deferred. The role cannot be both continuous and on-demand only.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 35.
