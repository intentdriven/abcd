---
schema_version: 1
id: "iss-2609251514129841"
slug: "the-site-renderer-is-backtick-only-while-the-block-walk-it"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/markdown.go"
---

The site renderer is backtick-only while the block walk it renders is not: site/markdown.go RenderBlock opens a fence only on a first line starting with three backticks (and r.fence assumes exactly that run), so a tilde fence, which Blocks now reads as one block by mdrecord's rule, renders as a paragraph with its delimiters and code inlined into prose, silently; a four-backtick fence renders with its extra backtick in the language class. The site's own doctrine is that its quietest failure is the one to refuse. No tilde fence sits in docs/ or .abcd/development today, so no page is wrong yet.
