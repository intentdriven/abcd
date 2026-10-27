---
schema_version: 1
id: "iss-2610262040189116"
slug: "the-lint-contract-says-rules-are-named"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 922 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-43]
finding_ref: cfk-669af085835aa8e5
---

The lint contract says rules are named, never numbered; itd-43 says a `GL002` lint catches regressions

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"
- End B: `.abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:40` — "and the `GL002` lint catches regressions"

06-lint says the engine has no numbered code catalogue: rules are named, not numbered, and are defined in the two JSON configs. itd-43 says its spec term-file rename is guarded by the `GL002` lint. A numbered rule code has no place in a lint contract that says no numbering exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 922.
