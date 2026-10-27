---
schema_version: 1
id: "iss-2610262041301883"
slug: "shipped-itd-24-has-embark-surface-past"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1005 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-ae0fd856ce38c871
---

Shipped itd-24 has embark surface past retrospectives during its press-release interview; the verification matrix says embark's press-release interview is staged and what ships is probe-then-write

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:55` — "`/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:62` — "What ships is probe-then-write with the coverage blanks a human must answer surfaced first"

itd-24 is shipped, and its scope says `/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview, with its decision 2 ranking those lessons on embark. The verification matrix marks the embark press-release interview as staged, with no shipped surface: what ships is probe-then-write, and embark curates nothing. Embark cannot surface retrospectives inside an interview the brief says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1005.
