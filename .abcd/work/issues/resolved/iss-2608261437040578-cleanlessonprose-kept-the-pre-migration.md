---
schema_version: 1
id: "iss-2608261437040578"
slug: "cleanlessonprose-kept-the-pre-migration"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/lifeboat/graveyard_lessons.go"
resolution: "cleanLessonProse delegates to termsafe.CleanProse with the unchanged cap"
impact: fix
resolved_by:
  commit: "2d00855e"
---

cleanLessonProse kept the pre-migration body the CleanProse consolidation claims routed