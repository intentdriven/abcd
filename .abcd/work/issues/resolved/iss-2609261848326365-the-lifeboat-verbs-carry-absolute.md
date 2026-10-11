---
schema_version: 1
id: "iss-2609261848326365"
slug: "the-lifeboat-verbs-carry-absolute"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/pack.go"
resolution: "Every lifeboat report directory field (pack dest, embark lifeboat_dir and target_dir, graveyard lessons, principles, press-release and review lifeboat_dir) goes through fsutil.RedactHome where the result is built."
impact: fix
resolved_by:
  commit: "09f7dcdf"
---

The lifeboat verbs carry absolute directories into their --json and text reports: PackResult.Dest (disembark pack), EmbarkPlan and EmbarkResult LifeboatDir and TargetDir (embark probe, embark from), LessonsResult.LifeboatDir (disembark graveyard), and PrinciplesResult, PressReleaseResult and ReviewResult LifeboatDir (disembark principles, press-release, review). A lifeboat or target directory under the home names the developer, and the iss-81 rule is that machine output never carries a developer-identity path; the site verbs' OutDir fields had the same shape (iss-2608291957114882).

## Grounds

- pursued: each of the nine fields reports a directory under HOME as ~/… (TestLifeboatReportsNameDirectoriesWithoutTheHomePath, watched failing on all nine at the base); an absolute home path in any lifeboat --json would show it wrong
