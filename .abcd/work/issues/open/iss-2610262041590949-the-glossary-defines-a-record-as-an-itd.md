---
schema_version: 1
id: "iss-2610262041590949"
slug: "the-glossary-defines-a-record-as-an-itd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1038 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-2609151138388536]
finding_ref: cfk-50ecb56ae32f812d
---

The glossary defines a record as an itd, spc, adr, iss or rdg id; the decisions intent mints decision records with ids of their own.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:4` — "an itd-N, spc-N, adr-N, iss-N or rdg-id"
- End B: `.abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:26` — "allocated through the same seam that mints captures, intents, specs and"

The record entry defines a record as one identified, filed document that a command mints and a gate reads, and enumerates the ids as 'an itd-N, spc-N, adr-N, iss-N or rdg-id'. The record-families page has no decision row. The shipped decisions intent makes every decision 'a record with an id', allocated through the same seam that mints captures, intents, specs and ADRs, and minted by `abcd decide` as dcn- records. The glossary's closed list of record kinds does not cover the family this intent created.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1038.
