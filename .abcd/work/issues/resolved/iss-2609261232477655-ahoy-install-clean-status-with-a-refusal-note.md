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
resolution: "Already fixed at the base by 973a73d18, which proves the local tier from the resolved checkout root: the refusal the report saw was the false one a checkout entered through a symlinked path drew, which the banlist step then overtook, leaving clean beside the note. At the base that run is clean with no refusal, and a real refusal (a symlink planted at the tier) reports partial; 52c9bcc2f pins both statuses in the local-tier tests. Other notes on a clean run are advisory (PATH reach, the plugin-cache degradation, the unasked artefact kind), not refusals a later step overtook."
impact: internal
resolved_by:
  commit: "973a73d18"
---

ahoy install reports its status as clean while also printing a refusal note for a step, because a later step (the banlist scaffold) creates the local tier before the final status check runs, so the note and the status disagree in one run's output. The status should reflect every step's refusal, or the note should say the refusal was overtaken.

## Grounds

- pursued: a run whose local-tier step refuses never reports clean; a clean install result carrying a local-tier refusal note would show it wrong
