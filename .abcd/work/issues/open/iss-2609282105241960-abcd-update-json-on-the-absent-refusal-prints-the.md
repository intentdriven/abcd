---
schema_version: 1
id: "iss-2609282105241960"
slug: "abcd-update-json-on-the-absent-refusal-prints-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/update.go"
---

`abcd update --json` on the absent refusal prints the dispatch-refusal receipt (with an empty origin key, update.Report's Origin has no omitempty) and then the root error envelope: two JSON documents on stdout where a machine reader expects one. Found by the v0.11.1 crosscheck (x-051); unchanged since v0.11.0.
