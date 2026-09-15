---
id: spc-55
slug: an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t
intent: itd-177
---

# Claim typing and scope-condition identity

## Summary

spc-55 delivers [itd-177](../../intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md):
the intent record's three claim kinds become machine-readable, and every scope
condition gains a minted identity that survives a rewrite of its own text. The
recording requirements themselves are not this spec's to invent: they are the
[claim-recording-gradient discipline](../../intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md)'s
rule, and the two sections are
[adr-51](../../decisions/adrs/0051-intents-declare-mechanism-and-scope-conditions.md)'s
format. What lands here is the parse, the mint, the render, and the refusal.

The identity is the load-bearing half. A disposition attaches to a condition,
not to a sentence, so [spc-59](spc-59-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md)
has something stable to key on when the fidelity verdict arrives.

## Scope

- **Claim parse** (`internal/core/intent/claims.go`, new): One reader for the
  `## Mechanism` and `## Scope Conditions` sections, returning the three byte
  states the gradient distinguishes (absent, empty, nullity token) plus the
  parsed conditions with their identity markers. It composes the section
  primitives `audit.go` already owns (`sectionBody`, `headingRe`, `bulletRe`),
  so the corpus keeps one notion of what a section and a top-level bullet are.
- **Readiness gate** (`internal/core/intent/ready.go`): Two new checks,
  `mechanism_claim` and `scope_conditions`, added to `ReadyResult.Checks` in
  fixed order after `acceptance_criteria`. `Ready` stays a read-only reporter:
  the checks report, they never stamp.
- **Identity mint** (`internal/core/intent/lifecycle.go`): `Plan` stamps a
  marker onto every unmarked scope-condition bullet before it moves the record
  drafts → planned, and does that step alone — no spec, no bucket move — when
  run on a record already in `planned/`, so a condition written after planning
  still reaches the mint and the gate's remedy is a command that works. A run
  with nothing unmarked refuses. It uses `recordid.Minter.Mint` (the
  [adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md)
  mint). Markers are never hand-typed.
- **Create-path scaffold** (`internal/core/intent/create.go`): `seedDraft`
  gains both sections with their one-line contracts, so a drafted intent
  arrives already carrying the prompt rather than being told about it later.
- **Surface** (`internal/surface/cli/cli.go`, `commands/intent.md`): The new
  checks render in both the text and `--json` forms of `abcd intent ready`;
  the plugin surface's planning interview gains the mechanism prompt and the
  scope-condition elicitation as named steps.
- **Corpus pass**: The `planned/` bucket is populated with conditions or the
  nullity token in the same change set that promotes the refusal.

## Approach

**The parse is one function, not four call sites.** `ParseClaims(content)
Claims` returns `Mechanism ClaimState`, `Conditions []ScopeCondition` and
`ConditionsState ClaimState`, where `ClaimState` is one of `absent`, `empty`,
`nullity` or `stated`. Everything downstream (the gate, the `Plan` stamp, the
`--json` render, and spc-59's verdict validation) asks this one reader, on the
`countAcceptanceCriteria` precedent: three gates already share one criterion
parser precisely so they cannot disagree about what a criterion is.

**The nullity token is exact.** `None stated.` alone on its line under the
heading, matched anchored after a `\r` trim, is the only spelling that reads as
a recorded nullity. Prose that merely contains the word "none" is `stated`, and
a heading with nothing but blank lines under it is `empty`. The three states
map to three gate outcomes and are never collapsed.

**Decision — the marker grammar.** A condition's identity is an HTML comment
carried anywhere in the bullet, written at the end of its first line:
`<!-- cond: cond-<16 digits> -->`. It is READ anywhere in the bullet, including
a folded continuation line, because a file wrapped at 80 columns gets rewrapped
and a positional read would orphan the identity (iss-2608300235377731). The comment
form is the repository's existing machine-marker idiom (`audit.go`'s
`<!-- abcd-review: … -->`), it survives markdown rendering invisibly, and it
keeps the condition's prose exactly what a human wrote. The family tag `cond`
is minted through `recordid.Minter.Mint("cond")`, which already validates a
lowercase family before it is spliced anywhere. `cond` is deliberately not a
record store: the marker identifies a claim inside a record, and `abcd <id>`
dispatch is untouched.

**Decision — the observable surface is `abcd intent ready --json`, not
`abcd intent --json`.** itd-177 names the latter, but `abcd intent` with no
argument renders `intent.Status`, a corpus-wide count-and-link summary with no
per-record body; folding every intent's conditions into it would make a status
render read every file for a payload nobody asked for. The per-intent gate
already takes an `itd-N`, already emits `--json`, and is the exact moment the
identities matter, so `ReadyResult` gains `Conditions []ScopeCondition` and the
identity criteria assert against that payload.

**The gate refuses, it does not repair.** A missing or duplicated marker is
reported by name with the remedy `abcd intent plan <itd-N>` (the write-capable
verb), never silently minted at read time: a reporter that writes is a reporter
whose output depends on who ran it. `Plan` is idempotent here, so re-running it
on a record whose conditions were edited stamps only the unmarked bullets —
which is why the verb accepts a planned record for the stamp step and not only
a draft (iss-2608300210588874).

