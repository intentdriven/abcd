---
schema_version: 1
id: "iss-2611031337115764"
slug: "a-finished-agent-is-stranded-when-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Before the implement stage asks for a fresh agent, it reads the receipt path it awaits and verifies a receipt already there (as implement receipt would), so a refusal between an agent's finish and its collection loses nothing; a test refuses the collecting step once and then steps the run to validate without a new launch."
related_intents: [itd-2611031342011261]
---

A finished agent is stranded when the step that would collect it is refused: on run-2611031131030791 (2026-11-03) the routed implementer ran 1h37m, committed and wrote its receipt at 13:32, but the next implement step was refused on the run-state schema trap (iss-2611020719494343), and after the state was put right every further step asked the host to start a fresh implementer, though the receipt sat at the path the stage awaited. Only handing it in by hand with implement receipt moved the run on; an unattended run would have launched a second implementer and paid for the work twice.
