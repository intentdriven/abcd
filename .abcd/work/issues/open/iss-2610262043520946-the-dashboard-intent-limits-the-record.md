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
remedy: "In the dashboard intent mark decision 13 (record site promised for abcd's own record only) as superseded for the record site by the 2026-10-14 ruling in itd-2610091010348892 and by the shipped site setup, and add the same note to the 2026-10-04 ruling in the itd-140 discipline."
related_intents: [itd-2610032150577708, itd-2610091010348892]
finding_ref: cfk-2142c261055ce57c
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The dashboard intent limits the record site to abcd's own record; the record-pages intent builds a record site for every project that opts in

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling."
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "abcd's release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages"

itd-2610032150577708 records a ruling that the record site and the dashboard are promised for abcd's own record only. itd-2610091010348892 says the opposite: any project that switches it on gets its about page and development record built at each release and published on its own GitHub Pages, with abcd's own record built 'exactly the way any other project's is'. The two planned intents disagree on whether the record site serves other projects at all.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1168.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
