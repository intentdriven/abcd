---
schema_version: 1
id: "iss-2610262031239413"
slug: "06-framing-calls-its-construal-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 124 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/06-framing.md:13"
remedy: "Edit 06-framing.md (opening paragraph and construal): say 'the provenance ledger's' construal and 'the provenance ledger work', not the bare ledger."
finding_ref: cfk-6d7764452ff61392
resolution: "06-framing.md's opening paragraph and construal now name the provenance ledger in full ('the provenance ledger's' construal, 'the provenance ledger work'), so bare 'the ledger' no longer reads as the issue ledger under the glossary's rule."
impact: internal
resolved_by:
  commit: "9ffe4fa3a5da08f145ca6d1a262728c682cbda7a"
---

06-framing calls its construal the ledger's, bare, meaning the ledger work; the ledger entry reserves bare 'the ledger' for the issue ledger.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/06-framing.md:13` — "construal below is the ledger's — the frame the ledger's own design reasons"
- End B: `.abcd/development/brief/glossary/core/ledger.md:44` — "only where the issue ledger is unambiguous"

06-framing says the 'construal below is the ledger's — the frame the ledger's own design reasons' inside, which is the ledger work the construal describes, not the captured-issue store. The glossary's ledger entry says bare 'the ledger' means the issue ledger and may be used 'only where the issue ledger is unambiguous', every other ledger being named in full. Read under the glossary's rule, the framing chapter assigns its construal to the issue ledger.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 124.

## Grounds

- pursued: no bare 'the ledger' is left in 06-framing.md (detector.sh passes), and the glossary's ledger senses stay consistent; a remaining unqualified 'the ledger' in the chapter, or a reader taking the construal as the issue ledger's, would show it wrong.
