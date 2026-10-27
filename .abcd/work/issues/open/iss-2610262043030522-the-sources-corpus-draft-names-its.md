---
schema_version: 1
id: "iss-2610262043030522"
slug: "the-sources-corpus-draft-names-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1111 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609292108089653-ingesting-a-source-and-consulting-the.md:43"
remedy: "none (filed automatically)"
related_intents: [itd-2609292108089653, itd-39]
finding_ref: cfk-637125c56f80c63a
---

The sources-corpus draft names its ingest, consult and lint 'source add', a consult verb and a corpus lint; itd-39 says the store is already reached as library ingest, ask and lint.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609292108089653-ingesting-a-source-and-consulting-the.md:43` — "is refused at `source add` and"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:79` — "its store is now reached as `library ingest`, `library ask` and `library lint`"

itd-2609292108089653 proposes moving ingest and consult into the binary as new verbs: a consult verb, a quality check that refuses a keyword stub 'at source add', and 'a corpus lint'. itd-39 records that the same store is already reached as 'library ingest', 'library ask' and 'library lint'. One surface has two sets of names, and a builder taking the draft at its word would add a second verb family beside the one itd-39 says exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1111.
