---
schema_version: 1
id: "iss-313"
slug: "check-attribution-cases-sh-scratch-repos"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "scripts/check-attribution-cases.sh"
resolution: "check-attribution-cases.sh unsets the ambient git-location vars before the first git call."
impact: internal
---

check-attribution-cases.sh scratch repos are not hermetic so an inherited GIT_DIR makes the corpus report green while rewriting the ambient repository history
## Evidence
`scripts/check-attribution-cases.sh:495` — bare `git init -q "$repo"`; every later call is `git -C "$repo" ...` with no env scrub. An absolute inherited `GIT_DIR` overrides `-C`. Reproduced in a sandbox victim repo: `GIT_DIR=/victim/.git bash scripts/check-attribution-cases.sh` → `74 passed, 0 failed, exit 0` while the victim's `refs/heads/main` was moved to a bogus empty commit and 13 scratch commits + 12 reset --hard entries were written into its object store. A live instance of the shell shim iss-28 (severity major) explicitly deferred ("Go-only per maintainer Option A; cross-language scaffolding deferred").

## Adversarial verdict: CONFIRMED, understated (minor→moderate)
Not merely "confusing diagnosis" (the gpgsign scenario is loud): there is a real green-while-corrupting path via inherited GIT_DIR, absolute form as exported in hook / rebase -x / bisect contexts. Not exposed in CI (clean runner) but exposed to local `make check-attribution` (not a preflight prereq; pre-push unsets GIT_DIR but never calls it). Fix: hoist the existing convention — `unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS; export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1` at the top, and check the commit status in commit_as. iss-28 currently reads closed; this is its live shell instance.
