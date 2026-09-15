---
schema_version: 1
id: "iss-2608291814563403"
slug: "licence-whitespace-check-differs-between-source-shapes"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/schema.go"
resolution: "one requireLicence next to requireClass/requireCitation/requireDate trims and refuses on both shapes; a test drives empty, space and tab licences through the single-source and sources[] shapes"
impact: fix
---

ultra-v0.6.8 C5: internal/core/memory/schema.go checks licence with lic == "" in validateSourceEntry (the sources[] shape) but strings.TrimSpace(lic) == "" in the single-source branch. The YAML reader trims unquoted scalars but keeps interior whitespace in quoted ones, so licence: " " passes on a sources[] entry and is refused on the single-source shape; hasLicence in lint.go already trims. Fix: one shared requireLicence next to requireClass/requireCitation/requireDate, used by both branches.
