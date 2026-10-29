---
schema_version: 1
id: "iss-2610161904318905"
slug: "in-the-guided-design-on-a-phone-opening"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "design fix round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "Move focus into a phone sheet when it opens and return it to the opener on close, sharing the dialogs' focus helper; test it in layout_test.go."
---

In the Guided design on a phone, opening a bottom sheet (documents, suggestions, people, invite) does not move keyboard focus into the sheet, unlike the Send, invite and trust dialogs.
