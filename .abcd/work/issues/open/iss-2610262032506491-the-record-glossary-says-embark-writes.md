---
schema_version: 1
id: "iss-2610262032506491"
slug: "the-record-glossary-says-embark-writes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 323 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:104"
remedy: "none (filed automatically)"
finding_ref: cfk-c40d8e5e8fb3dcf4
---

the record glossary says embark writes four record families; embark reads and writes five, adding release retrospectives

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:104` — "The five record families — ADRs, issues, intents, specs, release retrospectives — plus the report-only files that inform the run."
- End B: `.abcd/development/brief/glossary/core/record.md:32` — "the four families `embark` writes"

core/record defines 'a record family' by pointing at the four families embark writes in 03-embark, and its disagreement note quotes that chapter as counting four (ADRs, issues, intents, specs). 03-embark reads the lifeboat's five record families, adding release retrospectives, and lands all five. The glossary's anchoring example describes a chapter that no longer says what it cites.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 323.
