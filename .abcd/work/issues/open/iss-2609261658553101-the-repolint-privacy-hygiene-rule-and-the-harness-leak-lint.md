---
schema_version: 1
id: "iss-2609261658553101"
slug: "the-repolint-privacy-hygiene-rule-and-the-harness-leak-lint"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint/rule_privacy.go"
---

The repolint privacy-hygiene rule and the harness_leak lint rule read each committed line raw, so the JSON-escape spellings the scanner reads through its decoded views (jsonescape.go) pass both: a third-party home after a \n escape, a home written with the solidus escape, a Windows home with doubled separators, and a token or a session URL written straight after a \n escape (the patterns anchor on a leading word boundary, and the escape letter is a word byte). A committed JSON fixture, export or transcript carrying any of them is not refused. Reading the views in the lint is not contained: Go interpreted string literals use the same escapes, so the views would surface every test string that puts a \n escape before a /home root and a name in committed Go source, and the waivers on those lines do not all exist. The CI gitleaks history scan covers the token half; the home-path and session-URL halves have no other gate. Detector: privacyLeak and harnessLeakOnLine report each of the four spellings on a committed line.
