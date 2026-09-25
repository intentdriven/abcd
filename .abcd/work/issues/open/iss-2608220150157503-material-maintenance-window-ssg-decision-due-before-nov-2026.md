---
schema_version: 1
id: "iss-2608220150157503"
slug: "material-maintenance-window-ssg-decision-due-before-nov-2026"
severity: "major"
category: "tech-debt"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "docs/requirements.txt"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): After the Zensical vs Hugo/Hextra comparison, which static site generator succeeds Material?"
---

The docs toolchain is on a closing maintenance window: MkDocs 1.x has had no release since August 2024, Material for MkDocs announced maintenance mode in November 2025 (critical bug fixes and security updates for 12 months at least, no new features), and MkDocs 2.0 removes plugins entirely. An SSG decision (Zensical, Hugo/Hextra, or other) is due as an ADR before the window closes, around November 2026; adr-47 keeps generation outside the SSG so the migration touches only mkdocs.yml, overrides and the build command