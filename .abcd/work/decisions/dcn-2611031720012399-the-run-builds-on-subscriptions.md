---
id: dcn-2611031720012399
slug: the-run-builds-on-subscriptions
date: 2026-11-03
---
2026-11-03 — The run builds on the person's subscriptions: the build loop's roles are routed to the host, so each stage runs as a sub-agent of the coordinating session on its subscription, since the claude runner's --bare launch always bills a key; the keys stay a fallback at 95% of a subscription window, per dcn-2611031615094092. The main work moves to the account that always runs the latest build, and a second account keeps the latest release.
