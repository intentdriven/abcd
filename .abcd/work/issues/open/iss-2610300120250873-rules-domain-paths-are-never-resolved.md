---
schema_version: 1
id: "iss-2610300120250873"
slug: "rules-domain-paths-are-never-resolved"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "one-home principle review (PR #1243), 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a lint that resolves every repository path named in a rules domain (the bundled defaults in internal/core/rules/defaults/rules.json and a repository's own .abcd/rules.json) and fails on one that does not exist, so a moved home breaks the check that names its pointer."
---

A rules domain points at its home with a bare repository path, and nothing checks that the path still resolves. The link check reads markdown links only, and the one test that reads an OPINIONS path checks the users-directory principle alone. So renaming or moving a principle leaves its OPINIONS rule pointing at a file that is gone, the loader keeps injecting the dead pointer into every matching prompt, and every check stays green.
