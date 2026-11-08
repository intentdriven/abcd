---
schema_version: 1
id: "iss-2610121528179959"
slug: "abcd-has-no-shared-handover-place-for"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "multi-session close-down before a usage-limit reset, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "the local tier's handover convention (NEXT.md)"
remedy: "Give abcd a declared handover store, for example a per-checkout or machine store keyed on the root commit with one file per session plus a derived index, written by a verb such as abcd handover write that fills branches, worktrees, PRs, queue tickets and records from git and the ledger, and read by session start so a fresh session sees every open handover; relate it to itd-2610050548126044 (refresh reminder) and the worktree store (itd-2609091014076309)"
---

abcd has no shared handover place for concurrent sessions. On 2026-10-12 seven sessions had to close at once (usage limit) and resume days later as fresh sessions; the only convention, .abcd/.work.local/NEXT.md, holds one session's note and was stale from 2026-10-07, each session's scratchpad (lane briefs, cloud prompts, peer notes, security plans) is deleted when the session closes, and the coordinator improvised a per-session folder and template (.abcd/.work.local/handover-2026-10-14/) by message. Nothing lists which branches, worktrees, PRs, records on unmerged branches, private local-only branches and memory files each session leaves behind, so work can be orphaned across a closure.

## Checked at the default branch 2026-10-28

Still true: abcd has no handover verb. It overlaps in part with iss-2610130501353955, which asks abcd to write the cross-session handover index itself; this record is the shared place a handover lives, and that one the index of them.
