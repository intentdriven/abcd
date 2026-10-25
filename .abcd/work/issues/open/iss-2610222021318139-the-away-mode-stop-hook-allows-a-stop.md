---
schema_version: 1
id: "iss-2610222021318139"
slug: "the-away-mode-stop-hook-allows-a-stop"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "abcd-50 review pair of the RootCommit adaptation on PR #1101"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_stop.go"
remedy: "On a Resolve error that is not ErrNoRootCommit, keep allowing the stop but print one stderr line naming git's failure (and record it in the away record where the store is reachable); fix the comment at hook_stop.go:139 and the matching drop in owedread.go:427."
---

The away-mode Stop hook allows a stop silently when git does not answer for the repository's root commit. Resolve now returns ErrNoRootAnswer or ErrGitTimedOut (wrapped) instead of ErrNoRootCommit, and hook_stop.go returns nil on any Resolve error under a comment that says the folder is not a checkout abcd can key a store on. On a loaded machine where the rev-list is killed, an away session stops with nothing on stderr and nothing in the away record, though the hook's help says every allowed stop is explained. owedread.go drops the same error the same way, so recorded branches leave the stop check without a note.
