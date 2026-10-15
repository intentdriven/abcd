---
id: dcn-2610110000000001
slug: the-merge-queue-builds-at-most-two
date: 2026-10-11
---
2026-10-11 — The merge queue builds at most two entries at once and waits 90 minutes for a group's checks (main protection ruleset 19969675, was five and 45; the technical facilitator's ruling). Every queue entry and every pull-request push runs a macOS job, and the account runs five macOS jobs at a time at about 25 minutes each, so with five parallel builds the waiting entries passed the 45-minute timeout and were dropped. Each drop rebuilt the queue, and the superseded group's runs, whose `gh-readonly-queue` refs are gone, kept holding macOS runners, so nothing merged from 03:36 to past 08:40 on 2026-10-11 while ten stale runs were cancelled by hand. Fewer parallel builds make each entry wait its turn longer; the cancellation of superseded queue runs in CI is a separate fix.
