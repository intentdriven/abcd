---
schema_version: 1
id: "iss-2610050259233425"
slug: "docs-fidelity-apply-writes-each-drafted-replacement-into"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.13.0 release cut, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/brief-review-flags.json"
remedy: "Have the flags file record the sentence as committed: refresh each flagged entry's text from the chapter at commit time (or at the next docs fidelity record), and have record-lint report a flagged replacement no chapter line contains."
---

docs fidelity --apply writes each drafted replacement into .abcd/work/brief-review-flags.json, but the sentence a person then tidies by hand before committing (grammar, a half-replaced wrapped line, a flag name the chapter prose may not carry) is not what the flags file records: in the v0.13.0 cut, five chapters (10-docs, 15-prepare-this-repo, 22-site, 23-reading, 27-implement) landed wording that differs from the flagged replacement, so the record of what the product thinker must read names text that is not in the brief.
