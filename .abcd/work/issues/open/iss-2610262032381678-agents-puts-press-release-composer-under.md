---
schema_version: 1
id: "iss-2610262032381678"
slug: "agents-puts-press-release-composer-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 297 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:20"
remedy: "In 01-agents.md move press-release-composer out of the cite-or-be-dropped group and say, as for the changelog composer, that its payload is refused whole; also correct the blanket 'validating mode is cite-or-be-dropped' sentence in 02-disembark.md."
finding_ref: cfk-08b1eb1c28a8a307
---

agents puts press-release-composer under cite-or-be-dropped validation; disembark says the press release is accepted or refused as a whole

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:20` — "each feeding one verb that validates its output under a cite-or-be-dropped rule"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:185` — "The document stands or falls whole"

The agent catalog groups press-release-composer with the agents whose output is validated under a cite-or-be-dropped rule, and contrasts that with the changelog composer, whose payload is refused whole. The disembark chapter says the press release stands or falls whole: a payload citing none of the required paths is refused with exit 2. By the catalog's own terms, that is refusal as a whole, not cite-or-be-dropped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 297.
