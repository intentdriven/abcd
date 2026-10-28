---
schema_version: 1
id: "iss-2610241550257430"
slug: "when-the-site-build-inlines-an-svg"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b12 reading iss-2608220750024303"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/assets.go"
remedy: "In the SVG branch of internal/core/site/assets.go, give the wrapping span role img and the page's alt as its aria-label with the inlined drawing marked aria-hidden, and mark the span aria-hidden when the alt is empty, so the site and GitHub name the picture alike; proven by a site test that a page writing an SVG image with alt text x renders a picture whose accessible name is x. Rests on WCAG 2.2 technique H37 and the W3C images decision tree, both already cited on iss-2608220750024303."
---

When the site build inlines an SVG picture, the alt text the docs page gives it is dropped: the SVG branch of the asset renderer (internal/core/site/assets.go:425-432) returns the drawing wrapped in a span and never uses alt, so the picture's accessible name on the site is whatever the file itself declares (docs/explanation/artefacts.md:13 gives The brief while docs/assets/img/artefact-brief.svg declares brief), and nothing requires an inlined SVG to declare a name at all, so a drawing without one would reach the site unnamed while its page's alt is discarded.
