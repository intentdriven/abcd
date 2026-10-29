---
schema_version: 1
id: "iss-2610262042403190"
slug: "both-itd-106-and-itd-118-have-abcd"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1085 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-106, itd-118]
finding_ref: cfk-7f48956d835b9b37
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032556158. The same delete-branch-on-merge setting that iss-2610262032556158 asks a ruling on; ownership between itd-106 and itd-118 only matters if that ruling lets abcd write it."
duplicates: [iss-2610262032556158]
---

Both itd-106 and itd-118 have abcd switch on deletion of merged PR branches

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:24` — "switches on auto-deletion of merged branches"
- End B: `.abcd/development/intents/drafts/itd-118-merged-work-leaves-no-residue-abcd.md:12` — "abcd-managed repos delete a PR branch on merge"

itd-106's repository-settings reconciliation switches on auto-deletion of merged branches. itd-118's title has abcd-managed repos delete a PR branch on merge. The same setting is claimed by two drafts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1085.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032556158. The same delete-branch-on-merge setting that iss-2610262032556158 asks a ruling on; ownership between itd-106 and itd-118 only matters if that ruling lets abcd write it.
