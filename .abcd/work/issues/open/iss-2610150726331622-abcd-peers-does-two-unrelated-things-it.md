---
schema_version: 1
id: "iss-2610150726331622"
slug: "abcd-peers-does-two-unrelated-things-it"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "planning interview of itd-2609151838312703, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/peers.md"
remedy: "When the session register (itd-2609150819440345) is built, make abcd peers list the sessions running now, move its local duplicate check to abcd status --local and drop its reservation listing in favour of abcd status, updating AGENTS.md, the rules domains and every page that names abcd peers (about 46 files) in the same change; until then leave peers as it is."
related_intents: [itd-2609151838312703]
---

abcd peers does two unrelated things: it lists records that sibling worktrees and local branches hold that this checkout does not (an agent's duplicate check before filing), and it fetches and lists the reservations on the shared remote. The second duplicates abcd status from itd-2609151838312703, the people's view of what everyone else is doing. The product thinker ruled on 2026-10-15 that peers keeps its name for its future meaning, which sessions are running now, local first and later over a private network such as Tailscale or WireGuard (the session register itd-2609150819440345), and that the local duplicate check belongs under abcd status --local.
