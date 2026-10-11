---
schema_version: 1
id: "iss-2609251728586400"
slug: "two-scanner-test-nits-from-review-3"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "Both nits fixed in meter_test.go: a rootless over-long separator run pins the bound's over-report (fails with the loop bound removed), and the fixtures past the bound use a fixed overLongSeparatorRun guarded to stay past maxSeparatorRun."
impact: internal
resolved_by:
  commit: "c43694401b04b28fd24f71ca1e32897d592ca6a8"
---

Two scanner test nits from review 3: meter_test.go:196 pins the separator-run cap with a true account position (C:\Users followed by 65 backslashes), which passes with or without the cap, so the over-report direction for a rootless run is not itself pinned; and the meter fixtures scale with maxSeparatorRun, so inflating the constant to probe the no-cap case allocates gigabyte strings (remove the loop bound instead).

## Grounds

- pursued: removing the separator-run bound from endsWithPathFold now fails TestBoundedContextHelpersKeepTheFinding; that mutation passing, or a fixture again derived from maxSeparatorRun, would show it wrong
