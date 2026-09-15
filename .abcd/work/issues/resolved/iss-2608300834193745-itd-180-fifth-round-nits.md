---
schema_version: 1
id: "iss-2608300834193745"
slug: "itd-180-fifth-round-nits"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-180 fifth-round ruthless review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, internal/core/capture/reading.go, internal/core/capture/standingparity_test.go"
resolution: "The two readers share one record byte cap (issueschema.RecordReadLimit), the Unsafe finding carries the reason it declined instead of one directory-shaped sentence, and the parity test's leading comment describes the whole-standing-set expectation the contested change introduced."
impact: internal
---

itd-180 fifth-round nits: the two readers of one record family use different byte caps (lint's citation page limit of 8 MiB, capture's record read limit of 1 MiB), so an oversized disposition stands on the board and is refused by the verb — loud, not silent; the Unsafe finding text describes a directory but is now also emitted for a symlinked record file; the parity test's leading comment still says the expectation is built from the first standing id, which the contested change removed.
