---
schema_version: 1
id: "iss-2610040744587676"
slug: "the-site-screenshots-workflow-s-path-filter-rests-on-a"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "site-screenshots failing on every PR since 2026-10-03 23:37Z"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/site-screenshots.yml"
remedy: "Waits on a ruling on audit cost: either add the record stores (.abcd/development/** and .abcd/work/issues/**) to the pull_request paths so a record edit is audited on its own pull request, at the price of running the browser job on most pull requests, or keep the filter and rewrite the comment to state the residual honestly (a record edit can move layout; the wrapping nets in site.css bound the known class, and the next site-touching pull request is where anything else surfaces). Grounds: this incident, where the failure surfaced on pull requests that did not cause it."
---

The site-screenshots workflow's path filter rests on a premise this repository has falsified: its comment says a record edit under .abcd/ moves page content without moving the layout the audit measures, so record-only pull requests do not run the audit. #795 added two ADRs whose titles quote a path with no break in it, which moved the /record/ layout past 360 px; the audit did not run on #795 and first failed on the next pull request that touched site sources, an unrelated one, which then carried a red check it did not cause. The wrapping nets of iss-2610040729344770 and iss-2610040732240935 close the one class seen so far by construction, but the comment still states the premise as fact.
