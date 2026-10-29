---
schema_version: 1
id: "iss-2610262033534032"
slug: "the-glossary-says-embark-writes-only-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 446 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:66"
remedy: "In record.md replace 'the four record families embark writes' (the senses row and the example) with the five embark writes: ADRs, issues, intents, specs and release retrospectives."
finding_ref: cfk-1f1d1935dae770b2
---

The glossary says embark writes only the four record families; the reflect chapter says embark also writes retrospectives back, sealed with the other record families.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:66` — "`embark` writes only the four record families, verbatim, into their canonical locations."
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:203` — "the other record families. The embark write puts them back into the new"

glossary/core/record.md fixes 'a record family' as one of the four families embark writes (ADRs, issues, intents, specs), with the example that embark writes only those four. 09-reflect says a lifeboat carries retrospectives sealed 'with the other record families' and that 'The embark write puts them back into the new voyage's store'. Either retrospectives are a fifth family that embark writes, or embark writes only four.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 446.
