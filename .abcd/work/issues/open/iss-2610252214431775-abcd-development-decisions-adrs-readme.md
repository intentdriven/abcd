---
schema_version: 1
id: "iss-2610252214431775"
slug: "abcd-development-decisions-adrs-readme"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/README.md"
remedy: "Correct adrs/README.md:33, capture the consolidation onto recordid.CanonADRID/ADRFileID under one-canonical-primitive, and cut the brief paragraph to a pointer at that record."
---

.abcd/development/decisions/adrs/README.md:33 says every ADR-id reader admits both vintages 'through one derivation', naming the record_schema and citation-currency gates, the site index and the lifeboat; 24-decide.md:142-154 and the code show four derivations (internal/core/lint/schema.go:86, internal/core/site with no recordid import, internal/core/lifeboat/graveyard_abandoned.go:435-441). The brief calls consolidating them 'open work', and no ledger record holds it.
