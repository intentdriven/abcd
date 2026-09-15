---
schema_version: 1
id: "iss-2608300914553076"
slug: "frontmatter-shape-refusal-fires-on-body-rules"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-183 seventh-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (unresolvableFrontmatterShape, inFirstBlock), internal/core/site (isFenceLine)"
resolution: "The frontmatter shape refusal and the setext block bound ran on the first three dashes found anywhere, so an ordinary documentation page with a thematic break was refused: a lone rule read as an unclosed block, and a rule above an image or anchor-like line read as a YAML tag or anchor. Every docs page is admitted, so that was a floor refusing the corpus it exists to pass. Both are scoped to a block opening at line 0, a byte-order mark allowed, which is the only place the stripper recognises one. The shared pre-existing residues are disclosed in the charter: a four-space-indented fence delimiter is accepted here as a fence where CommonMark reads an indented code block, so a heading between two such lines is masked; and a double-quoted scalar continued across lines with a brace on a continuation line refuses, because the scan reads one line at a time."
impact: internal
---

unresolvableFrontmatterShape and the setext first-block bound run on the first --- found anywhere, so a frontmatter-less admitted doc with a thematic break refuses the whole assembly (a lone rule reads as an unclosed block; a rule followed by an image line reads as a YAML tag; a rule followed by an anchor-like line or a dots paragraph likewise); every docs page is admitted and several open with image lines. Scope both to a block opening at line 0 (BOM allowed), which is the only place the stripper recognises one. Residue, pre-existing and shared with the site renderer: a four-space-indented fence delimiter is accepted as a fence while CommonMark reads an indented code block, so a heading between two such lines is masked.
