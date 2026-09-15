---
id: itd-189
slug: what-the-widening-reading-proposes-is-admitted-or-declined-o
spec_id: spc-67
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# What the widening reading proposes is admitted or declined on the record — grounds on admission, dispositions on declines, and surprises as their own entries

## Press Release

> **Declining a proposal costs nothing epistemically; admitting one is
> where the frame is engaged.** When the widening reading proposes
> candidate framings, every admission into the candidate set carries
> recorded grounds, and every declined proposal carries a disposition — so
> the record can show that not every contributed proposal was taken, since
> uniform adoption is equally consistent with abdication. Separately, a
> surprise entry records what was unexpected, distinct from the
> disposition record, because the reading's output and the researcher's
> response are different acts — and the surprise that occasions abduction
> is a third thing again.

## What's In Scope

- The admission-grounds record: keyed to the proposal, grounds free text,
  written at admission (the analogy of the `disposition_grounds` required on
  rejection).
- The declined-proposal disposition: keyed to the proposal, on the ledger
  side.
- The surprise entry: a distinct record shape, keyed to whatever
  occasioned it (a detection, a consequence), never folded into a
  disposition.
- Schema now; **[HAND]** in Iteration 1 (no reading runs, so nothing to
  record yet); enforced at the command in Iteration 2.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a widening proposal admitted to the candidate set, **when**
  the admission is recorded, **then** it carries grounds, and a blank is
  refused once command enforcement lands.
- **Given** a declined widening proposal, **when** the session ends,
  **then** its disposition exists on the ledger side.
- **Given** a surprise, **when** it is recorded, **then** the surprise
  entry is a distinct record from any disposition it relates to.

## Grounds

- pursued: Pursued now because declining a proposal costs nothing epistemically while admitting one is where the frame is actually engaged, and uniform adoption of everything a reading proposes is equally consistent with careful judgement and with abdication. Only a record that carries grounds on each admission and a disposition on each decline can tell the two apart, and it has to exist before the first widening run rather than after it.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-6b45c296b217 -->
Fidelity review — receipt rcp-6b45c296b217 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:cdecb6e299d97e8168d22d1789d66040578701979edb31f38015f8b55b0ab775
Input attestations: diff:932629f9...build/itd-189 (803acdad)@sha256:e7b265907cf8c5903db50e7bbe7b8c9092bbf13ba751091f668b92147659e931;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The admission record exists, declares `grounds` required from ONE list, and the gate that refuses a blank one is wired to the real config and to preflight — not to a test-only fixture. I ran it rather than reading it: over 25 spellings of `grounds` driven end to end through `Lint` against the adm store, the nine spellings the enumeration names (empty, whitespace, '""', "''", [], {}, ~, null, NULL, !!null) are each refused as a blocker, and an absent `grounds` is refused too. CONCERNS, three, and the first is the consequential half of this verdict. (i) The refusal is a SPELLING test and its complement is live: sixteen nothing-carrying spellings pass GREEN — `!!null null`, `!<tag:yaml.org,2002:null>`, `!!str ''`, `!!str ""`, `!!null ~`, `&anchor`, `*alias`, `!!seq []`, `!!map {}`, a bare U+200B zero-width space, and six trailing-comment forms including `"" # nothing here`, `null # x`, `[] #x`, `~ #x` and a comment-only value. That is not the whole of it: I then drove the SAME records through the outstanding report and every one of the six I tested returned 0 findings — the gate accepts an admission that admits nothing AND the report then calls the proposal answered, which is precisely the harm class this branch exists to stop. Open, unwidened by deliberate ruling (iss-2608301808198621, iss-2608301744268001). The user-facing doc was corrected to stop claiming completeness and now names the tag/anchor/alias and trailing-comment classes — but it does not name the zero-width space or `!!map {}`, so the corrected enumeration is itself still narrower than the measured gap. (ii) The criterion's second clause ('a blank is refused once command enforcement lands') has no producer: no verb writes an adm-N on any surface, and no intent id is named anywhere in the record for that Iteration-2 work — spc-67 and commands/capture.md both say 'the next iteration' without naming it, so itd-192's obligation to name the closing intent is unsatisfied. (iii) The Given clause is unreachable in production: nothing on any surface mints an rdi-N, because `capture.IngestReading` has no CLI or plugin caller and itd-185, the ingest door, is still in planned/ and not started. Judged MET_WITH_CONCERNS under itd-192 because THIS phase does wire the enforcement it promised for this cycle (the gate), but the universal reading of 'it carries grounds' is FALSE as measured.
  evidence: internal/core/issueschema/admission.go:70 — "var AdmissionRequired = []string{\"schema_version\", \"id\", \"run\", \"proposal\", \"grounds\"}"
  evidence: internal/core/lint/schema.go:394 — "requiredFields: issueschema.AdmissionRequired, knownFields: issueschema.AdmissionKnown,"
  evidence: .abcd/record-lint.json:265 — "\"adm\": \".abcd/work/issues/admissions\","
  evidence: Makefile:175 — "preflight: lint-reviews lint-issues lint-decisions record-lint docs-lint site-render"
  evidence: internal/core/lint/schema.go:1712 — "func isAbsentValue(value string) bool {  — measured: 16 nothing-carrying spellings pass green, 6 of them also return 0 outstanding findings"
  evidence: internal/core/lint/schema_test.go:2322 — "func TestIsAbsentValueIsASpellingTestNotANullTest(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2608301808198621-isabsentvalue-decides-on-literal-strings-rather-than-the-yam.md:1 — "OPEN: isAbsentValue decides on literal strings rather than the YAML null class"
  evidence: .abcd/work/issues/open/iss-2608301744268001-a-trailing-comment-on-a-frontmatter-key-defeats-every-blank.md:1 — "OPEN: a trailing comment on a frontmatter key defeats every blank spelling"
  evidence: commands/capture.md:261 — "That list is the set the gate reads, not every way YAML can carry nothing."
  evidence: internal/core/capture/reading.go:144 — "func IngestReading(req IngestReadingRequest) (IngestReadingResult, error) {  — no caller outside internal/core/capture"
  evidence: .abcd/development/intents/planned/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:1 — "id: itd-185 — still under intents/planned/, not started"
