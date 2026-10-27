---
schema_version: 1
id: "iss-2610262032467744"
slug: "the-matrix-says-the-disembark-read-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 316 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:234"
remedy: "none (filed automatically)"
finding_ref: cfk-48bc0762da27533a
---

the matrix says the disembark read-only test hashes the source and asserts it byte-identical; disembark says the evidence is a fingerprint of path, mode and size, not a content hash

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:234` — "The evidence is a fingerprint, not a content hash"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:38` — "a test hashes the source tree before and after the run and asserts it is byte-identical"

02-verification-matrix's Disembark read-only row states that a test hashes the source tree before and after the run and asserts byte-identity. 02-disembark says the evidence is a fingerprint, not a content hash: it accumulates each entry's path, mode and size and skips .git, so a size-preserving rewrite or a write under .git escapes it. The matrix claims a stronger guarantee than the test the surface chapter describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 316.
