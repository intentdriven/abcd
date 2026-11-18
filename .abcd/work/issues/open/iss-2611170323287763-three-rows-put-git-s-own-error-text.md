---
schema_version: 1
id: "iss-2611170323287763"
slug: "three-rows-put-git-s-own-error-text"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "Opus review pair of PR 1611's merge head"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/prune.go"
remedy: "Route the three texts through a cause-in-words helper like the causeInWords that PR 1611 adds to internal/core/ahoy (errno and exit status in words, git's message left out, naming the git command that shows the reason, its path quoted through shellquote.Single)"
---

Three rows put git's own error text into what the person reads. Two came in with aab27ce34, which reached main through PR 1609 on 16 Nov: ahoy worktree prune's Ignored.Why (internal/core/ahoy/worktree/prune.go:375) and a sweep row's Reason and Evidence Detail (internal/core/sweep/sweep.go:249). The third is older: prune's removal refusal, "git refused the removal: " followed by the first line of git's message (internal/core/ahoy/worktree/prune.go:360). Each is home-redacted and sanitised, but when git refuses a status read or a removal its bounded stderr comes through, and that can name a path outside the store worktree in home-relative form, such as the main checkout's .git/worktrees entry. PR 1611, open at this writing, makes the install's notes say why a step failed in words through a new causeInWords in internal/core/ahoy (iss-2610152204588951); once it lands these rows are the ones left that do not.

The sweep's automatic setting (internal/core/sweep/auto.go:76) is not one of them: its error comes from reading abcd's own layered config, which runs no git, and naming the file a misspelt key sits in is what it is for.
