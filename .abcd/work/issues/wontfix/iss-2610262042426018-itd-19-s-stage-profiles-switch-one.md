---
schema_version: 1
id: "iss-2610262042426018"
slug: "itd-19-s-stage-profiles-switch-one"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1087 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-19, itd-2609170822093401]
finding_ref: cfk-f9cc90411d1aec0d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037211310. Same draft edit as iss-2610262037211310, set against a different statement of the same current state."
duplicates: [iss-2610262037211310]
---

itd-19's stage profiles switch one oracle value per stage; itd-2609170822093401 says that repo-wide value is read by nothing and routes per agent tier instead

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:33` — "Validation: visibility=public-prep, dev_sync=read-only, scan.deep=true, oracle=rp+codex (max scrutiny)"
- End B: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26` — "backend is one repo-wide value that nothing reads"

itd-19 tightens scrutiny by setting a single oracle backend (in-session, rp, rp+codex) per development stage in the repository config. itd-2609170822093401 records that the repo-wide oracle value is read by nothing and replaces it with per-agent rows from a closed tier set. A stage profile written to the unread value would change nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1087.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037211310. Same draft edit as iss-2610262037211310, set against a different statement of the same current state.
