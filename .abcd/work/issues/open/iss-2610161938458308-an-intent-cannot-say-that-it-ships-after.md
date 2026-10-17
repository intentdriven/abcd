---
schema_version: 1
id: "iss-2610161938458308"
slug: "an-intent-cannot-say-that-it-ships-after"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/startcheck.go"
remedy: "Give an intent's dependency a step scope: an edge that names the spec step it gates, so the build check refuses only that step and the spec close that ships the intent refuses until the named intent has shipped, while the intent's other steps build freely."
related_intents: [itd-2610161733316518]
---

An intent cannot say that it ships after another without being blocked from building: blocked_by makes abcd build refuse every step until the named intent ships (startBlockedRow in internal/core/intent/startcheck.go), and builds_on says nothing about order, so an intent whose one step needs another's capability (the model-matrix run's judging step needs abcdesign.app's session) has no edge that holds its shipping and frees its other steps.
