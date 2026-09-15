---
id: spc-60
slug: the-record-s-discipline-failures-are-themselves-recorded-a-l
intent: itd-182
---
# The lapse log: a capture category, timestamped at the lapse

## Summary

spc-60 completes [itd-182](../../intents/shipped/itd-182-the-record-s-discipline-failures-are-themselves-recorded-a-l.md):
when the recording discipline is suspended, deferred or evaded, that is itself
captured, as an ordinary issue record carrying `category: lapse`.

Part of the intent is already delivered and this spec claims no credit for it.
Landed and shipping: the `lapse` value in `issueschema.Categories`
(`internal/core/issueschema/issueschema.go`), pinned by
`TestValidateStrictLapseCategory` in `internal/core/capture/parse_test.go`; and
the value's appearance in the two record pages that enumerate the taxonomy,
`.abcd/development/brief/04-surfaces/06-capture.md` and
`.abcd/work/issues/README.md`.

What remains is the half of the acceptance criterion the enum line does not
touch: **the timestamp at the lapse**. The ledger has no property that can carry
it, and this spec adds one.

## Scope

In: a `lapsed_at` property on the issue schema, required exactly when the
category is `lapse`; the capture flag that supplies it; the mirrored refusal in
the committed-ledger gate; the fixtures; the surface and record pages the new
flag touches; the first three lapse entries.

Out: everything under `## Out of scope`.

## Approach

### Why a new property, and why not an existing one

The acceptance criterion asks for "a timestamp at the lapse rather than at
write-up". Nothing in the issue schema can carry that today:

- `found_at` is a **location**, not a time. Its flag help reads "optional
  repo-relative path or conceptual location"
  (`internal/surface/cli/cli.go`), and both record pages document it that way.
  Reusing it would silently redefine a shipped field.
- `created` and `updated` are tolerated on read for legacy ledgers and written by
  nothing, and both would mean write-up time in any case.
- The record id itself is timestamp-numeric (adr-45), which is exactly the
  write-up instant the criterion distinguishes itself from. Its existence is the
  reason a second field is needed rather than a reason to skip one.

So `issueschema.Known` gains `lapsed_at`: an RFC 3339 timestamp in UTC, optional
for every category and required for one.

### The refusing gate

`validateStrict` in `internal/core/capture/validate.go` gains one conditional
rule: when `category` is `lapse`, `lapsed_at` must be present and must parse as
RFC 3339. Absent, blank or unparsable is a refusal, not a warning: the whole
working claim under test is that recording at the point of commitment beats
retrospective reconstruction, and a lapse entry with no lapse time is the
reconstruction wearing the evidence's clothes.

The flag `--lapsed-at` on `abcd capture` carries the value and **has no
default**. Every other provenance flag on that command defaults, and this one
must not: a default would be the wall clock at write-up, which is the one value
the criterion forbids, supplied silently.

`checkIssueRecordShape` in `internal/core/lint/schema.go` mirrors the rule, as
that function already does for the enum sets and the slug grammar, so the gate
over the committed ledger refuses exactly what the reader refuses. Both sides
read the one definition in `internal/core/issueschema`; neither restates it.

### `found_during` is already the point in the process

The criterion's other two elements need no new mechanism. `category` is the enum
value, already landed. The point at which the discipline gave way is
`found_during`, which the schema already marks required and which
`validateStrict` already refuses when blank. This spec states the mapping and
adds nothing, which is the whole of the reuse rule here.

### Surfaces and pages the flag reaches

`commands/capture.md` gains `--lapsed-at` beside `--found-at` in its flag
paragraph. `.abcd/development/brief/04-surfaces/06-capture.md` gains
`lapsed_at` in its frontmatter block and the conditional rule in the sub-verb
row. `.abcd/work/issues/README.md` gains one sentence naming the rule.
`docs/reference/cli/commands.md` is a derived artefact and is regenerated with
`go generate ./internal/surface/cli`, never hand-edited; its drift test is what
proves the regeneration happened.

### The first three entries

