---
schema_version: 1
id: "iss-2608300940486285"
slug: "scalar-position-needs-yaml-whitespace-and-heading-bound-blanking"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 eighth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (quotedSpanRe, rawHeadingOpenRe, rawHeadingEndFor, rawHeadingEnds, excludedKeyInFirstBlock)"
resolution: "A colon and a dash are YAML indicators only with the whitespace YAML requires, so a quote in a plain scalar (a:'b or a - 'b) no longer opens a span that blanks the excluded key beside it; the raw-heading scan runs over a copy whose angle brackets inside HTML comments and quoted attribute values are blanked length-preservingly, so a closing tag or a greater-than written as data no longer bounds a title or ends an opening tag; the key scan takes its block from firstBlockRange, closing the line-0 disagreement; and the per-element regexp cache keyed by a document-chosen element name is replaced by one static bound pattern with a case-folded name compare."
impact: internal
resolved_by:
  intent: "itd-183"
  spec: "spc-61"
---

itd-183 eighth-round security findings: the scalar-position rule treats a colon or a dash as an opener without the whitespace YAML requires, so a quote inside a flow plain scalar (a - 'b, origin: X or a:'b, origin: X) blanks the excluded key beside it — the commit's own test shape with one extra space; verified fix: open a scalar only at line start, after a line-start dash, after a colon followed by whitespace, or after a brace, bracket or comma. The raw-heading title bound stops at a closing tag inside an HTML comment or an attribute string and the opener at a greater-than inside an attribute value, so a heading every browser renders as the excluded title is judged as something else — blank comments and quoted attribute values length-preservingly before bounding, or disclose the three shapes. firstBlockRange recognises a block at line 0 only while excludedKeyInFirstBlock still takes the first --- anywhere (also in the ruthless record). The per-element regexp cache is keyed by an attacker-chosen element name and never evicted — drop it or use one static closer pattern with a case-folded name compare.
