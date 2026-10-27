---
schema_version: 1
id: "iss-2610262035256103"
slug: "the-history-chapter-keeps-the-history"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 570 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:15"
remedy: "none (filed automatically)"
finding_ref: cfk-dca7a4690ca08b74
---

The history chapter keeps the history directory for ahoy's registry with no transcripts, but the build chapter calls the transcript store 'abcd's history store'

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:15` — "ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:500` — "transcript lands in abcd's history store, keyed on the repository's root"

11-history places transcripts under ~/.abcd.noindex/transcripts/< root-sha>/records/ and says ahoy's registry stays under ~/.abcd.noindex/history/ and holds no transcripts. 34-build says the runner's transcript lands in abcd's history store. 'History store' therefore names both the registry, which holds no transcripts, and the transcript store, so a builder could write transcripts into the wrong tree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 570.
