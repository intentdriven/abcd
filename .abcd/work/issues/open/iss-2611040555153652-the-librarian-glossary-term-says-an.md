---
schema_version: 1
id: "iss-2611040555153652"
slug: "the-librarian-glossary-term-says-an"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/librarian.md"
remedy: "Mark the term's body as a design target naming the draft intent, until `library pack` ships."
---

The librarian glossary term says an agent runs it with `abcd library pack`, in the present tense (.abcd/development/brief/glossary/core/librarian.md:17). The binary registers no such command: it is refused as unknown, and the sub-commands are add, ask, cite-check, declassify, ingest, init, ledger, lint and sync-banlist. The backing intent itd-2610141646126492 is a draft, and the term's status: draft records term maturity, not that the verb is unbuilt.
