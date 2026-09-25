---
schema_version: 1
id: "iss-2609251728586400"
slug: "two-scanner-test-nits-from-review-3-meter-test-go-196-pins"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

Two scanner test nits from review 3: meter_test.go:196 pins the separator-run cap with a true account position (C:\Users followed by 65 backslashes), which passes with or without the cap, so the over-report direction for a rootless run is not itself pinned; and the meter fixtures scale with maxSeparatorRun, so inflating the constant to probe the no-cap case allocates gigabyte strings (remove the loop bound instead).
