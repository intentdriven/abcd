---
id: itd-2609020625400445
slug: a-committed-preset-declares-the-window-it-was-calibrated-for
spec_id: spc-2609020626048722
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-199, itd-198, itd-196]
severity: major
impact: fix
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A committed preset declares the window it was calibrated for, and the record holds every preset to its declaration

Typed links: `builds_on` [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (the committed presets), [itd-198](../shipped/itd-198-an-assembly-reports-what-it-would-cost-before-a-reading-is.md) (the size report), [itd-196](../disciplines/itd-196-a-capability-is-rehearsed-end-to-end-before-it-ships-over-a.md) (rehearsal before shipping); `refines` [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (one entry per position in three parts, a declared window per entry).

## Press Release

> **A reading can be handed to a reader.** The committed preset file holds one entry per position, and each entry states three things: the object set the run is about (which records and which delivered paths), the kinds admitted within it (every kind the position's definition reads, while the object set fits the reader's window; a kind that proves useless leaves by a commit), and the estimated-token window it was calibrated for. The assembler applies the entry for the position it was invoked at, with no operand, and its own eval assembles the entry for every assembling position by dry run and fails when a result exceeds its declaration, so drift past the window is caught by the cold-reading eval lane, which is not yet a required check, rather than by the reader. An entry whose object set names one record is proven to carry that record's material and nothing else. The assembler still enforces no budget at invocation; the entry states the bound, the eval checks it, and the size report goes on saying what a run would cost, stating the figure whenever it is over the two-hundred-thousand-token target. Changing any part of an entry is a commit, which is how the object set grows and how a kind leaves the default over time.

> "I want to point the entailment reading at the claim record and know before I spend a run that the bundle fits the reader," said an AI/agent researcher who commissions cold readings against their own repository. "And when the repository grows past what the entry was measured at, I want the build to tell me, not the reader."

## Why This Matters

The instrument was found correct and undeliverable one day before release: about 9.8 MB per position, roughly 2.45 million estimated tokens ([iss-2608311501186646](../../../work/issues/resolved/iss-2608311501186646-the-assembled-input-for-a-real-reading-of-this-repository-is.md)). [itd-198](../shipped/itd-198-an-assembly-reports-what-it-would-cost-before-a-reading-is.md) added the size report and [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) the committed presets. Measured on the v0.7.0 tree (commit 8f68ffb3) by dry-run assembly under the narrower of the two named presets that file then carried, the widening position assembles 257,835 bytes (about 67 thousand estimated tokens), the entailment position 942,074 bytes (about 245 thousand), and the detection position 751,612 bytes (about 195 thousand). Under the wider preset the detection position is 2,528,029 bytes (about 657 thousand). The entailment and detection assemblies exceed or sit at the edge of a 200-thousand-token reader, and nothing in the record says what window any preset was calibrated for.

Two gaps in itd-199's delivery compound this. Every committed preset carries an empty record list, and the only assembly under a record-id selector in the test corpus selects nothing, so the positive half of the selector grammar has never been shown to carry a record's material. And the widening reading's bound, stated in advance by the design, is a construal of one or two sentences and a glossary of three to six terms, which is a statement about what is passed, not about what the tree weighs.

The design fixes the invocation at a position and a target state and places what a reading is handed in the record; adr-2609021016286571 restores that, and the preset file is the record it points at. One entry per position, in the three parts the framework names, is the shape that lets a reviewer see in one place what a position reads, what it weighs and where it came from.

This intent lands in Phase A, before any reading runs, after the two-operand invocation and itd-194 and before the comparative channel and the reading-occasioned origin; the origin lands in Phase A too, before the first promotion of an accepted reading item, because the origin key is written only by commands.

This intent closes the critical issue above, resolves [iss-2609012259585189](../../../work/issues/resolved/iss-2609012259585189-nothing-reports-the-proportion-of-intents-that-carry-a-mecha.md), which asks that the entailment reading's yield bound be stated beside its findings, and resolves [iss-2608311501240566](../../../work/issues/resolved/iss-2608311501240566-three-of-the-four-reading-positions-receive-a-byte-identical.md), whose item-set distinctness itd-199 addressed by a selector-set proxy: the maintainer ruled on 2026-09-02 that the preset entry is the one configuration surface for a position's object set, kinds and window, so the entry each position is handed is what makes its item set its own, and the spec pins each default item set by digest.

## What's In Scope

- **One entry per position, in three committed parts.** The preset file moves to schema version 2, where each position's entry states its object set (records by id and delivered paths), the kinds admitted within it, and the window it was calibrated for. The named presets and their inheritance go with the operand that chose between them. At version 2 a position without a declaration is refused at load, and a version 1 file holding one preset goes on loading, so an adopter's committed file keeps working until they move it.
- **A declared window per entry** as an estimated-token figure the entry was calibrated to, using the size report's own byte-derived basis, together with the figure measured and the commit measured on.
- **An eval that assembles the entry for every assembling position by dry run** and fails when a result exceeds its declared window, naming the position, the measured figure and the declaration. It runs in the cold-reading eval lane; that lane becoming a gate that blocks a merge is the two recorded issues on it, which this intent names and does not close ([iss-2608311632382737](../../../work/issues/resolved/iss-2608311632382737-the-pre-push-gate-is-blind-to-both-eval-lanes-so-the-read-bl.md), [iss-2608311051046981](../../../work/issues/resolved/iss-2608311051046981-the-new-cold-reading-evals-ci-job-is-not-a-required-status-c.md)).
- **The default object set for Iteration 2 is Iteration 1's shipped state**: the fifteen workstream intents itd-177 to itd-189, itd-198 and itd-199, their specs spc-55 to spc-69, and the packages and pages they delivered that the deny list does not exclude. A test proves that an entry whose object set names one record carries that record's projected material and no other record's.
- **The default kinds are every kind the position's definition reads.** A reading of the object set includes all kinds when the object set fits the reader's window, on the ruling of 2026-09-02. The detection entry names `brief-section`, `glossary-term`, `discipline`, `spec`, `intent-projection`, `doc`, `config`, `source` and `test`, the glossary among them because the design framework's section 9 lists it among what the assembler reads for the position; measured on 2026-09-02, it comes to about 624,000 estimated tokens. The widening entry names `brief-section`, `glossary-term`, `discipline`, `spec`, `doc`, `config`, `source` and `test`, every kind the readings companion's section 5.2 lists for the position, "Brief current text including the construal section; `brief/glossary/`; `intents/disciplines/` including the selection-criteria discipline; `specs/`; the shipped tree where one exists, scoped per 4.5", with the object set narrowing the specs to spc-55 to spc-69 and the tree to the delivered paths; it comes to about 615,000. Both are over the target, stated by the size report on every assembly, and inside a million-token reader, and every source and test item is marked `unscanned` in the manifest. The entailment entry names the claim record and the constraint sources and no tree kind, because its definition's object holds no tree. A kind that proves useless (tests, say) leaves the default entry by a commit, which is recorded; two leaner entries are one such commit away at either tree position. The choice is recorded in each entry's own comment field. The `brief-section` kind reaches every entry that names it as the six chapters itd-194 admits, meta, product, constraints, surfaces, internals and delivery, so the figures stated here were measured over two chapters and are re-measured at landing by the window eval.
- **An assembly over a 200 thousand token target says so.** Two hundred thousand estimated tokens is the target an entry aims at, and any figure inside a reader's window is acceptable; a dry run or an assembly whose estimate exceeds the target prints one line naming the figure and the target, as the maintainer ruled on 2026-09-02, so the operator learns it from the tool rather than from the reader. The committed detection and widening entries are over the target and their declarations say so; the kinds are the lever that keeps the object set while cutting size.
- **A change to any part of an entry is a commit**, reviewed and inside the dirty gate, and the manifest names the entry applied and its hash. That is the mechanism by which the object set grows, a kind such as `test` enters or leaves the default for a position, and a window moves: each is recorded, and none is a flag.
- **The per-position entry resolves the byte-identical item sets.** The maintainer ruled on 2026-09-02 that the preset entry is the one configuration surface for a position's object set, its kinds and its window, so the change that delivers one entry per position resolves [iss-2608311501240566](../../../work/issues/resolved/iss-2608311501240566-three-of-the-four-reading-positions-receive-a-byte-identical.md): each position is handed its own entry, the default item set at each position is pinned by a digest recorded before the change, and two assemblies of one entry produce identical bundles.
- **The entailment size report states the mechanism proportion.** The readings companion bounds the entailment reading by how many intents carry a mechanism claim and asks that the proportion be reported beside the findings. At the entailment position the size report states how many projected intents carried a `## Mechanism` section, how many carried the `None stated.` nullity, and how many carried neither, as the maintainer ruled on 2026-09-02; the workstream's own fifteen shipped intents keep their absent claims as the Iteration 1 baseline and are reported, never backfilled.

## What's Out of Scope

- A budget enforced at invocation. The assembler reports; the entry declares; the eval checks. Nothing is refused for size, and there is nothing at the invocation to widen an entry with.
- A tokenizer. The basis stays byte-derived and labelled as such.
- The comparative position, whose object arrives by its own channel and is bounded by the widening run rather than by the tree; its entry is that channel's, in the shape fixed here.

## Mechanism

We expect a declared window checked by a dry-run eval to keep the instrument deliverable because the failure that reached release was one nobody measured, and a measurement that runs on every committed change in the eval lane is not one a builder forgets. It fails if the byte-derived estimate diverges from a reader's real count by more than the margin the declaration leaves, which the calibration sample recorded in spc-68 bounds.

## Scope Conditions

- The declared window is a property of the entry, not of any reader. Which reader a run is handed to is the operator's choice and is recorded on the run. <!-- cond: cond-2609020626042294 -->
- The eval measures the repository it runs in. A repository that adopts the presets measures its own tree; the figures recorded here are this repository's at the commit named. <!-- cond: cond-2609020626042487 -->
- Recalibration narrows what a position is handed, kind by kind, and is recorded as such in the preset file: A kind that proves useless leaves the default entry by a commit. A narrowing that changes what the position's definition names as its object is a ruling and is not made here. <!-- cond: cond-2609020626045182 -->
- **The impact is `fix`.** The change closes a critical defect, and the schema move is opt-in by version, so no adopter's committed file stops loading. <!-- cond: cond-2609020626048715 -->
- The widening reading receives the whole committed glossary, twenty-four term files across four contexts (ten core, three distribution, two interview and nine ledger), because the readings companion's section 5.2 names `brief/glossary/` whole; the companion's section 5.6 states the bound in advance as "the glossary three to six terms", so the run record's `bounds` list, which `reading ingest` populates from the manifest, states that bound as exceeded, on ruling M5 that a departure is "stated as a bound rather than passed off". The glossary figure the spec records was measured before the ledger context existed and is re-measured at landing. <!-- cond: cond-2609021140329660 -->
- The closing run uses the preset entry at the same hash the opening run's manifest records, because the design framework's section 13 requires the closing run to be over "the same object set"; where a prior run at the same position recorded a different preset hash, the closing run record's `bounds` list, which `reading ingest` populates from the manifest, states the mismatch as a bound on the comparison rather than repairing it. <!-- cond: cond-2609021140328523 -->

## Acceptance Criteria

- **Given** a preset file at schema version 2 in which one position carries no declared window, **when** it is loaded, **then** the assembler refuses and names the position.
- **Given** a preset file at schema version 1 holding one preset, **when** it is loaded, **then** its entries apply per position as they do today and the size report says no window is declared.
- **Given** the committed entries on the current tree, **when** the eval runs, **then** each of the three assembling positions' entries (widening, entailment and detection) measures at or below its declaration, and the eval passes.
- **Given** an entry whose declaration is set below its measured figure, **when** the eval runs, **then** it fails and names the position, the measured figure and the declaration.
- **Given** an entry whose object set names one shipped intent, **when** the entailment assembly runs, **then** the bundle carries that intent's projected fields and no other intent's, and the manifest lists exactly those items.
- **Given** a committed entry with a populated object set, **when** the assembly runs, **then** the bundle carries the listed records' material as the position admits it, the admitted kinds under the listed paths, and no unlisted record's.
- **Given** the preset file, **when** it is read, **then** each declaration carries the figure it was measured at and the commit it was measured on.
- **Given** an assembly or dry run whose estimate exceeds two hundred thousand tokens, **when** the size report renders, **then** it carries one line naming the figure and the target, and an assembly under the target carries no such line.
- **Given** an assembly or dry run at the entailment position, **when** the size report renders, **then** it states how many projected intents carried a `## Mechanism` section, how many carried the `None stated.` nullity, and how many carried neither, and the report at any other position carries no such statement.

## Prior Art

- [itd-198](../shipped/itd-198-an-assembly-reports-what-it-would-cost-before-a-reading-is.md) (the size report), [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (the committed presets), [itd-196](../disciplines/itd-196-a-capability-is-rehearsed-end-to-end-before-it-ships-over-a.md) (rehearsal before shipping).
- The cold-reading rulings of 2026-08-28 in the decision log; adr-2609021016286571 (the invocation is a position and a target state, and the committed preset for the position supplies the rest).

## Open Questions

None. Whether an object set should be expressible as a set of record families was narrowed at itd-199's verdict and is not reopened here.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0aba02f7c8b4 -->
Fidelity review — receipt rcp-0aba02f7c8b4 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:ea33cbf2b90dd31533bd639553432172638c73ea2229e3aa1de7e1323fa40ca3
Input attestations: diff:c1be0000e7a0117947b1e3fedc08244ca77c9aa9 (single commit on 255543c1fa30ae7a9794e98c896b523586ff29ed)@sha256:c8bd7e8580d85a33d11a94f23cf0aeedb7e4fe2d926cdc58fe0a0231ae6a7c8f;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: decodeV2 walks the entries in sorted position order and refuses a nil Window naming the position; the refusal text carries the position and TestPresetV2RefusesAPositionWithoutAWindow passes over a fixture file
  evidence: internal/core/reading/scope.go:509 — "the entry for %s declares no window; at schema_version 2 every position states the window it was calibrated for"
  evidence: internal/core/reading/window_test.go:49 — "func TestPresetV2RefusesAPositionWithoutAWindow(t *testing.T) {"
- ac-2 — MET_WITH_CONCERNS: decodeV1 keeps a one-preset version 1 file loading as the entry set with Window nil and renderSizeReport says so rather than rendering a zero; CONCERN, named: the new tree-narrowing rule retro-applies to version 1 entries, so 'apply per position as they do today' does not hold for a v1 entry that names tree kinds with no path, and this same commit had to add eight root paths to the repository's own version 1 baseline fixture to keep it handing tree material
  evidence: internal/core/reading/scope.go:478 — "out := PresetFile{SchemaVersion: 1, Positions: map[string]PositionEntry{}}"
  evidence: internal/surface/cli/reading.go:399 — "window: none declared (preset schema version 1)"
  evidence: internal/core/reading/window_test.go:86 — "func TestPresetV1LoadsAndReportsNoWindow(t *testing.T) {"
  evidence: internal/core/reading/scope.go:282 — "entry with no path hands nothing from the tree whatever kinds it lists."
  evidence: evals/testdata/cold-reading/baseline/.abcd/config/reading-presets.json:6 — ""paths": ["Makefile", "README.md", "docs", "fence.go", "go.mod", "main.go", "main_test.go", "sitefixture_test.go"]"
- ac-3 — MET: I re-ran the dry-run assembly myself at each assembling position on this tree and every measurement sits at or below its declaration (widening 799,981 of 800,000; entailment 377,674 of 380,000; detection 809,019 of 810,000), and TestEveryCommittedEntryFitsItsDeclaredWindow passes under make evals-cold-reading
  evidence: .abcd/config/reading-presets.json:30 — ""tokens_est": 800000, "measured_tokens_est": 799981,"
  evidence: .abcd/config/reading-presets.json:62 — ""tokens_est": 380000, "measured_tokens_est": 377674,"
  evidence: .abcd/config/reading-presets.json:94 — ""tokens_est": 810000, "measured_tokens_est": 809019,"
  evidence: evals/coldreading_window_test.go:55 — "func TestEveryCommittedEntryFitsItsDeclaredWindow(t *testing.T) {"
- ac-4 — MET: the eval's negative control lowers one declaration to one below the figure just measured, commits it in the detached clone, and requires exactly one breach naming the position, the measured figure and the declaration; TestTheWindowCheckReportsABreach passes
  evidence: evals/coldreading_window_test.go:77 — "func TestTheWindowCheckReportsABreach(t *testing.T) {"
- ac-5 — MET: the positive half of the selector grammar, which itd-199 never exercised, is now shown: an entry naming one shipped intent yields exactly that intent's five projected fields and the manifest item count equals the bundle item count
  evidence: internal/core/reading/window_test.go:262 — "func TestAnObjectSetNamingOneRecordCarriesThatRecordAlone(t *testing.T) {"
  evidence: internal/core/reading/window_test.go:292 — "want := []string{"Acceptance Criteria", "Mechanism", "Press Release", "Scope Conditions", "spec_id"}"
- ac-6 — MET: PositionEntry.selects decides admission per include-table row - a record row narrowed to the named records when the object set reaches it, a tree row narrowed to the listed paths, a constraint source handed whole - and three tests cover the record half, the path half and the whole-handing half
  evidence: internal/core/reading/scope.go:286 — "func (e PositionEntry) selects(c candidate, narrowRecords bool) bool {"
  evidence: internal/core/reading/window_test.go:310 — "func TestAnObjectSetSelectsOnlyListedRecords(t *testing.T) {"
  evidence: internal/core/reading/window_test.go:359 — "func TestAConstraintSourceKindIsHandedWhole(t *testing.T) {"
  evidence: internal/core/reading/window_test.go:392 — "func TestAPathSelectsOnlyTheEntrysKinds(t *testing.T) {"
- ac-7 — MET: every committed window block carries measured_tokens_est, measured_bytes and measured_at; the loader refuses a measurement that is not a sha of 7 to 40 hexadecimal digits and TestShippedEntriesDeclareMeasuredFigures holds the committed file to it; I confirmed 255543c1 is an ancestor of the delivered commit
  evidence: .abcd/config/reading-presets.json:33 — ""measured_at": "255543c1fa30ae7a9794e98c896b523586ff29ed""
  evidence: internal/core/reading/scope.go:519 — "names %q as the commit it was measured on, which is not a commit sha of 7 to 40 hexadecimal digits"
  evidence: internal/core/reading/window_test.go:488 — "func TestShippedEntriesDeclareMeasuredFigures(t *testing.T) {"
- ac-8 — MET: renderSizeReport prints the over-target line only under s.OverTarget, and TestSizeReportNamesAnOverTargetTotal asserts both halves - a small report is not marked over target and a report past 200,000 estimated tokens is; all three of my dry runs printed the line, each being over the target
  evidence: internal/surface/cli/reading.go:412 — "if s.OverTarget { fmt.Fprintf(w, " over target: %s estimated tokens against a target of %s"
  evidence: internal/core/reading/window_test.go:232 — "func TestSizeReportNamesAnOverTargetTotal(t *testing.T) {"
- ac-9 — MET: SizeReport.Mechanism is filled at entailment alone and the CLI prints one line from it; my dry runs show 'mechanism: 12 of 147 projected intents carry a mechanism claim; 0 state none; 135 carry neither' at entailment and no such line at widening or detection
  evidence: internal/surface/cli/reading.go:420 — "mechanism: %d of %d projected intents carry a mechanism claim; %d state none; %d carry neither"
  evidence: internal/core/reading/window_test.go:529 — "func TestEntailmentSizeReportStatesTheMechanismProportion(t *testing.T) {"
  evidence: internal/core/reading/window_test.go:577 — "func TestMechanismProportionIsAbsentAtOtherPositions(t *testing.T) {"

Gap audit:
- honoured:
  - the committed preset file holds one entry per assembling position, each in three parts: the object set, the kinds admitted within it, and the window it was calibrated for
    evidence: .abcd/config/reading-presets.json:2 — ""schema_version": 2, "positions": {"
  - the assembler applies the entry for the position it was invoked at, with no operand
    evidence: internal/core/reading/scope_test.go:879 — "func TestAssemblyAppliesTheCommittedPresetForThePosition(t *testing.T) {"
  - an eval assembles the entry for every assembling position by dry run and fails when a result exceeds its declaration, naming the position, the measured figure and the declaration
    evidence: evals/coldreading_window_test.go:55 — "func TestEveryCommittedEntryFitsItsDeclaredWindow(t *testing.T) {"
  - the default object set for Iteration 2 is Iteration 1's shipped state: itd-177 to itd-189, itd-198 and itd-199, spc-55 to spc-69, and the packages and pages they delivered
    evidence: .abcd/config/reading-presets.json:8 — ""itd-177", "itd-178", "itd-179", "itd-180", "itd-181", "itd-182","
  - an assembly over the two-hundred-thousand-token target prints one line naming the figure and the target, and nothing is refused for size
    evidence: internal/surface/cli/reading.go:413 — "over target: %s estimated tokens against a target of %s; the reader's window decides whether this is acceptable"
  - the entailment size report states how many projected intents carried a Mechanism section, how many the nullity, and how many neither, and nothing is backfilled
    evidence: internal/surface/cli/reading.go:420 — "mechanism: %d of %d projected intents carry a mechanism claim; %d state none; %d carry neither"
  - the per-position entry resolves the byte-identical item sets: each default item set is pinned by a digest and two assemblies of one entry produce identical bundles
    evidence: internal/core/reading/window_test.go:607 — "func TestDefaultItemSetsMatchTheRecordedDigests(t *testing.T) {"
    evidence: internal/core/reading/window_test.go:706 — "func TestTwoAssembliesOfOneEntryAreByteIdentical(t *testing.T) {"
  - a change to any part of an entry is a commit, reviewed and inside the dirty gate, and the manifest names the entry applied and its hash
    evidence: internal/core/reading/window_test.go:736 — "func TestAppliedEntryHashIsTheEntrysOwn(t *testing.T) {"
  - the departures the design documents bound are stated on the run record rather than passed off (ruling M5): the glossary count against companion 5.6 and a preset-hash mismatch against a prior run at the same position
    evidence: internal/core/reading/ingest.go:246 — "the glossary bound is exceeded: the readings companion's section 5.6 states the bound in advance as three to six terms"
- diverged:
  - the widening entry comes to about 615,000 estimated tokens and the detection entry to about 624,000 - delivered as 799,981 and 809,019, roughly thirty per cent higher, because itd-194 enlarged the brief-section rows from two chapters to six; the intent's own clause anticipating re-measurement at landing is what the delivery honoured, and the delta is disclosed in each entry's comment
    evidence: .abcd/development/intents/shipped/itd-2609020625400445-a-committed-preset-declares-the-window-it-was-calibrated-for.md:43 — "it comes to about 624,000 estimated tokens ... it comes to about 615,000"
    evidence: .abcd/config/reading-presets.json:95 — ""measured_tokens_est": 809019,"
  - the entailment entry sits under the two-hundred-thousand-token target - the spec's pre-landing table put it at about 121,000; delivered at 377,674, over the target and stated as over on every assembly at that position
    evidence: .abcd/development/specs/closed/spc-2609020626048722-a-committed-preset-declares-the-window-it-was-calibrated-for.md:384 — "| entailment | about 121,000 | 130,000 |"
    evidence: .abcd/config/reading-presets.json:63 — ""measured_tokens_est": 377674,"
  - the schema move is opt-in by version so a version 1 file goes on applying as it does today - delivered as loading only: the new rule that an entry with no path hands nothing from the tree retro-applies to version 1 entries, and the repository's own version 1 baseline fixture had to gain eight root paths in this same commit
    evidence: evals/testdata/cold-reading/baseline/.abcd/config/reading-presets.json:6 — ""paths": ["Makefile", "README.md", "docs", "fence.go", "go.mod", "main.go", "main_test.go", "sitefixture_test.go"]"
    evidence: internal/core/reading/fixture_test.go:266 — "var fixtureTreePaths = []string{"
  - every new behaviour has a test watched fail before the change and pass after - the loader was implemented before its tests were written, and red was shown instead by eleven mutations applied on a scratch copy; the implementer disclosed this rather than claiming the sequence
    evidence: CLAUDE.md:216 — "Every new behaviour has a test watched fail before the change and pass after."
  - the widening reading receives twenty-four glossary term files across four contexts (ten core, three distribution, two interview, nine ledger), the figure divergence register 17 and the scope condition both state - the delivered tree carries thirty-two term files (eighteen core) and the committed widening entry hands 38 glossary-term items, so the bound is exceeded by more than the register recorded
    evidence: .abcd/development/research/notes/2026-09-02-iteration-2-divergence-register.md:33 — "twenty-four term files across four contexts (ten core, three distribution, two interview and nine ledger)"
    evidence: .abcd/config/reading-presets.json:28 — ""kinds": ["brief-section", "glossary-term", "discipline", "spec", "doc", "config", "source", "test"]"
  - the committed preset file holds one entry per position - delivered as one entry per ASSEMBLING position; the comparative position carries no entry and is exempted by name in the window eval, which the intent's What's Out of Scope section states
    evidence: evals/coldreading_window_test.go:33 — "The comparative position is exempt BY NAME"
- missing:
  - the default object set IS Iteration 1's shipped state - nothing mechanical checks that the hand-listed set COVERS the workstream; TestShippedObjectRecordsAllResolve proves only that every listed id resolves to a record and every listed path exists, so a workstream record omitted from the list is caught by review alone
    evidence: internal/core/reading/window_test.go:427 — "func TestShippedObjectRecordsAllResolve(t *testing.T) {"
  - the fixture-leak test itd-194's spc-2609021003136831 ac-5 names lands with a coverage row - the test is delivered and passes, but no coverage row was added for it, the implementer's stated ground being that the itd-194 spec's enumeration assigns none
    evidence: evals/coldreading_test.go:768 — "func TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset(t *testing.T) {"

Scope-condition dispositions:
- cond-2609020626042294 — survived: the window lives on the entry as PositionEntry.Window and nothing in the assembler or the eval consults a reader; which reader a run was handed to is recorded separately on the run record as Instrument.Model
  evidence: .abcd/config/reading-presets.json:29 — ""window": { "tokens_est": 800000,"
  evidence: internal/core/reading/ingest.go:133 — "Model string `json:"model"`"
- cond-2609020626042487 — survived: the eval clones the repository's own HEAD detached and dry-run-assembles it, and the committed declarations name this repository's commit 255543c1 as what they were measured on, not any figure carried in from elsewhere
  evidence: evals/coldreading_window_test.go:55 — "func TestEveryCommittedEntryFitsItsDeclaredWindow(t *testing.T) {"
  evidence: .abcd/config/reading-presets.json:33 — ""measured_at": "255543c1fa30ae7a9794e98c896b523586ff29ed""
- cond-2609020626045182 — survived: each entry's comment records the measured leaner alternatives kind by kind (widening without test about 521,000, without source and test about 295,000), and an uncommitted preset file refuses to load, so a recalibration cannot happen except as a reviewed commit
  evidence: .abcd/config/reading-presets.json:5 — "measured here, without test it is about 521,000 estimated tokens and without source and test about 295,000, both still over the target"
  evidence: internal/core/reading/scope_test.go:334 — "func TestUncommittedPresetRefuses(t *testing.T) {"
- cond-2609020626048715 — narrowed: the intent's impact is fix and a version 1 file still parses and loads, but the new tree-narrowing rule retro-applies to version 1 entries, so an adopter's committed file can go on loading while handing strictly less than it did
  narrowing: holds for LOADING alone, not for behaviour: a version 1 entry that names doc, config, source or test with no path now hands nothing from the tree, and this same commit added eight root paths to the repository's own version 1 baseline fixture to keep it handing its tree material
  evidence: internal/core/reading/scope.go:282 — "entry with no path hands nothing from the tree whatever kinds it lists."
  evidence: evals/testdata/cold-reading/baseline/.abcd/config/reading-presets.json:6 — ""paths": ["Makefile", "README.md", "docs", "fence.go", "go.mod", "main.go", "main_test.go", "sitefixture_test.go"]"
- cond-2609021140329660 — narrowed: the mechanism the condition rests on is delivered - the widening entry admits glossary-term by kind alone so brief/glossary/ travels whole, and statedBounds names the companion's section 5.6 bound as exceeded on ruling M5 rather than narrowing the glossary silently - but the condition's own enumeration of the glossary is contradicted by the delivered tree
  narrowing: holds for the whole-glossary mechanism and the stated-bound path, not for the figure: the condition and divergence register 17 both say twenty-four term files across four contexts (ten core, three distribution, two interview, nine ledger), while the delivered tree carries thirty-two term files (eighteen core) and the committed widening entry hands 38 glossary-term items; and the bounds statement itself is demonstrated only over a synthetic manifest in unit test, no reading run having yet been ingested in this repository
  evidence: internal/core/reading/ingest.go:246 — "the glossary bound is exceeded: the readings companion's section 5.6 states the bound in advance as three to six terms"
  evidence: internal/core/reading/ingest.go:226 — "const glossaryTermBound = 6"
  evidence: internal/core/reading/window_test.go:832 — "func TestRunRecordCarriesTheStatedBounds(t *testing.T) {"
- cond-2609021140328523 — narrowed: priorRunUnderAnotherPreset compares a run's preset hash against the committed runs at the same position and statedBounds states the mismatch as a bound on the comparison rather than repairing it, exactly as the condition assumes, but nothing has yet run through it
  narrowing: holds as a delivered and unit-tested mechanism only: no reading run has been ingested in this repository, so neither an opening nor a closing run has exercised the preset-hash comparison, and the demonstration is over synthetic manifests written directly into a temporary root
  evidence: internal/core/reading/ingest.go:253 — "the object set is not the one a prior run at the %s position read"
  evidence: internal/core/reading/ingest.go:267 — "func priorRunUnderAnotherPreset(root *os.Root, m Manifest) (string, string, bool) {"
  evidence: internal/core/reading/window_test.go:846 — "t.Run("seven terms after a prior run under another hash", func(t *testing.T) {"
## Grounds

- pursued: we expect a declared window per preset checked by a dry-run eval on every change to keep the instrument deliverable, because the size defect reached release only because nobody measured; a preset that passes its eval and still overflows the reader it was calibrated for by more than the recorded margin would show it wrong
