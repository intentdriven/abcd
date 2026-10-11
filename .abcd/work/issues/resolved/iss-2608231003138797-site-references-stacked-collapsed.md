---
schema_version: 1
id: "iss-2608231003138797"
slug: "site-references-stacked-collapsed"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/csl.go"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

The references page's two panels (References & sources, Inspirations) read as two odd parallel columns of very different length. Maintainer's model is the relationship page's Browse-as-a-list disclosure: stack the two as full-width boxes one under the other, both COLLAPSED by default, so the page opens as two closed headings a reader chooses between. This supersedes the reading-grid exemption added earlier the same day, which only stopped the shorter column stretching to match the taller one (report B of the 2026-08-23 second pass).