---
schema_version: 1
id: "iss-2611040555249487"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Point the chapter at the command page and correct the help's parameter-expansion item."
---

.abcd/development/brief/04-surfaces/17-guard.md:1034-1036 calls the check's own help the fuller statement of the list, with a worked example for each. `abcd guard check --help` is the shorter list and lacks the chapter's symlink `..`, `~user`, relative `..` and variable carry-in items. It also lists `--$VAR` and a variable as the program among what an allow does not see, yet `git push --$VAR origin main` and `$GIT push --force` both exit 1, as .abcd/development/brief/04-surfaces/17-guard.md:803 says. The fuller statement in the tree is commands/guard.md.
