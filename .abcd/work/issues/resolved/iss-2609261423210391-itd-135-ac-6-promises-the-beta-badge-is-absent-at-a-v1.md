---
schema_version: 1
id: "iss-2609261423210391"
slug: "itd-135-ac-6-promises-the-beta-badge-is-absent-at-a-v1"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-135"
origin: researcher-authored
production_mode: hand-written
resolution: "TestBuildAtAV1ReleaseDropsTheBetaBadge builds the fixture at a 1.0.0 release and asserts no Beta badge and no beta footer meta."
impact: internal
resolved_by:
  commit: "5f20348c7d5b914be245cdd8d371bc5d673999eb"
---

itd-135 ac-6 promises the Beta badge is absent at a v1 release with no copy change, but no test builds the site against a v1 release: internal/core/site/build_test.go asserts the badge at a 0.x release (line 1087) and its absence with no release at all (line 1138), so the v1 half of the criterion rests on isPreOne (internal/core/site/build.go:572) with nothing watching it. Wanted: a build test at a v1.0.0 release asserting no class="beta" span renders.

## Grounds

- pursued: the v1 half of itd-135 ac-6 is now watched by a build; a predicate that renders the badge at v1 passing this test would show it wrong
