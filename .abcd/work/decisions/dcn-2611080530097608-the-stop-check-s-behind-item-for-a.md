---
id: dcn-2611080530097608
slug: the-stop-check-s-behind-item-for-a
date: 2026-11-08
---
2026-11-08 — The stop check's BEHIND item for a merge-queue pull request (iss-2610311339274372) goes in its own run, started once run-2611080159059570 lands, not into that run's open fix round. Grounds: it needs a new forge read (the live ruleset, never the in-tree mirror) with its own timeout and fail-closed tests, and adding it now would restart a definition of done that has queued for over an hour behind other preflights. Both touch internal/core/autonomy/owedread.go, so running them side by side would make the second reservation overlap the first. Reverse by folding it into the open lane in a further fix round.
