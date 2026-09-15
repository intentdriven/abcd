---
id: spc-56
slug: every-record-written-through-a-command-carries-its-origin-an
intent: itd-178
---

# Origin and production-mode keys, stamped by the command

## Summary

spc-56 delivers [itd-178](../../intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md):
two frontmatter keys, `origin` and `production_mode`, written only by the
commands that mint records, plus the lint that reports a record carrying them in
a shape no command could have produced. Neither key touches authorship. They are
disclosure at field granularity, on the same footing as the `Assisted-by:`
trailer at commit granularity.

Population is forward-only. Existing records are untouched, sparseness is
information, and an absent stamp is never backfilled.

## Scope

- **Vocabulary** (`internal/core/provenance/`, new leaf package): The two closed
  value sets and the one parser that reads and renders them, held as data on the
  `internal/core/issueschema` and `internal/core/changelog` precedent, so the
  writers (`core/intent`, `core/capture`, `core/spec`) and the gate
  (`core/lint`) read one copy. Standard library only; no new dependency.
- **Resolver support** (`internal/core/frontmatter`): None needed. Both keys are
  single-line scalars, so the shared line scanner reads them unchanged. This is
  a deliberate constraint on the encoding, not an accident.
- **Issue schema** (`internal/core/issueschema/issueschema.go`): Both keys added
  to `Known`. Without that, capture's reader refuses every stamped record and
  skips it, which is exactly how a previous flag shipped unable to execute.
- **Write paths**: `intent.seedDraft` (drafts), `spec.Create` (specs),
  `capture.commitCapture` via `buildIssueText` (issues), and the mutating
  transitions `capture.transition` and `capture.Promote` via `setScalarField`.
- **Attribution seam** (`internal/core/identity`): The repo's default production
  mode is read from the existing `.abcd/config/identity.json` pin (itd-91), as
  an optional member on `identity.Pin`. No second config file and no second
  reader.
- **Lint** (`internal/core/lint`): A new `record_provenance` rule reporting a
  record whose keys are in a state no write path produces.
- **Surfaces**: A closed-choice `--production-mode` flag on the record-minting
  verbs, and the two keys documented in
  [`brief/04-surfaces/05-intent.md`](../../brief/04-surfaces/05-intent.md) and
  `commands/capture.md`.

## Approach

**Decision — both keys are single-line scalars, and the reading pointer rides in
the `origin` value.** A nested mapping (the `resolved_by` shape) is readable by
capture's own YAML subset but invisible to `frontmatter.Fields`, the same-line
scanner every intent and spec reader uses; a nested `origin` would therefore
need a second record parser, which this repository forbids. So the grammar is:

```
origin: researcher-authored
origin: extracted-from-record
origin: contributed-by-reading <run-id>/<item-id>
production_mode: hand-written | dictated-and-formatted | scribe-transcribed
```

`provenance.ParseOrigin` is the one predicate that reads it, and the one the
lint calls. The value contains no `: ` sequence, so it stays a plain YAML scalar
to every reader in the corpus.

**Decision — an enum flag is not free text.** itd-178 bars a flag that carries
either key as free text; a closed-choice flag validated against the vocabulary
before anything is written is the same shape as `--impact` and `--severity`,
both of which already stamp machine-read enums. So `--production-mode` exists,
defaults to the repo's declared default (and to `hand-written` when the pin
declares none), and is refused outright on any value outside the set. `origin`
has no flag at all: it is derived from which command ran.

- `researcher-authored`: The default for a verb invoked by a person.
- `extracted-from-record`: Stamped automatically by `capture.Promote`, the one
  shipped path that derives a record from another record.
- `contributed-by-reading`: Stamped only by the reading-ingest verb, which
  carries the run and item identifiers already and never asks an operator for
  them.

**Decision — `origin` is stamped at mint and never rewritten; `production_mode`
may be restamped by a mutation that adds text.** A record's arrival path is a
fact about how it came to exist and does not change when someone resolves it. A
resolution note, by contrast, is new text with its own production mode, so
`capture.transition` restamps `production_mode` when the caller declares one and
leaves it alone otherwise. This keeps the pair meaningful without making either
key a running log.

**Decision — a restamp of a record carrying no `origin` is REFUSED, before any
write.** Forward-only population means every record committed before these keys
existed carries neither, and none is backfilled. Restamping such a record would
append a lone `production_mode`, which is precisely the shape `record_provenance`
reports as a state no command produced — the gate firing on a record the command
had just written. The pair is written together or not at all, so the transition
refuses (`this record predates disclosure; nothing to restamp`) and names the
remedy: re-run without `--production-mode`. The refusal is about the RESTAMP and
never about the transition — an unstamped record stays resolvable, because
forward-only population must not strand a record nobody can close.

**The attribution seam is extended, not duplicated.** `identity.Pin` gains an
optional `ProductionMode` member; `LoadPin` validates it against the vocabulary
and returns an error on an unknown value, exactly as it already errors on a
malformed pin. The self-contained pre-commit identity guard seds `name` and
`email` out of that file by name, so an added member is invisible to it, and
`WritePin`'s existing refusal of quotes, backslashes and control characters is
untouched. An absent member means `hand-written`.

