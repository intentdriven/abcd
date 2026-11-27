---
schema_version: 1
id: "iss-2611272337581253"
slug: "the-site-build-inlines-an-svg-drawing"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the inlined-SVG naming fix for iss-2610241550257430"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/assets.go"
remedy: "In checkInlinableSVG, read each start tag's raw bytes (the decoder's InputOffset brackets them, as rootStartTag does) and refuse a single-quoted attribute value or space on either side of an equals sign, naming the asset and the attribute; prove it with tests whose inlined output goes through parseHTML."
---

The site build inlines an SVG drawing verbatim, and its asset check (checkInlinableSVG in internal/core/site/assets.go) reads the drawing through the XML decoder, which normalises an attribute's spelling away. A drawing that writes an attribute value in single quotes, or puts space around the equals sign, passes the asset check, and the page's own reader (htmlscan) then refuses the page on its raw spelling, naming the page and not the drawing, so lint site and the site-render gate fail with nothing that points at the asset. Deferred out of iss-2610241550257430 by dcn-2611272337491591.
