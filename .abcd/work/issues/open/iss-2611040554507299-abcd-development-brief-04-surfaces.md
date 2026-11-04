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
---

.abcd/development/brief/04-surfaces/22-site.md:114-117 and 154-156 describe setup as writing the site's settings and workflow, say the build and gate refuse settings of the other shape, and say abcd's composed site is not a project site. Where .abcd/site.json is a composition, `site setup --project` writes .abcd/project-site.json and .github/workflows/project-site.yml beside it, as this tree carries, and `abcd site build --project` reads that file first (internal/core/site/project.go:168-180). The chapter names neither file.
