---
id: itd-2609020625402599
slug: the-scribe-s-context-is-assembled-and
spec_id: spc-2609020626045177
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-188, itd-183, itd-180, itd-185]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The scribe's context is assembled and its output is ingested by a verb, and the record can show that no session held both a reading and the ledger

Typed links: `builds_on` [itd-188](itd-188-machine-assistance-in-maintaining-the.md) (the scribe definition), [itd-183](itd-183-the-cold-reading-sees-exactly-what-the.md) (the assembler and manifest idiom), [itd-180](itd-180-a-cold-reading-s-findings-land-as.md) (the disposition validator), [itd-185](itd-185-one-ingest-verb-validates-every-cold.md) (the ingest idiom); `refines` [itd-188](itd-188-machine-assistance-in-maintaining-the.md) (the protocol becomes a verb).

## Press Release

> **The scribe is a verb, not a protocol.** `abcd scribe assemble` builds the scribe's context from the ledger's allow list and nothing else, and writes a manifest of what it passed. `abcd scribe ingest --scribe-json <path>` validates what the scribe returned, transcribes dispositions, admissions and surprises through the same validators the capture verbs use, and refuses a payload that authors anything. Every assembly, reading or scribe, carries a per-run stamp naming its kind and its run, and the session store gains a check that reports a retained transcript carrying both stamps of one run. Until now the scribe's inverse access rule was held by its definition alone; now it is held by construction, and a reader can check it.

> "The reading session and the ledger session were always meant to be two sessions," said an AI/agent researcher who maintains the ledger with machine help. "I want the tool to build the scribe's context so I cannot accidentally hand it the tree, and I want the transcript store to be able to say whether any session held both."

## Why This Matters

[itd-188](itd-188-machine-assistance-in-maintaining-the.md) specifies the scribe as a definition with an inverse access rule, hand-run until an ingest verb lands, with two acceptance criteria: the scribe's assembled context contains ledger content and no shipped-tree material, and each reading run and each scribe run is a distinct retained session with no session holding both. Iteration 1 shipped the definition with the access rule stated in its allow list, and its own text says "There is no ingest verb". The fidelity verdict found both criteria met by declaration rather than by mechanism: no assembler runs for the scribe, and session retention cannot show that no session held both.

The scribe exists so that machine assistance in maintaining the ledger remains available without any context holding both ledger content and a reading. That property is the mirror of the read block, and the read block is held by an assembler, a manifest and an eval. The scribe deserves the same three things, or its half of the wall is an assertion.

## Decisions flagged for the maintainer

Both were adopted by the maintainer on 2026-09-02 as [adr-2609021016275803](../../decisions/adrs/2609021016275803-no-session-holds-both-a-reading-and-the.md), which states the mechanism behind brief invariant 15 and amends the invariant to name the stamp and the check.

- **The invariant exists, and the ADR states its mechanism.** Brief invariant 15 states that no session holds both a reading and the ledger; the ADR states the per-run context stamp and the session check as the mechanism, and the invariant's amendment names them. The per-run stamp is a change to the reading bundle's shape and moves the assembler version, as itd-199 did for the bundle's selector block.
- **A new top-level verb.** `scribe` is a verb of its own rather than a sub-verb of `reading`, because the two contexts must never share a front door. The surface coverage and index gates require its plugin page, its brief surface chapter row and its entry in the released surface snapshot, all in scope below.

## What's In Scope

