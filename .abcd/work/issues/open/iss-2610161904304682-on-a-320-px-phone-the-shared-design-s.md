---
schema_version: 1
id: "iss-2610161904304682"
slug: "on-a-320-px-phone-the-shared-design-s"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "design fix round, renders at 320 px"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.css"
remedy: "Fold the guest toolbar's labels to icons below 360 px, as the member toolbar does, and add the 320 px guest render to the design tests' width set."
---

On a 320 px phone, the Shared design's toolbar for a guest overlaps: the Documents and Your suggestions buttons collide. 390 px is fine.
