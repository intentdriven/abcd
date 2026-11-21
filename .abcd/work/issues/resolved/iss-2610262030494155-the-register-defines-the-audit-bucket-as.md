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
remedy: "Widen the audit comparison in the bucket row of 04-naming.md (line 181) to say audit compares reality, or one recorded commitment against another (the consistency pass), against a recorded commitment."
finding_ref: cfk-74758eabafd246e1
resolution: "The bucket row in 04-naming.md now says audit compares reality against a recorded commitment, or one recorded commitment against another (the consistency pass), which matches 05-intent filing consistency under audit; a lint test holds the two together."
impact: internal
resolved_by:
  commit: "463534056fdcf0a10f8f62d609dbd07f6c460ce4"
---

The register defines the audit bucket as reality against a recorded commitment; the intent chapter files the document-against-document consistency pass under audit

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:181` — "`audit` compares reality against a recorded commitment (family-2 verdicts)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:62` — "| `consistency` | audit | shipped |"

04-naming's closed bucket vocabulary says audit compares reality against a recorded commitment, while 05-intent's sub-verb table puts consistency, whose opponent is other documents with no delivered reality involved, in the audit bucket. The pass uses the term in a sense the register does not give it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 49.

## Grounds

- pursued: the register's audit definition should cover every sub-verb the surfaces file under audit. It would be shown wrong if the consistency pass's verdicts were not family-2, or if adr-40 meant consistency to sit in a different bucket.
