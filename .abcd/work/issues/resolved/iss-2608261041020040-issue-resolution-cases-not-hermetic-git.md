---
schema_version: 1
id: "iss-2608261041020040"
slug: "issue-resolution-cases-not-hermetic-git"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: "scripts/check-issue-resolution-cases.sh"
resolution: "Hoisted the check-attribution-cases.sh hermetic git-env scrub (unset GIT_DIR family + GIT_CONFIG_GLOBAL=/dev/null) before the first git call."
impact: fix
resolved_by:
  commit: "47efe8e82d3d45d3f8f968d55c57cc8655dcb350"
---

check-issue-resolution-cases.sh runs its scratch-repo git commands under the inherited environment, so an ambient GIT_DIR or global core.hooksPath corrupts the caller's repo or breaks the gate. The harness builds ~60 'git -C $d' fixtures but omits the hermetic-env scrub its sibling check-attribution-cases.sh carries (iss-28, iss-313, resolved four days before this script was authored): an inherited absolute GIT_DIR overrides -C and redirects the fixture commits and config onto the ambient repository while every case still prints ok (green-while-corrupting, reachable on a direct make preflight under a git alias / rebase -x / bisect run); and an inherited GIT_CONFIG_GLOBAL / core.hooksPath — which this repo's .githooks/pre-push documents the maintainer uses — fires the developer's global pre-commit hook inside the scratch repos and turns make lint-issues red with no gate diagnostic. Fix: hoist the same unset GIT_DIR family + export GIT_CONFIG_GLOBAL=/dev/null scrub before the first git call.