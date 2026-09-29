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
resolution: "Fixed: every site classified. schema.go 154 (class), 170 (ingested_at), 436/439/442 (type, domain, slug) describe the value; 299 quotes a declared class only when it is in the closed enum and describes any other (the derived set passed requireClass and is kept); 425 names the undeclared keys through scanner.RedactRefusal, so ValidateDistilledPage takes the repository root; 491 and 531 describe the assembled filename. Kept: 196/429/397/403 (constant key names), 254/263 (a class already in the enum). Sibling found and fixed: ingest.go 297 named an uncited page by its filename, whose slug may be token-shaped; it names the page by position. The writer's judgeFilename echo is iss-2609290411321963."
impact: fix
resolved_by:
  commit: "4fe838324"
---

The memory page ingest echoes payload-chosen values into its refusals, the class iss-2609290218032954 fixed in scribe, release, ideate and lifeboat. A DistilledPage arrives host-produced (memory ingest --pages-json, memory ask --page-json), and internal/core/memory/schema.go quotes its values with %v or %q, sanitised at most and never redacted: a source class (154), an ingested_at (170), the declared classes (299), the undeclared keys themselves (425), type, domain and slug (436-442), and the assembled filename (491, 531). A token or a home path in any of them reaches the terminal and the transcript. Fix direction: describe a closed-shape value with termsafe.DescribeRefused, and name an undeclared key through scanner.RedactRefusal. Detector: a page carrying a marker and a home path in each field is refused without either in the error.

## Grounds

- pursued: a DistilledPage carrying a marker and a home path in each field is refused without either in the error and still names the field; a refusal quoting a payload value would show it wrong
