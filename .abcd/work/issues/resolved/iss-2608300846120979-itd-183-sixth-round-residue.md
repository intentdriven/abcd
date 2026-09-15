---
schema_version: 1
id: "iss-2608300846120979"
slug: "itd-183-sixth-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 sixth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go, internal/core/reading/assemble_test.go"
resolution: "The tag floor caught the double-bang shorthand alone; any line-initial bang is a tag now — local and verbatim included — any ampersand is an anchor, and any explicit-key line the readable-key pattern cannot fully read is refused rather than assumed. The raw-heading scan uses the fence mask only to disqualify an open tag sitting inside a fence: blanking every fenced line erased the heading's own text whenever the block ran through a fence delimiter, so a heading opened in plain prose was mis-titled and admitted. An element carrying a heading role joins the open-tag pattern, since it renders and is announced as a heading with no h-tag present. The renderedText comment states that a double-encoded reference is left as the different heading it renders as, deliberately, and the charter names that residue beside the blockquote and homoglyph ones and states the fold-or-render equality they are measured against. The amp-entity probe asserts through the shared helper instead of skipping its own assertion."
impact: internal
---

itd-183 sixth-round residue: the tag and anchor floor catches only the double-bang shorthand at line start — single-bang local and verbatim tags, and an explicit key carrying a tag, an anchor or an escape are all admitted (refuse any tag at line start and any explicit-key line the pattern cannot fully read); the raw-heading scan blanks fenced lines so an HTML block opened outside a fence that continues through fence delimiters is mis-titled and admitted (use the fence mask only to disqualify an open tag on a fenced line); a div with a heading role is neither caught nor disclosed; the renderedText comment does not say a double-encoded reference is deliberately left as the different heading it renders as; the amp-entity test still continues on error rather than asserting the refusal the commit and record claim.
