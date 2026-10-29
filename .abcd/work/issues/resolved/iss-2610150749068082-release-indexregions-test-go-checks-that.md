---
schema_version: 1
id: "iss-2610150749068082"
slug: "release-indexregions-test-go-checks-that"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "iss-2608220150157511 fix review pair"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release"
remedy: "Derive the cut's write set from the release writer itself (or run a cut in a fixture and diff the regions) rather than a hard-coded list."
resolution: "The generated-region check in the release package no longer compares against a hand-kept list of the cut's write paths: it runs a full cut through the writer seam in a fixture that makes every write a cut can make (target move, archive, page, the repository's declared digest, changelog) and checks the paths the writer actually wrote, so a write the cut gains is seen without a list to update. The old list had already missed the declared release digest."
impact: internal
resolved_by:
  commit: "9b47ae73b17a8a0e29b77d44f0b99941c80de24f"
---

release/indexregions_test.go checks that no generated region's source directory is one the release cut writes by comparing against a hard-coded list of the cut's write paths, so a new write path added to the cut is not noticed unless someone also updates the list.

## Grounds

- pursued: a generated index over the directory of any path the release writer writes is reported as one a cut makes stale; TestRegionCheckCoversEveryCutWrite would show it wrong by failing for any written path whose directory the check does not report (it failed on the old list for internal/core/releasedigest/digest.json).
