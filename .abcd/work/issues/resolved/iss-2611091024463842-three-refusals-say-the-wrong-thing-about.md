---
schema_version: 1
id: "iss-2611091024463842"
slug: "three-refusals-say-the-wrong-thing-about"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Map a worktree's admin dir to its checkout path through the admin dir's gitdir file, word an update-refs hold as a branch a stopped rebase will rewrite, and have the secret scan name a binary that could not be executed (exit 126 or a permission error) with the temp dir and a TMPDIR that allows execution, keeping exit 2."
resolution: "The branch refusals now name a worktree's checkout path, read from the admin dir's gitdir file, and say a stopped rebase will rewrite a branch only its update-refs list names; the secret scan says when the verified gitleaks could not be executed, naming the temp dir and the TMPDIR to set."
impact: fix
---

Three refusals say the wrong thing about why they refuse, though each refusal is right. The lane-branch delete refusal and the sweep's branch delete proof name git's admin dir for a linked worktree (the common dir's worktrees/<id>, whose id can differ from the checkout folder) instead of the worktree's checkout path. Both also call a branch held only by a stopped rebase's update-refs list checked out where a rebase or bisect stopped on it, when the rebase will rewrite it and no checkout stands on it. And the preflight secret scan, when the verified gitleaks cannot be executed because the temp dir is mounted noexec, says the archive unpacked a gitleaks that reports nothing, hiding the real cause.
