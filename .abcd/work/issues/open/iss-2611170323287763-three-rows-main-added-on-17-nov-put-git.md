---
schema_version: 1
id: "iss-2611170323287763"
slug: "three-rows-main-added-on-17-nov-put-git"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "Opus review pair of PR 1611's merge head"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/prune.go"
remedy: "Route the three Why and Reason texts through a cause-in-words helper like ahoy's causeInWords (errno and exit status in words, git's message left out, with the git command that shows the reason), keeping prune.go's existing 'git refused the removal' row in the same change"
---

Three rows main added on 17 Nov put git's own error text into what the person reads: ahoy worktree prune's Ignored.Why (internal/core/ahoy/worktree/prune.go:375, from aab27ce34), a sweep row's Reason and Evidence Detail (internal/core/sweep/sweep.go:249, the same commit), and the sweep automatic setting's Why (internal/core/sweep/auto.go:76, from dddab8bef). Each is home-redacted and sanitised, but when git refuses a status read its bounded stderr comes through, and that can name a path outside the store worktree in home-relative form, such as the main checkout's .git/worktrees entry. The install's notes say why a step failed in words through causeInWords (iss-2610152204588951, iss-2611162118486104); these rows do not.
