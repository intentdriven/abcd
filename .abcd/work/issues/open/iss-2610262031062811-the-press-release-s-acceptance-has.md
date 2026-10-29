---
schema_version: 1
id: "iss-2610262031062811"
slug: "the-press-release-s-acceptance-has"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 87 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:43"
remedy: "Reword 04-universal-patterns.md:90 to say these verbs write no report file of their own, and split the list: capture, intent, spec and identity write records, while lint, guard, banlist, rules and --version are read-only renders."
finding_ref: cfk-aba87e37c6230e40
---

The press release's acceptance has capture write an issue file, but universal patterns calls capture (and intent and spec) a read-only render

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:43` — "a structured issue at `.abcd/work/issues/open/iss-N-<slug>.md` exists with frontmatter populated and source provenance recorded"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders"

The press release's acceptance criterion says `/abcd:capture "<text>"` leaves a structured issue file under `.abcd/work/issues/open/`. Universal patterns lists `capture`, `intent` and `spec` with `lint`, `guard` and `banlist` as read-only renders. A verb that writes a record into the ledger is not a read-only render.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 87.
