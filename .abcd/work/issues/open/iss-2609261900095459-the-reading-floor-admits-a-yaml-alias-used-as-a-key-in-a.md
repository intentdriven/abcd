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
---

The reading floor admits a YAML alias used as a key in a frontmatter block, so an excluded key travels past the redaction under a manifest asserting its refusal. `k: &a origin` followed by `*a : X` (or `m: {*a : X}`) is admitted, and a YAML reader resolves it to {origin: X}. The anchor refusal in unresolvableFrontmatterShape fires only at line start or after a block indicator (nestedBlockEntry); an anchor in value or flow position is admitted, and nothing inspects `*` at all. Same class as iss-2608301237450573.
