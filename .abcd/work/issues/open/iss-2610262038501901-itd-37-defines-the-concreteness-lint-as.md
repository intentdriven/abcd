---
schema_version: 1
id: "iss-2610262038501901"
slug: "itd-37-defines-the-concreteness-lint-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 822 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:56"
remedy: "Rewrite itd-37's concreteness lint as named record-lint rules instead of MG001-MG004 numbered codes, and say they are designed, not built (06-lint.md: rules are named, not numbered)"
related_intents: [itd-37]
finding_ref: cfk-71e74d6553bf2bd5
---

itd-37 defines the concreteness lint as numbered codes MG001 to MG003; 06-lint says the engine has no numbered codes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:56` — "in the native intent lint with three codes:"
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"

itd-37 specifies its concreteness lint as three numbered codes, plus MG004 and VR001 elsewhere. 06-lint says the lint engine carries no numbered code catalogue and that rules are named in the JSON configs. Building itd-37 as written breaks the lint contract's naming rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 822.
