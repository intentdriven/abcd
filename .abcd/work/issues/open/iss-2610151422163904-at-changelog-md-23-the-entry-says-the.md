---
schema_version: 1
id: "iss-2610151422163904"
slug: "at-changelog-md-23-the-entry-says-the"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Correct CHANGELOG.md:23 so it states what the shipped binary does, as the finding describes."
---

At CHANGELOG.md:23: The entry says the build page 'lists all four refusals the pick commit's worktree stage can raise'. pickCommit (internal/core/implement/loop/pickcommit.go) raises more: a signing prompt or hook still waiting, an entry it could not write, a grounds-writer defect, and a hook refusal that shares the identity message. commands/build.md:128-130 does cover the hook and signing cases, so the page is fine; only the count of four is overstated. (v0.14.2 release docs-currency review, raised by opus.)
