---
schema_version: 1
id: "iss-2610252214358706"
slug: "34-build-md-679-680-says-the-validate"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S05"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md"
remedy: "Delete 'one fresh agent at a time', or replace the paragraph with a pointer to commands/implement.md:402."
---

34-build.md:679-680 says the validate stage hands the head to validators 'one fresh agent at a time'. validate.go:4-5 and commands/implement.md:402 run them side by side up to the ceiling (ruling DR6, state v9), as does the brief's own 406-414.
