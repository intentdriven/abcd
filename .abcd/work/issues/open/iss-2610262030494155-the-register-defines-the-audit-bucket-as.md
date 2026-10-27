---
schema_version: 1
id: "iss-2610262030494155"
slug: "the-register-defines-the-audit-bucket-as"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 49 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:181"
remedy: "none (filed automatically)"
finding_ref: cfk-74758eabafd246e1
---

The register defines the audit bucket as reality against a recorded commitment; the intent chapter files the document-against-document consistency pass under audit

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:181` — "`audit` compares reality against a recorded commitment (family-2 verdicts)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:62` — "| `consistency` | audit | shipped |"

04-naming's closed bucket vocabulary says audit compares reality against a recorded commitment, while 05-intent's sub-verb table puts consistency, whose opponent is other documents with no delivered reality involved, in the audit bucket. The pass uses the term in a sense the register does not give it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 49.
