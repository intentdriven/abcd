---
schema_version: 1
id: "iss-2609300756163382"
slug: "the-shell-rules-domain-teaches-only-the-bundled-hazard"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane teachRepoGuard, ruling CK1; question raised by lane teachPlane)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/shell.go"
remedy: "Rebuild SHELL on every rules load from the registry the guard enforces in the repository (guard.LoadRepo: the bundled entries merged with .abcd/guard.json), through the same generator (Lessons/RecallTerms), marking every lesson whose words are the repository's with (repo) after its entry id so provenance is visible (GHSA-22f8-qf5r-gjgq); a guard.json the guard refuses is refused here too, loudly (a load note on stderr naming the file and reason) and never taught, while SHELL teaches the registry the guard falls back to. Grounds: ruling CK1 verbatim, and J10's single-source rule (the registry is the one source)."
---

The SHELL rules domain teaches only the bundled hazard registry: an entry a repository adds in its own .abcd/guard.json is refused by the guard but never taught before shell work, so the teaching plane and the execution plane of itd-103 part in exactly the repositories that extended the registry. Ruling CK1 (2026-09-29, the product thinker): teach a repository's own guard entries in the SHELL domain, generated the same way as the bundled registry.
