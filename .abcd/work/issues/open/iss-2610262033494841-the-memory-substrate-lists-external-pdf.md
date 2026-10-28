---
schema_version: 1
id: "iss-2610262033494841"
slug: "the-memory-substrate-lists-external-pdf"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 440 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:51"
remedy: "none (filed automatically)"
finding_ref: cfk-59e891f126aedae0
---

The memory substrate lists external_pdf pages as ingested through abcd library ingest; the library chapter says the binary rejects PDF sources.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:51` — "User-ingested PDF via `/abcd:library ingest <path>`"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:355` — "And PDF is a later seam: the binary rejects PDF sources with a clear error until a text-extraction dependency is wired."

07-memory's closed source-class enum carries external_pdf, described as 'User-ingested PDF via /abcd:library ingest < path>' and immutable after ingest. 07-library says PDF is a later seam and the binary rejects PDF sources with a clear error until a text-extraction dependency is wired. No page can carry a class whose only producer refuses its input.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 440.
