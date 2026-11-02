---
schema_version: 1
id: "iss-2611012309408370"
slug: "a-receipt-s-model-is-copied-into-the-run"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "landing run-2611012100266247 on the API route, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Record the model a runner ran on the role's route as soon as the runner reports it (its stored transcript already holds it), so a lane whose receipt never arrives or omits the model still knows it; refuse a receipt that reports no model at implement receipt, where the agent can still add it, rather than at land; and give a lane at land a sanctioned way to correct its receipts' reported model and resolves (a re-read of the receipt file it points at, or a verb), recorded in the run log, so no one edits the run's state by hand."
resolution: "The brief asks for the model and the receipt verifier refuses a receipt without one; a lane whose receipt verified without a model takes its runner's reported model, or else its corrected receipt file's, at the landing, with the run log naming the source."
impact: fix
resolved_by:
  commit: "da2ae9c40"
---

A receipt's model is copied into the run's state when the receipt verifies and is never read again, and the land stage refuses a records commit whose receipts report no model, so a receipt verified without one leaves the lane with no way to land: on 2026-11-01 a host-written receipt for run-2611012100266247 omitted the model, the lane passed three review rounds, and the landing refused; correcting the receipt file changed nothing, implement step --restart refuses a lane at land, and the remedy's fix round cannot be reached, so the model had to be written into state.json by hand (internal/core/implement/loop/land.go, assistedByTrailers).

## Grounds

- pursued: no lane is stuck at land for want of a model; shown wrong if a lane whose receipt file or route reports a model is still refused at land, or one needs its state edited by hand
