---
id: itd-70
slug: launch-release-retention-newest-per-line
severity: minor
related_issues: [iss-282]
---

# Drawn-out intent

Every `launch ship` prunes superseded releases under a newest-per-line (MAJOR.MINOR) retention policy: publishing vX.Y.Z removes the superseded vX.Y.(Z-1) tag and (once cut) its GitHub Release + assets, while the terminal release of every other line survives.

_Drawn out from a human brief edit by the brief-change derivation gate (itd-61 / spc-75). Grill + flesh out before planning._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).
The finding reverses this stub's direction.

- **Published versions are immutable by convention**: Semantic Versioning
  rule 3 says a released version's contents "MUST NOT be modified"; a change
  is a new version.
- **Ecosystems retract rather than delete**: the Go module mirror caches "to
  avoid breaking builds", keeps serving a version its origin removed, and
  tells authors to retract a bad version and publish a new one.
- **The forge now offers immutable releases (generally available
  2025-10-28)**: once published, a release's "assets can't be added, modified,
  or deleted" and its tag "can't be deleted or moved"; the release can still be
  deleted, after which the tag name cannot be reused; a release attestation is
  minted. `release.yml` already runs the draft-then-publish flow "immutable
  releases require", so the setting is a repository switch. It does not block
  the prune: a superseded Release and its tag can still be deleted, at the
  price that the tag name can never be reused and no asset of a published
  release can be swapped in place.
- **Provenance outlives the asset**: a public repository's attestations sit in
  an "immutable transparency log" whatever happens to the Release.
- **Where "newest per line" is practised it is local disk retention**: the
  harness's installer keeps the two newest versions on disk; Homebrew removes
  old local versions on upgrade. Neither deletes anything published.
- **The prune has run by hand once** (v0.6.7 pruned v0.6.1 to v0.6.6, tags
  kept, per the 2026-08-27 pilot note), and the advisories published for that
  cut name those versions.

**Path 3 — deliberate divergence from the brief's own retention design**,
argued from the field's immutability convention (semver rule 3, Go's
retract) and from the advisories whose `patched_versions` point at pruned
versions, not forced by the forge, which still allows the prune: enable
immutable releases, keep every release,
turn `ComputeRetention` into a report of what each line holds, and let the
only retention act be the status edit an immutable release still allows (a
superseded patch loses "latest"). The local preference overturned is brief
§3's "shipping a patch removes the superseded patch and its Release assets";
under prefer-sota it is revisited through an ADR at this intent's planning,
not silently kept or broken. No code beyond the setting and a rename; no
dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Does the ADR retire the brief's retention paragraph
  outright, or keep newest-per-line as a report that names the superseded
  patches without deleting them?
- (SOTA 2026-10-23) Are the already-pruned v0.6.x Releases left as they are
  (tags present, no Release), and does the launch report say so where an
  advisory's `patched_versions` points at them?