The lapse log is content, not mechanism, and the distinction matters here: a
defect in the mechanism is fixed, a defect in the content is disclosed
(`.abcd/work/DECISIONS.md`, 2026-08-28). Three entries are written at the outset
rather than discovered later:

1. **The pre-tooling window.** Which entries were hand-authored before the
   surfaces that would have written them existed. Its pointer is already
   recorded: the facilitator's 2026-08-30 decision resolves pre-tooling entry 01
   to the ledger's construal filing.
2. **Anticipation.** Those populating the record know what the readings will
   look for, and the instrument is specified alongside the record it will read.
3. **Commitments made outside the tooling during the build.**

Each is an ordinary `abcd capture` call with `--category lapse`, a
`--found-during` naming the point in the process, and a `--lapsed-at` carrying
the instant the discipline gave way.

## Acceptance criteria mapping

itd-182 states one criterion with three elements. Each is mapped separately,
because two are already met and one is what this spec builds.

| Element of the criterion | How spc-60 satisfies it | Test |
|---|---|---|
| The entry carries the category | `lapse` in `issueschema.Categories`, validated by the same strict path every issue takes. **Already delivered** | `TestValidateStrictLapseCategory` (landed) |
| The entry carries the point at which the discipline was suspended, deferred or evaded | `found_during`, already required and already refused when blank. No change | `TestValidateStrictLapseCarriesFoundDuring` |
| The entry carries a timestamp at the lapse rather than at write-up | The new `lapsed_at` property, conditionally required, no default on the flag, mirrored in the committed-ledger gate | `TestValidateStrictLapseRequiresLapsedAt`, `TestValidateStrictLapsedAtMustBeRFC3339`, `TestCaptureLapsedAtHasNoDefault` |

## Tests

Every case below is watched to fail before its change lands.

- `internal/core/capture/parse_test.go` :
  `TestValidateStrictLapseRequiresLapsedAt` (a `lapse` record with no
  `lapsed_at` is refused), `TestValidateStrictLapsedAtMustBeRFC3339` (a
  free-text or date-only value is refused), `TestValidateStrictNonLapseAcceptsAbsentLapsedAt`
  (every other category is unaffected), `TestValidateStrictLapseCarriesFoundDuring`
  (a blank point in the process is refused).
- `internal/core/capture/serialize_test.go` :
  `TestLapsedAtRoundTrips` (written, re-read, and identical, so the value is not
  quietly normalised to write-up time).
- `internal/surface/cli/capture_surface_test.go` :
  `TestCaptureLapsedAtHasNoDefault` (a `lapse` capture with the flag omitted
  exits non-zero, writes nothing, and names the flag),
  `TestCaptureLapsedAtWritesTheGivenInstant`.
- `internal/core/lint/schema_parity_test.go` :
  `TestIssueRecordShapeFlagsLapseWithoutLapsedAt` (the committed-ledger gate
  refuses what the reader refuses), and a corpus fixture: one well-formed lapse
  record that the gate must pass. It sits in the parity file beside the other
  reader-mirroring issue-shape cases, which is what it is one of.
- The generated-reference drift test in `internal/surface/cli` fails until
  `docs/reference/cli/commands.md` is regenerated, which is the mechanism that
  keeps the new flag documented.

## Out of scope

- A separate lapse store, a separate enum, or a lapse-specific record type.
  Ruling (6) (2026-08-28) settles it as a value in capture's validated category
  list, and that ruling is not reopened here.
- Any judgement about whether a lapse *should* have been recorded. The log
  records what was disclosed; a gate that inferred undisclosed lapses would be
  asserting knowledge nothing in the repository has.
- Backfilling lapses from before this cycle. The first three entries are the
  ones written at the outset, and inventing earlier ones would be the
  reconstruction the intent exists to measure.
- Any change to `found_at`, `created` or `updated`.
- The `origin` and production-mode keys (itd-178), which will later say how a
  record was written. A lapse entry says the discipline gave way; the origin keys
  say who or what wrote a record. They are different questions and different
  changes.
