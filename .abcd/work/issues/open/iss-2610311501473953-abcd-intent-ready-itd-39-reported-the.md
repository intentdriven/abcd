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
remedy: "Waits on the product thinker's ruling: making intent ready gate on the build's record-only start checks reverses their ruling on iss-2609091009111294, which kept the mechanism, scope-condition and grounds rows advisory. The two answers are to reverse it, so that READY means the build's record checks pass, or to keep READY advisory and have intent ready name the build refusal it predicts (for example 'build will refuse at claim_sections'). Either way, reuse intent.StartChecksIn and add no second row named steps."
---

abcd intent ready itd-39 reported the intent READY on 2026-10-31, listing scope_conditions as an advisory row ('no ## Scope Conditions section — the context claim is unrecorded'), and abcd build itd-39 then refused it at claim_sections for the same missing section. ReadyWithQuestions (internal/core/intent/ready.go) appends only the open-questions start check to the readiness verdict, so READY still does not mean the build will start, the defect iss-2610101610446956 fixed for open questions alone.
