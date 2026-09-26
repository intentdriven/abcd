---
schema_version: 1
id: "iss-2609251823551349"
slug: "capture-disposition-writes-disposition-grounds-and-exit"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

capture disposition writes disposition_grounds and exit_condition redacted but never through termsafe.EncodeHiddenRunes (internal/core/capture/reading.go:471-479), so a bidi or zero-width rune in --exit-condition lands in the committed record verbatim, the class iss-2608301206073609 closed for other free-text writes (review-capture 2).
