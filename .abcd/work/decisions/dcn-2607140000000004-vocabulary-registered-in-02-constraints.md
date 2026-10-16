---
id: dcn-2607140000000004
slug: vocabulary-registered-in-02-constraints
date: 2026-07-14
---
2026-07-14 — Vocabulary registered in `02-constraints/04-naming.md`, fulfilling a
claim adr-4 made and never kept (`voyage/`, `manifest_sha256`, `_provenance.json`,
`history.jsonl` were absent from the registry). Added with them: `coverage.json`,
`graveyard/`, and two new controlled enums — coverage `status ∈ {grounded, partial,
blank}` and source `tier ∈ {git, conventions, abcd-native}`, both with the Go enum
named as the machine-readable source of truth. The brief's `"sufficient"` oracle
verdict — a member of NO registered enum — is retired in favour of the registered
`{SHIP, NEEDS_WORK, MAJOR_RETHINK}`; no third verdict family is minted (four
brief locations).
