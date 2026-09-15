---
schema_version: 1
id: "iss-2608291814565781"
slug: "memory-frontmatter-parser-is-bom-unaware"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/yaml.go"
---

ultra-v0.6.8 below-cap reuse-5: the memory frontmatter parser (frontmatterOpenIndex in internal/core/memory/yaml.go) trims lines with strings.TrimSpace, which does not strip U+FEFF, while the record-lint parser uses frontmatter.TrimBOM, so a BOM-led memory page is a page with frontmatter to record-lint and a page without it to memory. Pre-existing, not introduced by the reviewed slice.
