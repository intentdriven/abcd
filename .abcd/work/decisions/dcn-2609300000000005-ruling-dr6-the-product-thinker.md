---
id: dcn-2609300000000005
slug: ruling-dr6-the-product-thinker
date: 2026-09-30
---
2026-09-30 — Ruling DR6 (the product thinker, 2026-09-29, verbatim: "per-run agent limit: WORK IN PARALLEL — a run may build several pieces and run reviewers concurrently up to its limit (new build-loop work; then AC6 is testable).") is designed into spc-2609202134341288 as its piece 6 and recorded as decision 7 of itd-2609201925079472 (lane dr6Spec of autonomous run A; records only, nothing built). A slot is one outstanding await in the run's state file, implementers and validators alike; the validators of a round run side by side; a spec step runs beside earlier ones only when it declares `- needs:` (absent, it needs every earlier step, so itd-2609212103565953's in-order landing holds for every spec written so far); a lane lands one at a time and, when a sibling landed after its base, is merged with the default branch (never rebased) and judged by a fresh round, a sync that does not count against `--fix-rounds`. Criterion 6 is tested through the spec's criteria C1 to C12.
