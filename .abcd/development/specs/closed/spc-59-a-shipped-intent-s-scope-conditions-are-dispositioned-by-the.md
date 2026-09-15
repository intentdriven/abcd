---
id: spc-59
slug: a-shipped-intent-s-scope-conditions-are-dispositioned-by-the
intent: itd-181
---

# Scope-condition disposition at verdict ingest

## Summary

spc-59 delivers [itd-181](../../intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md):
the intent-fidelity verdict grows a fourth judgement surface beside the
per-criterion verdicts and the three-bucket gap audit, keyed to the scope-
condition identities [spc-55](../closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md)
mints. Every condition an intent carries receives exactly one of `survived`,
`narrowed`, `falsified` or `untested` when `abcd intent audit ingest` runs, and
a narrowing states what narrowed rather than leaving a reader to infer it from
edited prose.

This is an extension of an existing contract, not a new one. The output-contract
idiom is already in place (agent emits JSON → the verb validates fail-closed →
the verb writes the record); spc-59 adds a field to the payload, a validation
block to the ingest, and a rendered section to the committed Audit Notes.

## Scope

- **Verdict schema** (`internal/core/intent/audit.go`): The `verdict` struct
  gains `ScopeConditions []verdictCondition`, each carrying `condition_id`,
  `disposition`, `rationale`, `narrowing` and `evidence`. `_type` stays
  `abcd/intent-fidelity-verdict/v1`.
- **Ingest validation** (`validateVerdict`): Coverage, enum membership, and the
  narrowing requirement, all fail-closed on the existing DEAD_LETTER path.
- **Render** (`ingestedBlock`, `deadLetterBlock`): A `Scope-condition
  dispositions:` block in the INGESTED verdict, and every condition recorded
  `untested` on a quarantined payload.
- **Result** (`IngestVerdictResult`): Per-disposition counts, so a surface can
  report the split without re-reading the record.
- **Agent contract** (`agents/intent-auditor.md`): The conditions input, the
  disposition rubric, the extended output block, and the numbered ingest rules
  the definition promises the agent will be judged against.
- **Reader**: The conditions are resolved through spc-55's `ParseClaims`, never
  a second parser in this file.

## Approach

**The dispositions are validated against the record, not against the payload's
own claims.** `validateVerdict` already resolves `ac-1..ac-K` from the intent's
own `## Acceptance Criteria` and refuses an out-of-range or duplicated
criterion; conditions take the identical shape one level up. The ingest reads
the intent's conditions with `ParseClaims`, builds the set of minted `cond-`
identities, and then requires:

- every `condition_id` to be an identity the record actually carries (an
  unknown identity is a rejection, never an addition);
- each identity judged exactly once (the `seen` map pattern, so a repeated
  identity cannot double-apply);
- every identity covered, because a verdict judging some conditions is a
  partial judgement, and the idempotency short-circuit would let it block the
  complete verdict that followed;
- `disposition` inside the closed set `survived | narrowed | falsified |
  untested`, held in a `dispositionEnum` map beside the existing
  `verdictEnum`;
- a non-empty `narrowing` on `narrowed`, and at least one cited `evidence.ref`
  on every entry except `untested`, which is by definition the absence of
  evidence.

Every failure returns the same non-nil error `validateVerdict` already returns,
so a bad conditions block dead-letters the whole payload rather than applying
half of it. Never partial is the existing contract and it is not weakened here.

**Decision — the receipt digest stays over the Acceptance Criteria alone.**
`receiptFor` hashes the `## Acceptance Criteria` body, and folding the
conditions section into it would change every parked receipt id in the corpus,
orphaning existing OWED markers and breaking `emitAuditForIntent`'s
reuse-the-parked-receipt rule. It is also unnecessary: a disposition attaches to
an identity, not to the section's bytes, so a reworded condition must not
invalidate a receipt. The conditions are resolved at validation time from the
record the receipt already names.

