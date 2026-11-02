---
schema_version: 1
id: "iss-2610311501473953"
slug: "abcd-intent-ready-itd-39-reported-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "starting the product thinker's selected builds"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/ready.go"
remedy: "Gate intent ready now on the start checks no ruling covers, read from intent.StartChecksIn with no second row named steps: hold, blocked, the landed-steps refusal and the malformed-steps refusal, where spec.ReadSteps fails and Ready's stepsCheck reports it only as advisory. The claim_sections half waits on the product thinker in iss-2610312234315875 and is not built here."
---

abcd intent ready itd-39 reported the intent READY on 2026-10-31, listing scope_conditions as an advisory row ('no ## Scope Conditions section — the context claim is unrecorded'), and abcd build itd-39 then refused it at claim_sections for the same missing section. ReadyWithQuestions (internal/core/intent/ready.go) appends only the open-questions start check to the readiness verdict, so READY still does not mean the build will start, the defect iss-2610101610446956 fixed for open questions alone.
