---
schema_version: 1
id: "iss-55"
slug: "corpus-stub-backfill"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: the stub entries live in the user-home sources store outside the repo; backfill and corpus lint wait on iss-27 corpus-in-core). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Six public corpus entries are keyword stubs with no document text — five practitioner SDLC pieces and one harness-engineering post — and they yielded zero extraction candidates in the 2026-07-09 tool-extraction pass, so the corpus claims coverage it cannot deliver. The fix serves retrieval honesty: backfill the full texts via the ingest path, and attach the on-disk PDF to the already-registered research-integrity paper entry so its text is queryable too. Detector: a corpus lint flags any text.md stub whose ledger entry claims a fetchable URL; acceptance is a lint run with zero such stubs and a consult query over the six entries returning document text rather than keywords.