---
schema_version: 1
id: "iss-2610040758095861"
slug: "ahoy-install-seeds-abcd-docs-lint-json-with-roots-docs"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/docs-lint.json"
remedy: "Seed roots from what exists at install time (docs only when that folder exists, README.md only when that file exists), or have the install name a seeded root that is missing and how to fix it; test: an install in a fixture repository with no docs/ leaves a docs-lint config that abcd lint docs runs to exit 0 or 1, never 2."
resolution: "ahoy install now seeds .abcd/docs-lint.json with roots from what exists at install time: docs only when that folder exists, README.md only when that path exists. With neither, roots is empty, so abcd lint docs runs, reads nothing and warns that nothing was checked at exit 0, never refusing with exit 2 over a root the seed named itself. An existing config is never changed."
impact: fix
---

ahoy install seeds .abcd/docs-lint.json with roots ["docs", "README.md"] whatever the repository holds, so in a repository with no docs/ folder the documentation check it has just armed cannot run: abcd lint docs exits 2 with 'roots entry "docs" does not exist', and neither the install receipt nor bare abcd ahoy says the seeded check is broken. Reproduced at 57d5ec9fa in a scratch repository holding only a README.md, after an install given --scan-deep false. A downstream lab reported the check as silently disarmed; at tip the lint fails loudly on a missing root (since iss-2608270500208736, which already named this seed as the trigger), so the defect is the seed its fix left in place, not a silent pass.
