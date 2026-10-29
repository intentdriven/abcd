---
schema_version: 1
id: "iss-2610262035387982"
slug: "the-lint-chapter-says-its-site-target"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 586 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:9"
remedy: "Reword the sentence at 04-universal-patterns.md:90: it is about reports, so say these verbs write no report pair, not that they are read-only renders with zero writes. capture, intent and spec write records, guard writes a branch record and the question marker, identity and banlist have write paths, and lint site builds into its output directory when it holds no index.html (16-lint.md:9)."
finding_ref: cfk-f6e5defdf3bcb034
---

The lint chapter says its site target writes a render; universal patterns says lint and its targets are read-only renders and that `abcd lint` is documented as performing zero writes.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:9` — "It is **read-only** but for one target: bare and every other target perform"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`lint` and its targets, `capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders, and `abcd lint` in particular is documented as performing zero writes"

16-lint says the verb is read-only except for one target: the site target renders the site into its output directory when that directory holds no `index.html`. 04-universal-patterns lists 'lint and its targets' among the read-only renders and says `abcd lint` is documented as performing zero writes. That is not what the lint chapter documents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 586.
