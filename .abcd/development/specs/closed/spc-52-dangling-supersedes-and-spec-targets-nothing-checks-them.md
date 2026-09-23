---
id: spc-52
slug: dangling-supersedes-and-spec-targets-nothing-checks-them
intent: itd-160
---
# dangling-supersedes-and-spec-targets-nothing-checks-them

## Summary

Arms a red-gate detector for dangling typed cross-references — a `supersedes`
naming a record absent from the tree, or a `spec_id` naming a `spc-N` with no
file — ratcheted through `.abcd/site-baseline.json` so the existing eight-entry
backlog is baselined and only *new* dangles fail. The reference graph and the
baseline machinery already exist in the site build; this spec is the design that
wires them into the gate, seeds the backlog, and proves the four ratchet
behaviours.

## Scope

In:

- `internal/core/site/recordjson.go` — `measureHealth` (line 416), which walks
  `graph.Dangling` and counts unresolved refs including `supersedes` and the
  `spec_id` graph field.
- `internal/core/site/check.go` — `checkBaseline` (line 1369), the ratchet red
  gate that fails a live unresolved ref outside the baseline and passes a
  baselined one.
- `.abcd/site-baseline.json` — the seeded `unresolved_references` backlog (the
  eight entries: adr-22→adr-14/15/17, adr-25→adr-8, adr-27→adr-16, adr-28→adr-18,
  adr-35→adr-4, itd-3→spc-1).
- Tests in `internal/core/site/check_test.go` and `health_test.go`.

Out:

- No decision on how tombstones/stubs *render* (itd-136/itd-137 owns that); this
  spec gates the dangle, it does not add a target page for it.
- No change to the reference parser itself (`internal/core/lint/graph.go`,
  `schema.go`) beyond consuming what it already emits.

## Approach

**The reference graph already distinguishes dangles.** `LoadRecordGraph`
(`lint/graph.go:87`) reuses the `record_schema` scan; the parsed reference fields
(`schema.go:79–91`) include `supersedes`/`superseded_by` (handle fields) and
`spec_id`/`intent` (graph fields). The graph emits absent-target refs as
`g.Dangling` and retired ids as `g.Retired` (graph.go:101–149).

**Count the dangle, not-excusing supersedes.** `measureHealth`
(`recordjson.go:416`) walks `graph.Dangling` into `Health.Unresolved`. The
load-bearing line is recordjson.go:429: a retired target is excused *except* for
`supersedes` (`if e.Field != "supersedes" && retired[e.To] { continue }`) — so a
`supersedes` naming an absent id counts even when that id is retired, and a
`spec_id` naming an absent `spc-N` counts as a graph-field dangle. This is the
detector the intent asks for; the design confirms both field kinds reach
`Health.Unresolved`.

**Ratchet as the red gate.** `checkBaseline` (`check.go:1369`) reads
`Health.Unresolved` against `.abcd/site-baseline.json` (`Baseline`/`BaselineEntry`
in `site/paths.go:27–36`, loaded by `LoadBaseline`, path
`BaselineRelPath=".abcd/site-baseline.json"`). Direction: a live unresolved ref
**not** in the baseline is `c.fail(...)` — a red gate (check.go:1401–1408); a
baseline entry whose ref now resolves is a shrink invitation (`c.note`), never a
failure (check.go:1413–1421). This is exactly the ratchet semantics the intent
specifies. The eight backlog entries are seeded into the baseline so they do not
newly fail; a ninth, new dangle fails red.

## How it satisfies each acceptance criterion

- *A new `supersedes` naming an absent record fails as a red gate* —
  `measureHealth` counts the supersedes dangle (recordjson.go:429 does not excuse
  it), and `checkBaseline` fails it because it is not in the baseline. Test
  (`check_test.go`): a record introducing `supersedes: adr-999` (no such file) <!-- record-lint: illustrative -->
  produces a red gate.
- *A `spec_id` naming a `spc-N` with no file fails as a red gate* — the
  `spec_id` graph-field dangle reaches `Health.Unresolved` and fails the same
  way. Test: an intent whose `spec_id` names a missing spec fails red.
- *The existing backlog is baselined and does not newly fail* — the eight entries
  seeded in `.abcd/site-baseline.json`. Test: with the seeded baseline, the build
  passes; assert the eight known dangles are matched by baseline entries, not
  reported as failures.
- *A baselined dangle whose target is later added still passes* — `checkBaseline`
  treats a now-resolving baseline entry as a shrink `c.note`, not a failure
  (check.go:1413–1421). Test: add the target for a baselined entry and assert the
  gate still passes (and invites the baseline to shrink).

## Decisions

Gate at the site build via the existing ratchet, not at record-lint: the intent
says the *planned site build* arms the detector via the baseline, and the
reference graph plus `checkBaseline` already live there, so seeding the backlog
and confirming both field kinds reach the gate is the whole delivery. The
tombstones-or-stubs rendering question is deliberately deferred to itd-136/itd-137
— this spec makes the dangle *fail*, and leaves how a resolved target renders to
that decision.
