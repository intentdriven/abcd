---
schema_version: 1
id: "iss-2610111850237083"
slug: "the-readme-contents-part-builds-each"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of README parts step 2, run-2610110908176735"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Slug the heading's rendered text: render emoji shortcodes and decode HTML entities as the forge does before slugging, or omit such a heading's anchor with a stated reason."
resolution: "The contents part slugs each anchor from the heading's rendered text: character references are decoded as the forge does, and a heading carrying an emoji shortcode, with every later heading whose anchor it could take or number, is listed without a link and named in a note."
impact: fix
resolved_by:
  commit: "9802aa384c27ae0a13f63109c96a0b84212ee11e"
---

The README contents part builds each anchor slug from the heading's markdown source rather than its rendered text, so an emoji shortcode or an HTML entity in a heading gives an anchor the forge does not have: `## :rocket: Launch` is linked as `#rocket-launch` where the forge renders `#-launch`, and `## A &amp; B` as `#a-amp-b` where the forge renders `#a--b`.

## Grounds

- pursued: every contents link names an anchor the forge's page has; a contents link to an anchor missing from the rendered page shows it wrong
