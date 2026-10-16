---
schema_version: 1
id: "iss-2610161734179773"
slug: "interview-narrates-question-drafter"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "PT watching a peer session's planning interview, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question/asking.go"
remedy: "Add to the asking rule that sends questions through abcd:question-drafter (internal/core/question/asking.go, regenerated into the GRILL domain and the drafter page by make asking-sync) that a refused or over-limit draft is fixed and redrafted silently: the agent says nothing to the person about the drafter's refusals or retries, and the next thing they see is the question itself."
resolution: "The drafter rule in the asking rules now says to redraft silently after a refusal or an over-limit draft, so the person sees only the finished question."
impact: fix
resolved_by:
  commit: "5ad7f033b7b9a1a9508efaedf80f45e18d2287e8"
---

During an interview the orchestrating agent tells the person about the question drafter's internal round trips: the drafter refused a draft for running one row over the 24-row limit, and the agent narrated that refusal, its reasoning, and the redraft it was resuming. The person does not need any of it. Only the finished question should reach them; a refused or redrafted question is the agent's own business.

## Grounds

- pursued: we expect interviews to stop narrating drafter refusals; a session that still tells the person about a refused or redrafted question would show this wrong.
