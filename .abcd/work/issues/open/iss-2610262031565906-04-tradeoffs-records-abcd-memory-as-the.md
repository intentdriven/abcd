---
schema_version: 1
id: "iss-2610262031565906"
slug: "04-tradeoffs-records-abcd-memory-as-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 198 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:78"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-7737f2157ae95533
---

04-tradeoffs records /abcd:memory as the curated store's command with ingest/ask/lint; itd-39 says memory is now the agents' notes command and the store is reached through library.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:78` — "Since the verb split, `memory` is the agents' notes command"
- End B: `.abcd/development/brief/03-evidence/04-tradeoffs.md:37` — "ship `/abcd:memory` command initially with `ingest`/`ask`/`lint` sub-verbs"

The memory-unification tradeoff records the choice to 'ship `/abcd:memory` command initially with `ingest`/`ask`/`lint` sub-verbs'. itd-39 says 'Since the verb split, `memory` is the agents' notes command' and that the store is reached as library ingest, library ask and library lint. The name memory now refers to two different commands in the corpus, and the tradeoff entry carries no note that the verb was split.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 198.
