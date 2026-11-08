---
schema_version: 1
id: "iss-2611040337228910"
slug: "a-resume-does-not-report-how-far-a-lane"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Report commits behind before a preflight runs, and offer to sync the lane with main; a test runs the preflight step behind main and reads the report."
---

A lane's base can move a long way while it waits, and nothing warns before a preflight judges the old base: on 2026-11-04 a lane resumed on a base seven hours old, so its pull request was behind main by more than twenty commits and its preflight judged the old base.

Since itd-2611041549461012 shipped, the resume listing says how far each lane's base has moved (`base moved N commit(s)`, tested by `TestTheListingSaysHowFarALanesBaseHasMoved`). What is still missing is the same report before a preflight runs, and an offer to sync the lane with main when it is behind. The slug keeps the issue's original wording, which no longer describes the gap.
