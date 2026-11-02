---
schema_version: 1
id: "iss-2611011344205894"
slug: "every-change-that-mints-or-edits-an-adr"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "CI and local compute investigation, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/records-only.sh"
remedy: "Match a test's mention of a changed record by its repo-relative path or record id, and by bare file name only when the name is unique in the tree, so a generated index named README.md no longer turns every ADR change into a full gated preflight; keep the fail-closed default for any path a test does name."
---

Every change that mints or edits an ADR takes the full, gated preflight instead of the records-only path, because the ADR index .abcd/development/decisions/adrs/README.md is regenerated with it and scripts/records-only.sh treats a changed path as a test fixture when any test names its bare file name: 'full run: .abcd/development/decisions/adrs/README.md is named by a test (its file name README.md), so it is that test's fixture'. Dozens of files are named README.md and tests naming one of them do not read this one, so a records-only branch with an ADR waits at the load gate (1 to 2 hours on 2026-11-01) and runs the whole Go suite for no test it could affect.
