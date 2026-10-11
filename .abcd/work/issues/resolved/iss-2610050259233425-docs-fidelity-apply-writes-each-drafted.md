---
schema_version: 1
id: "iss-2610050259233425"
slug: "docs-fidelity-apply-writes-each-drafted"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.13.0 release cut, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/brief-review-flags.json"
remedy: "Have the flags file record the sentence as committed: refresh each flagged entry's text from the chapter at commit time (or at the next docs fidelity record), and have record-lint report a flagged replacement no chapter line contains."
resolution: "record-lint's new brief_flag_landed rule refuses a review flag whose replacement no line of its chapter contains, read through docfidelity.UnlandedFlags (registered in record-lint and the CLI); the six stale flags now record the committed wording, and the brief and commands/docs.md say the flag records the sentence as committed"
impact: internal
resolved_by:
  commit: "5d8b5e8cc4449d14bbfe5f0ef3fd00c15c71b20c"
---

docs fidelity --apply writes each drafted replacement into .abcd/work/brief-review-flags.json, but the sentence a person then tidies by hand before committing (grammar, a half-replaced wrapped line, a flag name the chapter prose may not carry) is not what the flags file records: in the v0.13.0 cut, five chapters (10-docs, 15-prepare-this-repo, 22-site, 23-reading, 27-implement) landed wording that differs from the flagged replacement, so the record of what the product thinker must read names text that is not in the brief.

## Grounds

- pursued: record-lint over the tree reports no brief_flag_landed finding and a hand-tidied replacement is reported on its flag's line; shown wrong by a flag whose replacement its chapter lacks passing record-lint, or a current flag being reported
