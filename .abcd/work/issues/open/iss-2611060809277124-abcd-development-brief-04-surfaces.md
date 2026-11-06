---
schema_version: 1
id: "iss-2611060809277124"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "Describe the timeline as a dashboard panel that `pages.timeline` switches off, under its displayed label, and correct the comment in `internal/core/site/timeline.go`."
---

`.abcd/development/brief/04-surfaces/22-site.md:160-161` counts the timeline among the pages every repository gets and the `pages` block switches off. No timeline page or route exists: `internal/core/site/explorer.go:533-536` renders it as a folded panel on the `/record/` dashboard, labelled 'Genealogy' (`site-src/ui.json:40`), and `pages.timeline` removes only that panel. The comment at `internal/core/site/timeline.go:3` still names `/record/timeline/`.
