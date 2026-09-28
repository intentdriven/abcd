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
resolution: "update --json writes one document on a refusal: the receipt carrying the refusal envelope's abcd, error and exit_code fields, with no empty origin; proved by TestUpdateJSONRefusalIsOneDocument, watched failing on a scratch archive first."
impact: fix
resolved_by:
  commit: "d11429b11fe08cf9c7ebf0df3d0e0caba3c81e8f"
---

`abcd update --json` on the absent refusal prints the dispatch-refusal receipt (with an empty origin key, update.Report's Origin has no omitempty) and then the root error envelope: two JSON documents on stdout where a machine reader expects one. Found by the v0.11.1 crosscheck (x-051); unchanged since v0.11.0.

## Grounds

- pursued: a machine reader decoding stdout of a refused update --json finds exactly one JSON value naming the refusal's shape, remedy and exit code; a second value on stdout, or a document missing abcd:error or the refusal block, would show it wrong
