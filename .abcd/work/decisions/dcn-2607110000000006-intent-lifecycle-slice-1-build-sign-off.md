---
id: dcn-2607110000000006
slug: intent-lifecycle-slice-1-build-sign-off
date: 2026-07-11
---
2026-07-11 — Intent-lifecycle slice 1 (build sign-off given): the pipeline is
**dogfooded** — itd-3 stays shipped as the reference fixture (option b), and a
new tightly-scoped intent **itd-80-intent-lifecycle-automation** (ACs = the
steel thread) is the pipeline's first real payload, driven drafts→planned→
shipped through the machinery it specifies. Slice scope: minimal native spec
store (`internal/core/spec`, directory-as-truth open/closed, `intent:` link),
`abcd intent` (plan/link/review-ingest + bare render) and `abcd spec`
(close + bare render) verbs, deterministic reconcile inside `spec close`
(no vendor event), host-delegated `intent-fidelity-reviewer` markdown agent
(Role 1 only) + async outbox/inbox verdict ingest to `## Audit Notes`.
