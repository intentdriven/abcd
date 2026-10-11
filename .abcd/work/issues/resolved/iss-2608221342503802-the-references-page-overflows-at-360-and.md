---
schema_version: 1
id: "iss-2608221342503802"
slug: "the-references-page-overflows-at-360-and"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "agent-finding"
found_at: "site-src/site.css"
resolution: "site.css carries overflow-wrap:anywhere on .refs/.insp list content and a .panel anchor overflow net; the CSS comment names this exact defect"
impact: fix
---

the references page overflows at 360 and 390 px (scrollWidth 484): long unbroken link tokens escape their column; found by the screenshot audit's first run while the static mobile gate reports ok on the same tree — the complementary-gates argument made concrete