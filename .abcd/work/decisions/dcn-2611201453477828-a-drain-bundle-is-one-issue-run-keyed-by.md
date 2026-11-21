---
id: dcn-2611201453477828
slug: a-drain-bundle-is-one-issue-run-keyed-by
date: 2026-11-20
---
2026-11-20 — A drain bundle is one issue run, keyed by its lead issue and carrying the other issues as members, with one lane, one branch, one review set, one preflight and one pull request, for itd-2611201450052326. Grounds: a lane already keeps a list of resolutions and the landing writes one Resolves line per entry (internal/core/implement/loop/land.go), so several runs sharing a branch would add coordination the run already does not need. Decided by the technical facilitator on delegation, 20 Nov 2026, while the person is away.
