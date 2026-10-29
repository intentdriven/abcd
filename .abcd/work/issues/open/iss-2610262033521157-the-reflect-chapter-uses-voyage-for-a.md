---
schema_version: 1
id: "iss-2610262033521157"
slug: "the-reflect-chapter-uses-voyage-for-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 445 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md:201"
remedy: "In 09-reflect.md:201-205 replace 'the voyage produced' and 'the new voyage's store' and 'the new voyage's brief' with 'the project's' and 'the target repository's' wording; the glossary fixes voyage as the operator-level log of disembark and embark runs."
finding_ref: cfk-e4102e56c9c60775
---

The reflect chapter uses 'voyage' for a project's lifetime ('every retrospective the voyage produced', 'the new voyage's store'); the glossary fixes voyage as the operator-level log of disembark and embark runs.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/09-reflect.md:201` — "A lifeboat pack carries every retrospective the voyage produced, verbatim, as"
- End B: `.abcd/development/brief/glossary/core/voyage.md:4` — "an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run)"

glossary/core/voyage.md defines a voyage as the operations namespace under ~/.abcd.noindex/voyage/< source-root-sha>/, an append-only record of what abcd did to produce a lifeboat. It warns against using the word for a project, and records the end-to-end lifecycle sense as retired by adr-35 and unsettled. 09-reflect says a lifeboat carries every retrospective 'the voyage produced', and that embark writes them into 'the new voyage's store' and ranks lessons against 'the new voyage's brief'. A run log produces no retrospectives and has no store or brief, so the chapter is using the retired lifecycle sense.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 445.
