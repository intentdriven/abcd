---
schema_version: 1
id: "iss-2610262032410731"
slug: "the-lint-contract-counts-previously-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 304 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:10"
remedy: "none (filed automatically)"
finding_ref: cfk-4cff3fe8fc53ee98
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. .abcd/docs-lint.json bans the bare word as a blocker and the launch tier is a separate narrower scan of the payload, so a bare 'previously' fails the first before it reaches the second."
---

the lint contract counts 'previously' as change narration; launch's change-narration gate passes a bare 'previously' as present-tense prose

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:10` — "change-narration (past-tense drift such as "previously" or "formerly")"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:303` — "Bare "now" and bare "previously" are present-tense prose and pass"

The lint contract defines the change narration it lints in docs/ as past-tense drift and gives 'previously' as an example. Launch's change-narration gate, run over the same Markdown under docs/, says a bare 'previously' is present-tense prose and passes. The term means different things in the two chapters, so the same sentence is drift to one check and acceptable to the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 304.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. .abcd/docs-lint.json bans the bare word as a blocker and the launch tier is a separate narrower scan of the payload, so a bare 'previously' fails the first before it reaches the second.
