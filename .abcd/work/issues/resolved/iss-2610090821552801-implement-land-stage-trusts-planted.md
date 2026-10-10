---
schema_version: 1
id: "iss-2610090821552801"
slug: "implement-land-stage-trusts-planted"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-qfpj-v8c5-c4fw, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Apply briefStage's check on the land stage, refusing before `EnsureRealDirAll` when the worktree is not the path `laneWorktree` derives for that run and lane; prove it with a loop test (watched fail first) that a state whose worktree is not the derived path leaves the other directory untouched, a derived-path land still creates the local tier when the landing closes, and a missing branch or landing still returns before any directory is created; sweep siblings (every stage that reads a path from the run state)."
resolution: "the brief stage's derived-worktree check is now one helper (loopWorktree) that the land, implement and validate stages and both worktree removals (hold discard, hand-back discard) apply, so a state file naming a worktree path the loop did not derive is refused before anything is made, run or removed there"
impact: fix
---

The implement loop's land stage accepts any non-empty worktree path from the run state file, so a planted state makes `abcd implement step` create `<that path>/.abcd/.work.local` (mode 0700) in a directory the state names.

Private security advisory GHSA-qfpj-v8c5-c4fw (draft, severity low). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `briefStage` refuses a worktree that is not the path `laneWorktree` derives (internal/core/implement/loop/brief.go:104-105). `landStage` only checks the string is non-empty and skips `syncLane` once a landing is recorded (internal/core/implement/loop/land.go:131, :139). `landRecords` uses that worktree when `landing.closes` is true (internal/core/implement/loop/land.go:237) and calls `fsutil.EnsureRealDirAll` there (internal/core/implement/loop/land.go:269). The state file is gitignored, so it arrives in a zip, not a clone. The result is an empty 0700 directory tree; no file body is written and no hook in the other repository runs.

Reproduction: on git 2.39.5, hand-write the state under `.abcd/.work.local/run/` for a run whose one lane is at stage land, with full base and head SHAs, a real branch at that tip, `landing.closes` true and `worktree` a relative path to another existing git repository; run `abcd implement step`. `<other>/.abcd/.work.local` exists mode 0700, then `intent.Reconcile` fails on the absent spec. A missing branch or a missing landing record returns before the mkdir.
