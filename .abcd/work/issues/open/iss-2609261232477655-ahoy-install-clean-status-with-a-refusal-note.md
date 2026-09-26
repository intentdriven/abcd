---
schema_version: 1
id: "iss-2609261232477655"
slug: "ahoy-install-clean-status-with-a-refusal-note"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix3-cutfix risks"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
---

ahoy install reports its status as clean while also printing a refusal note for a step, because a later step (the banlist scaffold) creates the local tier before the final status check runs, so the note and the status disagree in one run's output. The status should reflect every step's refusal, or the note should say the refusal was overtaken.
