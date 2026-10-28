---
schema_version: 1
id: "iss-2610262031492489"
slug: "invariant-15-refuses-any-path-at-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 183 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:167"
remedy: "none (filed automatically)"
related_intents: [itd-86]
finding_ref: cfk-36ab68450553b0c7
---

Invariant 15 refuses any path at a reading's invocation, but itd-86's cold reading takes exactly one target document.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:167` — "a repository path is never accepted at the invocation"
- End B: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29` — "A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone."

Invariant 15 says a reading's operands are only a position and a target state, that what it is handed comes from the committed preset for that position, and that a repository path is never accepted at the invocation (adr-2609021016286571, which supersedes adr-58's scope operand). itd-86, still listed by invariant 14 as a consumer, defines the cold reading as a surface that takes exactly one target document.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 183.
