---
schema_version: 1
id: "iss-2610262037533417"
slug: "itd-34-names-the-fidelity-auditing-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 756 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:683"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-3262440fe4b28694
---

itd-34 names the fidelity-auditing agent intent-fidelity-reviewer and gives it a third role; the build chapter names the same agent intent-auditor.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:683` — "on the lane whose landing closes the spec, an `intent-auditor`. The fidelity audit runs once"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "`intent-fidelity-reviewer` shape-classification role (third role)"

itd-34 calls the agent that performs single-document and cross-document fidelity audits intent-fidelity-reviewer, and adds a third, shape-classification role to it. 34-build.md hands the closing lane's fidelity audit to an intent-auditor. The one fidelity agent carries two names, so a reader following itd-34 looks for an agent the build never dispatches, or takes the two for different agents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 756.
