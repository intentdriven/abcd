---
schema_version: 1
id: "iss-2608220150157503"
slug: "material-maintenance-window-ssg-decision-due-before-nov-2026"
severity: "major"
category: "tech-debt"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "docs/requirements.txt"
deferred_after: v0.11.1
deferral_reason: "a research lane owed, then a ruling (re-deferred at v0.11.1 by run A's major-triage lane): ruling M8 (2026-09-23) is research first, a short Zensical vs Hugo/Hextra comparison against the docs' needs (overrides, tags, search; no Node toolchain; new dependencies need sign-off), then the product thinker's choice as an ADR before Material's maintenance window closes around November 2026. The comparison has not been commissioned, and it is due now: the window closes within the next cycle."
---

The docs toolchain is on a closing maintenance window: MkDocs 1.x has had no release since August 2024, Material for MkDocs announced maintenance mode in November 2025 (critical bug fixes and security updates for 12 months at least, no new features), and MkDocs 2.0 removes plugins entirely. An SSG decision (Zensical, Hugo/Hextra, or other) is due as an ADR before the window closes, around November 2026; adr-47 keeps generation outside the SSG so the migration touches only mkdocs.yml, overrides and the build command

## Deferral 2026-09-29

Deferred past v0.11.1: a research lane owed, then a ruling (re-deferred at v0.11.1 by run A's major-triage lane): ruling M8 (2026-09-23) is research first, a short Zensical vs Hugo/Hextra comparison against the docs' needs (overrides, tags, search; no Node toolchain; new dependencies need sign-off), then the product thinker's choice as an ADR before Material's maintenance window closes around November 2026. The comparison has not been commissioned, and it is due now: the window closes within the next cycle.
