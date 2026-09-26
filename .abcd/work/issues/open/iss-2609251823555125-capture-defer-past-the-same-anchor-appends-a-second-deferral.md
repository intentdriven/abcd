---
schema_version: 1
id: "iss-2609251823555125"
slug: "capture-defer-past-the-same-anchor-appends-a-second-deferral"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

capture defer past the SAME anchor appends a second ## Deferral section (internal/core/capture/deferral.go:114), where the doc says one per cycle (review-capture 3).
