---
schema_version: 1
id: "iss-2608220150157499"
slug: "mirrored-typed-references-278-stored-226-distinct"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: ".abcd/development"
resolution: "handled at tip: the site record export collapses mirrored typed references so each distinct link renders once (internal/core/site/recordjson.go, 766da566)"
impact: internal
resolved_by:
  commit: "766da56642e968b8e12b3ada4373703a2bd3f1ab"
---

The spec link is recorded from both ends (intent spec_id and spec implements, 30+ pairs) and 20 related_* pairs are listed in both files, so the record's 278 stored typed references are 226 distinct links. Harmless today, but every consumer that renders or counts links must collapse mirrored references or double-count; the site build is the first such consumer

## Grounds

- pursued: the first consumer of the record graph counts each distinct link once; shown wrong if the site export renders a mirrored pair as two edges
