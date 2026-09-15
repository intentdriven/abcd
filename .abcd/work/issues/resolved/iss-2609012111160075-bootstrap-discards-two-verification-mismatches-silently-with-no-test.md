---
schema_version: 1
id: "iss-2609012111160075"
slug: "bootstrap-discards-two-verification-mismatches-silently-with-no-test"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
resolution: "Both branches now say what they declined to do on the bootstrap's own stderr line: the migration seed refuses a mismatched root binary with a notice, and the PATH-copy refresh names the copy it left untouched (tilde form) and why. Sibling silent discards were made loud or left quiet with the reason stated in the fixing commit."
impact: fix
resolved_by:
  commit: "d2bd597ddeb59f8f58b441b291d3f3fdf13a5466"
---

itd-132 ac-5 promises that every promotion of a cached artefact re-verifies against its recorded binary_sha256 and 'refuses loudly' on a mismatch. Two branches discard silently instead: the migration-seed mismatch is ignored by spec design (spc-35 line 145) with no notice, and the bootstrap's PATH-refresh branch skips a copy whose hash no longer matches the recorded provenance with no notice and no test covering that branch. Both are refusals in effect, neither is loud, and a user whose PATH copy quietly stopped being refreshed has no signal until abcd update or ahoy tells them it is foreign. Surfaced by the itd-132 fidelity audit (receipt rcp-acde3e9ce729). Loud-staging says a stage that no-ops must say so.

## Grounds

- pursued: a reader whose seed or PATH refresh was declined now sees one line saying so, and the tests that pinned the file state also pin the notice; shown wrong if either branch still exits without a word on a mismatch
