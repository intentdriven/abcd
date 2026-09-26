---
schema_version: 1
id: "iss-2609261835118276"
slug: "the-consistency-ingest-and-the-reading-ingest-copy"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/consistency.go"
---

The consistency ingest and the reading ingest copy host-delegated prose into lint-bound records with no prose-citation check: capture.IngestConsistency files each finding's summary, quotes and explanation as an issue record, and the reading ingest writes reading items into the rdi store, and both stores are read by record-lint's prose_citation_resolves, so a finding or item citing a record id that names no record is written and then refuses the whole tree. The verdict ingest holds its rendered block to the gate before writing (intent.SetProseCitationGate over lint.UnresolvedProseCitationsInRecord, iss-2609231036448320); these two writers take no such check. capture cannot import core/lint (lint's tests import capture), so the fix is the same front-door seam, asked over the text each writer renders.
