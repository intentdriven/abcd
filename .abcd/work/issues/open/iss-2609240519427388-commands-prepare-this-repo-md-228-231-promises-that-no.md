---
schema_version: 1
id: "iss-2609240519427388"
slug: "commands-prepare-this-repo-md-228-231-promises-that-no"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/prepare-this-repo.md"
---

commands/prepare-this-repo.md:228-231 promises that no lint-config JSON is committed, but step 5 runs ahoy install, which writes .abcd/docs-lint.json into the target repository, so an adopter commits abcd-authored content the page said would not be there. Found by the v0.10.0 brief-surface crosscheck at fa744b41 (finding x-050).
