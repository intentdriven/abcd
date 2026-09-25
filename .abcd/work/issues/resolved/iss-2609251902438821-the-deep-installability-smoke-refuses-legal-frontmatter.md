---
schema_version: 1
id: "iss-2609251902438821"
slug: "the-deep-installability-smoke-refuses-legal-frontmatter"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/deepsmoke.go"
resolution: "The deep tier's frontmatter reader folds indented continuation lines into their key, reads quoted and non-ASCII keys, and accepts the YAML document end; TestRenderPageHelpLoadsLegalFrontmatter covers fourteen legal shapes and four that still refuse."
impact: fix
resolved_by:
  commit: "8bb8bb3c"
---

The deep installability smoke refuses legal frontmatter: pageFields (internal/core/launch/deepsmoke.go) drops indented continuation lines after a key with no block header, so a description written as a plain multi-line scalar reads as absent and a skill is refused for lacking one; a quoted key ("name": or 'name':) or a non-ASCII key is refused as not a YAML mapping entry; and a block closed by the YAML document end ... is refused as never closed. The cut wires the deep tier unconditionally (internal/surface/cli/ship.go), so the first page written in any of these shapes makes launch ship refuse under ErrPayloadPageUnloadable with no opt-out. Latent: every page shipped today passes.

## Grounds

- pursued: every legal top-level YAML mapping shape in the table loads with its name and description; a page in one of those shapes refused by the deep tier, or a duplicate spelled once quoted and once plain being accepted, would show it wrong
