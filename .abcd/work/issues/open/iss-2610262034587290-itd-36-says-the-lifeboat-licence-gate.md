---
schema_version: 1
id: "iss-2610262034587290"
slug: "itd-36-says-the-lifeboat-licence-gate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 537 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-36-memory-unification.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-222c3f31e71772e7
---

itd-36 says the lifeboat licence gate refuses to surface kept originals; the library surface page says that gate is not built and the ingest never refuses on a licence.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-36-memory-unification.md:44` — "the lifeboat licence gate refuses to surface `sources/` files without an explicit allowlist entry"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:362` — "is not built, and the shipped ingest classifies restrictive licences without"

itd-36 scopes keeping originals under .abcd/memory/sources/ behind a lifeboat licence gate that refuses to surface those files without an explicit allowlist entry. 07-library.md says the licence gate belongs to the lifeboat, that the gate is not built, and that the shipped ingest classifies restrictive licences without ever refusing on them. The intent's scope likewise lists a curator role on principle-distiller that the library page lists as not built, while the same page calls itd-36 non-decomposable. A shipped intent cannot rely on a gate the surface page says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 537.
