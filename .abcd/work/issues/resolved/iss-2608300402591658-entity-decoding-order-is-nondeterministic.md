---
schema_version: 1
id: "iss-2608300402591658"
slug: "entity-decoding-order-is-nondeterministic"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-183 fifth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (renderedText)"
resolution: "Entities were decoded by ranging a Go map, so the verdict for a heading such as Audit&amp;nbsp;Notes depended on whether &amp; happened to be applied before &nbsp; that time round — the same input giving two answers in a determinism instrument. Decoding is now html.UnescapeString, one pass, which is a function of its input and also covers the numeric and hex character references a hand list could never enumerate. The comment no longer claims that an entity outside the list is a refusal rather than a leak."
impact: fix
---

renderedText decodes entities by iterating a Go map, so the decoding order is random and the refusal verdict for a heading such as Audit&amp;nbsp;Notes is nondeterministic (200 calls: refused 77, admitted 123) — a determinism instrument with a coin-flip refusal; and a numeric or hex character reference inside an excluded title is outside the short list, slugs differently, and travels, while the comment claims an entity outside the list is a refusal not a leak. Replace the hand list with html.UnescapeString before slugging, which closes both, and correct the comment.
