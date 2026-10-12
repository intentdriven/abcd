---
schema_version: 1
id: "iss-2610120656366937"
slug: "rules-fence-covers-unreplaced-bundled"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "Fable review of security lanes B and B2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
remedy: "Fence only rule bodies the override actually supplied (rules replaced), leaving bundled bodies of a domain whose override changes only recall or state rendered as before; prove with a rules test"
---

The rules loader fences every rule of any domain a user or repo override names (internal/core/rules/rules.go ~965, fix for iss-2609020219265817), including a domain whose override only changes recall or state, so bundled rule bodies lose their markdown in that case. Over-fencing only; no attack path.
