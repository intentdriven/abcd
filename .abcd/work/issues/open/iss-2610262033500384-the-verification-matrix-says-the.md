---
schema_version: 1
id: "iss-2610262033500384"
slug: "the-verification-matrix-says-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 442 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:86"
remedy: "none (filed automatically)"
finding_ref: cfk-a911326ab8560f8f
---

The verification matrix says the lifeboat licence gate refuses to publish a kept original without an allowlist entry; the library chapter says that gate is not built.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:86` — "and the lifeboat licence gate refuses to publish it without an explicit allowlist entry"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:361` — "That gate is not built, and the shipped ingest classifies restrictive licences without ever refusing on them."

The matrix's Memory ingest row is not marked staged, so it describes what the binary carries today, and it says the lifeboat licence gate refuses to publish a kept original without an explicit allowlist entry. 07-library says 'That gate is not built' and that the shipped ingest classifies restrictive licences without ever refusing on them. A gate cannot both refuse today and not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 442.
