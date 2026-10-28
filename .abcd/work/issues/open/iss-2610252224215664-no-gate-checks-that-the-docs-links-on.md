---
schema_version: 1
id: "iss-2610252224215664"
slug: "no-gate-checks-that-the-docs-links-on"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "state-of-the-art passes of the hand-back triage, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/check.go"
remedy: "Add a check to the site build's verification that every /docs/ link on the composed pages and every site-src/redirects target names a file in the built output, and run it in the site workflow."
---

No gate checks that the /docs/ links on the composed site pages and the site-src/redirects targets resolve in the built output. abcd lint site drops docs/ from its page walk (internal/core/site/check.go), no workflow runs a link checker, and siteHref (internal/core/site/compose.go) hard-codes the docs generator's URL shape, so a generator change that moved those URLs would break the links with every gate passing.
