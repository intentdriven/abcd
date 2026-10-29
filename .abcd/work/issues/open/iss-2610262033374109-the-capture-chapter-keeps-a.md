---
schema_version: 1
id: "iss-2610262033374109"
slug: "the-capture-chapter-keeps-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 424 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:521"
remedy: "Mark section 4 of 06-capture.md (lines 519-527) as moot, since capture writes the ledger directly and nothing is left to migrate (03-configuration.md), or delete it; keep the ForceID seam note only if a migrator still needs it."
finding_ref: cfk-27e272564119cfa7
---

The capture chapter keeps a scratch-to-ledger migration as owed design, while the configuration chapter says that migration has nothing left to migrate.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:521` — "The migration rides the `dev-sync work` surface"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1200` — "so the migration this design was drawn around has nothing left to migrate"

06-capture describes a legacy scratch migration, not yet built, that rides the dev-sync work surface and has a migrator seam reserved for it. 03-configuration says capture writes the ledger directly, so the migration the dev-sync design was drawn around has nothing left to migrate. One document treats the migration as pending work and the other treats it as moot.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 424.
