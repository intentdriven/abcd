---
schema_version: 1
id: "iss-2608301455387735"
slug: "the-character-floor-is-answerable-by-padding-so-three-letter"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-4-recheck"
found_at: "internal/core/grounds/grounds.go"
resolution: "the floor's character half counts letters and is asked after the unit half, so padding with dots, digits or characters that render as nothing no longer answers it"
impact: fix
resolved_by:
  intent: "itd-179"
---

the character floor is answerable by padding so three letters plus seventeen non-letter runes clear it and an entirely invisible text passes

## Grounds

- pursued: we expect a floor answerable by characters that say nothing to be answered that way, because a caller who cannot meet it reaches for whatever the count accepts
