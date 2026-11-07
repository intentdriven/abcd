---
schema_version: 1
id: "iss-2611060809234702"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md"
remedy: "Point the sentence at the record's page and the facilitator view's intent rows, or render reservations on the bare view."
---

`.abcd/development/brief/04-surfaces/37-reserve.md:11-12` says a person sees reservations on the board (bare `abcd`) and on a record's page. The record's page holds. The bare view does not: `product` in `internal/core/board/board.go:271` reads only a row's title, lane and next-up mark, so it prints no reservation. The only rendering is `rowTag` (`internal/core/board/board.go:435`), reached through `statusRow` on the full board that `abcd --view facilitator` prints, and those rows are Now and Next intents, so a reservation on an issue appears on no board view.
