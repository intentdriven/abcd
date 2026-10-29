---
schema_version: 1
id: "iss-2610262031401131"
slug: "the-read-block-entry-puts-all-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 160 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/read-block.md:17"
remedy: "In ledger/read-block.md add the one positional exception: at the comparative position two body fields of one committed widening run's items are admitted as the candidate set (adr-2609021016272867, brief invariant 15); every other prior-run output stays inside the block."
finding_ref: cfk-bcbe37d864c222e3
refines: [iss-2609020626100041]
---

The read-block entry puts all the instrument's own outputs inside the block, while invariant 15 admits widening-run items at the comparative position

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/read-block.md:17` — "The instrument's own outputs, manifests and reading records, sit inside the read-block on the next run."
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:167` — "The instrument's own output is never its input, with one positional exception adr-2609021016272867 states"

The glossary says the instrument's own outputs, manifests and reading records sit inside the read-block on the next run, with no exception. Invariant 15 says the instrument's own output is never its input with one positional exception: at the comparative position, two body fields of a derived widening run's items are admitted as the candidate set. An assembler built from the glossary excludes what the invariant requires the comparative reading to receive.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 160.
