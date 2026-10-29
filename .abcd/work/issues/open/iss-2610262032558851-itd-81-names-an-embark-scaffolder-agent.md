---
schema_version: 1
id: "iss-2610262032558851"
slug: "itd-81-names-an-embark-scaffolder-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 336 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:126"
remedy: "In itd-81's non-judging agents bullet (lines 126-127) replace the examples chat-distiller and embark-scaffolder, which do not exist in agents/, with shipped non-judging agents such as press-release-composer or principle-distiller"
related_intents: [itd-81]
finding_ref: cfk-482ecd2873a56f29
---

itd-81 names an embark-scaffolder agent; embark says no scaffolder and no model sit in its write path

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:126` — "Agents that transform or scaffold rather than render a verdict (`chat-distiller`, `embark-scaffolder`) remain under itd-5 only."
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:101` — "No interactive scaffolder and no model sit in the write path."

itd-81 exempts `embark-scaffolder` from calibration as an agent that scaffolds rather than judges, which assumes embark runs such an agent. The embark chapter says embark is a deterministic Go run with no interactive scaffolder and no model in its write path. The discipline classifies an agent that the surface it is named for does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 336.
