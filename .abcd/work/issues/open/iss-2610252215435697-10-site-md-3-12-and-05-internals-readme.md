---
schema_version: 1
id: "iss-2610252215435697"
slug: "10-site-md-3-12-and-05-internals-readme"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md"
remedy: "Check the live root after the latest tagged deploy. If it serves the site build, remove the design-target mark from both pages, as the banner itself asks ('A shipping change removes the mark it lands')."
---

10-site.md:3-12 and 05-internals/README.md:16 still mark the abcdev.app root as a design target that 'still serves the MkDocs rendering' until the first tagged deploy; the release chain has called site.yml since 2026-08-22 (auto-release.yml:148), and 26 tags have been cut since. This describes outside state and is probably stale
