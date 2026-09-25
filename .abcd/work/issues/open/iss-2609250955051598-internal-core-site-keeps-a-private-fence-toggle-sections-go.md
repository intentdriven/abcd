---
schema_version: 1
id: "iss-2609250955051598"
slug: "internal-core-site-keeps-a-private-fence-toggle-sections-go"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/sections.go"
---

internal/core/site keeps a private fence toggle (sections.go isFenceLine, used by Sections, Blocks, and compose.go auditIsMet and releaseOf) that diverges from mdrecord.Mask, the tree's CommonMark reading, and returns wrong sections. isFenceLine matches any line whose trimmed text starts with three backticks and flips a boolean. So it never sees a tilde fence, closes a four-backtick fence on a three-backtick line, closes on a line that carries an info string, and cannot see an HTML comment. Probe at 70daf701 via Sections: a # line inside a ~~~ block becomes a section, and so does a # line inside a three-backtick block quoted in a four-backtick fence (both give sections Doc, a shell comment, Real). A heading parked in <!-- --> becomes the section Parked. An unclosed ~~~ fence is not refused, even though Sections refuses an unclosed backtick fence as its quietest failure. The same walk decides auditIsMet (a ~~~-fenced Acceptance rollup: MET 1 counts as a verdict, the failure iss-2609090951277880 closed for backtick fences) and releaseOf (a ~~~-fenced credit stamps a version). The fix is not a drop-in swap for mdrecord.Mask, because the site deliberately reads fences indented under list items and mdrecord recognises only the 0-3 space indent, so the indent rule needs a decision first.
