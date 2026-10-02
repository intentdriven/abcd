---
schema_version: 1
id: "iss-2610020722584184"
slug: "a-repository-s-oracle-route-that-cannot-apply-takes-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/config.go"
remedy: "Per the technical facilitator's rulings CD3 and CD4 of 2026-10-02: skip a repository route to a model a configured keyless provider does not list, with a diagnostic, and let the machine's route for the name apply (the machine's own unlisted route and any denylisted model still refuse); skip a repository route to an unconfigured provider where the machine routes the same name, with a diagnostic, so the owner's setting applies (no machine route: runs on the host as now); amend adr-25."
resolution: "A repository route to a model a configured keyless provider does not list, and one to an unconfigured provider where the machine routes the same name, are skipped with a diagnostic and the machine's route applies (rulings CD3, CD4 of 2026-10-02); adr-25 amended."
impact: fix
resolved_by:
  commit: "257879f0d"
---

A repository's oracle route that cannot apply takes the configuration or the person's setting down: a route in .abcd/config.json naming a model a configured keyless provider does not list refuses the whole provider configuration load (every command that reads it fails), and a route there naming a provider this machine has not configured wins over the machine's own route of the same name, so the role runs on the host and displaces the person's setting. Both disagree with ruling CD2's shape for a repository's route (skip with a diagnostic, the machine's route applying in its place).

## Grounds

- pursued: LoadAPI loads with one diagnostic and the machine's route in force for both cases (TestARepositoryRouteToAnUnlistedModelIsSkipped, TestARepositoryRouteToAnUnconfiguredProviderYieldsToTheMachineRoute); a machine route to an unlisted model or a denylisted model from the repository still refusing shows the skip is not a softening, and a refusal or a host fallback in either new test would show it wrong
