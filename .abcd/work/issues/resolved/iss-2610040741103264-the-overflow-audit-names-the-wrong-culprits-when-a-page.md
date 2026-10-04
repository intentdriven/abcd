---
schema_version: 1
id: "iss-2610040741103264"
slug: "the-overflow-audit-names-the-wrong-culprits-when-a-page"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "site-screenshots failing on every PR since 2026-10-03 23:37Z"
origin: researcher-authored
production_mode: hand-written
found_at: "site-src/audit/overflow-audit.js"
remedy: "In measure(), skip any element with an ancestor whose computed overflow-x is not visible (its overflow is contained there and cannot widen the document), and also walk text nodes, measuring each with a Range, naming the parent element of any unclipped text run past the edge. Grounds: CSS Overflow Module Level 3 makes a box whose overflow-x is not visible a scroll container, and content inside it contributes to that container's scrollable overflow rather than to its ancestors'; a local probe that applies exactly this filter named the real span on /record/ and the h1 text on the ADR page."
resolution: "Fixed in measure(): an element or text run inside an ancestor whose overflow-x is not visible is skipped, and text runs are measured by their Range and named by their parent element. Against the unfixed site the audit names the .list title span on /record/, the mono source-path span on a timestamp-id ADR page and the h1 text on an issue page, and no sub-navigation link or timeline svg."
impact: fix
resolved_by:
  commit: "b89a52d71"
---

The overflow audit names the wrong culprits when a page scrolls sideways. measure() in site-src/audit/overflow-audit.js lists the first five elements whose box passes the viewport edge, including elements inside their own scrolling container (overflow-x auto, scroll or hidden), which cannot widen the page. On /record/ at 360 px it named the four sub-navigation links and the genealogy svg, both inside their own scroll containers, and never the Latest decisions title span that set scrollWidth 482, which sent the bisect after the timeline renderer. It also reports nothing when the overflow is a text run rather than an element box, as on a record page whose h1 holds a long unbroken path.

## Grounds

- pursued: every culprit the audit names can widen the document; a failing run whose culprits include an element inside a scroll container, or a text-run overflow reported with no culprit, would show it wrong.
