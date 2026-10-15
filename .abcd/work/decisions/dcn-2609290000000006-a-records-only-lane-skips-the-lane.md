---
id: dcn-2609290000000006
slug: a-records-only-lane-skips-the-lane
date: 2026-09-29
---
2026-09-29 — A records-only lane skips the lane preflight, and the integration branch that lands it runs the one preflight that judges it (ruling by the run A orchestrator abcd-fc [aaba3e]; recorded by lane integRemedies of autonomous run A). A records-only lane changes no code, only record files, and it still runs record-lint, lint-issues, lint-decisions, docs-lint and `abcd lint outbound` before it reports. Running `make preflight` once per such lane would queue nine full runs behind the machine's one-preflight lock to judge a tree whose code no lane touched. The receipt the push gate needs is minted by the integration's preflight, on the merged tree that is actually pushed.
