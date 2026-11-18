---
schema_version: 1
id: "iss-2611040554507299"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "Describe the side-by-side layout and what the route refusal then applies to."
resolution: "The site chapter's project-site section now names the usual paths (.abcd/site.json, .github/workflows/site.yml) and describes the side-by-side layout: where .abcd/site.json is a composition, setup on the project route writes .abcd/project-site.json and .github/workflows/project-site.yml beside it, and the project route reads that file first; the refusal of a composition applies to .abcd/site.json only when that file is absent. TestBriefProjectSiteNamesTheSideBySideFiles pins the four paths to the section."
impact: fix
resolved_by:
  commit: "1b8493330c3d5ee3e455a09687e7a96874a95fbb"
---

.abcd/development/brief/04-surfaces/22-site.md:114-117 and 154-156 describe setup as writing the site's settings and workflow, say the build and gate refuse settings of the other shape, and say abcd's composed site is not a project site. Where .abcd/site.json is a composition, `site setup --project` writes .abcd/project-site.json and .github/workflows/project-site.yml beside it, as this tree carries, and `abcd site build --project` reads that file first (internal/core/site/project.go:168-180). The chapter names neither file.

## Grounds

- pursued: the chapter matches loadRouteFrom/routeAgrees (internal/core/site/project.go) and setupComposition (internal/core/site/setup.go); a project build in a composed repository that reads .abcd/site.json before .abcd/project-site.json, or a setup that writes elsewhere, would show it wrong
