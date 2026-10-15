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
---

release/indexregions_test.go checks that no generated region's source directory is one the release cut writes by comparing against a hard-coded list of the cut's write paths, so a new write path added to the cut is not noticed unless someone also updates the list.
