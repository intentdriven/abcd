---
schema_version: 1
id: "iss-2610262038176813"
slug: "universal-patterns-lean-on-a-lint-code"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 785 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:188"
remedy: "none (filed automatically)"
finding_ref: cfk-52769062ca7e1f5a
---

Universal patterns lean on a lint code reserved in the lint contract; the lint contract keeps no numbered code catalogue.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:188` — "Lint blocks if a namespace's curation behaviour disagrees with its declared class (lint code reserved at `06-lint.md`)."
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"

04-universal-patterns says lint blocks a namespace whose curation disagrees with its declared lifecycle class and that the lint code is reserved at 06-lint.md. 06-lint says rules are named, not numbered, that literal enumerations belong only in generated regions, and it reserves no such rule. The dependency points at a reservation the lint contract does not make.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 785.
