---
id: dcn-2610110000000002
slug: the-merge-queue-builds-four-entries-at
date: 2026-10-11
---
2026-10-11 — The merge queue builds four entries at once again (main protection ruleset 19969675; the technical facilitator's ruling, amending the same day's ruling of two). #933 took macOS off pull-request runs (one race pass of about 25 minutes) and queue-sweep cancels superseded queue runs, so only queue entries and the nightly run use the five macOS runners; four parallel builds leave one runner for the nightly run. The 90-minute check timeout stands.
