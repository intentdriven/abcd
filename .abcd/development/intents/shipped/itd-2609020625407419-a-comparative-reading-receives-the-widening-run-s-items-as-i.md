---
id: itd-2609020625407419
slug: a-comparative-reading-receives-the-widening-run-s-items-as-i
spec_id: spc-2609020626039834
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-199, itd-191, itd-186, itd-183, itd-180]
severity: major
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A comparative reading receives the widening run's items as its candidate set, and characterises them before anyone admits one

Typed links: `builds_on` [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (the comparative refusal and the committed presets), [itd-191](../disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md) (the criteria discipline), [itd-186](../shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md) (the read-block eval's prior-run exhaust), [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (field projection and the manifest), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (reading records); `refines` [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (withdraws its refusal of a comparative preset, flagged for the maintainer) and [itd-186](../shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md) (a positional exception to the prior-run exhaust, flagged for the maintainer).

## Press Release

> **The comparative reading can now be commissioned.** `abcd reading assemble --position comparative --target <commit>` derives from the record the one committed widening run at that target whose items carry no disposition and no admission, and hands the reading that run's returned configurations, projected to the two fields the widening body carries, together with the six declared selection criteria. No operand names the run: the invocation is a position and a target state, as the design specifies. It hands over nothing else from the readings store: no disposition, no admission, no manifest, no other run. If no widening run qualifies, or more than one does, the assembly refuses and lists the widening runs at the target with their item counts and disposition state, so the operator sees what to disposition. If any item of the derived run already carries a disposition or an admission, the assembly refuses, because the design fixes the order as characterise first and admit second. If the run returned fewer than two configurations, the assembly refuses and records that the position was not exercised, which is the interpretation fixed before any run. At ingest, every item names a candidate from that run and a criterion from the discipline, and an ordering, a score or a recommendation across candidates is refused as it already is.

> "I want to see how each of the reading's proposals ordinarily behaves against the criteria we said we would select on, before I commit to any of them," said an AI/agent researcher who runs the four positions against their own record. "The reading that widened my candidate set must not be the one that tells me which candidate to take, and the one that characterises them must not see which I took."

## Why This Matters

The cold-reading rulings adopted on 2026-08-28 fix the comparative reading's object as the widening reading's output, being the configurations returned at Step 2 before admission, and admit no other source; they fix the ordering as Step 2 before Step 4, with admission a warm act performed after the comparative reading, because running admission first would leave the comparative reading nothing uncommitted to characterise; and they fix the interpretation in advance: where the widening reading returns fewer than two configurations, the comparative reading has nothing to compare and is not exercised, and that outcome is recorded as such.

v0.7.0 ships the definition, the regime gate and the criteria discipline ([itd-191](../disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md)), and the position refuses to assemble because its object "is not repository material and has no channel today" ([itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md)). That refusal was the right move over serving the detection corpus, and it leaves one of the four positions structurally unavailable until the channel exists. itd-199 scoped the channel out as "a separate intent", and this is that intent.

The channel is delicate because the readings store sits inside the read block by ruling: prior manifests and reading records never reach a reading ([itd-186](../shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md), ac-4), and the general rule is that no reading sees another's output. The comparative object is the one ruled exception, and it is narrower than it sounds. What the reading needs is the candidate text as it was returned, which is cold: the widening reading produced it without ledger access. What it must not see is what has happened to those candidates since, which is warm: dispositions, admissions, surprises, and any other run. The channel therefore projects two fields out of one run's items and asserts the exclusion of everything else in the same store, which is exactly the field projection the assembler already performs on shipped intents.

## Decisions flagged for the maintainer

The decisions were adopted by the maintainer on 2026-09-02 as [adr-2609021016272867](../../decisions/adrs/2609021016272867-the-comparative-reading-receives-one-widening-run-s-candidat.md), which admits one widening run's candidates at the comparative position as a positional exception to the prior-run exhaust, with the run derived from the record and no operand added. Two shipped rulings move under it, and the record says so rather than having it discovered.

- **The prior-run exhaust gains one positional exception.** itd-186's ac-4 and the include table's exclusions state that the instrument's own output is never its input. This intent admits, at the comparative position only, two body fields of one widening run's items. That is a trust-boundary rule; the ADR states it and brief invariant 15 carries the amendment, and the intent carries the derivation, the projection and the refusals.
- **The comparative preset refusal is withdrawn.** itd-199's ac-10 refuses a preset entry for the comparative position. The committed preset file gains a comparative entry whose object set is the criteria discipline, carrying the window declaration the presets intent requires.
- **The run is derived, and the invocation gains nothing.** The design fixes the invocation at a position and a target state, and adr-2609021016286571 restores that letter. The widening run a comparative reading characterises is therefore derived from the record, as adr-2609021016272867 states it: the one committed widening run at the target whose items carry no disposition and no admission. The record already holds enough to select it, and the one ambiguous case, two undispositioned widening runs after the closing run, is disambiguated by dispositioning one run's items, which is exactly the act the design sequences after the comparative reading; so the assembler refuses in that case, lists the runs, and the operator's next step resolves it. A named operand was drafted and withdrawn on the same day, because it contradicted the design's letter for a convenience the record does not need.

## What's In Scope

- **A derived candidate run.** At the comparative position the assembler selects the one committed widening run at the target whose items carry no disposition and no admission. None, or more than one, refuses, and the refusal lists the widening runs at the target with each run's item count and disposition state. The invocation stays a position and a target state.
- **Projection of the derived run's items** to the two widening body fields, configuration and what admits it, plus the item identifier the comparative body must cite. The item's pattern, its envelope, and every other field stay behind.
- **The criteria discipline** ([itd-191](../disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md)) passed with the candidates, as the declared criteria the reading characterises against, and the definition's Object section updated to name both.
- **The ordering as a refusal.** If any item of the derived run has a standing disposition or an admission, the assembly refuses and names the item, because the candidate set is defined as pre-admission. The gate that makes this unreachable through any verb sits in the shared disposition writer (the admission intent): at the widening position no disposition can be written until a committed comparative run names the run.
- **The fixed interpretation as a refusal.** If the derived run holds fewer than two items, the assembly still stages a comparative run with an empty candidate set, and ingesting it commits a comparative run with an empty item set naming that widening run, which is the clean-run idiom the design fixes in advance. The comparative outcome for a widening run is therefore always a committed comparative run naming it, never a mutable file, and the admission verb reads exactly that.
- **The manifest** records the candidate run derived, the count of items it holds, the projected fields, and the exclusion of the rest of the readings store, so a reader can see what the assembler selected and that the comparative reading saw candidates and not their fate.
- **Ingest at the comparative position** checks that every item's candidate identifier names an item of the run the manifest records, and that every criterion is one the discipline declares.
- **The read-block eval** gains a comparative case: with dispositions, admissions and a second, dispositioned run planted, the comparative bundle carries the derived run's two candidate fields and nothing else from the store.
- **The include table is the whole account at this position.** The candidate channel is a table row of its own (source the readings store, two fields, kind candidate, comparative only), so the rendered charter and the assembler version carry it. Every other row excludes the comparative position except the disciplines row, which at this position is narrowed to the criteria discipline, so no preset entry can hand the comparative reading the tree, the brief or the intents: no other source is admitted.

## What's Out of Scope

- Admission itself, which is itd-189's record and the admission verb's to write.
- Any characterisation the assembler performs. It passes candidates and criteria; the reading characterises.
- Widening beyond the derived run. A comparative reading is about one widening run, and two runs' candidates are two comparative runs.
- An operand naming the run. adr-2609021016286571 fixes the invocation at a position and a target state, and the record supplies the run.

## Mechanism

We expect a field projection from one run's reading records to hold the read block where a path rule cannot, because the warm half of the readings store (dispositions, admissions, other runs) is separated from the cold half (returned candidate text) at field and directory granularity, and the assembler already asserts field exclusions fail-closed. This is falsifiable in one move: plant a disposition on a candidate and show its text in the comparative bundle.

## Scope Conditions

- The candidate set is exactly one widening run's items. A candidate set drawn from more than one run, or from anything other than a widening run, is out of scope and would need its own ruling. <!-- cond: cond-2609020626038511 -->
- The ordering refusal assumes the admission and disposition stores are the only places a candidate's fate is recorded. A new store recording something about a candidate is excluded from the reading by positive inclusion but is not, by itself, a reason to refuse assembly. <!-- cond: cond-2609020626033649 -->
- The pattern named by the widening item is withheld from the comparative reading on the ground that provenance is the envelope's and not the candidate's. If a criterion turns out to need it, the projection widens by one field and the manifest says so. <!-- cond: cond-2609020626032295 -->

## Acceptance Criteria

- **Given** a comparative invocation at a target where no committed widening run has every item free of a disposition and an admission, or where more than one has, **when** it runs, **then** the verb refuses and lists the widening runs at the target with each run's item count and disposition state; and where exactly one qualifies, the manifest names the run derived.
- **Given** a widening run with three items, none dispositioned or admitted, **when** the comparative assembly runs against it, **then** the bundle carries each item's identifier, configuration and what admits it, and the criteria discipline, and no other field of those items.
- **Given** the same run with one item carrying a standing disposition, **when** the comparative assembly runs, **then** it refuses and names that item.
- **Given** the same run with one item admitted, **when** the comparative assembly runs, **then** it refuses and names that item.
- **Given** a widening run with one item, **when** the comparative assembly runs, **then** it refuses, names the fixed interpretation, stages a comparative run with an empty candidate set and, once ingested, a committed comparative run with an empty item set names that widening run.
- **Given** a repository holding two widening runs, one of them dispositioned, and admissions, **when** the comparative assembly runs and derives the other, **then** no text from the dispositioned run, from any disposition or from any admission appears in the bundle, and the manifest asserts their exclusion.
- **Given** a comparative output whose item names a candidate not in the recorded run, **when** it is ingested, **then** the verb refuses and names the item.
- **Given** a comparative output whose item names a criterion the discipline does not declare, **when** it is ingested, **then** the verb refuses and names the item and the criterion.
- **Given** the read-block eval with a disposition planted on a candidate of the derived run, **when** it runs, **then** it fails if the disposition's text reaches the comparative bundle.

## Prior Art

- The cold-reading rulings of 2026-08-28 in the decision log, and the design framework and its readings companion held outside the repository; adr-2609021016272867 (the derived run) and adr-2609021016286571 (the two-operand invocation).
- [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md), [itd-186](../shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md), [itd-191](../disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md).

## Open Questions

None. The flagged decisions are adopted as adr-2609021016272867. The ordering and the fewer-than-two interpretation are ruled, and this intent implements them as refusals rather than reopening them.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e37a3b69db80 -->
Fidelity review — receipt rcp-e37a3b69db80 (verifier intent-auditor claude-opus-5).

Provenance: intent-auditor@claude-opus-5 · rubric_hash sha256:44f19418b0b8d56d7b17cbecbd5acd5d2cbed5abc4347428680ce46647f381d5 · prompt_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5
Input attestations: diff:3b62c967dae9488813faee5317c167a0045fc715 (single commit; git show 3b62c967)@sha256:38fa04b07272158bb4df9c7af7c10dd80c617f8841e6c913bc836bdac299e579;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: DeriveCandidateRun applies the exactly-one rule and both refusals carry the whole listing with each run's item count and disposition state; the manifest and result name the run derived, and the CLI renders the listing plain and as widening_runs; four tests pass, asserting the listing content by string.
  evidence: internal/core/reading/candidates.go:240 — "func DeriveCandidateRun(repoRoot, target string) (WideningRun, error)"
  evidence: internal/core/reading/candidates.go:66 — "fmt.Fprintf(&b, "%s — %d item(s)", r.ID, r.Items)"
  evidence: internal/core/reading/candidates_test.go:83 — "TestTwoUndispositionedWideningRunsRefuseNamingThem — asserts "3 item(s)", "2 item(s)", "disposition" in the refusal"
  evidence: internal/core/reading/candidates_test.go:62 — "if res.Manifest.CandidateRun != fixtureCandidateRun"
  evidence: internal/surface/cli/reading.go:150 — "WideningRuns []reading.WideningRun `json:"widening_runs"`"
- ac-2 — MET: The candidate row projects exactly configuration and what_admits_it keyed by the item id; the test asserts three candidates each travelling as exactly CandidateFields, the CANDIDATE carrier arriving, the ENVELOPE (pattern) sentinel absent, and the discipline present with no other row admitted.
  evidence: internal/core/reading/candidates_test.go:264 — "if strings.Join(fields, "|") != strings.Join(CandidateFields, "|")"
  evidence: internal/core/reading/candidates_test.go:275 — "if strings.Contains(text, sentinelEnvelope) { t.Error("the item's pattern reached the bundle") }"
  evidence: internal/core/reading/assemble.go:557 — "item.Candidate, item.Field = id, c.field"
  evidence: internal/core/reading/candidates_test.go:288 — "TestComparativeBundleCarriesTheCriteriaDiscipline"
- ac-3 — MET: capture.ItemFate reports the standing disposition and WideningRuns records it in Fated as a rendered phrase naming the item and the disposition, which the refusal prints; the test asserts the item id, dsp-7 and "standing disposition" appear and that untouched items are not blamed.
  evidence: internal/core/reading/candidates.go:218 — "run.Fated = append(run.Fated, fmt.Sprintf("%s carries the standing disposition %s", item, strings.Join(fate.Dispositions, ", ")))"
  evidence: internal/core/reading/candidates_test.go:374 — "for _, want := range []string{fixtureCandidateItems[1], "dsp-7", "standing disposition"}"
  evidence: internal/core/capture/itemfate.go:67 — "itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, item)"
- ac-4 — MET: The same probe reads admissions keyed on the (run, proposal) pair and the refusal names the item and the admission; a companion test proves non-vacuity by filing an admission under a different run and showing it does not refuse.
  evidence: internal/core/reading/candidates.go:224 — "run.Fated = append(run.Fated, fmt.Sprintf("%s carries the admission %s", item, strings.Join(fate.Admissions, ", ")))"
  evidence: internal/core/reading/candidates_test.go:397 — "for _, want := range []string{fixtureCandidateItems[2], "adm-4", "admission"}"
  evidence: internal/core/reading/candidates_test.go:407 — "TestAnAdmissionUnderAnotherRunDoesNotRefuse"
- ac-5 — MET: A one-item run returns PositionNotExercised naming the interpretation beside a written result whose bundle carries no candidate item and whose manifest carries candidate_run, candidates:1 and exercised:false; the ingest test reads the committed run record back with those three fields and zero reading records.
  evidence: internal/core/reading/assemble.go:614 — "so the comparative reading has nothing to compare and this position is NOT EXERCISED"
  evidence: internal/core/reading/candidates_test.go:480 — "if !res.Written { t.Fatal("the run was not staged") }"
  evidence: internal/core/reading/ingest_comparative_test.go:183 — "if run.CandidateRun != fixtureIngestCandidateRun"
  evidence: internal/core/reading/ingest_comparative_test.go:194 — "if len(res.Records) != 0"
- ac-6 — MET: The test plants a second widening run with dispositions, admissions and a surprise, derives the other, and asserts the FATE sentinel never reaches the bundle while the manifest asserts six derived directory exclusions plus the readings-store signal row; assertCandidateProjection is the fail-closed half and the eval repeats the check at suite level.
  evidence: internal/core/reading/candidates_test.go:555 — "if text := bundleText(res.Bundle); strings.Contains(text, sentinelFate)"
  evidence: internal/core/reading/candidates_test.go:563 — ".abcd/work/issues/dispositions", ".abcd/work/issues/admissions", ".abcd/work/issues/surprises""
  evidence: internal/core/reading/assemble.go:433 — "if err := assertCandidateProjection(cands, candidateRun.ID); err != nil"
  evidence: evals/coldreading_test.go:639 — "func TestComparativeChannelCarriesCandidatesAndNothingElse(t *testing.T)"
- ac-7 — MET: validateItems takes the manifest and refuses a candidate_id that names no manifest item as unknown-candidate, naming the ordinal and the id; the test mutates the second of three items so the other two still land, proving the refusal is item-level and not blanket.
  evidence: internal/core/reading/ingest_comparative_test.go:33 — "if r.Rule != ruleUnknownCandidate"
  evidence: internal/core/reading/ingest_comparative_test.go:39 — "for _, want := range []string{"rdi-999", fixtureIngestCandidateRun}"
  evidence: internal/core/reading/ingest_regime.go:1 — "validateItems(out Output, m Manifest, def Definition)"
- ac-8 — MET: A criterion outside manifest.Criteria is refused as undeclared-criterion naming the field, the offered criterion and the discipline; the criteria are parsed at assembly off the same bytes the bundle carries, and TestEveryDeclaredCriterionIsAccepted proves the check is not refusing everything.
  evidence: internal/core/reading/ingest_comparative_test.go:59 — "if r.Rule != ruleUndeclaredCriterion"
  evidence: internal/core/reading/ingest_comparative_test.go:65 — "for _, want := range []string{"Vibes", CriteriaDiscipline, fixtureIngestCriteria[0]}"
  evidence: internal/core/reading/criteria.go:43 — "func declaredCriteria(doc string) ([]string, error)"
  evidence: internal/core/reading/ingest_comparative_test.go:75 — "TestEveryDeclaredCriterionIsAccepted"
- ac-9 — MET_WITH_CONCERNS: The read-block eval case is delivered and passes with a negative control (the holed control's third hole relocates a fate into a projected candidate field), but the eval LANE it lives in is red at the delivered commit and green at its parent, so an operator running ac-9's own command gets a failure.
  evidence: evals/coldreading_test.go:734 — "func TestComparativeChannelCatchesAPlantedFate(t *testing.T)"
  evidence: evals/coldreading_oracle_test.go:220 — "{"holes", len(holes), 3}"
  evidence: evals/coldreading_window_test.go:65 — "2 committed entry(s) measure past the window they declare: widening 809238 vs 800000; detection 818275 vs 810000 — observed running `make evals-cold-reading` at 3b62c967, and PASSING on a scratch copy of the parent c1be0000"

Gap audit:
- honoured:
  - The run is derived from the record and the invocation gains nothing: a position and a target state, as adr-2609021016286571 and brief invariant 15 fix it
    evidence: internal/core/reading/assemble.go:399 — "candidateRun, err = DeriveCandidateRun(req.RepoRoot, target)"
    evidence: internal/core/reading/candidates.go:9 — "No operand names the run."
  - Two body fields of one run's items and nothing else; the pattern stays in the envelope
    evidence: internal/core/reading/candidates_test.go:275 — "if strings.Contains(text, sentinelEnvelope)"
    evidence: agents/cold-reading-comparative.md:1 — "the pattern each candidate was read under stays in its envelope rather than travelling with it"
  - Every other include row withdraws from the comparative position and the disciplines row is narrowed to itd-191
    evidence: internal/core/reading/include_test.go:1 — "TestEveryOtherRowWithdrawsFromComparative"
    evidence: internal/core/reading/criteria.go:23 — "const CriteriaDiscipline = "itd-191""
  - The fixed interpretation is delivered as the clean-run idiom the framework's section 13 states: a run recorded with an empty item set, at every position, not a comparative carve-out
    evidence: internal/core/reading/ingest_regime_test.go:1 — "TestIngestCommitsAnEmptyRunAtEveryPosition"
    evidence: internal/core/reading/ingest_comparative_test.go:169 — "TestIngestCommitsAnEmptyComparativeRun"
  - itd-199's ac-10 refusal of a comparative preset entry is withdrawn and the committed entry carries its window declaration
    evidence: .abcd/config/reading-presets.json:68 — ""comparative": { "object": { "records": ["itd-191"], "paths": [] }, "kinds": ["discipline"]"
    evidence: internal/core/reading/scope_test.go:1 — "TestComparativePresetIsAdmitted"
  - The exclusion rows are derived from issueschema.LedgerDirs(), so a family the ledger adds later is excluded the day its constant is declared
    evidence: internal/core/issueschema/ledgerdirs.go:1 — "func LedgerDirs()"
    evidence: internal/core/reading/include_test.go:1 — "TestComparativeExclusionRowsAreDerivedFromLedgerDirs"
- diverged:
  - The spec put the ordering guard in Assemble as a second pass over every candidate; the delivery folds it into the one derivation probe, so the refusal arrives as NoCandidateRun carrying the item that stopped the run. Same observable outcome, disclosed in the commit message.
    evidence: internal/core/reading/candidates.go:205 — "run.Fated = append(run.Fated, ...)"
    evidence: .abcd/development/specs/closed/spc-2609020626039834-a-comparative-reading-receives-the-widening-run-s-items-as-i.md:1 — "`Assemble` asks it for every candidate before anything is minted or written."
  - WideningRun carries Committed and Fated beyond the spec's four fields (ID, Items, Dispositioned, Admitted). Both serve the spec's own prose — an uncommitted run must be listed and a refusal must name the item that stopped it — and both are documented in place.
    evidence: internal/core/reading/candidates.go:48 — "Committed reports that the run reached its commit marker (`run.json`)."
    evidence: internal/core/reading/candidates.go:52 — "Fated names what stands over the items that carry a fate"
  - Two pinned eval counts the spec's delta list did not name moved: carriers 17 to 19 and materialClasses 10 to 11. Disclosed in the commit message; the spec's delta list named only sentinelClasses, excludedKeys, excludedFamilies, admittedRecordPaths and coverage.
    evidence: evals/coldreading_oracle_test.go:218 — "{"carriers", len(carriers), 19}, {"materialClasses", len(materialClasses), 11}"
  - The comparative window could not be measured the way the other three were — this repository holds no committed widening run — so it was taken by dry run over a scratch copy with a three-item run planted, 1,312 estimated tokens rounded up to 10,000. Stated in the entry's own comment with measured_at naming the scratch tree's commit.
    evidence: .abcd/config/reading-presets.json:79 — ""measured_at": "c1be0000e7a0117947b1e3fedc08244ca77c9aa9""
    evidence: evals/coldreading_window_test.go:35 — "windowPositions = []string{"widening", "entailment", "detection"}"
  - requireConfiguredStores tolerates an absent store directory for the rdi (readings) store alone, on the ground that the store is provisioned on first use; every other store still refuses, and a readings path pointing at a non-directory still refuses.
    evidence: internal/core/reading/assemble.go:1205 — "if os.IsNotExist(err) && row.Store == issueschema.ReadingItemFamily { continue }"
  - The spec's Tests section claims every test was watched fail before and pass after; the implementer discloses the red-first ordering was partial — leaf helpers watched red, the rest proved by mutation on a scratch copy. Not verifiable from the tree either way.
    evidence: .abcd/development/specs/closed/spc-2609020626039834-a-comparative-reading-receives-the-widening-run-s-items-as-i.md:1 — "Watched fail before, pass after; each refusal proved by a mutation that removes it."
- missing:
  - `make evals-cold-reading` does not pass at the delivered commit. TestEveryCommittedEntryFitsItsDeclaredWindow and TestTheWindowCheckReportsABreach fail because the widening (809238 vs 800000) and detection (818275 vs 810000) entries now measure past their declarations; the same two tests pass on a scratch copy of the parent c1be0000, so the delivery caused the breach by growing the tree those two positions read. Neither the commit message, the spec nor the intent discloses it, and no re-measured declaration ships with the change.
    evidence: evals/coldreading_window_test.go:60 — "t.Fatalf("%d committed entry(s) measure past the window they declare")"
    evidence: .abcd/config/reading-presets.json:76 — "the widening and detection entries' tokens_est declarations are unchanged by this commit"

Scope-condition dispositions:
- cond-2609020626038511 — survived: The candidate set is exactly one widening run's items: the derivation admits exactly one qualifying run and refuses more than one, a record returned at any other position refuses the assembly by name, and assertCandidateProjection refuses to emit a candidate outside the derived run's directory.
  evidence: internal/core/reading/candidates.go:251 — "case 0: return WideningRun{}, &NoCandidateRun{...}; default: return WideningRun{}, &AmbiguousCandidateRun{...}"
  evidence: internal/core/reading/candidates_test.go:417 — "TestComparativeRefusesANonWideningRun"
  evidence: internal/core/reading/candidates_test.go:1 — "TestCandidateProjectionRefusesAForeignRun"
- cond-2609020626033649 — survived: ItemFate reads exactly the dispositions and admissions stores and nothing else, and the surprises store — a third store that does record something about a candidate — is excluded from the bundle by a derived row while a planted surprise leaves the assembly succeeding, which is the condition's second sentence exercised rather than merely assumed.
  evidence: internal/core/capture/itemfate.go:67 — "itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, item)"
  evidence: internal/core/capture/itemfate.go:82 — "runDir := filepath.Join(issuesRoot, issueschema.AdmissionsDir, run)"
  evidence: internal/core/reading/candidates_test.go:546 — "writeFile(t, root, ".abcd/work/issues/surprises/srp-1.md", ... occasioned_by ...) — and the assembly then succeeds"
- cond-2609020626032295 — survived: The projection is the two declared fields and the pattern stays behind: the ENVELOPE sentinel is planted in every candidate's pattern and asserted never to arrive, the manifest states the projected field names, and the divergence register records the withholding as the deliberate reading of companion 7.4.
  evidence: internal/core/reading/candidates_test.go:275 — "t.Error("the item's pattern reached the bundle; provenance is the envelope's and not the candidate's, and the projection is two fields")"
  evidence: internal/core/reading/candidates_test.go:280 — "if strings.Join(res.Manifest.CandidateFields, "|") != strings.Join(CandidateFields, "|")"
  evidence: .abcd/development/research/notes/2026-09-02-iteration-2-divergence-register.md:39 — "23 | The pattern named, at the comparative position | ... | Withheld | Provenance is the envelope's, not the candidate's"
## Grounds

- pursued: we expect a field projection out of one widening run's items to give the comparative reading its ruled object without opening the read block to the rest of the readings store, because the cold half (returned text) and the warm half (dispositions, admissions, other runs) of that store are separable at field and directory grain; a planted disposition reaching the comparative bundle would show it wrong
