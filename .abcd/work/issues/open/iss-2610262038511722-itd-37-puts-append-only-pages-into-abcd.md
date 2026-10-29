---
schema_version: 1
id: "iss-2610262038511722"
slug: "itd-37-puts-append-only-pages-into-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 823 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:63"
remedy: "Amend section 8 to say a namespace's declared class is its dominant behaviour and entry-level immutability inside it is allowed (as already for the issue ledger and memory's immutable source classes), so itd-37's per-spec append-only pages can sit in .abcd/memory/"
related_intents: [itd-37]
finding_ref: cfk-d05351831fdb0337
---

itd-37 puts append-only pages into .abcd/memory/; universal pattern § 8 gives each namespace one lifecycle class and lists .abcd/memory/ as compounding-curated

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:63` — "Writes append-only `spec_modification_grammar_<spec_id>.md` to `.abcd/memory/` with `source.class: spec_modification_grammar`."
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:188` — "Lifecycle class is declared in the parent README of each artefact namespace."

itd-37 writes append-only per-spec pages into .abcd/memory/, beside a compounding-curated per-domain page, and cites § 8 for both lifecycles. 04-universal-patterns § 8 declares one lifecycle class per artefact namespace in its parent README, lists .abcd/memory/ as compounding-curated, and has lint block a namespace whose behaviour disagrees with its class. One namespace cannot hold both of itd-37's classes under that taxonomy.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 823.
