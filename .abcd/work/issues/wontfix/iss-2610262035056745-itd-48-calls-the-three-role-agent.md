---
schema_version: 1
id: "iss-2610262035056745"
slug: "itd-48-calls-the-three-role-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 546 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:54"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-3ddcc62a3cf0003d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032120773. Same A line and root cause; only the B document differs."
duplicates: [iss-2610262032120773]
---

itd-48 calls the three-role agent intent-fidelity-reviewer (agents/intent-fidelity-reviewer.md); the intent chapter calls the same agent intent-auditor

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:54` — "Extend `agents/intent-fidelity-reviewer.md` with a Role 2 prompt section"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:532` — "`intent-auditor` is a single agent in the catalog (per `05-internals/01-agents.md`) that owns three roles."

Both documents describe one agent that owns the per-intent fidelity role, the cross-document role and the kind-classification role. itd-48 names it intent-fidelity-reviewer and its file agents/intent-fidelity-reviewer.md. The intent chapter names it intent-auditor. That is two names for one concept, and the file path itd-48 asks to extend does not match the agent the chapter describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 546.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032120773. Same A line and root cause; only the B document differs.