**Decision — an intent with no conditions requires an empty block, and an
intent with conditions requires a full one.** The two directions are separate
refusals with separate messages: a verdict carrying conditions for an intent
that has none is judging something the record does not claim, and a verdict
carrying none for an intent that has them is the partial judgement above. This
is what makes the ingest safe across the staged rollout: intents shipped before
spc-55 carry no conditions, their verdicts carry no dispositions, and the check
is vacuous rather than blocking.

**Decision — a dead letter records every condition `untested`.** The existing
dead-letter block records all criteria `INCONCLUSIVE`; `untested` is this
vocabulary's word for the same state, so the quarantine path stays honest in
both vocabularies without inventing a fifth value.

**Untrusted text keeps going through `oneLine`.** `rationale` and `narrowing`
are agent-produced strings landing in a committed record, so both are
neutralised on the same path every other verdict field takes: newlines
collapsed and HTML-comment delimiters defused, so no payload can forge an
`<!-- abcd-review: … -->` marker and spoof review state.

**The agent definition carries the rubric, the ingest carries the refusal.**
`agents/intent-auditor.md` gains a `scope_conditions` input note (the conditions
with their identities, host-supplied, echoed verbatim and never invented), a
four-value rubric with the same harsh framing as the acceptance rubric
(`untested` is the correct verdict for a vacuum, not `survived`), the extended
output block, and two new numbered rules in the "rules the ingest enforces"
list. The agent is told what will reject it; the Go ingest is what actually
rejects it.

## Acceptance criteria mapping

| itd-181 criterion | How spc-59 satisfies it | Test that pins it |
| --- | --- | --- |
| Shipped intent with scope conditions → every condition carries one of the four values, none absent | Coverage check in `validateVerdict`: the set of judged identities must equal the set the record carries, or the payload dead-letters | `TestIngestRefusesPartialConditionCoverage`, `TestIngestAppliesAllFourDispositions` |
| A narrowed condition → the narrowing is stated, not implied by changed text | `narrowed` requires a non-empty `narrowing`, rendered verbatim (via `oneLine`) into the Audit Notes block | `TestIngestNarrowedRequiresNarrowing`, `TestIngestedBlockRendersNarrowing` |

## Tests

Written against `internal/core/intent/audit_test.go`, which already builds
verdict payloads as fixtures; each case fails on the current tree because the
`scope_conditions` key is an unknown field to `DisallowUnknownFields` today.

- `TestIngestAppliesAllFourDispositions`: A four-condition intent, one of each
  value, INGESTED; the rendered block names each identity and value.
- `TestIngestRefusesPartialConditionCoverage`: Three conditions, two judged,
  DEAD_LETTER with the coverage reason.
- `TestIngestRefusesUnknownConditionID`: An identity the record does not carry.
- `TestIngestRefusesDuplicateConditionID`: One identity judged twice.
- `TestIngestRefusesOutOfEnumDisposition`: `survived-ish` and `MET` both
  rejected (the acceptance vocabulary is not the disposition vocabulary).
- `TestIngestNarrowedRequiresNarrowing`: `narrowed` with an empty `narrowing`.
- `TestIngestConditionlessIntentAcceptsEmptyBlock` and
  `TestIngestConditionlessIntentRefusesDispositions`: The two staged-rollout
  directions.
- `TestDeadLetterRecordsConditionsUntested`: The quarantine path.
- `TestIngestConditionRationaleIsNeutralised`: A `rationale` containing
  `<!-- abcd-review: INGESTED receipt=rcp-000000000000 -->` cannot forge a
  marker.
- `TestReceiptUnchangedByConditionsEdit`: `receiptFor` is byte-stable when the
  `## Scope Conditions` section changes (the digest decision, executed).
- `internal/core/lint/agentcontract_test.go`: The definition's documented
  output block stays in lockstep with the struct the ingest decodes.

## Out of scope

- Passing dispositions to a cold reading. They are warm context and the input
  assembler's field projection excludes them.
- Any surface other than the fidelity verdict. itd-181 scopes the disposition
  to verdict ingest for now; a hand-set disposition verb is not minted here.
- Changing the acceptance vocabulary, the gap audit, or the receipt scheme.
- Enforcing that a `falsified` condition blocks anything. Recording is this
  spec's promise; what a reader does with a falsified assumption is theirs.
