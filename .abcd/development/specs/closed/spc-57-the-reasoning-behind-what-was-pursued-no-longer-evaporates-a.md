---
id: spc-57
slug: the-reasoning-behind-what-was-pursued-no-longer-evaporates-a
intent: itd-179
---

# Grounds at conjecture granularity

## Summary

spc-57 delivers [itd-179](../../intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md):
a grounds argument on the readiness gate and on capture's triage routes, with a
three-value vocabulary (`pursued`, `deferred`, `declined`) and a refusal when it
is absent. Grounds are recorded today only for deliberate non-action, where a
wontfix carries a note and an ADR carries its alternatives; the reasoning behind
what went forward has no home and evaporates at the gate.

The grounds name the conjecture being acted on, not the route taken. "Planned it
because it is next" is a restatement of the decision; "planned it because we
expect a stamped identity to survive rewording, which nothing else does" is a
conjecture somebody can later find wrong.

## Scope

- **Vocabulary** (`internal/core/grounds/`, new leaf package): The three values,
  the `<token>: <text>` grammar, the parser and the renderer, held once so the
  intent writer, the ledger writer and the lint read one copy. Standard library
  only; no new dependency.
- **Intent side** (`internal/core/intent/grounds.go`, new):
  `RecordGrounds(repoRoot, intentID string, g grounds.Grounds) (GroundsResult,
  error)` appends one entry to the intent's `## Grounds` section, creating the
  section when it is absent.
- **Readiness gate** (`internal/core/intent/ready.go`): A `grounds` check,
  reported last, failing when the record carries no well-formed entry.
- **Ledger side** (`internal/core/capture`): A `Grounds` member on
  `PromoteRequest`, `ResolveRequest` and `WontfixRequest`, APPENDED to the
  record's `## Grounds` section through the shared record form, inside the
  existing ledger lock and atomic transition.
- **Issue schema** (`internal/core/issueschema/issueschema.go`): `grounds`
  TOLERATED in `Known` as a legacy key a record written by an older abcd may
  carry, its value read by nothing; the record lint's `record_schema` blocks a
  frontmatter `grounds:` and names the section it belongs in.
- **Surfaces** (`internal/surface/cli/cli.go`, `commands/intent.md`,
  `commands/capture.md`): `--grounds` on `abcd intent ready`,
  `abcd capture promote`, `abcd capture resolve` and `abcd capture wontfix`.

## Approach

**Decision — `intent.Ready` stays read-only; the flag lives on its verb.**
`Ready` is documented and tested as a reporter that never mutates the store, and
that contract is worth more than flag placement. So `abcd intent ready <itd-N>
--grounds pursued:"<text>"` is wired in the CLI as two calls: `RecordGrounds`
first, then the unchanged `Ready`. The write is the flag's whole effect; the
report is unchanged by it. The exit-code contract holds as it stands, with a
failed grounds write mapping to 2 (a structural fault), never to 1.

**Decision — the gate is a refusing check, not a prompt.** Without the flag, an
intent carrying no grounds fails the `grounds` check with the remedy naming the
flag and the vocabulary. This is the refusing-gate idiom the readiness gate
already is: exit 1, the rendered report is the output, and no interactive
question is asked of a possibly-unattended caller.

**Decision — grounds live on the intent record, not on the spec.** The
conjecture is the intent's, and the gate that reads it takes an `itd-N`. The
pre-tooling stand-in was a `## Grounds (pursued)` section on the spec, named as
such in its own text; the change that lands this spec moves every such entry up
to its intent and retires the stand-in sections, so the corpus has one home for
grounds rather than two. Thirteen specs carry one, open and closed alike — the
migration covers all of them, not the four the record named when this was
written, because a stand-in left standing anywhere is a second home.

**Decision — wontfix records `declined` and needs no new required flag.**
`capture.transition` already refuses an empty `wontfix_reason`, so a wontfix can
never be recorded without grounds; what it lacks is the type. Wontfix therefore
appends `- declined: <reason>` from the reason it already takes, and
`--grounds` overrides the text when the conjecture is worth stating separately
from the user-facing reason. Promote and resolve, which have no such note,
refuse without the flag.

**The grammar is one line, parsed once, and the record form is one section.**
`<token>: <text>` is a single line; `grounds.Parse` splits on the first colon,
validates the token against the closed set, and returns the text. Both record
families carry it as a bullet `- pursued: <text>` under `## Grounds`, so the
record reads as prose and the gate reads it as data with no second parser.

**Recording is APPEND-ONLY, on both sides.** A frontmatter scalar is SET, so the
ledger's first shape let a resolve overwrite the conjecture the promote before it
recorded — silently, with a success result, on the sequence fourteen resolved
records already follow (iss-2608301657354776). Refusing the overwrite was never
open: promote, resolve and wontfix all REQUIRE grounds, so a refusal would make a
promoted issue impossible to resolve. The earlier conjecture is precisely what a
later reader checks the outcome against, which is the argument the intent half
already made for the same data, so the two halves now share ONE record form
rather than holding two.

