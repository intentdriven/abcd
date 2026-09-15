---
schema_version: 1
id: "iss-2609022252265901"
slug: "the-regime-operator-surface-guard-testnooperatorsurfacesetsa"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "Iteration 2 opening run, push of the readings branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/regime_surface_test.go"
resolution: "walkConfigFiles now skips the readings record family, derived from issueschema.ReadingsRecordDir rather than a literal, so the guard reads only surfaces an operator can actually write. A new test plants a tracked run.json and refusal.json carrying a regime and requires the walk to stay silent, and plants a regime key in .abcd/config/ and requires the walk to still report it."
impact: fix
---

The regime operator-surface guard (TestNoOperatorSurfaceSetsARegime) walks every tracked JSON under .abcd/ from the git index and treats each key as a configuration key an operator could write. The readings record family is not configuration: abcd reading ingest writes .abcd/development/readings/RUNID/run.json and refusal.json from the agent definition, and each carries the regime the design requires the definition to stamp and the output contract to carry, so both fail the guard the moment a real run is committed. No run record had been committed before Iteration 2's opening readings, so the over-reach was latent and passed every prior preflight. A record family a verb writes from a definition is not an operator surface, and the walk reaches into one.

## Grounds

- pursued: the regime guard should refuse only channels an operator can set, so a record family the verb writes from the position's definition is outside it and no committed run record can turn the guard red; what would show this wrong is a way for an operator to influence a run's regime by editing a file under .abcd/development/readings/, or a real configuration key escaping the walk because it sits under that prefix.
