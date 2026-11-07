---
schema_version: 1
id: "iss-2611060809191251"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Drop the deeper opt-in secret scan from the sentence at 04-launch.md:413."
---

.abcd/development/brief/04-surfaces/04-launch.md:413 says the gate suite does not carry a deeper opt-in secret scan. It does: gitleaks, opted into by `.abcd/config/gitleaks.json` and registered at cmd/abcd/main.go:24, joins the scan, and armed with no binary installed it is a hard fail, as lines 55-58 of the same chapter and 02-constraints/02-dependencies.md:61-68 say.