- **`abcd scribe assemble --run <rdg-N>`**, which requires an ingested run and builds the scribe's context from the ledger's directories as the record declares them (the readings, dispositions, admissions, surprises and reframes stores, and the issue ledger's three status directories), together with the researcher's supplied dispositions text, and writes a manifest of what it passed by path. The run's reading records come from the store, not from a raw reading output supplied twice. It refuses any path outside the allow list by construction, including the shipped tree, the brief, the intents, the specs, the decisions and the session-transcript store.
- **`abcd scribe ingest --scribe-json <path>`**, which validates the scribe's four outputs (records to file, fidelity flags, outstanding items, refusals), writes dispositions, admissions and surprises through the existing validators and stores, and refuses a payload carrying a field the scribe may not author: a ground, a resolution, a disposition the researcher did not supply.
- **Per-run context stamps.** A reading bundle and a scribe context each carry a stamp naming the kind of session, the run and a digest of the context, matched exactly, so a session that merely reads the documentation carries none. `abcd history` gains a check, run by `history list` and by the smoke lane, that reports a retained transcript carrying both stamps of one run by name, reports which runs it saw when none does, and says the property is unobserved when the store holds no transcript that carries any stamp.
- **The scribe's manifest** is parked in the local tier at assembly and promoted beside the run at ingest, inside the read block, so the next reading cannot see it.
- **Surfaces:** the plugin page `commands/scribe.md`, a row in the brief's surface chapter index, the released surface snapshot, and the definition's Delivery section rewritten to name the verb.

## What's Out of Scope

- Any judgement by the verb about the content transcribed. The scribe authors nothing and the verb authors nothing.
- A change to the scribe's allow list. The rule stays the definition's; the verb enforces it.
- Redaction policy. Records written through the capture validators inherit the redaction those verbs already apply.

## Mechanism

We expect an assembler with an allow list to hold the inverse access rule for the same reason the reading assembler holds the read block: positive inclusion excludes by default, and a manifest makes the exclusion checkable. We expect the session-kind stamp to reach a retained transcript because a session reads its bundle or its context through a tool whose result the transcript retains, so the stamp text is in the transcript when the content was. It fails where a host hands a session content the transcript does not retain, which the scope conditions disclose and the check reports as unobservable rather than as clean.

## Scope Conditions

- The scribe's context is assembled from committed ledger content and from supplied text. A scribe that needs uncommitted ledger content is outside this scope. <!-- cond: cond-2609020626046719 -->
- The session check reads the native session-transcript store and can only see what a host retained. Where the host assembles context before anything is retained, the check reports that it cannot observe the property, and the definition's protocol remains the gate. <!-- cond: cond-2609020626048270 -->
- The verb transcribes into the stores that exist. A record family the scribe is asked to write that has no store is refused, never invented. <!-- cond: cond-2609020626049512 -->

## Acceptance Criteria

- **Given** an ingested run and supplied dispositions, **when** `scribe assemble` runs, **then** its context contains the run's reading records from the store, the other ledger content, and the supplied text, its manifest names every path passed, and no path from the shipped tree, the durable record outside the ledger, or the transcript store appears.
- **Given** a scribe payload carrying a disposition the researcher supplied, **when** `scribe ingest` runs, **then** a disposition record exists through the disposition validator and nothing else was written.
- **Given** a scribe payload carrying a ground or a resolution the researcher did not supply, **when** `scribe ingest` runs, **then** it refuses and names the field.
- **Given** a scribe payload naming an item it was given no disposition for, **when** `scribe ingest` runs, **then** the item appears in the outstanding report and no record was written for it.
- **Given** a retained transcript carrying a reading stamp and a scribe stamp of one run, **when** the session check runs, **then** it reports the transcript by name.
- **Given** two retained transcripts, one carrying each stamp of one run, **when** the session check runs, **then** it reports that no retained transcript carries two stamps of one run.
- **Given** an empty store, **when** the session check runs, **then** it reports that the property is unobserved rather than clean.
- **Given** the read-block eval over a repository holding a scribe manifest, **when** it runs, **then** the manifest does not reach any reading.

## Prior Art

- [itd-188](itd-188-machine-assistance-in-maintaining-the.md) and its spec (the definition), [itd-183](itd-183-the-cold-reading-sees-exactly-what-the.md) (the assembler and manifest idiom), [itd-180](itd-180-a-cold-reading-s-findings-land-as.md) (the disposition validator), the `abcd history` store.
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None. The flagged decisions are adopted as adr-2609021016275803.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-6654dbf923b3 -->
Fidelity review — receipt rcp-6654dbf923b3 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:34a24bad1b9e20b4164136080330b158cd3074b20815d120269663ffd6d5dd36
Input attestations: diff:internal/core/scribe/, internal/core/history/separation.go, internal/core/sessionkind/, internal/surface/cli/history.go, commands/scribe.md, evals/{coldreading_fixture,smoke}_test.go at 4001d807 (git ls-tree -r)@sha256:965e74af30dcb1a630023cf07598e711ecfe826ea8d086e36692a106176ea973;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Assemble requires an ingested run, builds the context from the ledger directories plus the supplied text and writes a manifest of every path passed; the test plants sentinels in the shipped tree, the durable record and the transcript store and asserts none reaches the context, that the run's records come from the store, and that the manifest names every path
  evidence: internal/core/scribe/assemble.go:61 — "func Assemble(req AssembleRequest) (AssembleResult, error) {"
  evidence: internal/core/scribe/scribe.go:81 — "func AllowList() []string {"
  evidence: internal/core/scribe/assemble_test.go:41 — "func TestScribeContextIsLedgerAndSuppliedTextOnly"
  evidence: internal/core/scribe/assemble_test.go:82 — "func TestScribeContextCarriesTheRunsRecordsFromTheStore"
  evidence: internal/core/scribe/assemble_test.go:107 — "func TestScribeManifestNamesEveryPathPassed"
  evidence: internal/core/scribe/assemble_test.go:186 — "func TestScribeAssembleRefusesAnUncommittedRun"
- ac-2 — MET: Ingest writes a supplied disposition through capture.Disposition, the existing validator, and the test asserts exactly one disposition file carrying the supplied ground with the admissions and surprises stores untouched
  evidence: internal/core/scribe/ingest.go:223 — "r, err := capture.Disposition(capture.DispositionRequest{"
  evidence: internal/core/scribe/ingest_test.go:99 — "func TestScribeIngestWritesASuppliedDisposition"
- ac-3 — MET: a key outside the closed output shapes (a resolution on a disposition) is refused by name with the item it was on, a ground the supplied text does not carry is refused naming grounds and the item, on dispositions and admissions alike, and a disposition for an item the text never names is refused; every case proves the ledger unchanged
  evidence: internal/core/scribe/ingest.go:335 — "func refuseKeys(where string, obj map[string]json.RawMessage, allowed map[string]bool) error {"
  evidence: internal/core/scribe/ingest_test.go:139 — "func TestScribeIngestRefusesAnAuthoredField"
  evidence: internal/core/scribe/ingest_test.go:172 — "func TestScribeIngestRefusesAnUnsuppliedGround"
  evidence: internal/core/scribe/ingest_test.go:196 — "func TestScribeIngestRefusesAnUnsuppliedAdmissionGround"
  evidence: internal/core/scribe/ingest_test.go:212 — "func TestScribeIngestRefusesAnUnsuppliedDisposition"
- ac-4 — MET: an all-outstanding payload returns both items in the result's outstanding list with the ledger digest unchanged, an item both dispositioned and outstanding refuses, and an outstanding item of another run refuses by name; a silent item is refused too
  evidence: internal/core/scribe/ingest_test.go:230 — "func TestScribeIngestReportsOutstandingAndWritesNothing"
  evidence: internal/core/scribe/ingest_test.go:264 — "func TestScribeIngestRefusesASilentItem"
- ac-5 — MET: SessionSeparation reads the store's records and reports a transcript carrying a reading stamp and a scribe stamp of one run as a Violation naming the session and file; the test asserts the exact violation, and history list prints the summary while the CLI test exits one on a breach
  evidence: internal/core/history/separation.go:62 — "func SessionSeparation(repoRoot, rootSHA string) (SeparationReport, error) {"
  evidence: internal/core/history/separation_test.go:96 — "func TestSeparationNamesATranscriptCarryingBothStampsOfOneRun"
  evidence: internal/surface/cli/history.go:172 — "if rep, err := history.SessionSeparation(repoRoot, rootSHA); err == nil {"
  evidence: internal/surface/cli/history_separation_surface_test.go:47 — "func TestHistorySeparationNamesABreachAndExitsOne"
- ac-6 — MET: two transcripts each carrying one stamp of one run yield no violation, two transcripts and two stamped counted, the run named, and a summary saying no retained transcript carries two stamps of one run; a reading stamp and a scribe stamp of two different runs are likewise no violation
  evidence: internal/core/history/separation_test.go:139 — "func TestSeparationReportsNoTranscriptCarryingTwoStamps"
  evidence: internal/core/history/separation_test.go:119 — "func TestSeparationIgnoresTwoStampsOfTwoRuns"
- ac-7 — MET: an empty store, and a store whose transcripts carry no stamp, set Unobserved with a reason rather than reporting clean; the CLI renders it and the smoke lane runs history separation over an empty store expecting exit 0
  evidence: internal/core/history/separation.go:115 — "rep.Unobserved, rep.Reason = true, reasonNoTranscript"
  evidence: internal/core/history/separation_test.go:168 — "func TestSeparationReportsAnEmptyStoreAsUnobserved"
  evidence: internal/surface/cli/history_separation_surface_test.go:19 — "func TestHistorySeparationRendersUnobservedOnAnEmptyStore"
  evidence: evals/smoke_test.go:90 — "{[]string{"history", "separation"}, true, true},"
- ac-8 — MET: the manifest is promoted last through reading.WriteRunArtefact into the run's directory, which the exclusion floor denies to every assembly, and the read-block eval fixture plants a promoted scribe-manifest.json beside a run as EXHAUST that no reading may carry; the tagged eval passes at BASE
  evidence: internal/core/scribe/ingest.go:259 — "rel, err := reading.WriteRunArtefact(req.RepoRoot, out.Run, ManifestFileName, m)"
  evidence: evals/coldreading_fixture_test.go:230 — ""repo:.abcd/development/readings/rdg-2608300900000001/scribe-manifest.json","
  evidence: internal/core/scribe/ingest_test.go:395 — "func TestScribeIngestPromotesTheManifestLast"

Gap audit:
- honoured:
  - the allow list is derived from the ledger's declared directories and fails closed
    evidence: internal/core/scribe/assemble_test.go:171 — "func TestAllowListIsDerivedFromLedgerDirs"
    evidence: internal/core/scribe/assemble_test.go:147 — "func TestAssertAllowListFailsClosed"
  - a reading bundle and a scribe context each carry a per-run stamp matched exactly, and a session that merely reads the documentation carries none
    evidence: internal/core/scribe/assemble_test.go:208 — "func TestScribeContextCarriesThePerRunStamp"
    evidence: internal/core/reading/manifest_test.go:39 — "func TestBundleCarriesTheReadingStampOfItsRun"
    evidence: internal/core/sessionkind/sessionkind_test.go:18 — "func TestStampsArePerRunAndMatchedExactly"
    evidence: internal/core/history/separation_test.go:79 — "func TestARecordWithoutContextStampsStillParses"
  - ingest proves the context hash and lands admissions and surprises through the capture stores, meeting the ordering gate
    evidence: internal/core/scribe/ingest_test.go:285 — "func TestScribeIngestProvesTheContextHash"
    evidence: internal/core/scribe/ingest_test.go:443 — "func TestScribeIngestWritesAdmissionsAndSurprises"
    evidence: internal/core/scribe/ingest_test.go:379 — "func TestScribeIngestRefusesBeforeTheComparativeRun"
  - the manifest is parked at assembly and promoted only after every record landed, write-once beside the run
    evidence: internal/core/scribe/ingest.go:256 — "if len(res.Landed()) == 0 {"
    evidence: internal/core/reading/ingest_comparative_test.go:241 — "func TestWriteRunArtefactIsWriteOnce"
  - surfaces: the plugin page, the brief's surface index row and the definition's Delivery section name the verb
    evidence: commands/scribe.md:1 — "scribe"
    evidence: .abcd/development/brief/04-surfaces/README.md:47 — "| 32 | `/abcd:scribe` | shipped |"
    evidence: internal/core/lint/scribecontract_test.go:621 — "func TestScribeDeliveryNamesTheVerb"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609020626046719 — survived: assemble refuses a run without its commit marker and reads the ledger's committed directories plus the supplied text; nothing reads uncommitted ledger content
  evidence: internal/core/scribe/assemble_test.go:186 — "func TestScribeAssembleRefusesAnUncommittedRun"
  evidence: internal/core/scribe/assemble_test.go:41 — "func TestScribeContextIsLedgerAndSuppliedTextOnly"
- cond-2609020626048270 — survived: the check reads the native store's metadata only and reports unobserved with a reason when no transcript or no stamped transcript is retained, rather than clean
  evidence: internal/core/history/separation.go:117 — "rep.Unobserved, rep.Reason = true, reasonNoStamp"
  evidence: internal/core/history/separation_test.go:195 — "func TestSeparationReadsMetadataOnly"
- cond-2609020626049512 — survived: the payload is a closed shape: dispositions, admissions and surprises land through the existing stores and any other key is refused by name, so a family with no store cannot be written
  evidence: internal/core/scribe/ingest.go:335 — "func refuseKeys("
  evidence: internal/core/scribe/ingest_test.go:139 — "func TestScribeIngestRefusesAnAuthoredField"

## Grounds

- pursued: we expect an allow-list assembler with a manifest to hold the scribe's inverse access rule by construction as the reading assembler holds the read block, and a session-kind stamp to make the two-sessions property observable in the transcript store; a scribe context reaching shipped-tree material, or a stamp that never lands in a retained transcript, would show it wrong
