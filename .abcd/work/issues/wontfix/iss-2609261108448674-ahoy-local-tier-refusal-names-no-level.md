---
schema_version: 1
id: "iss-2609261108448674"
slug: "ahoy-local-tier-refusal-names-no-level"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cutfix item 2 sibling sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/statusline_apply.go"
wontfix_reason: "Unreachable as described: Install refuses a repository whose .abcd is a symlink or a non-directory before any step runs (abcdDirHazard, internal/core/ahoy/apply.go), naming .abcd, so the only level stepLocalTier's EnsureRealDirAll can refuse is .abcd/.work.local itself, which its message names. A probe test with .abcd symlinked got the upfront refusal, not this one."
---

ahoy install's local-tier refusal names no level: stepLocalTier (internal/core/ahoy/statusline_apply.go) maps fsutil.EnsureRealDirAll's not-a-real-directory error to 'refused to create .abcd/.work.local/: something that is not a real directory stands at that path' whichever level was refused, and drops the *os.PathError path. With the repository's .abcd a symlink, the user is sent to .abcd/.work.local, which does not exist, rather than to .abcd. The refusal should name the refused level, repository-relative. The sibling of the inbox refusal's unnamed level.

## Grounds

- declined: Unreachable as described: Install refuses a repository whose .abcd is a symlink or a non-directory before any step runs (abcdDirHazard, internal/core/ahoy/apply.go), naming .abcd, so the only level stepLocalTier's EnsureRealDirAll can refuse is .abcd/.work.local itself, which its message names. A probe test with .abcd symlinked got the upfront refusal, not this one.