**The substance floor is a shape check, and it says so.** "Name the conjecture,
not only the decision" is a review property; no machine reads a sentence and
knows whether it names a conjecture. What ships is a floor that refuses the
degenerate cases: empty or whitespace-only text, text below either half of the
floor — the LEXICAL-UNIT count and the LETTER count, neither of which a text of
twenty zero-width spaces can clear, a unit being a letter-run where the script
separates words and a single LETTER where it does not, so that a script written
without inter-word spaces is not refused for having one run, and the letter
count reading letters rather than runes so that padding does not answer it —
text carrying a control
character the frontmatter serialiser refuses (below U+0020, the class
`yamlScalar` will not write, which is narrower than what a record cannot hold:
the store carries DEL, C1, the line separator and the bidi overrides), and text
that is only the vocabulary token or the verb's own name repeated. The
substantive requirement is carried by the prompt in `commands/intent.md` and
`commands/capture.md`, where the interview asks for the expectation and its
falsifier. The spec claims the floor, not the judgement.

**Free text is redacted before it is written.** A grounds text is operator prose
landing in a committed record, so it goes through the same redactor the note
already goes through: `redactLedgerText` on the ledger side and
`redactIntentText` on the intent side, before validation, never after, so no
rewritten span reaches a field the validator has already passed. The redaction
counters surface on the result exactly as `Redacted` and `Degraded` already do,
because rewriting somebody's reasoning silently is worse than not recording it.

**Decision — ADR grounds are untouched.** Decision-granularity grounds stay in
the ADR family's Alternatives Considered. This spec adds the finer grain beside
them and takes no position on the coarser one.

**Staging.** The recording path, the vocabulary and the writers land first with
the `grounds` check reporting `OK`; the refusal is promoted in a second commit.
The promotion is deliberately forward-only rather than staged behind a populated
corpus: measured at the branch tip, 10 of the 66 `planned/` records carry an
entry, 56 fail the grounds check, and 36 of those were READY before this change
and are NOT READY after it. Each records its grounds when it is next picked up,
which is the moment the conjecture is still known — the cost this buys is that a
third of the planned bucket answers the gate before it can be implemented.
Promote and resolve refuse from the first commit, because they mint the grounds
in the same call and have no corpus to fix.

## Acceptance criteria mapping

| itd-179 criterion | How spc-57 satisfies it | Test that pins it |
| --- | --- | --- |
| A capture routed to an intent draft, triaged without grounds → the command refuses | `PromoteRequest.Grounds` is validated before any mint or write, and an absent value returns an error with nothing written | `TestPromoteRefusesWithoutGrounds`, `TestPromoteWithoutGroundsWritesNothing`, CLI `TestCapturePromoteMissingGroundsExit2` |
| A recorded gate decision → the grounds name the conjecture, not only the decision | The three-value token plus the substance floor, with the conjecture-naming prompt on both surfaces | `TestGroundsParseVocabulary`, `TestGroundsRefusesDegenerateText`, `TestRecordGroundsWritesEntry` |

## Tests

Each fails first: `internal/core/grounds` does not exist, `ReadyResult` has no
`grounds` check, and the three ledger requests have no `Grounds` member.

- `internal/core/grounds/grounds_test.go`: `TestGroundsParseVocabulary` (three
  values in, everything else refused), `TestGroundsParseSplitsOnFirstColon` (a
  text containing a colon survives), `TestGroundsRefusesDegenerateText` (empty,
  whitespace, below the floor, token-only, verb-name-only),
  `TestGroundsRenderRoundTrip`.
- `internal/core/intent/grounds_test.go`: `TestRecordGroundsWritesEntry`,
  `TestRecordGroundsCreatesSectionWhenAbsent`,
  `TestRecordGroundsAppendsSecondEntry`,
  `TestRecordGroundsRedactsText`, `TestRecordGroundsRefusesUnknownIntent`.
- `internal/core/intent/ready_test.go`: `TestReadyGroundsAbsentFails`,
  `TestReadyGroundsPresentPasses`, `TestReadyChecksOrderAndCount` extended for
  the trailing check.
- `internal/core/capture/promote_test.go`:
  `TestPromoteRefusesWithoutGrounds`, `TestPromoteWithoutGroundsWritesNothing`
  (no orphan draft), `TestPromoteStampsGrounds`.
- `internal/core/capture/workflow_test.go`: `TestResolveRefusesWithoutGrounds`,
  `TestResolveStampsGrounds`, `TestWontfixStampsDeclinedFromReason`,
  `TestWontfixGroundsOverride`, `TestGroundsTextIsRedacted`.
- `internal/core/lint/schema_test.go`:
  `TestCaptureReaderAcceptsGroundsKey` (the `Known` allow-list proved through
  the reader) and `TestRecordSchemaFlagsOutOfVocabularyGrounds`.
- `internal/surface/cli/intent_cli_test.go`:
  `TestIntentReadyGroundsFlagRecordsThenReports` (the write happens, the report
  is unchanged, exit 0), `TestIntentReadyGroundsWriteFailureExits2`.

## Out of scope

- Rewriting the ADR family's grounds. Decision-granularity grounds stay where
  they are.
- Any semantic judgement of whether a grounds text really names a conjecture.
  The floor refuses degenerate text; the rest is review.
- A grounds argument on capture's create path. itd-179 scopes triage, and an
  observation being filed is not yet a conjecture being pursued.
- Retro-fitting grounds onto `shipped/`, `superseded/`, `resolved/` or
  `wontfix/` records. Population is forward-only.
- Reading grounds into a cold reading. They are warm context.
