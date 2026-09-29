---
schema_version: 1
id: "iss-55"
slug: "corpus-stub-backfill"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
related_intents: [itd-2609292108089653]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J6 of 2026-09-29: the backfill of the placeholder source entries is part of the sources-tools intent of ruling J5 (ingest and consult as abcd verbs). Filed as draft itd-2609292108089653. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

Six public corpus entries are keyword stubs with no document text — five practitioner SDLC pieces and one harness-engineering post — and they yielded zero extraction candidates in the 2026-07-09 tool-extraction pass, so the corpus claims coverage it cannot deliver. The fix serves retrieval honesty: backfill the full texts via the ingest path, and attach the on-disk PDF to the already-registered research-integrity paper entry so its text is queryable too. Detector: a corpus lint flags any text.md stub whose ledger entry claims a fetchable URL; acceptance is a lint run with zero such stubs and a consult query over the six entries returning document text rather than keywords.