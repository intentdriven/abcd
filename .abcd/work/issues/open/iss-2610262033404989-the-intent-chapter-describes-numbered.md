---
schema_version: 1
id: "iss-2610262033404989"
slug: "the-intent-chapter-describes-numbered"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 429 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:472"
remedy: "none (filed automatically)"
finding_ref: cfk-3e2f283d6cc1ada7
---

The intent chapter describes numbered lint codes such as GL005, while the lint contract says rules are named, not numbered.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:472` — "Lint extractor for GL005 knows both shapes"
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"

05-intent describes, in the present tense, a lint extractor for GL005, beside the codes GL003 and GR002. 06-lint says the engine carries no numbered code catalogue and that rules are named, not numbered. Anyone following the intent chapter would create numbered codes that the lint contract rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 429.
