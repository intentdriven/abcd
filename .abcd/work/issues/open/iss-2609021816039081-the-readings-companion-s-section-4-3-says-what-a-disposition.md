---
schema_version: 1
id: "iss-2609021816039081"
slug: "the-readings-companion-s-section-4-3-says-what-a-disposition"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "Phase A readiness statement"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reading.go"
---

The readings companion's section 4.3 says what a disposition's grounds must contain varies by state (a rejection names the purpose protected, an admission or a decline states why) and that the variation is a lint rule rather than four fields. Nothing enforces it: the disposition command refuses only a blank or whitespace-only grounds field, and no lint reads the grounds against the state. Held by protocol today; the readiness statement for Phase A records it as not held by mechanism.
