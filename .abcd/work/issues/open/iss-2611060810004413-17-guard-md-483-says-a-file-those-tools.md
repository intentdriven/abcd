---
schema_version: 1
id: "iss-2611060810004413"
slug: "17-guard-md-483-says-a-file-those-tools"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "List the stop-store denial and the stage's write refusals beside the shared-checkout rule in the guard's reach paragraph."
---

`17-guard.md:483` says a file those tools write is judged only by where it lands in a shared main checkout, and anywhere else is written unchecked. `abcd guard hook` also denies a Write to a stop file or the shared-folder setting wherever it lands (`Write into the stop`, `internal/surface/cli/hook_occupancy.go:178`), and with `ABCD_STAGE_WRITES` set it denies a Write outside the stage and a Write to `Makefile` (`internal/surface/cli/hook_occupancy.go:156`).
