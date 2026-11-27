---
id: dcn-2611272337491591
slug: defer-out-of-the-inlined-svg-naming
date: 2026-11-27
---
2026-11-27 — Defer, out of the inlined-SVG naming change for iss-2610241550257430, the drawing whose attribute is single-quoted or written with space around the equals sign. The site build inlines such a drawing verbatim and the page's reader (htmlscan) refuses the page on its raw spelling, naming the page and not the drawing. Deferred because catching it needs the asset's raw bytes, not the XML decoder's tokens, which normalise both spellings away, so it is a separate change to the asset check with its own tests; a follow-up issue holds the work. Reverse by fixing it in this change instead: refuse, in checkInlinableSVG, a start tag whose raw text carries a single-quoted value or space around an equals sign, naming the asset.
