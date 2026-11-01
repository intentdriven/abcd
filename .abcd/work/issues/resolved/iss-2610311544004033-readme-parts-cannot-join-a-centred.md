---
schema_version: 1
id: "iss-2610311544004033"
slug: "readme-parts-cannot-join-a-centred"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "readme setup after the v0.16.0 update"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_marker.go"
remedy: "Let a repository choose an inline form for the README parts: the badge parts written as HTML images that sit together in its own centred badge row, and links and contents as one line of links, placed where the owner put them; an absent part goes after an HTML h1 too, not above it. Then set readme.parts back on for abcd's own README, whose layout-C header is hand-written until then."
resolution: "README parts whose opening mark continues an open block-level HTML block are written in an inline HTML form (badges as <a><img></a>, licence, links and contents as one line of links); an absent part beside an inline one joins its block, one with no part present goes after an HTML <h1> too, and removing an inline part leaves no blank line; abcd's own README has readme.parts on again with its layout-C header filled in place."
impact: additive
resolved_by:
  commit: "7ea1fe00d5824907d9f4b5beb391a2555d6658b5"
---

abcd's README parts cannot form a centred header. Each part is written as its own paragraph, so the three badges stack on three lines and cannot join a centred badge row written in HTML, and links and contents are always bulleted lists. With no Markdown level-one heading, the first absent part goes to the very top of the file. In abcd's own README, choosing all six parts put six blocks above the centred logo and title, with a second release badge, so the README was rewritten by hand and the parts turned off.

## Grounds

- pursued: a part placed inside a centred HTML header renders as a badge or link there, and setup fills abcd's layout-C header with the same rows in the same order; shown wrong if a part inside a <p> or <div> block is written as Markdown, an absent part lands above an HTML <h1>, a removal or insertion adds a blank line inside the block, or a second setup run rewrites the README.
