---
schema_version: 1
id: "iss-2610252214267121"
slug: "the-doc-fidelity-gate-checks-only-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md"
remedy: "Rule which file is the SSOT for verb behaviour. Either extend the gate to commands/<verb>.md and cut the chapter to rationale and the reasons for refusals, or keep the chapter and make the commands page point at it."
---

The doc-fidelity gate checks only the brief chapter (04-surfaces/10-docs.md:66-82, commands/docs.md:92). So 05-intent.md is the gated copy of intent-verb behaviour. commands/intent.md, the page the host runs and the more current one (chunking, finding keys), is ungated and carries a false line (:232).
