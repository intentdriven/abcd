---
schema_version: 1
id: "iss-2611040555387360"
slug: "bare-abcd-reading-prints-exclusion-rows"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reading.go"
remedy: "Word the row as the table's size, or print the count per position."
---

Bare `abcd reading` prints `exclusion rows: 40, asserted into every manifest` (internal/surface/cli/reading.go:439). Manifests carry 30 at detection, 28 at entailment and 31 at widening; the others bind to particular positions, such as comparative (comparativeExclusions, internal/core/reading/include.go:788). So 40 is the table's size. .abcd/development/brief/04-surfaces/23-reading.md:103 says only that the floor rides in every manifest, which holds.
