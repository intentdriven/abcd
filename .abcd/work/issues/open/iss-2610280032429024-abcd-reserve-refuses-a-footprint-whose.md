---
schema_version: 1
id: "iss-2610280032429024"
slug: "abcd-reserve-refuses-a-footprint-whose"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "lane E salvage, store and sweep reservation renewal"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/claim.go"
remedy: "Let reserve derive the footprint from a branch diff and compact it to fit the cap, by folding paths into their common directories or carrying a hash of the full list, and say what it folded; the hand-rolled fallback was a shorter explicit list that left paths unreserved."
---

abcd reserve refuses a footprint whose reservation message passes the 4096-byte cap, and takes only an explicit list of paths, so a large stacked build cannot name its whole diff: renewing the worktree store's reservation after a rebase with its 83 changed paths was refused at 4740 bytes, and the renewal had to fall back by hand to the store's original 50 paths, leaving the rebased-in paths unreserved.
