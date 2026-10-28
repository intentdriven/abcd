---
schema_version: 1
id: "iss-2610262036193721"
slug: "the-site-chapter-s-setup-publishes-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 640 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md:86"
remedy: "none (filed automatically)"
related_intents: [itd-2610091010348892]
finding_ref: cfk-555a57fb626ab99f
---

The site chapter's setup publishes a managed repository's site through a hosting provider adapter on a custom domain; itd-2610091010348892 publishes the same per-release record on the project's own GitHub Pages, with abcd running nothing.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/22-site.md:86` — "creates the host, routes the custom domain to it and reports the live address."
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "The project's own workflow builds it and GitHub hosts it; abcd runs nothing."

22-site's setup writes a workflow that renders on each release, and at its host stage the provider adapter creates the host, routes the custom domain to it and reports the live address. itd-2610091010348892 claims the same ground, a per-release build of an opted-in project's record, but has GitHub host it on the project's own Pages and moves abcd's own record to that route. The two describe one publication in contradictory ways, and the intent does not say what becomes of setup's host stage.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 640.
