---
schema_version: 1
id: "iss-2609261905354450"
slug: "the-reading-status-render-probes-an-ingest-stage-s-commit"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainRd"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
---

The reading status render probes an ingest stage's commit marker with an unbounded os.Lstat over the joined path, while its staged-runs probe reads through an os.Root over the repository. The two disagree on a symlink: with .abcd/development/readings symlinked outside the checkout, a stage whose run is parked makes Describe refuse (path escapes from parent), while a stage alone is classified as a leftover or an orphan by a marker read outside the repository. Both probes should go through the one root.
