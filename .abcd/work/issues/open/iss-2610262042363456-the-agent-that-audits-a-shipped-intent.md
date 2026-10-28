---
schema_version: 1
id: "iss-2610262042363456"
slug: "the-agent-that-audits-a-shipped-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1080 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-25]
finding_ref: cfk-606df9014334d862
---

The agent that audits a shipped intent is intent-auditor in itd-1 and intent-fidelity-reviewer in itd-25

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17` — "emitted by the `intent-auditor` agent's single-document role"
- End B: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:46` — "`intent-fidelity-reviewer` extension"

itd-1 names the agent whose single-document role judges a shipped intent's criteria intent-auditor. itd-25 extends the same shipping-time check under the name intent-fidelity-reviewer. Two names for one agent leave an implementer unsure which prompt the extension belongs in.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1080.
