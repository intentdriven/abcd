---
schema_version: 1
id: "iss-2610262039302287"
slug: "itd-50-names-the-fidelity-auditor"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 865 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-0fa333c3ac3cefaa
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031563379. Same End A; the agent catalog (05-internals/01-agents.md) lists only intent-auditor."
duplicates: [iss-2610262031563379]
---

itd-50 names the fidelity auditor `intent-fidelity-reviewer`; the agent catalog names the same agent `intent-auditor`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict and the voyage moves on"
- End B: `.abcd/development/brief/05-internals/01-agents.md:35` — "`intent-auditor`, which judges a shipped intent's promise against delivered reality"

itd-50's press release says the `intent-fidelity-reviewer` records the NOT_MET verdict on a shipped intent. The agent catalog says the agent that judges a shipped intent's promise against delivered reality is `intent-auditor`, and it lists no agent under the other name. So one agent has two names, and a reader of the planned intent looks for an agent prompt that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 865.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031563379. Same End A; the agent catalog (05-internals/01-agents.md) lists only intent-auditor.