**Identity lifecycle is a stamping rule, not a diffing engine.** An edit keeps
its marker because the marker is bytes in the bullet and nothing rewrites it. A
split keeps the marker on the first part and `Plan` mints for the unmarked
second. A merge keeps the surviving marker; the retired one simply stops
appearing, and its absence surfaces as `narrowed` at the next fidelity verdict
(spc-59's business). A deletion is the same shape: the marker stops appearing,
and nothing here reads the record's history to notice — the gate judges the
record in front of it, and a retired identity is spc-59's to account for. No
text similarity is computed anywhere:
the intent asks for identity that survives rewording, and a stamped token
delivers exactly that with no heuristic to be wrong.

**Discipline records are exempt.** `parseIntent` already carries the bucket, so
both checks return `OK` with the detail "discipline records carry no claim
sections" for `BucketDisciplines`, matching the gradient's own exemption.

**Staging is loud and two-step.** The parse, the `Plan` stamp, the scaffold and
the `--json` render land first with both checks reporting `OK` and a detail
naming what is missing. The refusal is promoted in the second commit, after the
`planned/` corpus pass, so the gate never arrives as a wall of pre-existing
failures. The window in which the format precedes its enforcement is stated in
`commands/intent.md` rather than papered over.

## Acceptance criteria mapping

| itd-177 criterion | How spc-55 satisfies it | Test that pins it |
| --- | --- | --- |
| No conditions and no nullity → gate exits non-zero, names the missing field | `scope_conditions` fails on `ClaimState` `absent`, detail naming `## Scope Conditions`, remedy naming the nullity token | `TestReadyScopeConditionsAbsent` (core), `TestIntentReadyMissingConditionsExit1` (CLI) |
| Edited condition text → identity unchanged | The marker is bytes inside the bullet; `ParseClaims` reads it positionally and `Plan` stamps only unmarked bullets | `TestConditionIdentitySurvivesEdit` |
| `## Mechanism` carries the nullity token → gate passes, reports the recorded nullity | `mechanism_claim` returns `OK` with detail "mechanism claim declined (nullity recorded)" on `ClaimState` `nullity` | `TestReadyMechanismNullityPasses` |
| `## Mechanism` present but empty → gate exits non-zero, names the section | `ClaimState` `empty` is a distinct state and fails with the write-the-claim-or-the-token remedy | `TestReadyMechanismEmptyFails` |
| Duplicated or missing marker → gate exits non-zero, names the fault | `scope_conditions` reports each offending bullet by its ordinal and, for a duplicate, by the repeated `cond-` id | `TestReadyConditionMarkerMissing`, `TestReadyConditionMarkerDuplicated` |

## Tests

Every case below is written to fail against the current tree first: `ParseClaims`
does not exist, `ReadyResult.Checks` is asserted at length four in
`ready_test.go`, and `Plan` writes no markers.

- `internal/core/intent/claims_test.go`: `TestParseClaimsThreeByteStates`
  (absent, empty, nullity, stated for both sections), `TestNullityTokenIsExact`
  (a "none stated" line without the full stop, or with trailing prose, is
  `stated`), `TestParseConditionsMarkerExtraction`,
  `TestConditionIdentitySurvivesEdit`, `TestConditionMarkerSurvivesAReflow`,
  `TestConditionWithTwoMarkersIsAFault`; the stamp's own cases
  (`TestStampScopeConditionsMarksOnlyUnmarkedBullets` — the split case,
  `TestStampScopeConditionsIsIdempotent`,
  `TestStampScopeConditionsRedrawsOnCollision`) and `Plan` end to end
  (`TestPlanStampsConditionIdentities`, `TestPlanStampsAPlannedRecordInPlace`,
  `TestPlanOnAPlannedRecordWithNothingToStampRefuses`,
  `TestSeedDraftCarriesClaimSections`). They live beside the code under test
  rather than in a `lifecycle_test.go`/`create_test.go` split, matching where
  this package already keeps its `Plan` and create cases.
- `internal/core/intent/claims_fence_test.go`: the fence, duplicate-heading,
  malformed-marker and mint-lock cases, plus
  `TestFenceAwareBoundLeavesEveryAuditReceiptUnchanged`, which pins every
  section body in the real corpus byte-identical across the fence-aware bound —
  the acceptance-criteria body's sha256 is a parked review receipt.
- `internal/core/intent/ready_test.go`: The five gate cases in the table, plus
  `TestReadyChecksOrderAndCount` (six checks, fixed order),
  `TestReadyDisciplineExemptFromClaimChecks`,
  `TestReadyClaimChecksNotApplicableInTerminalBuckets`,
  `TestReadyScaffoldPromptIsNotAClaim` and
  `TestReadyReportsStructuralConditionFaults`.
- `internal/surface/cli/intent_cli_test.go`:
  `TestIntentReadyJSONRendersConditionIdentities` (the identity's observable
  surface), `TestIntentPlanStampsAPlannedRecord` (the remedy the gate names
  runs on the record that printed it) and the exit-code cases for the two new
  refusals.

## Out of scope

- The gradient's rationale, its staging argument and its exemptions: the
  discipline record itd-190 owns them, and this spec implements rather than
  restates them.
- Scope-condition dispositions: spc-59 populates them at verdict ingest. spc-55
  only makes them attachable.
- Any auditor-side flag for a mechanism-nullity intent: the gradient defers it
  to the record that owns verdicts.
- Retro-fitting claims onto `shipped/` or `superseded/` records. Population is
  forward-only, and an absent stamp is never backfilled.
