---
schema_version: 1
id: "iss-2610241550119108"
slug: "ahoy-install-s-re-founding-lineage"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b11 reading iss-221"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/setup_question.go"
remedy: "Give SetupConfirmQuestion in internal/core/ahoy/setup_question.go meanings for the history.lineage id that say what registerRepo does with each answer (today: No and decide later register this root as a new project with no link to the named predecessor, and the question is not asked again), pinned by a setup_question test checked against registerRepo's behaviour."
---

ahoy install's re-founding lineage question tells the person that No and decide-later write nothing and that the next install asks again, but declining registers this root as a new project in the session store's index, and no later install asks again, because the question is put only while the root is unregistered.
