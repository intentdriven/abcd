---
id: dcn-2611170900489974
slug: the-macos-suite-s-shard-ceiling-rises-to
date: 2026-11-17
---
2026-11-17 — The macOS suite's shard ceiling rises to 160 minutes and its race pass's package timeout to 150, because on 17 Nov 2026 the shard holding internal/core/implement/loop alone ran past the 80-minute timeout on two runners in a row during the v0.17.0 cut, with its tests still moving and none failed, while the same package took 1976 seconds on the ubuntu runner; under dcn-2611062201286033 a slow runner may not fail a cut with no failing test, and no shard runs on the merge queue's clock. Decided by the technical facilitator on delegation, in the autonomous run.
