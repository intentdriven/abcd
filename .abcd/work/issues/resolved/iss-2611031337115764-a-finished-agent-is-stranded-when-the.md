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
resolution: "A later step reads the receipt of a routed implementer at the implement stage, where the run's state shows a runner was started for that await, and hands in what verifies through the verifier and lock implement receipt uses, so a refusal between that agent's finish and its collection loses nothing and no second runner is started over it. A receipt that does not verify is left in place and the step says why. Every other await keeps the path it had: a host hands its agent's receipt back, a reviewer's return and a fix round's receipt come back from their runner or by hand, and a hand-out starts its agent."
impact: fix
resolved_by:
  commit: "ada307e2db836473671d65f7f61f274f8f9d005a"
---

A finished agent is stranded when the step that would collect it is refused: on run-2611031131030791 (2026-11-03) the routed implementer ran 1h37m, committed and wrote its receipt at 13:32, but the next implement step was refused on the run-state schema trap (iss-2611020719494343), and after the state was put right every further step asked the host to start a fresh implementer, though the receipt sat at the path the stage awaited. Only handing it in by hand with implement receipt moved the run on; an unattended run would have launched a second implementer and paid for the work twice.
