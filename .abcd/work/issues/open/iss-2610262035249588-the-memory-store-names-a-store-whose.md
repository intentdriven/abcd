---
schema_version: 1
id: "iss-2610262035249588"
slug: "the-memory-store-names-a-store-whose"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 569 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:394"
remedy: "none (filed automatically)"
finding_ref: cfk-8f3ebf8c3c9cffbe
---

'The memory store' names a store whose ingest takes documents and web addresses in the history chapter, but the memory chapter's store takes notes on stdin and has no ingest

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:394` — "the memory store's ingest takes a document or a web address"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:74` — "Adding takes no path: the text arrives on stdin, so the verb never opens the"

11-history says the memory store's ingest takes a document or a web address and never a stored transcript. 37-memory describes `/abcd:memory`, whose store is the two memory.json layers. Adding to it takes no path, only text on stdin, and it has no ingest of documents or addresses. One name is used for two stores: the history chapter's is evidently the library corpus, and the memory chapter's is the notes store.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 569.
