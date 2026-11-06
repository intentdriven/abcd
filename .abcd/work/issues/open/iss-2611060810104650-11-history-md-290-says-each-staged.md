---
schema_version: 1
id: "iss-2611060810104650"
slug: "11-history-md-290-says-each-staged"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Say the sidecar holds the session and lineage while the filename carries the stamp and the lock key."
---

`11-history.md:290` says each staged transcript's sidecar holds its session and lineage, so nothing is ever encoded in the filename. Staged names encode the capture time and the key, `<timestamp>-<key>.raw` (`stagedFilename`, `internal/core/history/staging.go:290`), and `stagedLockKey` (`internal/core/history/staging.go:443`) derives the agent or session key from that name without reading the sidecar. Pending transcripts are named `<session-id>.raw` (`pendingPaths`, `internal/core/history/pending.go:107`).
