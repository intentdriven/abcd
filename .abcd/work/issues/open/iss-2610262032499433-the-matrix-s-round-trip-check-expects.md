---
schema_version: 1
id: "iss-2610262032499433"
slug: "the-matrix-s-round-trip-check-expects"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 322 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:126"
remedy: "none (filed automatically)"
finding_ref: cfk-ca4fcb40917e7aa3
---

the matrix's round-trip check expects the embarked repo's memory to be a faithful subset; embark says the memory store does not travel

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:126` — "Terminology, docs and the memory store are **not** embark families; they do not travel."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:44` — "the new repo's memory and decisions are a faithful subset"

02-verification-matrix's Round-trip row, unstaged, checks that after a disembark and an embark the new repo's memory and decisions are a faithful subset of the source's. 03-embark says terminology, docs and the memory store are not embark families and do not travel, so an embarked target receives no memory at all. The row checks for something embark is specified never to write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 322.
