---
schema_version: 1
id: "iss-2610262038174723"
slug: "the-lint-contract-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 784 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:70"
remedy: "none (filed automatically)"
finding_ref: cfk-164ae7fbf9ffdbd6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032399609. Same End A; 01-agents.md's design-roster note is the second of two reasons the sentence is false."
duplicates: [iss-2610262032399609]
---

The lint contract says `documentation-auditor` audits it on every disembark; the agent catalog lists that agent in the design roster, none of which exists.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:70` — "The `documentation-auditor` agent (per `01-agents.md`) audits this contract on every disembark"
- End B: `.abcd/development/brief/05-internals/01-agents.md:80` — "document holds their original plan), and none of them exists in"

06-lint says, in the present tense, that the `documentation-auditor` agent audits the lint contract on every disembark, and builds its acceptance on that, citing 01-agents. 01-agents puts `documentation-auditor` in the design roster still to be built and says none of those agents exists in `agents/`. An agent that does not ship cannot be auditing anything.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 784.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032399609. Same End A; 01-agents.md's design-roster note is the second of two reasons the sentence is false.