**The lint reports what no write path could have produced.** `record_provenance`
walks the record stores through `lint.LoadRecordGraph` (the one canonical record
scan `core/site` already consumes) and reports four states:

- a value outside its closed set;
- one key present without the other, because every write path stamps both
  together;
- `origin: extracted-from-record` on an intent carrying no `promoted_from`
  back-edge, which no promote could have written;
- `origin: contributed-by-reading` whose run and item identifiers do not resolve
  to a reading record.

**The residual is stated, not hidden.** A hand edit that types a legal value in
a legal combination is byte-identical to a command's write, and no lint over
committed bytes can tell them apart. Closing that gap needs a stamp the record
cannot forge, which is a keyed digest the repository has no secret to hold, and
itd-178 scopes no such mechanism. `record_provenance` therefore catches
implausible hand edits, not all of them, and the rule's own message says so.
The honest claim this spec makes is "the lint reports a hand-edited record", and
the honest bound is "in every shape a command could not have written".

**Reading-record resolution ships armed, exercised by fixture.** The reading
store is itd-180's to create. Until it lands, no command in this repository can
mint `contributed-by-reading`, so the fourth check is proved against a fixture
reading record and has no production input. That is a sequencing fact, disclosed
here rather than discovered later.

**Free text is redacted on the way in, as everywhere else.** Neither key carries
free text, so neither adds a redaction surface; the run and item identifiers are
shape-validated ids. This is stated because it is the question a reviewer will
ask of any new committed field.

## Acceptance criteria mapping

| itd-178 criterion | How spc-56 satisfies it | Test that pins it |
| --- | --- | --- |
| A record written through a command carries both keys, neither supplied as free text by the operator | All four mint paths stamp both; `origin` has no flag, and `--production-mode` is enum-validated before any write | `TestSeedDraftStampsProvenance`, `TestSpecCreateStampsProvenance`, `TestCommitCaptureStampsProvenance`, `TestProductionModeFlagRefusesFreeText` |
| `origin: contributed-by-reading` resolves its item and run identifiers to a reading record | `provenance.ParseOrigin` extracts both ids; `record_provenance` resolves them against the reading store and reports a dangling pair | `TestParseOriginReadingPointer`, `TestRecordProvenanceReportsUnresolvableReading` |
| A hand-edited record carrying either key is reported by the lint | The four `record_provenance` states above, over `LoadRecordGraph` | `TestRecordProvenanceLoneKey`, `TestRecordProvenanceOutOfVocabulary`, `TestRecordProvenanceExtractedWithoutPromotedFrom` |

## Tests

Each fails first: `internal/core/provenance` does not exist, `issueschema.Known`
refuses both keys today, and no write path emits them.

- `internal/core/provenance/provenance_test.go`:
  `TestParseOriginThreeForms`, `TestParseOriginRejectsUnknownKind`,
  `TestParseOriginReadingPointer` (a missing or malformed id pair is refused),
  `TestProductionModeVocabularyIsClosed`.
- `internal/core/intent/create_test.go`: `TestSeedDraftStampsProvenance`.
- `internal/core/spec/store_test.go`: `TestSpecCreateStampsProvenance`.
- `internal/core/capture/serialize_test.go`:
  `TestCommitCaptureStampsProvenance`, plus
  `TestCaptureReaderAcceptsProvenanceKeys` (the `Known` allow-list, proved
  through the reader rather than asserted about the map).
- `internal/core/capture/promote_test.go`:
  `TestPromoteStampsExtractedFromRecord`.
- `internal/core/capture/workflow_test.go`:
  `TestTransitionRestampsProductionMode`, `TestTransitionLeavesOriginAlone`,
  `TestTransitionRefusesRestampOnUnstampedRecord`.
- `internal/core/identity/identity_test.go`:
  `TestLoadPinReadsProductionModeDefault`,
  `TestLoadPinRefusesUnknownProductionMode`,
  `TestLoadPinAbsentMemberDefaultsHandWritten`.
- `internal/core/lint/schema_test.go`: The three `record_provenance` cases in
  the table, plus `TestRecordProvenanceSilentOnUnstampedRecord` (forward-only
  population is not a finding).
- `internal/surface/cli/capture_surface_test.go` and `intent_cli_test.go`:
  `--production-mode` accepted, defaulted, and refused out of vocabulary.

## Out of scope

- Passing either key to a cold reading. Both are excluded by the input
  assembler's field projection, and the read-block eval asserts it.
- The production-mode vocabulary itself, which is the ruled authorship account's
  decision. spc-56 ships the mechanism that stamps it.
- Backfilling existing records. Forward-only is the ruled population property.
- Any claim about authorship or credit. `ACKNOWLEDGEMENTS.md` and the
  `Assisted-by:` trailer are unchanged and untouched.
- Minting the reading store or the reading-ingest verb.
