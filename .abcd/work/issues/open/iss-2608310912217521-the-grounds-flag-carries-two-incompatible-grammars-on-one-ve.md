---
schema_version: 1
id: "iss-2608310912217521"
slug: "the-grounds-flag-carries-two-incompatible-grammars-on-one-ve"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-fidelity-audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

the grounds flag carries two incompatible grammars on one verb family with a floor on three routes and free text on the fourth

Found by the itd-179 fidelity audit.

`--grounds` now means two different things on one verb family:

- `capture promote|resolve|wontfix` and `intent ready` take
  `<token>: <text>` with a closed vocabulary and the substance floor.
- `capture disposition` takes `disposition_grounds` as FREE TEXT with no token
  and no floor.

So a caller who learns the grammar on one route is wrong on the next, and a
record's grounds mean different things depending on which verb wrote them. The
audit reached this while judging ac-1: `capture promote <rdi-N>` dispatches
before `requireGrounds` is consulted, so that route satisfies the criterion by
DELEGATING to the laxer artefact rather than by enforcing the stricter one.

Not a defect in either half on its own. The question it raises is whether one
flag name should carry two contracts, and that is a design decision rather than
a bug to fix.
