---
schema_version: 1
id: "iss-2611221058212814"
slug: "abcd-drain-s-sweep-refuses-to-prune-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "crash clean-up after the 22 Nov reboot"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/notes.go:185"
remedy: "Teach the notes archive abcd's own binary local-tier files, starting with reservation/generated: skip them, or copy them verbatim outside the redaction pass, instead of refusing the whole archive and keeping the worktree."
refines: [iss-2611181412255215]
---

abcd drain's sweep refuses to prune a worktree whose local tier holds .abcd/.work.local/reservation/generated, with 'the notes could not be archived: reservation/generated: not text, which the redaction pass cannot vouch for'. abcd writes that 160-byte file itself, the generated-set cache (internal/core/claim/generatedcache.go), and its fields are NUL-separated by design. The notes archive's only exemption for binary bytes is the built binary (isBuildProduct), so it refuses any worktree holding the cache; c-step3-1115 was kept this way during the 22 Nov clean-up.

This refines iss-2611181412255215, which records the same refusal from `abcd ahoy worktree prune`. In the sweep the cost is larger: failedReclaim (internal/core/sweep/apply.go) counts an archive refusal as a failed reclaim, not a keep, and returns an error, so each automatic sweep after a landing or at a drain's end reports a fault for every such worktree, and the worktree is never reclaimed.
