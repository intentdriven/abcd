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
resolution: "commands/prepare-this-repo.md already says the config ahoy install seeds, .abcd/docs-lint.json among them, is the repository's own and is committed; only hand-copied lint config is barred"
impact: fix
resolved_by:
  commit: "d0c1899262284daae9413703dbf4c203b992995f"
---

commands/prepare-this-repo.md:228-231 promises that no lint-config JSON is committed, but step 5 runs ahoy install, which writes .abcd/docs-lint.json into the target repository, so an adopter commits abcd-authored content the page said would not be there. Found by the v0.10.0 brief-surface crosscheck at fa744b41 (finding x-050).

## Grounds

- pursued: the page no longer promises an absence ahoy install contradicts; a page barring the seeded docs-lint.json would show it wrong
