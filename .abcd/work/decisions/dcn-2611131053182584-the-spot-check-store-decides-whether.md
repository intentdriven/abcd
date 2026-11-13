---
id: dcn-2611131053182584
slug: the-spot-check-store-decides-whether
date: 2026-11-13
---
2026-11-13 — The spot-check store decides whether this checkout already holds a record from what this checkout wrote, and from nothing else. Of the two ways to keep its own writes apart, the local copy (refs/abcd/spotcheck) holding only this checkout's writes was chosen over a second local ref, as the smaller change: the push lays the records origin lacks over origin's tip in a commit of its own and pushes that, and never moves the local copy onto another writer's files, and a write no longer reads origin's copy at all. So nothing fetched can stop a local write, a copy of origin's that cannot be listed changes nothing, and a merge-gate decision asked again over one head and answer is kept once across pushes. This replaces the exception dcn-2611130923116384 made for a record origin's copy also holds.
