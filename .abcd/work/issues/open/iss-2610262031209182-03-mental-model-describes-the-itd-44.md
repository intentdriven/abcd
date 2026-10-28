---
schema_version: 1
id: "iss-2610262031209182"
slug: "03-mental-model-describes-the-itd-44"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 116 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:74"
remedy: "none (filed automatically)"
finding_ref: cfk-69e39ea08041a000
---

03-mental-model describes the itd-44 capture-time classifier as running; 03-out-of-scope lists itd-44 as an uncommitted draft.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "The capture-time classifier (`intent classify-capture-kind`, itd-44) recognises this signature"
- End B: `.abcd/development/brief/06-delivery/03-out-of-scope.md:54` — "`itd-44` — A Capture-Time `decision` Verdict That Routes Infrastructure Choices Into The Existing ADR Store"

03-mental-model states in the present tense, with no staged mark, that 'The capture-time classifier (`intent classify-capture-kind`, itd-44) recognises this signature' and emits a decision verdict that routes capture to the ADR store. 03-out-of-scope lists itd-44, the capture-time decision verdict, in the drafts/ corpus, the uncommitted bench of work not yet under way. A classifier cannot both be current behaviour and the subject of an intent that has not been planned.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 116.
