---
schema_version: 1
id: "iss-2610262041020990"
slug: "the-site-chapter-lays-the-record-site"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 972 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:83"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does the 2026-10-04 give-up (the record site and dashboard are promised for abcd's own record only: dashboard intent decision 13, itd-140 ruling) still stand, given shipped site setup for any managed repo and the 2026-10-14 ruling that abcd provides the record for any managed repo? Answers: (1) the give-up stands; (2) the 2026-10-14 ruling overtakes it. Recommend (2): it is the later ruling, in the product thinker's own words, and the code already behaves that way."
related_intents: [itd-2610032150577708]
finding_ref: cfk-6bfe1c0568ebb09a
---

The site chapter lays the record site into any managed repository; the dashboard intent promises the record site for abcd's own record only.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:83` — "lays the site into a repository abcd manages"
- End B: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling."

10-site.md describes `abcd site setup` laying the rendered site into a repository abcd manages, and still lists itd-139 (the generic explorer shown on a second instance) as a capability built on this plumbing. itd-2610032150577708 decisions 2 and 13 replace itd-139 with the dashboard and give up the itd-140 promise, so the record site is for abcd's own record only. The brief offers for every managed repository what the planned intent withdrew.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 972.
