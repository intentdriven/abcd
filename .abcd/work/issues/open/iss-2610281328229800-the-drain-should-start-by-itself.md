---
schema_version: 1
id: "iss-2610281328229800"
slug: "the-drain-should-start-by-itself"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker request, 2026-10-28 coordinator session"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "treat a runner's out-of-credit response like the rate-limit checkpoint: record the lane as paused-for-credit with the reason, re-ask the runner's remaining quota (the budget check's own question) on a backoff, and resume when it reports credit; and let a scheduled drain start on its own when a configured runner reports credit. Grounds: reuses build's rate-limit checkpoint and budget check (34-build.md), so no new mechanism"
---

The drain should start by itself whenever a runner with an API key and remaining credit is available, and pause by itself when that credit runs out, picking up again once the account is recharged. Today a run that hits an exhausted balance stops, and a person has to notice and restart it. The build loop already ends a window early on a rate-limit response and writes next_eligible_at, and its budget check asks each runner for its remaining quota, but nothing treats an exhausted balance as a pause that ends when credit returns, and nothing starts a drain when a funded key appears. Raised by the product thinker on 2026-10-28.
