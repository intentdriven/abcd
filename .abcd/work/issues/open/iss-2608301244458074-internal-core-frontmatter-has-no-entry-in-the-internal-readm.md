---
schema_version: 1
id: "iss-2608301244458074"
slug: "internal-core-frontmatter-has-no-entry-in-the-internal-readm"
severity: "nitpick"
category: "documentation"
source: "user-observation"
found_during: "itd-179-round-3-builder"
found_at: "internal/README.md"
---

internal/core/frontmatter has no entry in the internal README package map

Surfaced by the itd-179 round-3 builder as an observation and captured rather
than fixed: pre-existing, and unrelated to the round's records.

`internal/core/frontmatter` carries no row in `internal/README.md`'s package
map. The gap matters slightly more after round 3 than before it, because that
package is now the canonical home of `Unquote` — the decoder consolidated out
of `capture.unquote` and `lint.unescapeScalar` — so a reader looking for the
one true scalar decoder has no map entry pointing there.
