---
schema_version: 1
id: "iss-2610262034143915"
slug: "itd-13-s-dev-sync-keeps-a-last"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 480 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-13-scheduled-dev-sync.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-13]
finding_ref: cfk-91c9e6ad04b361a9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. No dev-sync verb or code exists (03-configuration section 2 and 08-abcd.md both say so), and the chapter's 'permanently' is bounded by 'until a state substrate exists'; a draft proposing that substrate does not contradict it."
---

itd-13's dev-sync keeps a last successful run timestamp shown by `abcd dev-sync status`; the board chapter says even in the dev-sync design no last-run timestamp is recorded

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-13-scheduled-dev-sync.md:31` — "Health check: `abcd dev-sync status` shows last successful run timestamp"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:585` — "is migration logic rather than a durable last-run timestamp, and no config field"

itd-13 scopes a health check, abcd dev-sync status, that shows the last successful run timestamp of the scheduled sync. The board chapter says that even in the dev-sync design the surface is migration logic rather than a durable last-run timestamp, and that no config field or store record captures when it last ran, so its staleness section stays a known-state line. The two disagree about whether the dev-sync design carries the substrate the board would read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 480.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. No dev-sync verb or code exists (03-configuration section 2 and 08-abcd.md both say so), and the chapter's 'permanently' is bounded by 'until a state substrate exists'; a draft proposing that substrate does not contradict it.
