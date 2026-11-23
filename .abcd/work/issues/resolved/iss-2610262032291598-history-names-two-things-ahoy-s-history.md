---
schema_version: 1
id: "iss-2610262032291598"
slug: "history-names-two-things-ahoy-s-history"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 275 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:1204"
remedy: "In 01-ahoy.md (the uninstall paragraph near line 1340) replace 'the history store' with 'the history registry and the transcript store', naming both as left intact."
finding_ref: cfk-d87fecf34a711c14
resolution: "The uninstall paragraph of the ahoy brief chapter now says an uninstall leaves the entire .abcd/ namespace, the history registry and the transcript store intact, in place of the ambiguous 'the history store'."
impact: internal
resolved_by:
  commit: "297137b14ffee13042928811c13ae24e744e71b8"
---

'history' names two things: ahoy's 'history store' could be the registry directory ~/.abcd.noindex/history/ or the history verb's transcript store

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:1204` — "It leaves the entire `.abcd/` namespace and the history store intact."
- End B: `.abcd/development/brief/04-surfaces/11-history.md:15` — "ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts."

01-ahoy.md says uninstall leaves the history store intact, and its tree gives ~/.abcd.noindex/history/ to the registry only. 11-history.md says the history verb's store is the transcript store under ~/.abcd.noindex/transcripts/< root-sha>/records/, and that ahoy's registry under ~/.abcd.noindex/history/ holds no transcripts. So 'history' names both the registry directory and the transcript store, and ahoy's phrase does not say whether uninstall keeps the transcript corpus, the registry, or both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 275.

## Grounds

- pursued: ahoy.Uninstall in internal/core/ahoy/apply.go and every function it calls write only the marker files, README.md and .abcd/config.json, the harness settings and its backups, the tool copies and tools.json, the shell PATH line and its record, and the PATH entry and path-entry; none reaches historyRoot, historyDir or the registry writers in store.go, or the history package. A call from that path into either store would show the sentence wrong.
