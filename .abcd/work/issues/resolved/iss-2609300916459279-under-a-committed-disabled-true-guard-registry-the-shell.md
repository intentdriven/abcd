---
schema_version: 1
id: "iss-2609300916459279"
slug: "under-a-committed-disabled-true-guard-registry-the-shell"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/teach.go"
remedy: "Keep teaching the hazard (the teaching and guard switches are independent, spc-16) but open each lesson of a disabled registry with a lead that says the guard is off, such as 'Hazard (guard off)', on both the repository and the bundled plane. Grounds: review probe P4b of feat/guard-teach-repo-entries."
resolution: "A disabled registry's lessons open 'Hazard (guard off)' on the repository and bundled planes"
impact: fix
resolved_by:
  commit: "532b8aca4"
---

Under a committed disabled: true guard registry the SHELL teaching plane still opens every lesson with 'Refused by the guard' or 'Warned by the guard', a false sentence because a disabled registry refuses and warns about nothing; this holds for the repository's own entries and for the bundled ones alike.

## Grounds

- pursued: every SHELL lesson under a committed disabled guard.json opens with the guard-off lead (TestLessonsUnderADisabledRegistrySayTheGuardIsOff, TestShellLessonsUnderACommittedDisabledGuardSayItIsOff); any lesson there opening 'Refused by the guard' would show it wrong
