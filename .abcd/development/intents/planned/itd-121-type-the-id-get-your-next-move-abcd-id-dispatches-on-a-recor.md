---
id: itd-121
spec_id: spc-26
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
slug: type-the-id-get-your-next-move-abcd-id-dispatches-on-a-recor
---

# Type The Id, Get Your Next Move

## Press Release

Type the id, get your next move. `abcd itd-119` answers *what is this and what
happens next* — a planned intent whose spec body is written: implement it, then
`abcd spec close spc-24`. The twelve-step walk stops being a directory hunt:
every record names its own next verb, and nobody needs to know that shipping an
intent is a `spec` verb. "I stopped keeping the lifecycle in my head — the
record tells me where it stands and what I'd do next," says Nia, facilitator.

## Why This Matters

The record walk is observable only to someone who already knows the verb map.
`iss-`, `itd-`, `spc-` (and `adr-`) prefixes are globally unique and
regex-validated, so the id alone is the routing. The mental model, per the
process-coherence plan: bare `/abcd` answers *what can I do*; `abcd <id>`
answers *what is this, and what is my next move*. SD001-safe — a positional on
the namespace root is not a `show` sub-verb. Duplicate-checked against itd-86
(a blind document-review pass) and itd-112 (the startup banner): no overlap.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** `abcd <id>` where the positional matches
  `^(iss|itd|spc|adr)-[0-9]+$`, **when** it runs, **then** the record is
  located in its store (any status folder or bucket) and rendered read-only —
  what it is, its key fields and links, and the next move. Zero writes, ever.
- **Given** an `itd-N`, **then** the next move follows the lifecycle:
  `drafts/` → planning interview + `intent plan`; `planned/` with a `_Draft:`
  spec body → write the spec body (path shown); ready (the `intent ready`
  checks pass) → implement; `shipped/` → audit state shown; `superseded/` →
  pointer to the superseding record.
- **Given** an `iss-N`, **then**: open and unpromoted → `capture promote` /
  `resolve` / `wontfix`; promoted → the `itd-N` it graduated into; resolved or
  wontfix → the trail, including `resolved_by` when present.
- **Given** an `spc-N`, **then**: open with its linked intent ready →
  implement against the spec body, then `spec close`; closed → done, linked
  intent shown.
- **Given** an `adr-N`, **then** a read-only render — id, status, title,
  path; next move: none, decisions are read.
- **Given** a shape-matching id that exists in no store, **then** a structural
  fault with a diagnostic and non-zero exit.
- **Given** any other positional, **then** the current unknown-command
  behaviour is byte-for-byte unchanged.
- **Given** `--json`, **then** the same facts structured.
- **Given** the next-move mapping, **then** it lives in one Go table and a
  test asserts every recommended verb resolves to a registered command in the
  cobra tree — a rename breaks the test instead of shipping stale advice.
- **Given** the sweep, **then** the root surface page and the `04-surfaces`
  registry document the positional.

## SOTA

Id-dispatch on a root command is a common CLI affordance (`gh issue view
<number>`, `git show <ref>`-style polymorphic lookups, issue trackers' global
id search); nothing importable — the record stores and lifecycles are native.
**Chosen path: bespoke**, a regex-gated positional over the existing store
readers (`capture`, `intent`, `spec`, plus a thin adr reader). No new
dependency.

## Open Questions

_None gating. Extending dispatch to further families (e.g. plans, principles)
is a future consideration, deliberately not an AC._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
