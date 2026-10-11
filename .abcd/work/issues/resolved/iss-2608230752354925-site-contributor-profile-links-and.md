---
schema_version: 1
id: "iss-2608230752354925"
slug: "site-contributor-profile-links-and"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

Contributor author rows should link to forge profiles, derived deterministically (no API per adr-47) from the forge noreply email pattern, graceful absence otherwise. Includes the repo-side .mailmap fix: the old-format noreply REPPL@users.noreply.github.com (39 commits) is not folded into Alex Reppel; maintainer authorised the one-line unification 2026-08-23 (report 19).