---
schema_version: 1
id: "iss-2608311039531552"
slug: "the-strip-a-matched-quote-pair-then-frontmatter-unquote-idio"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
---

The strip-a-matched-quote-pair-then-frontmatter.Unquote idiom now exists in three places: capture's reader (internal/core/capture/parse.go decodeScalar), record-lint's schema gate (internal/core/lint/schema.go readerScalar) and the cold-reading definition locator (internal/core/reading/definitions.go scalar). Unquote's own doc says its argument is the scalar's INNER text, so every caller that reads a possibly-quoted frontmatter value has to do the strip first, and a caller that forgets it -- as the locator's first cut did -- refuses well-formed records with a message comparing a value against itself. The idiom belongs beside Unquote in internal/core/frontmatter, with the three call sites moved onto it.
