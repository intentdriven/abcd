---
schema_version: 1
id: "iss-2610262038595637"
slug: "itd-126-adds-abcd-source-share-and-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 832 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-126]
finding_ref: cfk-bc40715cff2cf09a
---

itd-126 adds abcd source share and abcd source ingest; the surfaces register folds the old source command into library

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17` — "`abcd source share` writes a public source's *citation data* into the repo's committed CSL-JSON references store"
- End B: `.abcd/development/brief/04-surfaces/README.md:494` — "The `library` command holds what the old `source` and `memory` commands held"

itd-126 adds abcd source share and abcd source ingest for the sources corpus. The surfaces register says the library command holds what the old source command held. The draft extends a verb family the register has folded into library, so the same corpus is reached under two verb names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 832.
