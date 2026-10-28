---
schema_version: 1
id: "iss-2610241549107888"
slug: "the-root-machine-records-index-in-abcd"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b02 reading iss-2608221342508040"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/README.md"
remedy: "Add a row to the root machine records table in .abcd/README.md for each optional config/ record the binary reads, oracle-routing.json first, naming its reader and its owning record, proven by a check that every .abcd/config/ path named in non-test code under internal/ has a row."
---

The root machine records index in .abcd/README.md lists five config/ records and two optional absent ones (pii.json, scripts-closure.json) but not config/oracle-routing.json, an optional record the shipped binary reads and ahoy offers to write (internal/core/ahoy/oracle_routing.go:14, :51), nor the other optional config/ records added since, so the index a reader starts from lags the directory it indexes.
