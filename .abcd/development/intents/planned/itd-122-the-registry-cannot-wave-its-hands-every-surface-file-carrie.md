---
id: itd-122
spec_id: spc-27
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
slug: the-registry-cannot-wave-its-hands-every-surface-file-carrie
---

# The Registry Cannot Wave Its Hands

## Press Release

The registry can no longer wave its hands. Every surface file under
`04-surfaces/` carries a sub-verb table — which bucket each verb is
(lint / review / audit / gate) and whether it exists — and `surface_coverage`
now checks every row against the binary's own command tree, both ways: a
sub-verb that ships without a row fails the build, and a row claiming an
unregistered sub-verb fails the build. "shipped means shipped, at every grain —
when a rename lands, the gate proves the migration complete instead of me
grepping and hoping," says Bob, staff engineer.

## Why This Matters

`surface_coverage` is blind inside a row, so `shipped` means "the top-level
verb exists" and every unbuilt sub-verb hides behind it — six of twenty rows
qualify themselves in prose the lint does not read, and documents outside the
registry then cite those sub-verbs as live (refines iss-246, its "two defects,
one fix"). This is the detector adr-40 needs (decision 6), and it is the
ordering keystone of the process-coherence plan: the check lands **armed, with
rows reflecting current names, before any rename** — so the renames are proved
complete by a gate rather than asserted complete by an agent. Per the
2026-08-16 planning rulings, the `Status` enum stays two-valued at both grains;
sub-verb rows are what carry the granularity.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** a surface file under `04-surfaces/`, **then** it carries a
  sub-verb table with two facts per verb: bucket (`lint` / `review` / `audit`
  / `gate`, or `—` for a non-assessment verb) and existence (`shipped` /
  `staged`).
- **Given** the extended `surface_coverage`, **then** a `shipped` row must
  match a registered sub-command in the committed command-tree snapshot
  (`.abcd/development/release/surface.json`, itself drift-checked against the
  cobra tree) **and** every registered sub-command of that surface must have a
  row — both directions are lint failures.
- **Given** a `staged` row, **then** no registered sub-command may back it.
- **Given** a host-delegated surface (no Go verb: `consult`, `ingest`,
  `prepare-this-repo`) or an operator-internal verb with no surface file
  (`spec`, `rules`, `hook`, `completion`), **then** the exemption is explicit
  rule config — mirroring today's `bare_command` mechanism — never a silent
  skip: host-delegated rows are exempt from the cobra check only; the
  operator-internal list is enumerated.
- **Given** the population pass, **then** all twenty surface files gain their
  tables in the **same change** the extended check arms — nothing ships
  unbucketed, and rows reflect current (pre-rename) names.
- **Given** the bucketings adr-40 calls arguable, **then** the maintainer's
  pre-rulings bind: `identity` is an **audit** (rendered surfaces vs the
  recorded canonical block), the launch changelog guardrail is a **gate**, and
  `guard check` is a **gate**. Any other genuinely ambiguous bucket is a STOP
  for an unattended run, never a guess into the closed list.
- **Given** the bucket enum, **then** it is registered under Reserved
  vocabulary in `02-constraints/04-naming.md` (closed list, PR-to-extend).
- **Given** the `Status` enum, **then** it stays two-valued (`shipped` /
  `staged`) at both grains — no `partial`.
- **Given** the sweep, **then** the `04-surfaces/README.md` prose describing
  the check, and `CHANGELOG.md`, reflect the extended rule.

## SOTA

Registry-vs-implementation cross-checks are standard drift tooling (OpenAPI
spec-vs-handler checkers, terraform schema drift, this repo's own
`surface_coverage` and `index_drift` rules); nothing importable — the registry
format and command tree are native. **Chosen path: bespoke**, extending the
existing `checkSurfaceCoverage` lint against the existing committed
`surface.json` snapshot (no import-cycle: both inputs are committed
artefacts). No new dependency.

## Open Questions

_None gating. The three arguable bucketings were pre-ruled at the grill,
2026-08-16, and are binding on the build._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
