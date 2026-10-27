---
schema_version: 1
id: "iss-2610262031582207"
slug: "04-scope-defines-operator-internal-verbs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 203 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:75"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-dab67fb82053c4ab
---

04-scope defines operator-internal verbs, rules and spec among them, as wiring with no commands/ file; itd-146 says operator_internal only means 'needs no surface chapter' and treats rules as user-facing.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:75` — "`record-lint.json` means *needs no surface chapter*, which is a lint"
- End B: `.abcd/development/brief/01-product/04-scope.md:19` — "the class is defined by the absence of a `commands/` file rather than by the presence of one"

04-scope lists rules and spec among the operator-internal verbs, as wiring rather than user-facing surface, and says 'the class is defined by the absence of a `commands/` file rather than by the presence of one'. itd-146 says the operator_internal key in '`record-lint.json` means *needs no surface chapter*, which is a lint' exemption rather than a verdict on user visibility, and keeps rules listed because the marker block tells every managed repository to run it. The term operator-internal has two definitions, and they disagree on whether rules is user-facing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 203.
