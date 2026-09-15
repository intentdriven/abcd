---
schema_version: 1
id: "iss-2608300402598718"
slug: "itd-183-fifth-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 fifth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go, .abcd/development/readings/README.md"
resolution: "The raw-HTML scan runs over the unfenced body joined rather than line by line, matching an opening tag alone, so the unclosed, self-closing, attribute-carrying and multi-line forms are covered; the match offset maps back to a line so the refusal still names one. Its comment is corrected: the site's page reader refuses a raw HTML block rather than admitting h1-h6. A tag, an anchor and a block-scalar explicit key join the escape-is-the-signal set, refused as constructions whose keys resolving would mean a YAML parser. Every raw-line scan starts at the first body line, because running them over the frontmatter refused a block scalar ending in the excluded title as an underlined heading — a true refusal reached by a false reading, which teaches the wrong fix. And the charter and the code now name the two shapes the floor does not see: a heading nested in a blockquote or list item, and a title reaching the excluded one through a homoglyph or a format character."
impact: internal
---

itd-183 fifth-round residue, all Low: a heading nested in a blockquote or list item and a homoglyph or format-character title leak, and neither residue is disclosed in the charter or the code although the fourth-round record named them; the raw-HTML refusal's comment says the site's markdown subset admits h1-h6 when the site refuses raw HTML blocks, and the per-line pattern misses an unclosed, multi-line, self-closing or attribute-carrying heading and a div with a heading role; a tagged key, an anchored key and a block-scalar explicit key are spellings the escape-is-the-signal rationale covers but the pattern set does not; the setext scan runs over frontmatter lines so a block scalar ending in the excluded title before the closing --- is refused with the wrong diagnosis.
