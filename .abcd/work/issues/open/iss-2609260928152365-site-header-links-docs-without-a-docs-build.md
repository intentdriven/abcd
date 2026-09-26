---
schema_version: 1
id: "iss-2609260928152365"
slug: "site-header-links-docs-without-a-docs-build"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: sitesetup lane report risk note"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
---

The site header always links /docs/ (compose.go:334), so a managed repository set up with site setup and no docs build serves a navigation link that 404s on every page. The renderer's page shapes were out of the sitesetup lane's scope; the link should follow whether a docs tree is rendered.
