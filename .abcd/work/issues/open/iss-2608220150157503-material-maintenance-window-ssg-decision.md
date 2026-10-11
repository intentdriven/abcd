---
schema_version: 1
id: "iss-2608220150157503"
slug: "material-maintenance-window-ssg-decision"
severity: "major"
category: "tech-debt"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "docs/requirements.txt"
remedy: "Waits on ruling M8: stay on Material for MkDocs now (critical maintenance runs to 2027-05-05 and docs/ uses no feature it is losing); move to Zensical at 0.1.0 or later (from 2026-11-05) in one change that swaps the pinned package and the build command and pins the full resolved set with hashes, precondition a scratch build of the unchanged mkdocs.yml rendering all 14 pages with working search and a passing abcd lint site; Hugo with Hextra is the fallback if that build fails or 0.1.0 slips past 2027-02. Grounds: research at .abcd/development/research/notes/2026-09-30-docs-site-generator-sota.md (Zensical reads mkdocs.yml natively, same Python toolchain, lowest reversal cost; Hextra needs Hugo extended and a config rewrite)."
deferred_after: v0.11.1
deferral_reason: "research at .abcd/development/research/notes/2026-09-30-docs-site-generator-sota.md; ruling owed: M8, the product thinker's choice of the docs generator as an ADR, which also signs off the dependency swap (the note proposes staying on Material now and moving to Zensical at 0.1.0 or later; Material's critical maintenance is extended to 2027-05-05)"
---

The docs toolchain is on a closing maintenance window: MkDocs 1.x has had no release since August 2024, Material for MkDocs announced maintenance mode in November 2025 (critical bug fixes and security updates for 12 months at least, no new features), and MkDocs 2.0 removes plugins entirely. An SSG decision (Zensical, Hugo/Hextra, or other) is due as an ADR before the window closes, around November 2026; adr-47 keeps generation outside the SSG so the migration touches only mkdocs.yml, overrides and the build command

## Deferral 2026-09-30

Deferred past v0.11.1: research at .abcd/development/research/notes/2026-09-30-docs-site-generator-sota.md; ruling owed: M8, the product thinker's choice of the docs generator as an ADR, which also signs off the dependency swap (the note proposes staying on Material now and moving to Zensical at 0.1.0 or later; Material's critical maintenance is extended to 2027-05-05)
