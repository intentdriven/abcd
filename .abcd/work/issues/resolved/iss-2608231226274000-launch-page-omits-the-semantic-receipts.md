---
schema_version: 1
id: "iss-2608231226274000"
slug: "launch-page-omits-the-semantic-receipts"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "v0.6.2 release failure post-mortem 2026-08-23"
found_at: "commands/launch.md"
resolution: "commands/launch.md now carries a Semantic receipts section as part of the ship flow: the two host-run passes, the two-commit release-branch shape, where receipts live, and that a HOLD stops the release. The brief's launch surface page matches."
impact: fix
resolved_by:
  commit: "1ff105b"
---

`commands/launch.md` documents the release cut as three steps over two Go entry
points: emit the cut, compose the prose, ingest and write the heading. It never
mentions the two host-run semantic passes, or the receipts they produce, or the
two-commit branch shape the receipts require.

That shape is mandatory. `release.yml` arms `receipt_gate` against the release
CONTENT commit, derived as `<merge>^2^` on the auto-release path, so the release
branch must be exactly two commits: the CHANGELOG roll, then a commit recording
the receipts that name it. A receipt can never sit in the tree of the commit it
names, because adding it would change that commit's sha.

The procedure is written down correctly, but in
`.abcd/development/release-gate/README.md` — a development-tier runbook that
the launch page references only in passing, as one of the files
`launch scaffold` writes. Nothing on the path a user actually follows says the
step exists.

Field hit 2026-08-23: a release cut followed the launch page end to end,
produced a one-commit release branch, merged, tagged v0.6.2, and failed at
`Semantic-gate receipts (fail-closed)` with no receipt for the armed content
commit. Everything before it passed. The page's own flow was completed exactly
as written and still could not ship.

The fix is on the page, not in the runbook: the ship flow gains the semantic
passes and the two-commit shape as first-class steps, so following the surface
produces a shippable release.