---
schema_version: 1
id: "iss-2610262031462724"
slug: "the-naming-register-makes-abcd-ahoy-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 175 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:14"
remedy: "Rewrite itd-57 so it does not name `abcd configure` (no such verb; ahoy is the setup surface) and reconcile it with the shipped `abcd intent hold` and `unhold`, or supersede it by them."
related_intents: [itd-57]
finding_ref: cfk-927ad4807524f895
---

The naming register makes `/abcd:ahoy` the verb that installs abcd, but itd-57 scaffolds its sentinel with `abcd configure`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:14` — "install abcd into it, or report what is installed"
- End B: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "With this, `abcd configure` plants a single `spc-0-manual-hold` sentinel spec"

04-naming defines `/abcd:ahoy` as the verb that installs abcd into a project or reports what is installed, and its exempt list has no `configure` verb. itd-57 has `abcd configure` (or 'the equivalent first-run path') plant its sentinel spec, which gives the setup surface a second name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 175.
