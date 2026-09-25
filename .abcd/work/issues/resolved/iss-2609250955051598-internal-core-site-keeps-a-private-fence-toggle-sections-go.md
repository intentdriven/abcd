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
resolution: "The site's Sections, Blocks, auditIsMet and releaseOf read fences and HTML comments by mdrecord's ListNested rule; the private isFenceLine toggle is gone. Tilde fences, a quoted shorter run, a closer with an info string and a commented heading no longer make phantom sections, an unclosed tilde fence or comment is refused, and a tilde-fenced or commented rollup or credit no longer counts. The fix is two commits: 7f46da18 moved the site onto mdrecord, and at that commit site-render still refused the committed corpus (an issue record quoting a frontmatter block in a top-level fence with an indented closer read as an unclosed list-item fence); 7de521bc made ListNested read a run at up to three columns as TopLevel does, which is what made site-render pass, so it is the commit this resolution names."
impact: fix
resolved_by:
  commit: "7de521bc"
---

internal/core/site keeps a private fence toggle (sections.go isFenceLine, used by Sections, Blocks, and compose.go auditIsMet and releaseOf) that diverges from mdrecord.Mask, the tree's CommonMark reading, and returns wrong sections. isFenceLine matches any line whose trimmed text starts with three backticks and flips a boolean. So it never sees a tilde fence, closes a four-backtick fence on a three-backtick line, closes on a line that carries an info string, and cannot see an HTML comment. Probe at 70daf701 via Sections: a # line inside a ~~~ block becomes a section, and so does a # line inside a three-backtick block quoted in a four-backtick fence (both give sections Doc, a shell comment, Real). A heading parked in <!-- --> becomes the section Parked. An unclosed ~~~ fence is not refused, even though Sections refuses an unclosed backtick fence as its quietest failure. The same walk decides auditIsMet (a ~~~-fenced Acceptance rollup: MET 1 counts as a verdict, the failure iss-2609090951277880 closed for backtick fences) and releaseOf (a ~~~-fenced credit stamps a version). The fix is not a drop-in swap for mdrecord.Mask, because the site deliberately reads fences indented under list items and mdrecord recognises only the 0-3 space indent, so the indent rule needs a decision first.

## Grounds

- pursued: every document shape in the capture yields only its live sections under Sections (TestSectionsReadFencesAndCommentsByTheCommonMarkRule) and, from 7de521bc on, the site package's committed-corpus tests and the site-render gate pass over the committed corpus (at 7f46da18 alone site-render refused it); a phantom section, an unrefused unclosed span, or a counted fenced rollup or credit would show it wrong
