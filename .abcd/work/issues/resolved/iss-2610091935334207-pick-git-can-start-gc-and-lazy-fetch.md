---
schema_version: 1
id: "iss-2610091935334207"
slug: "pick-git-can-start-gc-and-lazy-fetch"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane W sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/pickcommit.go"
remedy: "Prepend -c gc.auto=0 -c maintenance.auto=false to pickGit and add GIT_NO_LAZY_FETCH=1 to its environment, proved by a test that a low gc.auto with loose objects starts no gc, watched fail first."
resolution: "pickGit pins gc.auto=0 and maintenance.auto=false on the command line and runs with GIT_NO_LAZY_FETCH=1, so the pick commit and the sync merge start no automatic gc or maintenance (and so no gc.recentObjectsHook) and a merge in a partial clone fetches nothing."
impact: fix
---

The implement loop's pickGit (pick commit, sync merge) runs with ScrubbedEnv, so it lacks gc.auto=0, maintenance.auto=false and GIT_NO_LAZY_FETCH: a pick commit can trigger gc --auto (and gc.recentObjectsHook on git 2.42+), and a merge in a partial clone can lazy-fetch through a promisor transport. Found by the security-drain-2026-10-09 lane W sweep; kept uncommitted until its fix lands.
