---
schema_version: 1
id: "iss-2611041338455722"
slug: "trimming-the-go-build-cache-by-age-does"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "review of the Go cache trim, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/cachetrim.go"
remedy: "Give the trim a free-space or size target that removes oldest entries first until it is met, and name how it behaves when nothing old enough remains."
refines: [iss-2610290717018523]
resolution: "The Go build cache's trim now works to a size limit: abcd deletes the oldest entries until the rest fit, 100 GiB by default, set by load.go_cache_limit_gib in the person's own configuration and never by a repository, and a cache under its limit loses nothing to age. Every entry used in the last 3 hours stays whatever the limit; if those alone exceed it they stay and the answer says so. Besides prune it runs through a new verb, ahoy worktree trim-cache, which the build loop starts detached when it makes a lane, at most once an hour across the account, so a machine that goes days without a prune stays bounded; the lane does not wait, and the next lane says on stderr what a detached run could not do. The load gate's refusal now offers the verb. Prune's JSON drops unused_days for limit_bytes, keep_hours, size_bytes, left_bytes and over_limit. Decided in dcn-2611070751183761."
impact: additive
---

Trimming the Go build cache by age does not bound it: on 2026-11-03 the cache reached 323 GB with every entry used within two days, so a prune's trim frees nothing on a machine that busy, and the load gate still stops work on a full disk.
