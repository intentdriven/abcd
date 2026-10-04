---
schema_version: 1
id: "iss-2610040732240935"
slug: "record-pages-scroll-sideways-at-360-px-whenever-a-record-s"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "site-screenshots failing on every PR since 2026-10-03 23:37Z"
origin: researcher-authored
production_mode: hand-written
found_at: "site-src/site.css"
remedy: "Set overflow-wrap:break-word on body in site-src/site.css and its seeded copy internal/core/site/setupsrc/site.css, as a net under every block of text: it breaks a token only where the line would otherwise overflow and, unlike anywhere, leaves min-content sizing alone, so tables, grids and flex rows lay out as before (CSS Text Module Level 3, overflow-wrap). Make the overflow audit also sample, per type, the record whose title or source path holds the longest run without a space or hyphen, so the worst case for wrapping is measured on every pull request rather than the shortest. A Go test pins the body rule; the audit run proves it in a browser."
---

Record pages scroll sideways at 360 px whenever a record's title or source path holds a long unbroken token, and the overflow audit cannot see it. Every ADR minted with a timestamp id shows its source path, .abcd/development/decisions/adrs/<16 digits>-<slug>.md, as one mono span in the side panel's reclinks paragraph, and the run up to the first hyphen is about fifty characters with no break opportunity: /record/adr/adr-2609091248200336/ measures scrollWidth 374 at 360 px. adr-2610031751065746 and adr-2610031751066232 also carry a home-relative path in their titles, which overflows the record page's h1 (scrollWidth 455). The stylesheet sets no overflow-wrap on body, the title or reclinks, so a block whose text has nowhere to break widens the page. The site-screenshots audit samples only the lowest-numbered record of each type (adr-1, itd-1, iss-1), whose paths and titles are short, so these pages have overflowed since timestamp ADRs began without the job noticing.
