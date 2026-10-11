---
schema_version: 1
id: "iss-367"
slug: "commands-launch-md-documents-a-top-level"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "commands/launch.md"
resolution: "Reworded commands/launch.md to reference bundle.files (an array to count) instead of a nonexistent top-level files key."
impact: fix
---

commands/launch.md documents a top-level files count in launch --dry-run --json, but DryRunReport has no such key; the file list is bundle.files, an array; the bullet violates the file's own dotted-path convention
## Evidence

- `commands/launch.md:30` — "- `files` — how many files the bundle would include."
- `internal/core/launch/dryrun.go:38-48` — `DryRunReport` top-level keys are version, bundle, scan, lockstep, retention, smoke, gates, would_publish, would_refuse_on; no `files`.
- The file list is `bundle.files` (array of objects). The `files int` count lives on `PayloadRenderResult` (`render.go`), which `--dry-run` never emits.
- Surrounding bullets use dotted paths (`scan.hard_fails`, `smoke.ok`), so bare `files` reads as a nonexistent top-level key.

## Adversarial verdict

CONFIRMED (nitpick). Datum recoverable from `bundle.files`; violates the file's own dotted-path convention. Fix: reword the bullet to `bundle.files` (count the array).
