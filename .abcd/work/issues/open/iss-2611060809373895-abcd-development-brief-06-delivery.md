---
schema_version: 1
id: "iss-2611060809373895"
slug: "abcd-development-brief-06-delivery"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Correct the passage to say what the code does."
duplicates: [iss-2611040555120214]
---

`.abcd/development/brief/06-delivery/02-verification-matrix.md:63`: The verification matrix says a capture reports its severity, and it does not. Severity is not reported. Text output is 'captured iss-… (open) — <path>'. A `--severity major --json` run returned id, slug, path, status, uncommitted, no_location, match and ledger, with no severity field.
