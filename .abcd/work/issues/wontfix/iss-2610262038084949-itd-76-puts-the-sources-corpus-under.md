---
schema_version: 1
id: "iss-2610262038084949"
slug: "itd-76-puts-the-sources-corpus-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 774 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-69836f6a0ccaecee
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 29-report states the current machine-scoped layout; itd-76's older home name predates the rename."
---

itd-76 puts the sources corpus under `~/.abcd/sources/`, abcd's user-level home; 29-report puts the sources store beside the inbox in the machine-scoped store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"
- End B: `.abcd/development/brief/04-surfaces/29-report.md:95` — "The inbox is a machine-scoped store beside the history, transcript, worktree, sources, labs and run stores."

itd-76 names `~/.abcd/` as abcd's user-level home and the default location of the sources corpus. 29-report states that the inbox, which it places at `~/.abcd.noindex/inbox/`, is a machine-scoped store beside the sources store among the others, which puts the sources store under `~/.abcd.noindex/`. The two give the same store different homes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 774.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 29-report states the current machine-scoped layout; itd-76's older home name predates the rename.
