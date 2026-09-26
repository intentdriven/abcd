---
schema_version: 1
id: "iss-2609261206437257"
slug: "the-site-renderer-links-a-repository-file-on-the-forge"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
---

The site renderer links a repository file on the forge through a hard-coded blob/main path (internal/core/site/compose.go siteHref and seven sibling sites in compose.go, csl.go, explorer.go and graphpage.go), so a managed repository whose default branch is not main serves forge links that 404; the build reads no forge, and blob/HEAD names the default branch whatever it is
