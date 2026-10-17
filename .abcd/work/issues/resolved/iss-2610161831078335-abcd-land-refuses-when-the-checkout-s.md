---
schema_version: 1
id: "iss-2610161831078335"
slug: "abcd-land-refuses-when-the-checkout-s"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd land post-release review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Ask origin for its default branch with git ls-remote --symref through a new gitutil.RemoteHeadBranch (check-ref-format on the answer, an error when origin cannot be asked, no fetch and no write); when it agrees with the forge's defaultBranchRef but the local origin/HEAD is stale or missing, land goes on with that default and fetches it into refs/remotes/origin/<branch> itself, never writing origin/HEAD; refuse only when the forge and origin disagree or origin cannot be asked; drop the needless set-head hint where the branch is the default."
resolution: "abcd land now asks origin for its default branch (gitutil.RemoteHeadBranch, git ls-remote --symref, checked with check-ref-format, no fetch and no write) and goes on with the default the forge and origin agree on when the local origin/HEAD is stale or missing, reporting it in an origin step, fetching the branch when it is not tracked and never writing origin/HEAD; a forge and origin that disagree, or an origin that cannot be asked, are refused; the default-branch refusal no longer carries a set-head hint"
impact: fix
---

abcd land refuses when the checkout's local origin/HEAD is stale or missing and tells the person to run git fetch and git remote set-head themselves (land.go, the forge-default check), though abcd can settle the gap by asking origin which branch is its default. The product thinker has ruled twice against a refusal that hands a person a manual step abcd can resolve itself (adr-2609301720596683, adr-2610030720195401). The refusal for the branch being the default also appends a set-head hint that has nothing to do with that case. Found by a peer session and confirmed on main, 2026-10-16.
