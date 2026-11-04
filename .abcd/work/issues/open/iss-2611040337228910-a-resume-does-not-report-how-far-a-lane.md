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
remedy: "Report commits behind at a resume and before a preflight, and offer the sync; a test resumes behind main and reads the report."
---

A resume does not report how far a lane's base has moved: on 2026-11-04 a lane resumed on a base seven hours old, so its pull request was behind main by more than twenty commits and its preflight judged the old base.
