---
id: dcn-2610181003467233
slug: a-pull-request-carries-a-reservation-s
date: 2026-10-18
---
2026-10-18 — A pull request carries a reservation's work only through its head branch or a closing trailer (facilitator-on-delegation, itd-2610180912189125 review round 2, blocker 4; refines dcn-2610180936568358): the link is the record id in the pull request's head branch name, or on a `Resolves:` or `Delivers:` line of its body, matched on word boundaries so that iss-5 never matches iss-50. A `Refs:` line, which the issue-resolution gate requires of any pull request that only names an issue, carries the work only when the head branch is already this copy's own by another route, and never makes a branch own, so a sibling copy's pull request that mentions the record neither leaves this session waiting nor becomes its branch (grounds: plan review round 2, blocker 4; the RS004 rule in scripts/check-issue-resolution.sh)
