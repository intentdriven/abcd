---
schema_version: 1
id: "iss-2610262035280605"
slug: "the-decide-chapter-says-a-decision"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 574 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/24-decide.md:68"
remedy: "In 04-universal-patterns.md:196 move the decisions folder (and the issue ledger's records) out of the Compounding-curated row into Append-only: a decision record is created exclusively and never edited (24-decide.md:68), and the same table already lists issue-ledger entries as append-only."
finding_ref: cfk-41b679b1f51bc544
---

The decide chapter says a decision record is never edited; the lifecycle taxonomy puts the decisions folder in the compounding-curated class, whose pages are modified, contradicted and deprecated.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/24-decide.md:68` — "A record is never edited; a correction is a new"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:196` — "Accumulates across sessions/runs; pages added, modified, contradicted, deprecated by curator. Carries provenance per entry; lint surfaces drift between curated form and source-of-truth. | `.abcd/memory/` (multi-upstream knowledge substrate per itd-36), `.abcd/work/` (shared working record — CONTEXT.md, the `.abcd/work/decisions/` folder"

24-decide states that a decision record is never edited and that a correction is a new record, which is the append-only class's behaviour. 04-universal-patterns places `.abcd/work/decisions/` under Compounding-curated, defined as pages added, modified, contradicted and deprecated by a curator. The taxonomy says lint blocks a namespace whose curation behaviour disagrees with its declared class, so the folder cannot be both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 574.
