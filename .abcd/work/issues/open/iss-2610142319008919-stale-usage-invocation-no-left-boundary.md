---
schema_version: 1
id: "iss-2610142319008919"
slug: "stale-usage-invocation-no-left-boundary"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Anchor the invocation head on a left word boundary before abcd, and require \\s+ (or the : of the /abcd: slash form) between abcd and a non-empty known path instead of \\s*; test that a fenced `myabcd build next --x` or `abcdbuild next --x` does not document build next --x, watched fail first."
---

The invocation regexp in synopsisCarriesFlag (internal/surface/cli/staleusage.go) has no left word boundary before abcd and joins the known path with \s*, which allows zero whitespace: a fenced line `myabcd build next --x` or `abcdbuild next --x` counts as an invocation of `abcd build next` that documents --x, so the stale-binary note can be granted on a line that never invokes abcd. Found by the Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15.
