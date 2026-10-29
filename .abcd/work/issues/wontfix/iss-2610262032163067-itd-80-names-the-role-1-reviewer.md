---
schema_version: 1
id: "iss-2610262032163067"
slug: "itd-80-names-the-role-1-reviewer"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 246 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-11afbca3dccf8837
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The agent directory holds intent-auditor and no intent-fidelity-reviewer; itd-123 names the rename."
---

itd-80 names the Role 1 reviewer `intent-fidelity-reviewer`; the brief's press release says Role 1 belongs to `intent-auditor`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:35` — "A host-delegated `intent-fidelity-reviewer` markdown agent"
- End B: `.abcd/development/brief/01-product/01-press-release.md:26` — "After shipping, `/abcd:intent audit` (Role 1 of `intent-auditor`) reviews delivered reality against the press release"

itd-80 introduces 'a host-delegated `intent-fidelity-reviewer` markdown agent' as the single-document (Role 1) reviewer. The brief's press release says `/abcd:intent audit` is Role 1 of `intent-auditor`. The same agent appears under two names in the record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 246.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The agent directory holds intent-auditor and no intent-fidelity-reviewer; itd-123 names the rename.
