---
schema_version: 1
id: "iss-2610262040315792"
slug: "the-voyage-glossary-says-the-voyage"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 937 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-17e90bb0ac5d09c3
---

The voyage glossary says the voyage records every disembark and embark run; the verification matrix says the voyage ledger has a disembark half only.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:4` — "an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run)"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:76` — "The voyage ledger has a disembark half only"

The voyage entry defines the namespace as the append-only record of every disembark and embark run and lays out embark/provenance.json and embark/from/. The matrix's Voyage embark side row is staged: embark from writes no operator-level record, and the ledger has only a disembark half. Embark runs cannot be both recorded and unrecorded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 937.
