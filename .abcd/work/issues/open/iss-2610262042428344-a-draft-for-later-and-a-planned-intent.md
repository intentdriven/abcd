---
schema_version: 1
id: "iss-2610262042428344"
slug: "a-draft-for-later-and-a-planned-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1088 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-2610040754453237, itd-2610091010348892]
finding_ref: cfk-3139b2c3977ee837
---

A draft for later and a planned intent both publish any managed project's record site

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md:18` — "publishes the public record site for their own project, the timeline, the decisions and the announcements, as abcd publishes its own."
- End B: `.abcd/development/intents/planned/itd-2610091010348892-anyone-who-opens-abcdev-app-finds-the.md:20` — "abcd's release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages"

itd-2610040754453237, filed as later and not now, has any abcd-managed team publish the public record site for its own project as abcd does. itd-2610091010348892, already planned, builds an opted-in project's about page and record at each release onto its own GitHub Pages and moves abcd's own record to the same route. The same capability is held by two records at different stages.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1088.
