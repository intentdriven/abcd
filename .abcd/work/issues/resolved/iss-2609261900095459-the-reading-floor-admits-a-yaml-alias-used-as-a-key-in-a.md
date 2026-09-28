---
schema_version: 1
id: "iss-2609261900095459"
slug: "the-reading-floor-admits-a-yaml-alias-used-as-a-key-in-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainRd"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The floor refuses a YAML alias at every key position: line start, behind { [ or , (node properties allowed between), with the block-indicator and explicit-key positions already refused by the compact-mapping and unreadable-key rules. The rule is on the alias, the smaller complete one; an alias in a value position stays admitted."
impact: fix
resolved_by:
  commit: "17e795d5bb5d27bdbf5c296cab2467a5c24304c1"
---

The reading floor admits a YAML alias used as a key in a frontmatter block, so an excluded key travels past the redaction under a manifest asserting its refusal. `k: &a origin` followed by `*a : X` (or `m: {*a : X}`) is admitted, and a YAML reader resolves it to {origin: X}. The anchor refusal in unresolvableFrontmatterShape fires only at line start or after a block indicator (nestedBlockEntry); an anchor in value or flow position is admitted, and nothing inspects `*` at all. Same class as iss-2608301237450573.

## Grounds

- pursued: every alias-as-key shape the review named and its siblings (flow continuation, tag, CRLF, anchors in flow and after a tag) is refused while a value alias and a harmless merge are admitted, and detection/widening/entailment dry-run assemblies keep 394/336/237 items and assembler_version 1.8.0+3c58dd; a YAML reader resolving an admitted frontmatter to an origin key would show it wrong
