---
schema_version: 1
id: "iss-2609281627055603"
slug: "exclusion-floor-closes-its-frontmatter"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainFm"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The floor's key and shape scans run to blockScanEnd, the later of its own prefix close and frontmatter.Close, so a four-dash line or a delimiter carrying text no longer ends them early; the body scans keep the earlier close."
impact: fix
resolved_by:
  commit: "3b6a81ad5"
---

The reading corpus's exclusion floor closes its frontmatter block earlier than the canonical reader: blockCloser (internal/core/reading/project.go) closes on any column-0 line opening with three dashes, so a column-0 `----` or `--- x` ends the floor's block while frontmatter.IsDelimiter, frontmatter.Fields and a YAML reader read on to the real `---`. The keys between the two closes are frontmatter to the canonical reader and invisible to excludedKeyInFirstBlock and unresolvableFrontmatterShape, so an excluded key there reaches the corpus under a manifest asserting its refusal. Probe: a block opened by `---`, holding `foo: 1`, then `----`, then `secret: x`, then `---` — floor close is line 2, canonical close line 4, Fields reports secret, and excludedKeyInFirstBlock does not refuse it. The allowlist reason in delimiter_canonical_test.go (the floor 'refuses more, never less') is false for this shape.

## Grounds

- pursued: an excluded key between a `----` (or `--- x`, BOM-led or CRLF) and the canonical close is refused, and every committed markdown file reads identically through the floor; a document Fields reads a key from that the floor admits would show it wrong
