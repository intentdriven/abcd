---
schema_version: 1
id: "iss-319"
slug: "development-readme-record-map-table"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/README.md"
resolution: "development/README.md record-map table adds specs/, release/, release-gate/ and corrects the personas.json note."
impact: internal
---

development README record-map table omits specs release and release-gate and carries a stale personas-json migration promise
## Evidence
`.abcd/development/README.md:6-7` frames the table as "one canonical home per concept" but lists only brief/intents/principles/decisions/roadmap/plans/research — omitting `specs/`, `release/`, `release-gate/`, all load-bearing (record-lint's spec_lifecycle/spec_id_unique/record_schema → specs; surface_coverage → release/surface.json; gate_lockstep → release-gate/README.md). `:19` says personas.json "migrates to embedded Go data when the intent surface is built" — under-describing its now-shipped role as the persona_registry SSOT read by a blocker lint.

## Adversarial verdict: CONFIRMED table omission (nitpick); "migration condition met" REFUTED
specs/ is a first-class record family (record_schema registers spc as a peer store; `abcd <id>` dispatches spc-N), not a sub-artefact of intents. The `directory_coverage` warn covers only specs/'s own missing README, not the parent table. The personas "condition met" framing does not survive (grill sub-verb unshipped → the intent surface is arguably not complete), so the residual is only that the parenthetical omits personas.json's shipped SSOT/lint role. Fix: add three table rows + one clause on personas.json's persona_registry role; ideally an index_drift region so the row set is gated (the iss-38 shape). Not prior art: iss-42 corrected a different row of this table.
