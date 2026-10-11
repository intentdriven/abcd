---
schema_version: 1
id: "iss-2610091010558682"
slug: "the-research-the-routing-of-itd"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-10-09 product thinker intent on abcdev.app"
origin: researcher-authored
production_mode: hand-written
found_at: "abcdev.app multi-repository rendering"
remedy: "Write a dated research note under .abcd/development/research/notes/ from primary sources (GitHub's REST API rate-limit, authentication and terms documentation, and the opt-in and build models of two or three hosted documentation services), comparing scheduled, owner-triggered and release-triggered builds, how a signed-in person's own GitHub allowance (an OAuth or GitHub App user token) pays for their builds, how anonymous read-only builds are rate-limited so bots cannot exhaust the site's allowance, and how a shared cache serves public pages to everyone while a private page is served only to people the repository admits; before the planning interviews of itd-2610091010348892 and itd-2610091016262335."
---

The research the routing of itd-2610091010348892 owes before planning has not been done: how abcdev.app can build pages from other public GitHub repositories within the GitHub API's limits (60 unauthenticated requests an hour, adr-47) and its terms, whether to fetch on a schedule, on an owner's request or from the owner's own release, and how comparable hosted documentation services handle opt-in, rebuilds and takedown.
