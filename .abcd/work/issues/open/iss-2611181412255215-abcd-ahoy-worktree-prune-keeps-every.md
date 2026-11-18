---
schema_version: 1
id: "iss-2611181412255215"
slug: "abcd-ahoy-worktree-prune-keeps-every"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "worktree prune dry run after #1633 merged"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/notes.go"
remedy: "Skip the generated-set cache in the notes archive the way the built binary is skipped, by its name together with its 'abcd generated set' header, so a merged, clean worktree holding it is reclaimed while text a person writes at that path is still archived."
---

abcd ahoy worktree prune keeps a merged, clean worktree whose local tier holds the generated-set cache, .abcd/.work.local/reservation/generated (internal/core/claim/generatedcache.go). The cache is written in any worktree where a reservation view runs (peers, reserve, build, drain, and the load gate's renewal pass, which every preflight's implement load --wait makes) while a live, footprinted reservation is on origin, whoever holds it; the loop's conflict check (claim.GeneratedFiles, in sync.go and drain_finish.go) writes it with no reservation needed. So a worktree that never reserved anything gets it too. Its fields are NUL-separated, and the notes archiver refuses a file that is not text (internal/core/ahoy/worktree/notes.go, 'not text, which the redaction pass cannot vouch for'), so the row reads kept rather than reclaimed. The other reservation files, fetched-at, copy-key and renew.lock, are text or empty and archive fine. On 18 Nov, 52 worktrees in the store held the cache, 24 of them without a copy-key, which every reservation mints, so those 24 never reserved; and most recent merged worktrees were kept for this reason alone.
