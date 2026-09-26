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
---

ahoy install falsely refuses the local tier of a checkout reached through a symlinked path, and its refusal names no level. stepLocalTier (internal/core/ahoy/statusline_apply.go) proves .abcd/.work.local with fsutil.EnsureRealDirAll from a.cwd, and a.cwd is filepath.Abs(os.Getwd()) (internal/core/ahoy/apply.go, internal/surface/cli/cli.go), the shell's logical working directory. EnsureRealDirAll proves its BASE real before any level below it, so `cd ~/proj` where `~/proj -> ~/src/proj` makes `abcd ahoy install --yes --adopt` exit 0 with "note: refused to create .abcd/.work.local/: something that is not a real directory (a symlink, or a file) stands at that path" while no symlink stands at or below the checkout's .abcd, and while .abcd/.work.local itself stands as a real directory (the banlist step created it earlier in the same run). The refused level is the checkout path the user entered through, which the message does not name; it sends the user to .abcd/.work.local instead. abcdDirHazard (internal/core/ahoy/apply.go) Lstats only <cwd>/.abcd, so it neither catches nor explains this. Reproduced on a scratch repository with a real .abcd, entered through a symlink, under a temp HOME. The fix resolves the checkout root through its symlinks before the proof (the way internal/surface/cli/cli.go's strayStoreNotes resolves cwd), so only a symlink at or below the checkout's .abcd is refused, and names the refused level repo-relative. The sibling of the inbox refusal's unnamed level.

## Grounds

- declined: Unreachable as described: Install refuses a repository whose .abcd is a symlink or a non-directory before any step runs (abcdDirHazard, internal/core/ahoy/apply.go), naming .abcd, so the only level stepLocalTier's EnsureRealDirAll can refuse is .abcd/.work.local itself, which its message names. A probe test with .abcd symlinked got the upfront refusal, not this one.
