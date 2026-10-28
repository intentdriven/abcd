---
schema_version: 1
id: "iss-2610262038269520"
slug: "the-glossary-says-that-in-the-build-loop"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 795 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:18"
remedy: "none (filed automatically)"
finding_ref: cfk-04707417f0b777b5
---

The glossary says that in the build loop "step" names only a spec's piece; the agent catalog calls part of a lane's validate stage the implement loop's audit step.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:18` — "In the build loop the word names the spec's piece and nothing else"
- End B: `.abcd/development/brief/05-internals/01-agents.md:229` — "and to the implement loop's audit step on the"

step.md says that in the build loop the word names the spec's piece and nothing else, and a lane goes through stages. 01-agents calls the auditor's pass on the closing lane the implement loop's audit step, which is part of the validate stage and not one of a spec's steps.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 795.
