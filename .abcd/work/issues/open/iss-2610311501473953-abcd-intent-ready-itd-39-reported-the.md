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
remedy: "Have intent ready append every record-only start check the build runs (claim_sections, hold, blocked, steps where gating) as gating rows, read from intent.StartChecksIn, so READY means the build's record checks pass."
---

abcd intent ready itd-39 reported the intent READY on 2026-10-31, listing scope_conditions as an advisory row ('no ## Scope Conditions section — the context claim is unrecorded'), and abcd build itd-39 then refused it at claim_sections for the same missing section. ReadyWithQuestions (internal/core/intent/ready.go) appends only the open-questions start check to the readiness verdict, so READY still does not mean the build will start, the defect iss-2610101610446956 fixed for open questions alone.
