---
schema_version: 1
id: "iss-2609281911015826"
slug: "itd-63-criterion-4-the-safety-gate-s"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-63"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-63-setup-wizard-explains-installs.md"
resolution: "itd-63's Why and References no longer assert that the safety gate blocks on a missing scanner: they say itd-62 is a draft and name the history store's armed-gitleaks refusal as the shipped consumer; the itd-62 draft's References carries the constraint that its missing-scanner refusal calls tools.Missing and tools.Install. Criterion 4 stays NOT_MET in the audit notes until the gate exists."
impact: internal
resolved_by:
  commit: "013be0bc4"
---

itd-63 criterion 4 (the safety gate's missing-scanner case routes through the explain-then-install mode) is NOT_MET at ceb4b6dbb: no safety gate exists on main, itd-62 (pluggable-safety-gate) is still in intents/drafts/, so nothing surfaces the prerequisite the criterion names. The spec's close note substitutes the history store's armed-gitleaks refusal (internal/core/history/history.go calls tools.Missing), which is transcript capture, not the gate. The shipped record names a first consumer that does not exist; when itd-62 is planned its missing-scanner path must call tools.Missing/tools.Install rather than print a bare command, and itd-63's References/Why should stop asserting that the safety gate already blocks on a missing scanner.

## Grounds

- pursued: the shipped record states no consumer that does not exist; a reader of itd-63 again told the gate blocks on a missing scanner today would show it wrong
