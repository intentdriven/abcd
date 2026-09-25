---
schema_version: 1
id: "iss-2609251939466588"
slug: "the-receipt-gate-s-sha-pattern-admits-7-to-64-hex-digits-so"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

The receipt gate's sha pattern admits 7 to 64 hex digits, so an abbreviated receipts directory for the same commit ties with its full-sha twin and the derivation refuses. The reviews charter, the release protocol and the runbook all say the directory is named by the full sha; the pattern should require 40 or 64 hex digits.
