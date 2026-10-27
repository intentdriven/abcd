---
schema_version: 1
id: "iss-2610262033559124"
slug: "the-intent-chapter-makes-the-grill"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 452 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:319"
remedy: "none (filed automatically)"
finding_ref: cfk-c231d8c4e992f9f2
---

The intent chapter makes the grill glossary-aware once a terminology/ directory exists; the glossary's home is .abcd/development/brief/glossary/.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "Glossary-aware once `terminology/` exists."
- End B: `.abcd/development/brief/glossary/README.md:66` — "The glossary lives at `.abcd/development/brief/glossary/` (the home, per adr-30)."

05-intent says the grill step is 'Glossary-aware once terminology/ exists'. The glossary README says the one canonical glossary lives at .abcd/development/brief/glossary/ (the home, per adr-30), and its grill design keys glossary-aware mode on that directory. Two names point at one directory, and a builder following 05-intent would test for a directory the record does not use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 452.
