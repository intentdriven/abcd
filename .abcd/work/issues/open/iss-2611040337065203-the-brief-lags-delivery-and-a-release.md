---
schema_version: 1
id: "iss-2611040337065203"
slug: "the-brief-lags-delivery-and-a-release"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/docfidelity"
remedy: "Run the fidelity review in the lane that delivers an intent, against the chapters it changes, so a cut starts clean, and record the rounds a cut takes as the measure; a sentence stating a universal (every, never, only) is checked against the code's early returns first, since those made up most findings."
---

The brief lags delivery and a release cut pays for it: the v0.17.0 cut's docs review went through at least five rounds on 2026-11-04, returning 10, 14, 1 and 9 false brief sentences, nearly all in one chapter, 35-drain.md, because itd-82 shipped with a review that only had to exist at its close. Each round costs a reviewer and a fixer while every merge is held by the release reservation.
