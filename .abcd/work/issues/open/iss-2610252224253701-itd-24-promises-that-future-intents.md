---
schema_version: 1
id: "iss-2610252224253701"
slug: "itd-24-promises-that-future-intents"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "gap audit of the owed fidelity reviews, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reflect/lessons.go"
remedy: "Have the canonical intent create or the planning interview surface the lessons whose terms overlap the new intent, read through ReadLessons; or narrow itd-24's promise to the lifeboat view and record why."
refines: [itd-24]
---

itd-24 promises that future intents reference past lessons, but nothing on the intent side reads the retrospective store: lessons are read only by embark's lifeboat view (ReadLessons at internal/core/reflect/lessons.go:28, used from internal/core/lifeboat/retrospectives.go:54).
