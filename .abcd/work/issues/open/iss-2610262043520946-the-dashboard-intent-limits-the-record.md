---
schema_version: 1
id: "iss-2610262043520946"
slug: "the-dashboard-intent-limits-the-record"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1168 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-2610032150577708, itd-2610091010348892]
finding_ref: cfk-2142c261055ce57c
---

The dashboard intent limits the record site to abcd's own record; the record-pages intent builds a record site for every project that opts in

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling."
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "abcd's release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages"

itd-2610032150577708 records a ruling that the record site and the dashboard are promised for abcd's own record only. itd-2610091010348892 says the opposite: any project that switches it on gets its about page and development record built at each release and published on its own GitHub Pages, with abcd's own record built 'exactly the way any other project's is'. The two planned intents disagree on whether the record site serves other projects at all.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1168.