- ac-2 — MET_WITH_CONCERNS: The decline is on the ledger side and the mechanism is real, reused rather than reinvented, and reachable from production. `declined` is available at the widening position in the shared availability table; the disposition record it lands in is spc-58's, written by the wired `abcd capture disposition <rdi-N> --state declined` verb into .abcd/work/issues/dispositions/rdi-N/, a store the real record-lint config gates; and spc-67's addition — the report that notices a widening proposal with neither an admission nor a decline — is wired both as a lint finding and as an `unadmitted` line on the CLI board. I ran the leg across all four states rather than trusting the table: `declined` alone silences it (0 findings), `accepted` and `rejected` each still report the proposal unadmitted, and `held` renders its own hold line with the exit condition. That is exactly the discrimination the criterion needs. CONCERNS, three. (i) 'when the session ends' has no producer at all — nothing runs at session end, and the report is pinned to `info` in code whatever the config asks, so a declined proposal whose disposition was never written is REPORTED and never REQUIRED; the criterion's obligation is a notice, not a gate. (ii) Deviation 2 lands squarely on this criterion: the padding and bucket legs share a tail asserting the named item 'goes on being reported as unanswered', which is false of exactly the `declined` item ac-2 is about — the message states as fact something the walk never consulted (open iss-2608301755006875). (iii) Same unreachable Given as ac-1: no surface mints an rdi-N, so no widening proposal can exist to be declined until itd-185 lands. Judged MET_WITH_CONCERNS under itd-192 — this phase wires the answering mechanism; the intent that closes the producer gap is itd-185, unstarted.
  evidence: internal/core/issueschema/reading.go:194 — "DispositionDeclined: true,"
  evidence: internal/surface/cli/cli.go:2781 — "Use: \"disposition <rdi-N> --state <accepted|rejected|declined|held> [--grounds <text>] ...\""
  evidence: .abcd/record-lint.json:263 — "\"dsp\": \".abcd/work/issues/dispositions\","
  evidence: internal/core/lint/readingoutstanding.go:674 — "o.Item + \" (run \" + o.Run + \") is a widening proposal with neither an admission nor a decline — outstanding. \""
  evidence: internal/surface/cli/cli.go:2488 — "fmt.Fprintf(w, \"  unadmitted %s (run %s) — a widening proposal with neither an admission nor a decline\\n\","
  evidence: internal/core/lint/reading_outstanding_test.go:523 — "func TestAdmissionLegSeverityIsInfoNotBlocker(t *testing.T) {"
  evidence: internal/core/lint/schema.go:1027 — "\"nothing and the \" + target.noun() + \" it names goes on being reported as unanswered\""
  evidence: internal/core/lint/schema.go:1081 — "\" it names goes on being reported as unanswered with no sign that an answer was written\""
  evidence: .abcd/work/issues/open/iss-2608301755006875-the-shared-unanswered-tail-on-the-padding-and-bucket-legs-is.md:22 — "A widening item carrying a `declined` or a `held` disposition IS answered"
  evidence: .abcd/development/intents/planned/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:1 — "id: itd-185 — the ingest door, not started"
