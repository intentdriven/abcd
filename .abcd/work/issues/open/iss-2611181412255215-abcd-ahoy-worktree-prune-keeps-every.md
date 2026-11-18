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
remedy: "Treat the reservation cache files abcd writes (reservation/generated, fetched-at, copy-key, renew.lock) as abcd's own state rather than notes: skip them in the archive as build products are skipped, or regenerate them, so a merged, clean worktree that reserved is reclaimed."
---

abcd ahoy worktree prune keeps every worktree that ever reserved a record, because the reservation cache abcd itself writes in the worktree's local tier, .abcd/.work.local/reservation/generated, is NUL-separated binary (the generated-set cache, internal/core/claim/generatedcache.go), and the notes archiver refuses a file that is not text (internal/core/ahoy/worktree/notes.go, 'not text, which the redaction pass cannot vouch for'). The row reads kept, not reclaimed, so merged and clean worktrees pile up: on 18 Nov the store held 121, and most recent merged ones were kept for this reason alone.
