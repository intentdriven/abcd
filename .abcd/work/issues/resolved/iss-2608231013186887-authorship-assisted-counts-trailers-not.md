---
schema_version: 1
id: "iss-2608231013186887"
slug: "authorship-assisted-counts-trailers-not"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/contributors.go"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

Authorship.Assisted counts trailer OCCURRENCES but every surface renders it as a count of COMMITS. contributors.go increments a.Assisted once per Assisted-by value inside the per-commit loop, and 8 commits in this history carry more than one trailer, so the contributors tile reads '1037 / 1454 commits disclose AI assistance' when the true commit figure is 1029 (1030 commits carry at least one trailer, one of which declares None). The displayed 71% is right only by rounding coincidence. The same conflation makes the panel note (1037) disagree with its own bars (1038, including None). Separate the two quantities and name each for what it is: commits that disclose (a commit-level count) against trailer occurrences per model (an occurrence-level tally, where one commit may appear twice), and publish the undeclared count (424) which the struct already computes and no page renders. Found while redesigning the contributors page (report D follow-up, 2026-08-23).