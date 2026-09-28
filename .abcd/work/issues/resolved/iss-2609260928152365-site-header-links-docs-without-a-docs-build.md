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
resolution: "the header's Docs entry and the docs routes follow the composition's docs block, so a site with no docs surface links no /docs/ tree and sends a docs-page link to the forge's view"
impact: fix
resolved_by:
  commit: "fbc516d1"
---

The site header always links /docs/ (compose.go:334), so a managed repository set up with site setup and no docs build serves a navigation link that 404s on every page. The renderer's page shapes were out of the sitesetup lane's scope; the link should follow whether a docs tree is rendered.

## Grounds

- pursued: a managed repository's setup-written site carries no href to /docs/ while abcd's own keeps its Docs link; any /docs/ href on a site whose composition names no docs index would show it wrong
