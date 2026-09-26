---
schema_version: 1
id: "iss-2609251939468296"
slug: "a-pre-release-changelog-head-such-as-1-0-0-rc-1-date-is"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
resolution: "A strict reader (changelog.ReleasedVersionIn) refuses a newest release heading the dated-heading reader cannot parse, so a pre-release head no longer binds to the previous release."
impact: fix
resolved_by:
  commit: "2ccf16b73b3b870a493bf9a526dac30cbb330c9c"
---

A pre-release CHANGELOG head such as '## [1.0.0-rc.1] - <date>' is invisible to the dated-heading reader, so the release gate's version binding compares against the previous version and the previous release's receipts admit the release. The release workflow's tag pattern accepts -rc and +build suffixes, so a hand-pushed pre-release tag reaches the gate. The derivation should refuse when the newest release heading is one the reader cannot parse.

## Grounds

- pursued: a pre-release, build-metadata or undated newest heading refuses naming the heading (TestDeriveReleaseContentSha_RefusesAnUnparseableNewestHeading, review probe S5); deriving the previous cut's receipts under such a head would show it wrong.
