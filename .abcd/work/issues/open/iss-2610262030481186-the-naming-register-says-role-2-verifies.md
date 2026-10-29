---
schema_version: 1
id: "iss-2610262030481186"
slug: "the-naming-register-says-role-2-verifies"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 48 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:155"
remedy: "Reword 04-naming.md line 155: registration is held by review today (the VR001 check is unbuilt) and `abcd intent consistency` reports terminology drift on demand; nothing runs it at plan-review."
finding_ref: cfk-876673ffc1c8a109
---

The naming register says Role 2 verifies registration on every plan-review; the intent chapter runs Role 2 on demand only

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:155` — "The Role 2 cross-document audit verifies registration on every plan-review."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:614` — "The judgement half runs on demand as the consistency sub-verb, host-delegated as the audit is"

04-naming relies on the Role 2 cross-document audit running on every plan-review to verify vocabulary registration, while 05-intent says Role 2's judgement runs on demand as the consistency sub-verb, with no aggregator and any hook wiring deferred. Nothing runs it at plan-review, so the registration check 04-naming counts on does not happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 48.
