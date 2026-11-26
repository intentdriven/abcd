---
schema_version: 1
id: "iss-2611252038564722"
slug: "install-a-checked-release-unasked-where"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs"
remedy: "Plan it as an intent after the standing check ships: install a checked release between sessions where the person's own machine setting, never a repository's, says the facilitator is auto, with the rollback mark held, and refine the ADR in the same change."
---

abcd installs a newer release of itself only before a drive, on the person's setting; every other copy goes stale until someone acts. The product thinker ruled on 2026-11-25 that outside a drive it may install unasked where the technical facilitator is auto, and not by default. Nothing builds it, and adr-2611251808065076 still says nothing applies a release unasked. Its third point stands: no repository file gives the yes, and facilitator.mode can come from a repository file.
