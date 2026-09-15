---
schema_version: 1
id: "iss-2609011207114761"
slug: "the-changelog-composer-cannot-see-the-previous-release-surfa"
severity: "major"
category: "process"
source: "user-observation"
found_during: "the v0.7.0 cut, docs-currency gate"
found_at: "agents/release-changelog-composer.md"
origin: researcher-authored
production_mode: hand-written
resolution: "The composer writes only Added and Fixed until it can see the previous release: writableSections in internal/core/release is the one declaration of the set, the ingest refuses Changed, Deprecated, Removed and Security by name and by ruling, every derived section carries one sentence under its heading saying what the notes list and do not claim, and the composer prompt's table is pinned to the set by test. A breaking record is an Added line that states the break."
impact: fix
resolved_by:
  commit: "d45a2312"
---

The changelog composer cannot see the previous release surface, so its `Changed`
and `Removed` sections rest on trust. Its inputs are `added[]`, `removed[]` and
the record bodies at their paths. None of those is the shipped surface of
`base_tag`, yet three of the six Keep a Changelog sections are claims about
exactly that: `Changed`, `Deprecated` and `Removed` each assert that something a
user could reach in the previous release is now different, deprecated or gone.
The composer has no way to check any of them.

The failure is not hypothetical. In the v0.7.0 cut the composer wrote three
lines about a verb that did not exist in v0.6.9:

- `CHANGELOG.md:39` — "an invocation carrying only `--position` and `--target` —
  anything scripted against the previous release — stops working until a scope
  is added" (itd-199)
- `CHANGELOG.md:40` — "`abcd reading ingest` takes `--reading-json` where it took
  `--output-json`. The old spelling is gone rather than aliased"
  (iss-2608311725218136)
- `CHANGELOG.md:44`, under `### Removed` — "The comparative position no longer
  assembles." (itd-199)

Verified against the tag: `git show v0.6.9:internal/surface/cli/reading.go` and
`git show v0.6.9:commands/reading.md` are both absent, and `output-json` appears
zero times in the CLI reference at v0.6.0, v0.6.5 and v0.6.9. The whole `abcd
reading` verb ships for the first time in 0.7.0. The same section's `### Added`
announces it as new (itd-183, itd-184, itd-185), so the release record
contradicts itself and instructs a user to migrate a surface they never had.

The records are not at fault and must not be edited to compensate. itd-199's
`impact: fix` is a deliberate maintainer ruling whose fidelity verdict records
`cond-2608312031020321` as **survived**. iss-2608311725218136's body states the
disambiguating fact outright — "the verb shipped today and has no users" — in
the very text the composer read before writing that scripts against the previous
release stop working. Every fact needed was present; the prompt asked no question
that would have used them.

## Grounds

- pursued: the prompt-side half is fixed in this same change, adding a `base_tag`
  baseline rule to the section-choice guidance, because a release record that
  fabricates a migration is falsifiable by anyone who reads the previous tag

- pursued: the prompt-side half is fixed in this same change, adding a `base_tag`

## What is still open

The prompt rule is a trusted instruction, not a checkable one: the composer is
told to ask whether a capability was reachable in `base_tag` while still having
no way to look. Two candidate rungs, both binary-side:

- hand the composer the previous tag's surface as cut input, so the question it
  is asked is one it can answer from evidence rather than inference
- validate at ingest instead: refuse a payload whose `Changed`/`Deprecated`/
  `Removed` line cites only records absent from `base_tag`, which needs no new
  agent input and fails closed in the binary where the bijection already lives

The second is closer to the existing shape — `launch ship --changelog-json`
already re-derives the cut and proves the prose against it, so a baseline check
would join the completeness bijection rather than form a new gate. Deciding
between them is the work this issue holds.
