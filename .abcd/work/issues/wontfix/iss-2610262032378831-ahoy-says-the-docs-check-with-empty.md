---
schema_version: 1
id: "iss-2610262032378831"
slug: "ahoy-says-the-docs-check-with-empty"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 295 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:1017"
remedy: "none (filed automatically)"
finding_ref: cfk-378f137505137b22
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The tri-state exit grammar governs findings of warn severity; the empty-roots case prints a 'nothing was checked' notice on the diagnostic stream with exit 0 by an explicit ruling recorded in 10-docs.md, so the two do not clash."
---

ahoy says the docs check with empty roots warns at exit 0; the lint contract says a warning raises the exit code to 1

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:1017` — "document and warns that nothing was checked, at exit 0"
- End B: `.abcd/development/brief/05-internals/06-lint.md:34` — "Raises the exit code to 1 on an otherwise-clean run"

The ahoy chapter says a seeded docs-lint config with roots [] runs, warns that nothing was checked, and exits 0. The lint contract says the docs gate follows the tri-state exit rule, where a warning raises the exit code to 1 on an otherwise clean run. Under that rule, the same run cannot both warn and exit 0.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 295.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The tri-state exit grammar governs findings of warn severity; the empty-roots case prints a 'nothing was checked' notice on the diagnostic stream with exit 0 by an explicit ruling recorded in 10-docs.md, so the two do not clash.
