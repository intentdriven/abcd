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
---

The README contents part builds each anchor slug from the heading's markdown source rather than its rendered text, so an emoji shortcode or an HTML entity in a heading gives an anchor the forge does not have: '## :rocket: Launch' is linked as #rocket-launch where the forge renders #-launch, and '## A &amp; B' as #a-amp-b where the forge renders #a--b.
