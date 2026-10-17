---
schema_version: 1
id: "iss-2610142319016723"
slug: "stale-usage-unstamped-path-build-no"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Name the source build in the note for an unstamped binary outside every plugin root (rebuild it from its source checkout, for example with make build or go install ./cmd/abcd), so it is told how to catch up without being called a stale release copy; test that branch names the rebuild, watched fail first."
---

An unstamped abcd binary on PATH, built by go install or make build from a source checkout, now gets no advice on how to catch up when typed a verb or flag the plugin surface documents: staleUsageNote's unstamped branch (internal/surface/cli/staleusage.go) says only that the thing is not in this build, with no abcd update and no rebuild hint. This is the intended trade-off of iss-2610120921425348 (an unstamped build's age is not what the page proves), recorded so it stays visible. Found by the Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15.
