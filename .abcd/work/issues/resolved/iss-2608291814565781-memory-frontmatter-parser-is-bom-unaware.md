---
schema_version: 1
id: "iss-2608291814565781"
slug: "memory-frontmatter-parser-is-bom-unaware"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/yaml.go"
resolution: "frontmatterOpenIndex strips a leading byte-order mark from the first line through the canonical frontmatter.TrimBOM, so a BOM-led memory page parses with its frontmatter in every memory reader, as it does in record-lint"
impact: fix
resolved_by:
  commit: "780a1f80d69683c85360a927272f0e4094a2b6d7"
---

ultra-v0.6.8 below-cap reuse-5: the memory frontmatter parser (frontmatterOpenIndex in internal/core/memory/yaml.go) trims lines with strings.TrimSpace, which does not strip U+FEFF, while the record-lint parser uses frontmatter.TrimBOM, so a BOM-led memory page is a page with frontmatter to record-lint and a page without it to memory. Pre-existing, not introduced by the reviewed slice.

## Grounds

- pursued: every memory frontmatter reader shares frontmatterOpenIndex, so one strip there aligns them with record-lint; a BOM-led page that textOpensFrontmatter, parseFrontmatter, splitFileFrontmatter or frontmatterKeyLine still reads as frontmatter-less would show it wrong
