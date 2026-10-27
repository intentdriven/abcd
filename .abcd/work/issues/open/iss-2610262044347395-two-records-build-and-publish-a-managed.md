---
schema_version: 1
id: "iss-2610262044347395"
slug: "two-records-build-and-publish-a-managed"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1215 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-2610091010348892, itd-2609061543533170]
finding_ref: cfk-b74f1e84a3783eda
---

Two records build and publish a managed project's record site at each release by different routes: the shipped provider adapter, and the planned GitHub Pages route

A scope leakage (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "The project's own workflow builds it and GitHub hosts it; abcd runs nothing."
- End B: `.abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:42` — "one adapter ships (the provider abcd's own site uses); a second is a later intent, not a change to the verb"

itd-2609061543533170 shipped one verb that sets up a managed repository's site through a provider adapter, the provider abcd's own site uses. itd-2610091010348892 publishes an opted-in project's record on its own GitHub Pages through the project's workflow, with abcd running nothing, and moves abcd's own record to that route. Neither record names the other, so the same ground is covered twice with different hosts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1215.
