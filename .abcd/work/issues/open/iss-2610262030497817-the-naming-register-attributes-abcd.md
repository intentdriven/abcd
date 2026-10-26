---
schema_version: 1
id: "iss-2610262030497817"
slug: "the-naming-register-attributes-abcd"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 50 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:116"
remedy: "none (filed automatically)"
finding_ref: cfk-95a54f7b5c112bd8
---

The naming register attributes /abcd:audit lifeboat to itd-16; the intent chapter attributes it to itd-35

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:116` — "`/abcd:audit chain` and `/abcd:audit lifeboat` (itd-16)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:659` — "(lifeboat-artefact integrity per itd-35 — a later phase)"

04-naming lists /abcd:audit chain and /abcd:audit lifeboat as staged under itd-16, while 05-intent says the lifeboat sub-verb is the lifeboat-artefact integrity check of itd-35, with itd-16 backing chain alone. The two disagree on which intent owns the sub-verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 50.
