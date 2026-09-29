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
deferred_after: "v0.11.1"
deferral_reason: "ruling AL owed to the product thinker: the honest fix changes what a person types on released verbs. Three grammars still stand: capture promote, resolve and wontfix and intent ready take <token>: <text> with the floor, capture disposition takes free text, and intent condition takes free text held to the floor with no token. The options are the record's: (a) split the flag name, (b) put disposition --grounds through grounds.New, which fails because a disposition carries its own state vocabulary, or (c) record the divergence as intended. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
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
