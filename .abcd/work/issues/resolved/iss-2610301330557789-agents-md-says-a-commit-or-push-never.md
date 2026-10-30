---
schema_version: 1
id: "iss-2610301330557789"
slug: "agents-md-says-a-commit-or-push-never"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Add repo-scoped entries to .abcd/guard.json that refuse the hook-skipping flag on git merge, rebase, am and pull, matching the short form only on am, where it means that flag, each entry with red and green fixtures."
resolution: "The repository's own .abcd/guard.json now refuses the hook-skipping flag on git merge, rebase, am and pull, each entry with known-bad and known-good fixtures that internal/core/guard/repo_guard_test.go holds to the bundled admission gate and to a fixed table of verdicts. The bundled entries already cover commit and push. The short form -n is matched only on am, where git reads it as that flag. cherry-pick has no entry because git gives it no such flag."
impact: internal
---

AGENTS.md says a commit or push never skips the repository's hooks, but in this repository the guard refuses the hook-skipping flag only on git commit and git push. git merge, git rebase, git am and git pull each take the same flag, and the guard allows all four, so for those verbs the rule is prose alone.

## Grounds

- pursued: git's own option tables (git 2.52, each verb's -h) are the authority on which verbs take the hook-skipping flag and what -n means on each: it is that flag on commit and am, the diffstat switch on merge, rebase and pull, dry-run on push, and no-commit on cherry-pick, which has no such flag. A later git that adds the flag to another verb, or gives -n that meaning elsewhere, would show the entry set incomplete.
