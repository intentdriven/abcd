---
schema_version: 1
id: "iss-2608231226347380"
slug: "tagging-precedes-the-receipt-gate-so-a-failure-wedges-the-version"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "v0.6.2 release failure post-mortem 2026-08-23"
found_at: ".github/workflows/auto-release.yml"
resolution: "The tag is made by release.yml's own tag job, which needs verify and runs only on auto-release's fresh-tag path (create_tag). Order is detect -> verify -> tag -> build -> publish, in the committed workflows and both scaffold profiles; TestTheTagWaitsOnTheVerifyGate pins it. Also answers the duplicate iss-2609100513521322's first acceptance; its hand-pushed-tag heal loop and git-revert trailer points are not addressed here."
impact: fix
resolved_by:
  commit: "9092a7b036cf1a2d64101c9f4ae7b9f0f617b754"
---

`auto-release.yml` runs `detect` -> `tag` -> `release`, and the semantic
`receipt_gate` runs inside the `release` job, after the environment approval.
The tag is therefore created BEFORE the gate that can refuse the release.

When the gate refuses, the result is not a blocked release but a wedged one:

- The tag exists and is immutable by design — `detect` states it is NEVER moved.
- The heal path re-releases FROM the tagged commit, resolved to its immutable
  sha, so every retry checks out the same tree and derives the same content
  commit.
- That tree can never gain the missing receipts, because the receipts must live
  in a commit that is an ancestor of the released commit and name its
  predecessor. Landing them on the default branch afterwards does not change the
  tagged tree.

So the version is consumed. Recovery requires deleting a tag the workflow treats
as immutable, or abandoning the version and moving to the next one. Neither is a
path the release machinery offers.

Field hit 2026-08-23: v0.6.2 tagged at b06fa80, receipt gate refused, no Release
published, and no sequence of pushes to the default branch can heal it.

The deterministic gates do not have this problem — they run in `verify`, before
`tag`. Only the semantic gate sits on the wrong side of the tag. Whether the fix
is to arm `receipt_gate` in `verify`, to gate tagging on receipt presence in
`detect`, or to accept the wedge and document the recovery, is an ADR-shaped
question rather than a patch: it touches the immutability guarantee that makes
the heal path safe. Related: iss-2608231226342272 (the preview is silent about
this gate) and iss-2608231226274000 (the surface never documents the step).

**Decision (adr-52, accepted 2026-08-28): arm the receipt gate in `verify`.** Partially implemented: the gate now runs in `verify` (refusing before build/publish) and the content-commit derivation is receipts-dir based (iss-355). RESIDUAL: the auto-release path's `tag` job still pushes the tag before invoking `release.yml`/`verify`, so to fully stop a version being consumed the `tag` job must also be gated on the semantic verify. That is a maintainer-verified release-workflow change (CI cannot exercise it outside a real release); this record stays open for it.

## Grounds

- pursued: a refused verify (deterministic or semantic) on the auto-release path leaves no tag; shown wrong by any auto-release run that pushes a tag while its release.yml verify job is red, or by a workflow edit that lets the tag job run without verify success
