---
schema_version: 1
id: "iss-2609261327506636"
slug: "drain-history-notice-bypasses-scrubpaths"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: integ4, landing review3-drain's note"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
---

The drain's history-walk notice (internal/surface/cli/intent_drain.go, runOwedDrain) prints site.LoadHistory's error to stderr through termsafe.Sanitize alone, bypassing scrubPaths, so git's quoted stderr, which can name an absolute path inside the repository (a corrupt loose object is reported with the file it is stored in), reaches the terminal unredacted: the one print in the drain front door no redactor touches. Found by review3-drain as a nit.
