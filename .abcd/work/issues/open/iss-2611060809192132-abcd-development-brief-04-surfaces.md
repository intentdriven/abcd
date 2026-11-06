---
schema_version: 1
id: "iss-2611060809192132"
slug: "abcd-development-brief-04-surfaces"
severity: "major"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Scope the surface guard to a repository that ships abcd's own command tree, or add a verb that writes a managed repository's baseline, and state which in the chapter."
deferred_after: v0.16.1
deferral_reason: "The fix needs a design for who lays the release snapshot in a managed repository and what its remedy names, and abcd's own cut does not take this path; it is in the follow-up release."
---

`.abcd/development/brief/04-surfaces/04-launch.md:48` says the gate's inputs are the ledger, the anchor tag and the version location, kind-independent, so the cut runs unchanged for every kind, and lines 25-30 name a release path for a repository that ships no plugin. The cut also needs `.abcd/development/release/surface.json` in the anchor tag and at HEAD (`internal/core/changelog/guard.go:103`, `:184`), equal to abcd's own compiled command tree plus the repository's manifests (`internal/surface/cli/surface.go:38`). Only `cmd/abcd-gen-surface` in abcd's own tree creates that file: `abcd ahoy install` does not lay it, and the cut only refreshes one already there (`refreshSurfaceSnapshot`, `internal/surface/cli/ship.go:187`), after the guard has passed. A repository declaring `binary`, with a tag and a shipped intent, is refused under `surface-guard` by `abcd launch --dry-run`, and the stale-snapshot remedy names `go generate ./internal/surface/cli`, a package that repository does not have.

## Deferral 2026-11-06

Deferred past v0.16.1: The fix needs a design for who lays the release snapshot in a managed repository and what its remedy names, and abcd's own cut does not take this path; it is in the follow-up release.
