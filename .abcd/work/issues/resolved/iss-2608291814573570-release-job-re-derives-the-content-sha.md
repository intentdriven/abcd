---
schema_version: 1
id: "iss-2608291814573570"
slug: "release-job-re-derives-the-content-sha"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: ".github/workflows/release.yml"
resolution: "verify's semantic-gate step exports the admitted sha as the content_sha job output; the release job receives and shape-checks it instead of re-deriving it. TestReleaseConsumesTheContentShaVerifyGated pins it."
impact: internal
resolved_by:
  commit: "bad409cc52f3cd15f0c85bd1c8c05fda9ae2f550"
---

ultra-v0.6.8 below-cap eff-2/simp-1: the release job in .github/workflows/release.yml re-derives the content sha (go run ./cmd/record-lint --derive-content-sha) instead of consuming the verify job's output — a second compile plus a full-history walk per release, and a second derivation that can in principle diverge from the one the gate passed. Cleanup: emit the sha as a verify job output and consume it.

## Grounds

- pursued: one derivation per release, and the release job signs the receipts of the commit verify gated; shown wrong by a derive-content-sha call in the release job or a release run whose CONTENT_SHA differs from verify's
