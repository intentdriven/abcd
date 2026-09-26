---
schema_version: 1
id: "iss-2609260057127611"
slug: "three-ahoy-install-writes-still-swallow-their-errors-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
---

Three ahoy install writes still swallow their errors, the class iss-227 fixed for their siblings: the identity pin (stepIdentityPin drops a failed identity.WritePin, and an unset git identity, without a word), the .gitignore visibility block (stepVisibility drops applyVisibilityBlock's error), and the conventions-file marker block (stepMarker drops installMarkerFile and removeMarkerFile failures). A run that wrote none of them reports with no reason.
