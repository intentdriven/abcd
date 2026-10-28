---
schema_version: 1
id: "iss-2610262039093344"
slug: "itd-30-calls-the-fidelity-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 842 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-68b3380b01e91e54
---

itd-30 calls the fidelity agent `intent-fidelity-reviewer`, but the agent catalogue calls it `intent-auditor`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "and `intent-fidelity-reviewer` reads either dialect against the same delivered reality"
- End B: `.abcd/development/brief/05-internals/01-agents.md:35` — "`intent-auditor`, which judges a shipped intent's"

In itd-30, `intent-fidelity-reviewer` reads both intent formats against delivered reality. The agent catalogue gives the agent that judges a shipped intent's promise against delivered reality the name `intent-auditor`, and ships no agent under the other name. One agent has two names, so a builder following itd-30 would look for, or extend, a prompt that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 842.
