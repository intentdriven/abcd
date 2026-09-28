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
resolution: "Describe opens one os.Root over the repository when anything is parked or staged, and the stage's commit-marker probe and the parked run's outcome probe both read through it, so a readings directory symlinked out of the checkout refuses the render for both."
impact: fix
resolved_by:
  commit: "97fa2de399b6c9c65ef48852482f4aa32a233ef8"
---

The reading status render probes an ingest stage's commit marker with an unbounded os.Lstat over the joined path, while its staged-runs probe reads through an os.Root over the repository. The two disagree on a symlink: with .abcd/development/readings symlinked outside the checkout, a stage whose run is parked makes Describe refuse (path escapes from parent), while a stage alone is classified as a leftover or an orphan by a marker read outside the repository. Both probes should go through the one root.

## Grounds

- pursued: with the readings directory symlinked outside the checkout and only a stage present, Describe refuses instead of classifying the stage by a marker read outside the repository, and the existing leftover/orphan/staged-runs render tests stay green; a status render that reads a marker through a symlink out of the checkout would show it wrong
