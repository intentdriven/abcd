---
schema_version: 1
id: "iss-2610262038415265"
slug: "itd-1-cites-lint-code-il002-and-points"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 811 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:30"
remedy: "In itd-1 line 30 delete 'Lint code: IL002 ...' and the pointer to 05-internals/06-lint.md, and name the intent package's own check instead."
related_intents: [itd-1]
finding_ref: cfk-1a0d08947b0b51a8
---

itd-1 cites lint code IL002 and points to 06-lint.md for it; 06-lint says the engine has no numbered codes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:30` — "Lint code: `IL002` (delivered by spc-8 (predecessor store); see `05-internals/06-lint.md`)."
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"

itd-1 names its drafts-to-planned hard block as lint code IL002 and points to 06-lint.md. 06-lint says the engine carries no numbered code catalogue and that rules are named in the JSON configs. The code itd-1 cites does not exist in the document it points to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 811.
