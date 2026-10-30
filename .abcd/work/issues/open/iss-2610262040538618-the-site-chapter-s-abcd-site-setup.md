---
schema_version: 1
id: "iss-2610262040538618"
slug: "the-site-chapter-s-abcd-site-setup"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 962 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:83"
remedy: "Supersede draft itd-2610040754453237 by planned itd-2610091010348892 (every opted-in project's record on its own Pages route) with abcd intent reclassify, naming shipped itd-2609061543533170 as the setup half already delivered; or re-scope the draft to what neither covers."
related_intents: [itd-2610040754453237]
finding_ref: cfk-a44feb861636f010
---

The site chapter's abcd site setup already lays the site into a managed repository; itd-2610040754453237 claims publishing that site for every managed project as later work.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:83` — "lays the site into a repository abcd manages"
- End B: `.abcd/development/intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md:18` — "Any team that manages its project with abcd publishes the public record site for their own project"

10-site documents abcd site setup as plumbing that lays the site into a repository abcd manages: the composition, the static inputs, a render-on-release workflow and host configuration. The draft claims as new, deferred work that any abcd-managed project publishes its own public record site the way abcd does. Both claim the same ground, so the capability is either delivered already or the draft's scope is wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 962.
