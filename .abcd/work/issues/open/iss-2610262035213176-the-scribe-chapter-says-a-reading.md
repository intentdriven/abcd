---
schema_version: 1
id: "iss-2610262035213176"
slug: "the-scribe-chapter-says-a-reading"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 566 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/32-scribe.md:6"
remedy: "In 32-scribe.md state the reading assembler's no-ledger rule with its one positional exception: at the comparative position the assembler hands two body fields of one derived widening run (adr-2609021016272867)."
finding_ref: cfk-95a407deafc6e442
---

The scribe chapter says a reading receives no ledger, but the reading chapter hands the comparative position a prior widening run's ledger items

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/32-scribe.md:6` — "a reading receives a positively included slice of the shipped repository and no ledger"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:62` — "its object is a prior widening run's pre-admission output"

32-scribe states the reading assembler's access rule as absolute: a reading receives a slice of the shipped repository and no ledger, and it counts reading records among the issue ledger's own directories. 23-reading says the comparative position's object is a prior widening run's pre-admission output, those reading records projected to two body fields, and calls this the one positional exception. The scribe chapter's inverse-access premise leaves that exception out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 566.