- ac-3 — MET_WITH_CONCERNS: Separateness is the strongest thing this branch delivers, and it is structural rather than conventional: a distinct family prefix (srp), a distinct FLAT store (surprises/, deliberately not keyed by occasion so it cannot inherit one family's key), a distinct required set whose whole join is `occasioned_by`, and a closed allow-list. I exercised it rather than reading it: a well-formed surprise is clean; a surprise carrying a disposition's own fields draws two blocker findings ('unknown frontmatter property state', 'unknown frontmatter property disposition_grounds'), so a surprise cannot be spelled as a disposition even by accident; and an `occasioned_by` naming a record the corpus does not hold is a finding, so the join is resolved rather than declared. The reading record's own envelope reserves `occasioned_by` as DORMANT and refuses a populated value, which closes the other direction. It is wired to the real config and to preflight. CONCERNS, two, both narrower than the first two criteria's. (i) No producer: no verb writes an srp-N and the doc says so plainly ('Neither shape has a sub-verb'); the store directory does not exist in the repository, because no reading has run. No intent id is named for the writing verb. (ii) A residual the spec itself declares: `abcd <id>` dispatch does not cover srp-N or adm-N. Judged MET_WITH_CONCERNS under itd-192 for the absent producer alone — the distinctness the criterion actually asserts is realised without qualification and is gate-enforced today.
  evidence: internal/core/issueschema/admission.go:82 — "var SurpriseRequired = []string{\"schema_version\", \"id\", \"occasioned_by\"}"
  evidence: internal/core/issueschema/admission.go:59 — "SurprisesDir = \"surprises\""
  evidence: internal/core/lint/schema.go:405 — "requiredFields: issueschema.SurpriseRequired, knownFields: issueschema.SurpriseKnown,"
  evidence: internal/core/lint/schema.go:407 — "field: \"occasioned_by\", why: \"a surprise is keyed to whatever occasioned it, and a join naming nothing joins nothing\""
  evidence: internal/core/issueschema/reading.go:139 — "var ReservedSurpriseFields = []string{\"occasioned_by\"}"
  evidence: .abcd/record-lint.json:266 — "\"srp\": \".abcd/work/issues/surprises\""
  evidence: commands/capture.md:252 — "Neither shape has a sub-verb — this surface writes no `adm-N` and no `srp-N`"
  evidence: .abcd/development/specs/closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md:1 — "Out of scope: Dispatching `abcd <id>` on `adm-N` or `srp-N`"

Gap audit:
- honoured:
  - Every admission into the candidate set carries recorded grounds — and the refusal is armed at the gate this cycle rather than promised for a verb.
    evidence: internal/core/issueschema/admission.go:70 — "var AdmissionRequired = []string{\"schema_version\", \"id\", \"run\", \"proposal\", \"grounds\"}"
    evidence: internal/core/lint/schema_test.go:1025 — "func TestAdmissionRecordRequiresGrounds(t *testing.T) { — measured: blank AND absent both blocker"
  - The admitted set is keyed on the (run, proposal) PAIR, with the run taken from the directory the record sits in and cross-checked against the field.
    evidence: internal/core/lint/schema.go:392 — "{prefix: \"adm\", noun: \"admission\", nodeType: \"admission\", bucketRe: readingRunBucketRe, ... bucketField: \"run\","
    evidence: internal/core/issueschema/admission.go:49 — "AdmissionsDir holds one directory per run: admissions/<run-id>/adm-<N>.md."
  - A declined proposal is not a second record type — it is spc-58's disposition in its `declined` state, and only that state answers the widening leg.
    evidence: internal/core/issueschema/reading.go:194 — "DispositionDeclined: true,"
    evidence: internal/core/lint/reading_outstanding_test.go:472 — "func TestDeclinedDispositionSatisfiesTheAdmissionLeg — measured: declined 0 findings, accepted/rejected 1 each"
  - A surprise is its own record and can never be read as, or overwrite, a disposition: separate family, separate flat store, `occasioned_by` rather than a shared key, closed allow-list.
    evidence: internal/core/issueschema/admission.go:82 — "var SurpriseRequired = []string{\"schema_version\", \"id\", \"occasioned_by\"}"
    evidence: internal/core/lint/schema.go:405 — "requiredFields: issueschema.SurpriseRequired — measured: `state` and `disposition_grounds` each refused as unknown properties"
  - The report notices a widening proposal that is neither admitted nor declined, as one branch of the existing rule rather than a twin, pinned at info, and reachable from the production entry point.
    evidence: internal/core/lint/readingoutstanding.go:674 — "is a widening proposal with neither an admission nor a decline — outstanding."
    evidence: internal/surface/cli/cli.go:2488 — "fmt.Fprintf(w, \"  unadmitted %s (run %s) — a widening proposal with neither an admission nor a decline\\n\","
  - The nine-store reader enumeration that was wrong four consecutive times is now executable — every answer established by running the reader, with a coverage gate that refuses a new store arriving without a row.
    evidence: internal/core/lint/duplicatekeyreaders_test.go:167 — "func TestEveryStoreHasADuplicateKeyReaderRow(t *testing.T) {"
    evidence: internal/core/lint/duplicatekeyreaders_test.go:58 — "func TestDuplicateTopLevelKeyReaderByReader — 14 subtests pass for 9 stores"
- diverged:
  - The press release's implied universal — that the record can show which proposals were taken — is delivered as a SPELLING test, not a null test. Sixteen nothing-carrying spellings of `grounds` pass the gate green, and six I drove further also return zero outstanding findings, so the gate accepts an admission that admits nothing AND the report then calls the proposal answered. Deliberately not widened; the doc was narrowed instead, and the narrowed doc still omits the zero-width-space and `!!map {}` classes I measured.
    evidence: internal/core/lint/schema.go:1712 — "func isAbsentValue(value string) bool {"
    evidence: commands/capture.md:265 — "and pass green (iss-2608301808198621)."
    evidence: .abcd/work/issues/open/iss-2608301808198621-isabsentvalue-decides-on-literal-strings-rather-than-the-yam.md:1 — "OPEN"
  - A trailing YAML comment defeats the blank refusal across fields, not only `grounds`. Measured: an admission with every required field spelled `"" # x` draws three findings, and none of them is a required-field finding — `schema_version` and `grounds` pass entirely; the other three are caught only by secondary grammar checks (filename, bucket, join), which is coincidence, not coverage.
    evidence: .abcd/work/issues/open/iss-2608301744268001-a-trailing-comment-on-a-frontmatter-key-defeats-every-blank.md:1 — "OPEN: defeats every blank spelling at once"
    evidence: internal/core/lint/schema.go:1686 — "A trailing comment defeats every test here at once, because the shared same-line scanner strips no comments"
  - The padding and bucket legs assert that the named item 'goes on being reported as unanswered' — false of a `declined` or `held` item, and `declined` is the exact state ac-2 is about. A gate stating as fact something about a report it never consulted.
    evidence: internal/core/lint/schema.go:1027 — "\"nothing and the \" + target.noun() + \" it names goes on being reported as unanswered\""
    evidence: internal/core/lint/schema.go:1081 — "\" it names goes on being reported as unanswered with no sign that an answer was written\""
  - NOT ON THE DECLARED LIST — spc-67 states that the surprise entry 'is declared once, in spc-58's family' and that this spec 'declares nothing a second time'. It does declare it a second time: the family, the store directory, the required set and the allow-list are all in spc-67's own admission.go, while spc-58's reading.go reserves only the dormant `occasioned_by` key on the reading envelope. The code's doc comment is honest about this; the spec's prose is not. Harmless in substance, wrong as a statement of where the shape lives.
    evidence: internal/core/issueschema/admission.go:41 — "SurpriseFamily = \"srp\""
    evidence: internal/core/issueschema/reading.go:139 — "var ReservedSurpriseFields = []string{\"occasioned_by\"}"
    evidence: .abcd/development/specs/closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md:1 — "The declaration lives with the reservation, in spc-58; spc-67 states its keying and its separateness and declares nothing a second time."
  - Three minor divergences left standing by ruling: an unpinned bucketField/bucketed-store coupling, two mark paths (judged on id, judged on slug) unreachable in the capture surface and left as deliberate redundancy. Both open.
    evidence: .abcd/work/issues/open/iss-2608301634527391-the-bucketfield-implies-a-bucketed-store-rule-is-asserted-in.md:1 — "OPEN: the bucketField-implies-a-bucketed-store rule is asserted in ..."
    evidence: .abcd/work/issues/open/iss-2608301634520703-mark-judged-on-id-and-on-slug-are-both-unreachable-in-the-ca.md:1 — "OPEN: mark judged on id and on slug are both unreachable in the capture ..."
- missing:
  - No command writes an adm-N or an srp-N on any surface. Declared out of scope as 'Iteration 2', but NO intent id names that work anywhere in the record — so itd-192's requirement that a MET_WITH_CONCERNS verdict name the intent which closes the gap has nothing to name. Filing that intent is the concrete next step this verdict asks for.
    evidence: commands/capture.md:252 — "Neither shape has a sub-verb — this surface writes no `adm-N` and no `srp-N`, and the command-side refusal is the next iteration's."
    evidence: .abcd/development/specs/closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md:1 — "Out of scope: The commands that write these records. Iteration 2, by itd-189's own words"
  - NOT ON THE DECLARED LIST — no widening proposal can be produced at all. `capture.IngestReading` mints rdi-N but has no caller outside its own package: no CLI verb, no plugin surface, no cmd/. Every one of ac-1, ac-2 and ac-3 has a Given clause that presupposes a reading item, and itd-185 — the ingest door — is still in intents/planned/ and has not been started. The three schemas are armed over a corpus that cannot be populated.
    evidence: internal/core/capture/reading.go:144 — "func IngestReading(req IngestReadingRequest) (IngestReadingResult, error) { — grep for callers outside internal/core/capture returns nothing"
    evidence: .abcd/development/intents/planned/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:1 — "id: itd-185 — under intents/planned/"
  - ac-2's 'when the session ends' has no mechanism. Nothing runs at session end; the admission leg is pinned to info in code whatever the configuration asks, so an unanswered proposal is reported and never required. The obligation the criterion states as a deadline is delivered as a notice.
    evidence: internal/core/lint/readingoutstanding.go:654 — "checkReadingOutstanding renders the report as findings, every one of them at severityInfo whatever the configuration says."
    evidence: internal/core/lint/reading_outstanding_test.go:523 — "func TestAdmissionLegSeverityIsInfoNotBlocker(t *testing.T) {"
  - `abcd <id>` dispatch does not resolve an adm-N or an srp-N — the cited-id grammar still covers the four id-bearing families only. Declared by the spec as a residual it shares with spc-58.
    evidence: .abcd/development/specs/closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md:1 — "Dispatching `abcd <id>` on `adm-N` or `srp-N`, which shares spc-58's residual"