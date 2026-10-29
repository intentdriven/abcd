---
schema_version: 1
id: "iss-2610262035312375"
slug: "the-banlist-chapter-says-this-repository"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 578 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md:79"
remedy: "In the docs-currency bullet at 06-lint.md:10 add that the names/ banned tokens also read the configuration's name_roots (.abcd, scripts, commands and the other shipped surfaces), every text file, as 20-banlist.md:72-80 and internal/core/lint/lint.go lintNameRoots do."
finding_ref: cfk-11d838950e712344
---

The banlist chapter says this repository's docs-lint `names/` entries read `name_roots` that include `.abcd`; the lint contract limits docs-currency to `docs/` and the repo-root prose.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/20-banlist.md:79` — "This repository's `name_roots` are `.abcd`, `AGENTS.md`,"
- End B: `.abcd/development/brief/05-internals/06-lint.md:10` — "lints `docs/` and the repo-root prose for change-narration"

20-banlist says the `names/` entries of `.abcd/docs-lint.json` also read the configuration's `name_roots`, and that this repository's are `.abcd`, `AGENTS.md`, `scripts` and the other shipped surfaces. 06-lint describes the docs-currency gate, configured by that same file, as linting `docs/` and the repo-root prose, and assigns `.abcd/development/` to record-currency. Both cannot describe the docs gate's reach: by the banlist chapter it reads the design record, and by the lint contract it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 578.
