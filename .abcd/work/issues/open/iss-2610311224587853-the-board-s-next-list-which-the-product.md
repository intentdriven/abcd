---
schema_version: 1
id: "iss-2610311224587853"
slug: "the-board-s-next-list-which-the-product"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "selecting intents to build from the board"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/board/board.go"
remedy: "Per dcn-2610311251013647: the board's Next applies the build's record-only pre-start checks (intent.StartChecksIn) and its footprint check (specPackages and noFootprint in internal/core/implement/loop) to every row, and lists a failing intent under Later with the failing check named."
---

The board's Next list, which the product thinker picks build runs from, shows as ready two intents that abcd build refuses from the record alone, before any reservation is read: itd-50, whose spec spc-2609211924346308 has no ## Footprint packages line, and itd-39, whose ## Open Questions section the build counts as asking one question. The pick-order head skips such intents, but Next lists them unmarked, so a person who chooses from Next gets a refusal instead of a run and cannot tell the two kinds of ready apart.
