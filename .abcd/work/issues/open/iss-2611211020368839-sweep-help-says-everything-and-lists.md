---
schema_version: 1
id: "iss-2611211020368839"
slug: "sweep-help-says-everything-and-lists"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "the research leg of an /abcd:ideate run, 2026-11-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/sweep.go"
remedy: "List stashes and stale lock files in the default listing and clear neither (a stash is uncommitted work); take the word everything out of the help and out of the sweep chapter of the brief. Naming the kinds a sweep lists in its help, and the line naming what a sweep is known not to look at, are left to itd-2611211024360819, so that each is built once. Grounds: the reframing of the ideate verdict dcn-2611211021077161 (a cheap gap is fixed in the default run), loud-staging and enforcement-claims-are-facts."
---

The help of abcd sweep opens by saying it lists everything this repository has left lying around, and then names five kinds: worktrees, local branches, branches on origin, reservation branches and reports waiting in the inbox. Other things a repository leaves lying around are outside it, and nothing says so: git stashes, lock files under `.abcd/` (an untracked `.abcd/work/.decisions.lock` sat in the main checkout on 2026-11-20), dogfood builds and local-tier scratch. The code under `internal/core/sweep` names no stash, no lock file, no dogfood build and no scratch folder. A person who reads a clean sweep as a clean repository is told more than was checked. Read from the help and the code; the sweep was not run for this record.
