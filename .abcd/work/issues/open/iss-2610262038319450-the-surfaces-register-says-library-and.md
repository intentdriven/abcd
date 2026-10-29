---
schema_version: 1
id: "iss-2610262038319450"
slug: "the-surfaces-register-says-library-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 800 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:494"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is a page distilled from an ingested source part of the library or of memory? Answer A: the memory store (07-library, 07-memory and the code agree), so amend the register's library row to say the library holds the corpus and its cited pages sit in the memory store (recommended). Answer B: the library side, as the register's row says, which would move where those pages live."
finding_ref: cfk-5512bc6932693807
---

The surfaces register says library and memory are not one store under two names; the memory chapter's memory store holds the outside material the library ingests.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:494` — "the two are not one store with two names: the library is for what a decision may need to cite"
- End B: `.abcd/development/brief/05-internals/07-memory.md:51` — "User-ingested PDF via `/abcd:library ingest <path>`"

04-surfaces/README separates the library (outside material a decision may cite) from memory (lessons an agent would otherwise relearn) and says they are not one store with two names. 07-memory's `.abcd/memory/` takes user-ingested PDFs, articles and transcripts through `/abcd:library ingest` as its page classes, so "memory" names the library's cited material there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 800.
