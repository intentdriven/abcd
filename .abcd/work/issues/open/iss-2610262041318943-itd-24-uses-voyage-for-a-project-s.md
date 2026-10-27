---
schema_version: 1
id: "iss-2610262041318943"
slug: "itd-24-uses-voyage-for-a-project-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1006 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-80d43c55ff1da7e2
---

itd-24 uses "voyage" for a project's lifetime of work; the glossary defines voyage as the operations namespace of disembark and embark runs and forbids it for a project

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:55` — "of the voyage's retrospectives into the lifeboat"
- End B: `.abcd/development/brief/glossary/core/voyage.md:59` — "Do not use it for a "project" (too generic)"

itd-24 speaks of packing all of the voyage's retrospectives, and of a new or previous voyage, using voyage to mean a project's arc of releases. The glossary pins voyage to the operator-level operations namespace that records disembark and embark runs, and says not to use it for a project. adr-35 retired the lifecycle sense, and the entry's open question leaves it unnamed. Read with the glossary's meaning, "the voyage's retrospectives" names nothing that holds retrospectives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1006.
