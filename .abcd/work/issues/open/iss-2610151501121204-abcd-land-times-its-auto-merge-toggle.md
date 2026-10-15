---
schema_version: 1
id: "iss-2610151501121204"
slug: "abcd-land-times-its-auto-merge-toggle"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "abcd land review round 6, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "When the forge reports more check contexts than land read (pageInfo.hasNextPage), page through them with the endCursor before timing the toggle grace, or refuse to toggle and say so, so a required check past the first 100 that passed last still moves the clock"
---

abcd land times its auto-merge toggle grace from the newest time a required check passed, read from the first 100 check contexts the forge lists. In a repository with more than 100 checks on a pull request, a required check beyond that page that passed last never moves the clock, so the grace can end early and land can toggle an arming the forge was about to queue. The holder report already says when more checks follow; the toggle does not use that. Rare: abcd's own pull requests carry 13 checks. Found by the round 6 reviewer of the abcd land change, 2026-10-15.
