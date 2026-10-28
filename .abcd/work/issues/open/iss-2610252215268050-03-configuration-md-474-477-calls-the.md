---
schema_version: 1
id: "iss-2610252215268050"
slug: "03-configuration-md-474-477-calls-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Say the rule is enforced by that test, with the assembler's exclusion row as its one declared exception (02-constraints/03-invariants.md:167)."
---

03-configuration.md:474-477 calls the one-owner rule for the transcript store 'a convention with nothing behind it', but TestOnlyTheHistoryPackageNamesTheStorePath (internal/core/history/store_boundary_test.go:162) enforces it, with the assembler's exclusion row as its declared exception
