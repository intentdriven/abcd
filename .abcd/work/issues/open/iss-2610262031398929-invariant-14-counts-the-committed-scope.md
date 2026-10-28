---
schema_version: 1
id: "iss-2610262031398929"
slug: "invariant-14-counts-the-committed-scope"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 159 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/construal.md:25"
remedy: "none (filed automatically)"
finding_ref: cfk-f78a16f9e8ee5739
---

Invariant 14 counts the committed scope as part of the construal, while the glossary says scope is never the construal

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/construal.md:25` — "Do not call the brief's purpose or scope the construal; those are products of the framing, and the construal is the frame itself."
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:165` — "the construal *as it presently stands* — the framing section's statement, the glossary's committed terms, the committed scope — is committed record"

Invariant 14 says the construal as it presently stands is the framing section's statement, the glossary's committed terms and the committed scope. The glossary entry makes the construal one of three framing surfaces beside the terms and the scope, and says not to call the brief's purpose or scope the construal. The invariant uses the term in the sense the glossary forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 159.
