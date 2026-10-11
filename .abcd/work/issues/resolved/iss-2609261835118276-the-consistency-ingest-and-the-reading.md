---
schema_version: 1
id: "iss-2609261835118276"
slug: "the-consistency-ingest-and-the-reading"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/consistency.go"
resolution: "The consistency ingest and the reading ingest now ask record-lint's prose-citation gate through the one registration the front doors make (intent.UnresolvedProseCitations, fail closed), and refuse naming the finding or item, the field and the id, with nothing written — the verdict ingest's behaviour, with no illustrative escape."
impact: fix
resolved_by:
  commit: "e8593b899"
---

The consistency ingest and the reading ingest copy host-delegated prose into lint-bound records with no prose-citation check: capture.IngestConsistency files each finding's summary, quotes and explanation as an issue record, and the reading ingest writes reading items into the rdi store, and both stores are read by record-lint's prose_citation_resolves, so a finding or item citing a record id that names no record is written and then refuses the whole tree. The verdict ingest holds its rendered block to the gate before writing (intent.SetProseCitationGate over lint.UnresolvedProseCitationsInRecord, iss-2609231036448320); these two writers take no such check. capture cannot import core/lint (lint's tests import capture), so the fix is the same front-door seam, asked over the text each writer renders.

## Grounds

- pursued: a finding or reading item citing a record id that names no record is refused before any write, so no ingested record can fail prose_citation_resolves; a record written by either ingest that the gate then refuses would show it wrong
