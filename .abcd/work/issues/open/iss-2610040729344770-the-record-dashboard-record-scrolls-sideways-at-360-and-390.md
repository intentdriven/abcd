---
schema_version: 1
id: "iss-2610040729344770"
slug: "the-record-dashboard-record-scrolls-sideways-at-360-and-390"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "site-screenshots failing on every PR since 2026-10-03 23:37Z"
origin: researcher-authored
production_mode: hand-written
found_at: "site-src/site.css"
remedy: "Give the non-id spans of a .list row the treatment .links li>span already has: .list li>span:not(.id){overflow-wrap:anywhere;min-width:0} in site-src/site.css and its seeded copy internal/core/site/setupsrc/site.css, the id column keeping white-space:nowrap. Grounds: overflow-wrap:anywhere, unlike break-word, lowers the element's min-content width (CSS Text Module Level 3, overflow-wrap), and min-width:0 removes the grid item's content-based automatic minimum (CSS Grid Layout Level 1, section 6.6), so the 1fr track gives whatever title arrives; the stylesheet's own comment above .refs li records the same reasoning for bibliography addresses. A Go test pins the rule and the row markup it matches; the overflow audit proves it in a browser at 360/390/768/1360 px."
---

The record dashboard (/record/) scrolls sideways at 360 and 390 px: the site-screenshots job measures scrollWidth 476 to 482 there on every pull request since 2026-10-03 23:37Z. The cause is not the genealogy timeline, whose svg sits inside its own overflow-x:auto panel and cannot widen the page, and not the four sub-navigation links the job names first, which scroll inside their own bar; it is the Latest decisions panel. A .list row is a two-column grid (auto 1fr) whose title is a plain span, and a grid item's automatic minimum width is its min-content width, so a title holding one long unspaced token sets the width of the row, which spills past the panel and the page. The two ADRs from #795 (adr-2610031751065746 and adr-2610031751066232) carry such titles, a home-relative path with no break in it, and became the two newest decisions on 2026-10-03. Every .list on the site (dashboard decisions, health, status, development, supersessions) has the same shape, so any record whose title holds a long token reproduces it.
