---
schema_version: 1
id: "iss-2610262031293547"
slug: "02-context-gives-abcd-library-six-corpus"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 135 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:13"
remedy: "none (filed automatically)"
finding_ref: cfk-3031d2c311078bf3
---

02-context gives /abcd:library six corpus sub-verbs beside ingest, ask and lint; 07-memory says the shipped library surface carries ingest, ask and lint and nothing else.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:13` — "keep the sources corpus (`init` / `add` / `declassify` / `ledger` / `sync-banlist` / `cite-check`, per itd-76)"
- End B: `.abcd/development/brief/05-internals/07-memory.md:197` — "The shipped surface carries `ingest`, `ask` and `lint` and nothing else."

02-context says /abcd:library will 'keep the sources corpus (`init` / `add` / `declassify` / `ledger` / `sync-banlist` / `cite-check`, per itd-76)' and curate the memory store with ingest, ask and lint, two stores under one command. 07-memory's retrieval section, discussing a recall sub-verb of abcd library, says 'The shipped surface carries `ingest`, `ask` and `lint` and nothing else.' Both describe the sub-verbs of the one library command and disagree on whether six of them exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 135.
