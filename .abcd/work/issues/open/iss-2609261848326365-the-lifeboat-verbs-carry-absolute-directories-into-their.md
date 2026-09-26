---
schema_version: 1
id: "iss-2609261848326365"
slug: "the-lifeboat-verbs-carry-absolute-directories-into-their"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/pack.go"
---

The lifeboat verbs carry absolute directories into their --json and text reports: PackResult.Dest (disembark pack), EmbarkPlan and EmbarkResult LifeboatDir and TargetDir (embark probe, embark from), LessonsResult.LifeboatDir (disembark graveyard), and PrinciplesResult, PressReleaseResult and ReviewResult LifeboatDir (disembark principles, press-release, review). A lifeboat or target directory under the home names the developer, and the iss-81 rule is that machine output never carries a developer-identity path; the site verbs' OutDir fields had the same shape (iss-2608291957114882).
