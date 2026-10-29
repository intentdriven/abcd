---
schema_version: 1
id: "iss-2610262034520240"
slug: "itd-34-calls-the-three-role-fidelity"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 530 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-deb52e454f961e73
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ holds intent-auditor only, and itd-123 states the rename and its changelog line, so the old name in itd-34 is a historical spelling."
---

itd-34 calls the three-role fidelity agent intent-fidelity-reviewer; the intent surface page calls the same agent intent-auditor.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "`intent-fidelity-reviewer` shape-classification role (third role)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:532` — "`intent-auditor` is a single agent in the catalog"

itd-34 attaches the shape-classification role as the third role of `intent-fidelity-reviewer`, the agent it says performs the single-document and cross-document fidelity audits. 05-intent.md says `intent-auditor` is the single agent in the catalog that owns those three roles. The old name survives in a shipped record and names no agent the catalogue holds, so a reader looking it up finds nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 530.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ holds intent-auditor only, and itd-123 states the rename and its changelog line, so the old name in itd-34 is a historical spelling.
