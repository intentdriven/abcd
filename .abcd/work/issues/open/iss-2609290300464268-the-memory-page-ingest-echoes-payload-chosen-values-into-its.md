---
schema_version: 1
id: "iss-2609290300464268"
slug: "the-memory-page-ingest-echoes-payload-chosen-values-into-its"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/schema.go"
refines: [iss-2609290218032954]
---

The memory page ingest echoes payload-chosen values into its refusals, the class iss-2609290218032954 fixed in scribe, release, ideate and lifeboat. A DistilledPage arrives host-produced (memory ingest --pages-json, memory ask --page-json), and internal/core/memory/schema.go quotes its values with %v or %q, sanitised at most and never redacted: a source class (154), an ingested_at (170), the declared classes (299), the undeclared keys themselves (425), type, domain and slug (436-442), and the assembled filename (491, 531). A token or a home path in any of them reaches the terminal and the transcript. Fix direction: describe a closed-shape value with termsafe.DescribeRefused, and name an undeclared key through scanner.RedactRefusal. Detector: a page carrying a marker and a home path in each field is refused without either in the error.
