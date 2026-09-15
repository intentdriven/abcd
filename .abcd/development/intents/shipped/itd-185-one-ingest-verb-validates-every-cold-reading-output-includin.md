---
id: itd-185
slug: one-ingest-verb-validates-every-cold-reading-output-includin
spec_id: spc-63
kind: bundle-member
suggested_kind: bundle-member
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# One ingest verb validates every cold-reading output — including what the reading was licensed to produce, not only what it saw

## Press Release

> **A reading that quietly exceeds its licence is refused at ingest.** The
> output contract carries, per item, an identifier the ingest verb mints
> (adr-45 mint, run-scoped — the reading itself holds no mint, so ids are
> assigned at validation, never self-supplied) and, per run, the
> instrument name, manifest reference, target state, and regime value. Ingest validates before any
> durable record is written — malformed output is rejected without partial
> writes — and then checks the supply regime: a detection-position output
> attaching a proposed resolution, a comparative output containing an
> ordering, a score or a single named recommendation, an entailment output
> in which a surfaced claim arrives already dispositioned — each is refused
> and the offending item named. The failure this catches is silent
> everywhere else: such an output would pass every structural test while
> violating the one property the position is defined by.

## Ruled

- **Ruled (maintainer, 2026-08-28; decision log):** build the supply-regime check now, as
  a regime field on the output contract validated at ingest — the form
  drafted here. Ground: a reading that quietly proposes or aggregates
  passes every other planned test; the read-block eval covers what a
  reading saw, never what it was licensed to produce, and the failure is
  silent. The ruling matches the shape implemented below — structural
  signatures at `evaluative` (aggregation) and `registrative`
  (fix-proposing), `explicative` checked as record shape rather than
  prose, and the record-and-flag degradation path for noisy signatures.
  Still open (recorded open question): whether the signatures lint cleanly in practice.

## What's In Scope

- The ingest verb: JSON in, validation, durable reading records out; the
  read-block and contract written in one place.
- The regime table, checked structurally at ingest against a strict schema
  (unknown fields are refused, so every violation is a named field, never
  a guess): `generative` (no regime-specific refusal — the constraint
  falls on admission at the dispositioning end); `explicative` (the input
  schema carries no disposition field, so a dispositioned claim is refused
  as an unknown field — the violation is impossible to express, not merely
  caught); `evaluative` (refuses an explicit rank or score field, or an
  item marked recommended; arrangement order is never refused — items
  arrive in document order by mandate); `registrative` (refuses a
  resolution field attached to a detection). Semantic violations — prose
  that ranks, settles, or proposes without the fields — are checked too;
  any signature degrades only on observed noise, per the rung bullet
  below, never pre-emptively.
- Item shapes are position-typed and validated per position: generative —
  configuration / what-admits-it; explicative — claim / type /
  what-implies-it; evaluative — candidate / criterion / characterisation;
  registrative — tension / constraint-in-play / why. The pattern named is
  an envelope field, validated once for every position; the strict schema
  is per-position and unknown fields still refuse.
- The regime's source of truth is the definition, resolved through the
  run's position: an output's self-declared regime that mismatches it is
  itself a refusal; no operator input can set a regime.
- Named provenance is enforced here, for every regime: each item carries a
  non-empty pattern-basis field or ingest refuses. The definitions
  instruct it; this contract enforces it; nothing else does.
- Ingest is staged: outputs validate into a write-aside area, records move
  together, and the run-metadata record lands last as the commit marker —
  an orphaned stage found on the next invocation is reported and cleared,
  so a crash mid-ingest leaves evidence, never half a run.
- The manifest reference is the content hash of the assembler's manifest —
  the one unforgeable form; a reference that resolves to nothing, or to a
  manifest whose hash disagrees, refuses the run.
- Instrument identity in run metadata (ruled 2026-08-28): "instrument
  name" comprises the model identity, the definition's content hash, and
  the assembler version — two runs claiming the same instrument are
  thereby provably the same, which the closing-run comparison requires.
- Refusal granularity: an item-level violation refuses the item and lands
  the rest; a list-level violation refuses the run. A refused run still
  writes a refusal record — run metadata and the named reason, no
  items — so the event is durable and a rerun is a new run.
- Enforcement per the ruling: every check ships enforced —
  refusal is the default at birth. The degradation path is reserved, not
  pre-taken: only where a signature proves noisy in practice does that
  check degrade to record-and-flag, and the degradation is itself a
  recorded decision that weakens the claimed property from enforced to
  observed, said out loud per `loud-staging`. (2026-08-28 review: the earlier
  observed-first stance for semantic signatures pre-empted the recorded open question, and is corrected here. Standing tension with
  the repo's widen-options promotion clause — "calibrated before it
  gates" — recorded as a standing tension; the ruled design governs the instrument
  meanwhile.)

## What's Out of Scope

- The definitions that state each regime — a sibling intent.
- Whether the regime signatures lint cleanly — untested; the degradation
  path exists precisely because of it.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a malformed reading output, **when** it is ingested, **then** ingest
  refuses and names the offending field, and no durable record exists for that
  run in the reading-record family, in the readings tree, or in the stage.
- **Given** a fault injected after the reading records are staged and before the
  run-metadata commit marker is written, **when** ingest runs, **then** no
  reading records and no run metadata are durable for that run, and the next
  invocation names the orphaned stage and clears it.
- **Given** a run, **when** its metadata is read, **then** the manifest
  reference resolves to that run's manifest — the stored hash equalling the
  content hash of the manifest itself — and a reference that resolves to
  nothing, or to a manifest whose hash disagrees, refuses the run.
- **Given** a `registrative` output whose item carries a reserved name
  (`resolution`, `fix`, `remedy`), **when** it is ingested, **then** ingest
  refuses and names the item's ordinal, the field, and the licence breached.
- **Given** a `registrative` output whose item body REPORTS a fix the passed
  material carries — quoting the record's own instruction, or naming the section
  that says it is merged while another says it is pending — **when** it is
  ingested, **then** ingest lands the item and raises nothing against it. Only a
  `fix`, `remedy` or `resolution` carried as one of the item's OWN keys refuses
  (ruling of 2026-09-01 on iss-2608311518056854; the registry of prose signatures
  this criterion was written against is withdrawn).
- **Given** an `evaluative` output carrying a `rank`, `score`, `order` or
  `recommended` field, **when** it is ingested, **then** ingest refuses and
  names the field.
- **Given** an `evaluative` output whose items are merely arranged in an order
  and carry no reserved field, **when** it is ingested, **then** ingest accepts
  it: arrangement order is never inspected and never refused.
- **Given** an `explicative` output in which a surfaced claim carries a
  disposition-bearing field — `disposition`, `status`, or any field outside the
  explicative body schema — **when** it is ingested, **then** ingest refuses and
  names the field.
- **Given** an `explicative` output whose claim body REPORTS a disposition the
  passed material carries — quoting a record's `disposition:` line, or saying
  that a clause settles a licensing question — **when** it is ingested, **then**
  ingest lands the item and raises nothing against it. Only a `disposition` or
  `status` carried as one of the item's OWN keys refuses (ruling of 2026-09-01 on
  iss-2608311518056854; the registry of prose signatures this criterion was
  written against is withdrawn).
- **Given** a run refused at list level, **when** the refusal completes, **then**
  a refusal record exists carrying the run metadata and the named reason, and no
  reading records exist for that run.
- **Given** an item at any of the four regimes whose `pattern_named` envelope
  field is empty or absent, **when** it is ingested, **then** ingest refuses that
  item, without exception at any regime.
- **Given** an output whose self-declared regime disagrees with the regime stated
  by the definition its position resolves to, **when** it is ingested, **then**
  ingest refuses the run at list level.
- **Given** an accepted output, **when** its reading records are read, **then**
  every item carries an identifier the verb minted, and a payload supplying its
  own item identifier is refused as an unknown field.

**Disclosed residue (ac-5 and ac-9).** The gate is STRUCTURAL, and both criteria
now say so: a reserved name is matched against the item's own keys and never
against the words inside a value. The residue is what that costs, and it runs in
one direction only.

**Under-catching** is the whole of it. A reading that proposes a fix, ranks a
candidate or dispositions a claim in PROSE, without carrying a field for it,
raises nothing — because nothing reads prose any more. That is deliberate. The
registry that did read prose could not tell a reading that PROPOSES from one that
REPORTS somebody else proposing, and the second is most of what a reading
legitimately does: measured against a constructed corpus of thirty-four realistic
outputs, fourteen were caught, every one for quoting the document it read. The
detector was withdrawn rather than softened, because recording the hit instead of
refusing it leaves a gate that still reads prose and still cannot make the
distinction. `internal/core/reading/ingest_corpus_test.go` carries that corpus and
holds the gate to it in both directions.

**Over-catching** is closed by construction, and what closes it is the same fold
that used to serve the detectors: keys are compared folded, so a reserved name
respelled in code points that render the same — a fullwidth `ｄｉｓｐｏｓｉｔｉｏｎ`,
an `ﬁ` ligature in `fix`, a zero-width space inside `score` — refuses as itself,
while no fold ever reaches a value. A script-**confusable** key is still not
folded — a Cyrillic that is not the Latin one, and no normalisation equates them —
so a reserved name spelled in a confusable script lands as an unknown field
rather than as a named licence breach. It refuses either way; what is lost is the
account of WHY. Closing that needs a confusables table, which is a new dependency
and a maintainer's decision; until it is taken, this class is open.

Their structural halves — ac-4, ac-6 and ac-8 — carry no residue of either kind,
because a field is present or it is not.

**Two verdicts in the ingested audit describe superseded behaviour.** The
fidelity review under receipt `rcp-fe3450ca55ff` records ac-5 and ac-9 as MET on
the strength of a REFUSAL, which is what the verb did when that review ran. The
same review's own diverged findings are what unmade it: the semantic signatures
became review flags on 2026-08-31 and were withdrawn altogether on 2026-09-01,
and ac-5 and ac-9 above are rewritten twice over to match, so those two verdicts
now describe behaviour this repository no longer has. The Audit Notes below are
left exactly as they were ingested — a receipt records what was found, not a
document to amend — and the two criteria need re-issuing at the next audit. No
other verdict in that receipt is affected.


## Grounds

- pursued: This conjecture is pursued now because the failure it catches is silent everywhere else: a reading that quietly proposes, ranks, or arrives already dispositioned passes every structural test while violating the one property its position is defined by, and the read-block eval covers only what a reading saw, never what it was licensed to produce. Building the gate later would mean trusting outputs produced before anything checked them.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-fe3450ca55ff -->
Fidelity review — receipt rcp-fe3450ca55ff (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:9949d1a3804fb6e68dafb44570193b4d81fdc675cbd0788d44cb457f925370c6
Input attestations: diff:78f6bd1d..8ec6f42f on build/itd-185 (9 commits), merged as 1d84ce26@sha256:ddfe21ba83c51358b6b72aff6780300798143ff1b2fae279e42ea820b753b45e;

Acceptance rollup: MET 11 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: I built the binary from a checkout of 1d84ce26 and ran nine distinct malformed shapes through the real `abcd reading ingest` — not JSON at all, a wrong `_type`, an unknown top-level field, an unknown field nested inside `instrument`, a traversal run_id (`../../etc`), an unknown position, a manifest_sha256 that is not a digest, an empty item list, and a second document appended after the first. Every one exited 2 and named the offending field or value verbatim in the message. The durable half is the part I measured rather than argued: I snapshotted the union of `.abcd/work/issues/readings/**`, `.abcd/development/readings/**` and `.abcd/.work.local/scratch/reading-ingest/**` before the nine runs and after them, and the two sets were identical — no reading record, no run directory, no refusal record, no stage. The ordering that buys this is in the code and it is the order the criterion needs: `Ingest` sweeps, reads, decodes and checks the envelope before it opens anything to write, and `refuse()` — the only writer of a durable refusal — is not reachable until `resolveParkedManifest` has proven the run's identity. Strict decoding is genuine at every declared level: `DisallowUnknownFields` plus a `dec.More()` check, so a trailing second document is refused rather than ignored.
  evidence: internal/core/reading/ingest.go:449 — "dec.DisallowUnknownFields() ... if dec.More() { return Output{}, errors.New(\"reading: the output carries trailing content after the document\") } — measured: nine malformed shapes, all exit 2, zero durable delta across the three locations"
  evidence: internal/core/reading/ingest.go:465 — "func checkEnvelope(out Output) (Position, error) { — run id checked FIRST among the values a path is built from; measured: run_id \"../../etc\" refused before any open"
  evidence: internal/core/reading/ingest.go:394 — "// The run's identity is proven from here ... A refusal below is therefore recordable against that run — nothing reaches refuse() before resolveParkedManifest returns"
- ac-2 — MET_WITH_CONCERNS: The first clause is fully realised and I executed it at both windows of the protocol rather than reading it. An injected fault after the stage marker and again after the ledger batch each leave no run metadata and no reading records for the run, and leave the stage as the evidence a later invocation finds. I proved that can fail, twice: neutering `rollbackRun` so the sweep clears the stage but leaves the ledger records turns TestRunMetadataLandsLast/after-ledger and TestOrphanedStageIsReportedAndCleared red; writing the commit marker before the ledger batch turns TestRunMetadataLandsLast/after-ledger red with `the run metadata landed despite a fault before it`. I then reproduced the sweep independently of the seam at the CLI, against a state I built by hand — a stage plus a committed rdi record plus a person's NOTES.md plus a stage entry named `not-a-run` — and a valid next invocation printed `cleared: orphaned stage(s) of rdg-2608310000009999` and `rolled back: rdi-2608310000009001 removed from the ledger`, removed exactly that record, and left the other two standing. THE CONCERN, and it is the criterion's own second clause failing on a reachable path: the next invocation NAMES the orphan only if it survives to a render. The sweep is the first thing under the lock, before the payload is even read, but the CLI renders the result only on success or on a refusal that recorded a run — so an invocation whose payload fails early clears the orphan and says nothing. I demonstrated it: with an orphan staged and its record `rdi-2608310000008001.md` sitting in the committed ledger, I ran an ingest whose payload fails at `_type`. The record was deleted from the committed tier and the entire operator output was the `_type` error; under `--json` the render is `{"error": ...}` alone, with `rolled_back_records` and `cleared_stages` both dropped. That is precisely the outcome the code's own comment says must not happen, and a retry after a crash carrying a typo is an ordinary way to reach it.
  evidence: internal/core/reading/ingest.go:546 — "func write(...) — stage marker, then capture.IngestReading, then the manifest, then RunFileName LAST; measured: moving the marker earlier turns TestRunMetadataLandsLast red"
  evidence: internal/core/reading/ingest_stage_test.go:40 — "func TestRunMetadataLandsLast(t *testing.T) { for _, step := range []string{faultAfterStage, faultAfterLedger} — both windows; measured red on two independent mutations"
  evidence: internal/surface/cli/reading.go:185 — "if res.RefusalPath != \"\" { _ = render(...) } — the ONLY render on the error path; measured: a payload failing at _type deleted rdi-2608310000008001 from the committed ledger and reported nothing, in both text and --json"
  evidence: internal/core/reading/ingest.go:243 — "// RolledBack names the reading records the sweep REMOVED from the committed ledger ... \"cleared an orphaned stage\" does not tell an operator that records left the ledger with it. — the field is populated at ingest.go:368 and then discarded by the front door"
- ac-3 — MET: Four adversarial probes against the real binary, plus the read-back clause the criterion opens with. A run id that resolves to no parked manifest refuses, naming the path it looked at. A manifest_sha256 of sixty-four zeros refuses, printing both the claim and the manifest's true digest. Tampering with the parked manifest AFTER assembly — I rewrote its target_commit — refuses on the same rule, which is what makes the reference unforgeable rather than merely present. Substituting another run's manifest and citing that manifest's own true hash still refuses, because the manifest is then read back for its own RunID and position. On the read-back half: after an accepted run I recomputed the hashes myself, and run.json's `manifest_sha256` equals the sha-256 of the manifest promoted beside it in the readings tree AND of the parked original, and the promoted bytes are byte-identical to the parked bytes — so the stored hash equals the content hash of the manifest the reference resolves to, through the package's own encoder, with no drift.
  evidence: internal/core/reading/ingest.go:505 — "func resolveParkedManifest(root *os.Root, out Output) (Manifest, error) — measured: a citation resolving to nothing refuses naming the path"
  evidence: internal/core/reading/ingest.go:525 — "if got := sha256Hex(raw); got != out.ManifestSHA256 { — measured: both a forged hash and a manifest tampered after assembly refuse here"
  evidence: internal/core/reading/ingest.go:533 — "if m.RunID != out.RunID { — measured: another run's manifest, cited by its own true hash, still refuses"
  evidence: internal/core/reading/ingest.go:586 — "writeJSONIn(root, runRel+\"/\"+ManifestFileName, m) — measured: sha256(promoted) == sha256(parked) == run.json manifest_sha256, promoted bytes identical to parked bytes"
- ac-4 — MET: All three registrative reserved names were run through the real binary at the detection position and each refused with every element the criterion demands present in one message: the item's ordinal, the field, and the licence breached. `resolution` produced `item 1 carries the reserved registrative field "resolution": a registrative reading names a tension and the constraint in play; proposing the resolution is not its licence`; `fix` and `remedy` together produced the same shape naming both. I also confirmed the granularity the criterion implies rather than states: with one offending item and one clean item in the same payload, the run LANDS one record and reports `item 1 (reserved-name)` — the reserved name refuses the item, not the run. The reserved-name check runs before the strict key-set check on purpose, so a licence breach gets an account of the licence rather than a bare `unknown field`. Recorded under diverged because it is real and it is not a criterion failure: the standing guard for this criterion is mutation-vacuous — emptying `ReservedNames[RegimeRegistrative]` leaves the whole `internal/core/reading` package green, because the test iterates the very table it is testing.
  evidence: internal/core/reading/ingest_regime.go:51 — "RegimeRegistrative: {\"fix\", \"remedy\", \"resolution\"}, — measured: each refuses, naming ordinal, field and licence"
  evidence: internal/core/reading/ingest_regime.go:287 — "if named := present(fields, ReservedNames[def.Regime]); len(named) > 0 { ... Rule: \"reserved-name\" — checked FIRST, before the unknown-key rule"
  evidence: internal/core/reading/ingest_regime.go:66 — "RegimeRegistrative: \"a registrative reading names a tension and the constraint in play; proposing the resolution is not its licence\" — the licence sentence the refusal quoted verbatim"
- ac-5 — MET: A detection item whose `why_a_tension` reads `The fix is to rewrite the charter.` was refused by the real binary with `item 1 (RG-REG-FIXPROPOSAL): item 1 matches the registered signature RG-REG-FIXPROPOSAL` — the item and the signature id both named, which is exactly what the criterion asks for. The gate is over the item BODY, and it is not evaded by the classes that used to defeat it: I confirmed at the CLI that the same phrase with a non-breaking space, an ideographic space, a soft hyphen inside `remedy`, a zero-width space inside `resolution`, and a fi-ligature `the ﬁx is` all refuse. I proved the fold is load-bearing rather than decorative by two mutations: removing `foldForMatching` from `bodyText`, and dropping the NFKC step alone, each turn TestTheRegimeGateIsNotEvadedByInvisibleRunes red. I also proved the enforce mode is guarded — degrading a signature to `flag` turns TestEverySignatureShipsEnforced red, so the reserved degradation path cannot be taken silently. The disclosed residue holds as written: a Cyrillic homoglyph in `the fiх is` is accepted, which is the class itd-185 and spc-63 both name as open. Two things outside the criterion's letter but inside its subject are recorded under diverged: the registry refuses a great deal of legitimate quotation, and the `pattern` envelope field is a signature-free channel through which the registry's own phrasing lands durably.
  evidence: internal/core/reading/ingest_regime.go:120 — "ID: \"RG-REG-FIXPROPOSAL\", Regime: RegimeRegistrative, Mode: SignatureEnforce, — measured: refuses, naming item and signature id"
  evidence: internal/core/reading/ingest_regime.go:345 — "if s.Pattern.MatchString(bodyText(fields, bodyFields)) { — measured: removing the fold from bodyText turns TestTheRegimeGateIsNotEvadedByInvisibleRunes red"
  evidence: internal/core/reading/ingest_regime.go:481 — "func foldForMatching(text string) string — measured: dropping norm.NFKC alone also turns that test red"
  evidence: internal/core/reading/ingest_regime_test.go:296 — "func TestEverySignatureShipsEnforced(t *testing.T) { if len(Signatures) < 4 { — measured: degrading RG-EXPL-DISPOSITION to flag turns it red; the count floor stops the table being emptied"
- ac-6 — MET: All four names the criterion enumerates were run separately through the real binary at the comparative position, and each refused naming the field: `rank`, `score`, `order`, `recommended`, every one producing `item 1 carries the reserved evaluative field "<name>": an evaluative reading characterises candidates against criteria; ordering, scoring or recommending among them is not its licence`. The table in the code is exactly the criterion's four names and no more. As with ac-4, the behaviour is delivered and the standing guard for it is not: emptying `ReservedNames[RegimeEvaluative]` leaves the package green, so nothing would notice if a name left the table. Recorded under diverged rather than held against the criterion, because what ingest actually does is what ac-6 promises.
  evidence: internal/core/reading/ingest_regime.go:52 — "RegimeEvaluative:   {\"order\", \"rank\", \"recommended\", \"score\"}, — measured: all four refuse, each naming its field"
  evidence: internal/core/reading/ingest_regime_test.go:127 — "for _, field := range ReservedNames[RegimeEvaluative] { — measured: emptying the table leaves the package green, so the loop body never runs"
- ac-7 — MET: A comparative payload of three items in a deliberate arrangement — candidate ids C-best, C-mid, C-worst, characterised as cheapest, middling and dearest, in that order and carrying no reserved field — was ACCEPTED by the real binary: three records landed, zero refusals, exit 0. The criterion's stronger clause, that arrangement order is never inspected, is a property of the code rather than of one test, and I checked it directly: the item loop's index is used for one thing only, the ordinal that a refusal message quotes, and no comparison anywhere in the ingest path reads an item's position in the list. The reserved table deliberately names a FIELD rather than a property of the list, which is what makes the criterion structurally true rather than accidentally true.
  evidence: internal/core/reading/ingest_regime.go:188 — "for i, raw := range out.Items { ordinal := i + 1 — the index's only use; no ordering comparison exists anywhere in the path"
  evidence: internal/core/reading/ingest_regime_test.go:151 — "func TestEvaluativeDocumentOrderIsNeverRefused(t *testing.T) { — measured at the CLI too: three deliberately arranged items landed, RefusedItems empty"
- ac-8 — MET: The criterion has three arms and I ran all three at the entailment position. `disposition` refuses as a reserved name with the explicative licence quoted; `status` refuses the same way; and an arbitrary field outside the schema — I used `verdict_of_the_reader` — refuses as `unknown-field` naming that field and listing the explicative body (`claim_surfaced`, `claim_type`, `what_implies_it`), which is the criterion's `any field outside the explicative body schema` arm. The intent's design claim that a dispositioned claim is impossible to EXPRESS rather than merely caught is realised: the item key set is closed against the position's own body fields, so there is no way to phrase a disposition-bearing field that is not refused by name. The same guard vacuity as ac-4 and ac-6 applies to the reserved half and is recorded under diverged; the unknown-field arm is guarded independently and is not vacuous.
  evidence: internal/core/reading/ingest_regime.go:54 — "RegimeExplicative:  {\"disposition\", \"status\"}, — measured: both refuse as reserved-name with the licence stated"
  evidence: internal/core/reading/ingest_regime.go:311 — "if len(unknown) > 0 { ... Rule: \"unknown-field\" — measured: verdict_of_the_reader refused, naming the field and the explicative body"
  evidence: internal/core/reading/ingest_regime.go:179 — "allowed := map[string]bool{PatternField: true}; for _, f := range bodyFields { allowed[f] = true } — the closed key set per position"
- ac-9 — MET: An entailment item whose `claim_surfaced` reads `This claim is already accepted by the maintainer.` was refused by the real binary with `item 1 (RG-EXPL-DISPOSITION): item 1 matches the registered signature RG-EXPL-DISPOSITION` — both the item and the signature id named. The signature ships in enforce mode and I proved that is guarded: changing its Mode to SignatureFlag turns TestExplicativeProseDispositionRefused AND TestEverySignatureShipsEnforced red, so a degradation cannot land unnoticed. The detector reads the folded claim body, and I confirmed at the CLI that the narrow no-break-space form of the same phrasing also refuses. The residue is disclosed and accurate: a disposition written in a confusable script is not caught. The false-refusal side of this signature is the largest in the registry and is recorded under diverged — `\bdisposition\s*[:=]` refuses any explicative claim that quotes the token `disposition:` from the document it read, and this repository's own records carry that token everywhere.
  evidence: internal/core/reading/ingest_regime.go:128 — "ID: \"RG-EXPL-DISPOSITION\", Regime: RegimeExplicative, Mode: SignatureEnforce, — measured: refuses naming item and signature id"
  evidence: internal/core/reading/ingest_regime.go:130 — "`(?i)\\b(?:this\\s+claim\\s+is|the\\s+claim\\s+is|it\\s+is)\\s+(?:already\\s+)?(?:accepted|rejected|declined|settled|resolved)\\b|...|\\bdisposition\\s*[:=]`"
  evidence: internal/core/reading/ingest_regime_test.go:244 — "func TestExplicativeProseDispositionRefused(t *testing.T) { — measured: degrading the signature to flag mode turns this and TestEverySignatureShipsEnforced red"
- ac-10 — MET_WITH_CONCERNS: The mainline is realised and I verified it three ways at the CLI. A payload whose self-declared regime disagreed with the definition wrote `.abcd/development/readings/<run>/refusal.json` carrying the run id, position, regime, target_commit, manifest_sha256, the sanitised instrument and the reason WHOLE — uncut, which is the point of carrying it whole — with no reading records for that run and no surviving stage. A run in which every item was refused produced the same. The list is bounded in both dimensions the payload controls: a 100-item all-refused run produced a 4,095-byte reason ending `and 80 more item(s) refused` and 25 terminal lines rather than 100. THE CONCERN is that `a run refused at list level` is broader than the set of refusals that route through `refuse()`, and I demonstrated two classes that refuse the whole run and write no refusal record at all. First, and the ordinary one: a `LoadDefinition` failure is returned bare, AFTER the run's identity is proven. I edited a definition to state another position's regime and ingested a run whose manifest resolved cleanly — the run was refused, exit 2, and `.abcd/development/readings/<run>/` was never created. That contradicts the plugin page's own corrected sentence, which promises that from the identity point a list-level refusal writes `refusal.json`. Second, the write-phase one: an item whose `recordBytes` estimate passes at 1,044,215 but whose assembled record is 1,235,274 bytes — I built it from 520 KB of a repeated IPv4 the ledger redactor lengthens — refuses the run through `capture.IngestReading`, again with no refusal record, leaving a stage and an empty committed directory behind.
  evidence: internal/core/reading/ingest.go:656 — "func refuse(root *os.Root, res *IngestResult, out Output, m Manifest, def Definition, cause error) error { ... Reason: cause.Error() — measured: run metadata plus the whole named reason, no items, no reading records"
  evidence: internal/core/reading/ingest.go:402 — "def, err := LoadDefinition(repoRoot, pos); if err != nil { return err } — a bare return AFTER identity is proven; measured: a drifted definition refuses the run and creates no run directory at all"
  evidence: commands/reading.md:140 — "**A refusal leaves a record once the run's identity is proven** ... From that point a list-level refusal writes `refusal.json` — contradicted by the LoadDefinition leg above"
  evidence: internal/core/reading/ingest.go:567 — "if err != nil { return fmt.Errorf(\"reading: writing the records of run %s: %w\", out.RunID, err) } — the second class; measured reachable at 1,235,274 assembled bytes, and it wrote no refusal record"
- ac-11 — MET: This is the criterion the host asked me to be hardest on, and it holds without exception across forty CLI probes: four regimes multiplied by ten forms of an empty-or-absent pattern — absent, empty string, ASCII spaces, U+200B zero-width space, U+00AD soft hyphen, U+FE0F variation selector, U+034F combining grapheme joiner, U+00A0 non-breaking space, U+2060 word joiner, and U+3164 Hangul filler. Every one of the forty refused the item with rule `named-provenance`, naming the field. The zero-width hole the host flagged is genuinely closed at all four regimes, and I proved the closure is load-bearing by mutation: reverting the check to `strings.TrimSpace(fields[PatternField])` turns TestEmptyPatternNamedRefusesItemAtEveryRegime red at all four positions at once. The same hole IS closed for the body fields — the identical folded-blankness rule runs over every declared body field, and U+200B, soft hyphens and U+00A0 in a body field each produce `missing-body-field`. The ordering matters and is right: the fold-blankness check runs BEFORE the hidden-rune encoder, so a pattern of one tab is judged blank rather than judged as the string `%09`. Two things recorded under diverged rather than against the criterion: U+2800 BRAILLE PATTERN BLANK is accepted as a pattern although it renders as nothing, which contradicts the residue's claim that every invisible rune is dropped; and the criterion names the field `pattern_named` while the wire name is `pattern`.
  evidence: internal/core/reading/ingest_regime.go:299 — "if strings.TrimSpace(foldForMatching(fields[PatternField])) == \"\" { ... Rule: \"named-provenance\" — measured: 40 of 40 forms refuse across all four regimes"
  evidence: internal/core/reading/ingest_regime.go:329 — "if strings.TrimSpace(foldForMatching(fields[f])) == \"\" { missing = append(missing, f) — the body half; measured: ZWSP, soft hyphens and NBSP each produce missing-body-field"
  evidence: internal/core/reading/ingest_regime.go:201 — "// Encode the hidden runes on the way OUT, once the item has been judged ... a pattern of one tab encodes to \"%09\", which is no longer blank"
  evidence: internal/core/reading/ingest.go:98 — "const PatternField = \"pattern\" — the wire name; ac-11 says `pattern_named`"
- ac-12 — MET: A detection payload declaring `regime: generative` was refused at LIST level by the real binary — no item refusals, the whole run — with `the output declares the generative regime and agents/cold-reading-detection.md states registrative; the regime is the definition's property, resolved from the run's position`. The refusal record landed carrying the run metadata and that reason, no reading records existed for the run, and no stage survived. The comparison is against the definition file resolved from the position, not against a table, and I proved the check is guarded: making `checkRegime` unconditionally return nil turns TestSelfDeclaredRegimeMismatchRefusesRun red. On the adjacent case the criteria do not name and the host asked about: I edited `agents/cold-reading-detection.md` to state `regime: evaluative` and ran the binary. Both the bare verb and `reading ingest` exit 2 with `a definition stating another position's licence is refused rather than resolved` — the definition is refused, not resolved into a changed licence, which is what commit 7fa4bbbc claims. The drifted definition is refused BEFORE the payload's regime is ever compared, so a lying definition cannot hand the ingest the wrong licence. Two residues of that adjacent case are recorded under diverged: it leaves no refusal record, and the issue it closes was left open in this checkout.
  evidence: internal/core/reading/ingest_regime.go:141 — "func checkRegime(out Output, def Definition) error { if out.Regime == def.Regime { return nil } — measured: making this unconditional turns TestSelfDeclaredRegimeMismatchRefusesRun red"
  evidence: internal/core/reading/ingest.go:408 — "if err := checkRegime(out, def); err != nil { return refuse(root, res, out, manifest, def, err) } — list level, and it writes the refusal record"
  evidence: internal/core/reading/definitions.go:95 — "states regime %q under position %s, which carries the %s regime ... refused rather than resolved — measured: the real binary exits 2 on both `abcd reading` and `abcd reading ingest` with a drifted definition"
- ac-13 — MET: Both halves executed against the real binary. On acceptance, the committed reading record carries `id: "rdi-2608311447227460"` — a family tag plus a sixteen-digit UTC-stamp-and-entropy id in adr-45's mint shape — and the run record's `records` array names the same id and its path; nothing in the payload supplied it, because the payload schema has nowhere to put one. On refusal, I tried two names the criterion's spirit covers: `id` and `item_id`. Each was refused as `unknown-field`, naming the field and stating in the message itself that `the item identity is the verb's to mint and the envelope is the verb's to compose, so neither has a field here` — the criterion's exact requirement that a payload-supplied identifier is refused as an unknown field, not silently dropped and not honoured. The standing test pins three literal names (`id`, `rdi`, `item_id`) rather than iterating a table, so unlike ac-4/ac-6/ac-8 this guard is not vacuous. Builder deviation 3 verified: the mint is capture's, taken under the ledger lock where the collision probe can see the tree, and the test asserts the mint's SHAPE against `^rdi-[0-9]{16}$` and per-run uniqueness rather than injecting a clock the verb does not own.
  evidence: internal/core/reading/ingest.go:556 — "written, err := capture.IngestReading(capture.IngestReadingRequest{ — the mint is capture's, under the ledger lock; measured: rdi-2608311447227460 landed with no payload contribution"
  evidence: internal/core/reading/ingest_identity_test.go:149 — "for _, field := range []string{\"id\", \"rdi\", \"item_id\"} { — literal names, not a table; measured at the CLI for `id` and `item_id`"
  evidence: internal/core/reading/ingest_identity_test.go:18 — "var mintedItemID = regexp.MustCompile(`^rdi-[0-9]{16}$`) — the shape check builder deviation 3 declares, in place of an injected clock"

Gap audit:
- honoured:
  - WIRED, and this is the phase's keystone claim. capture.IngestReading now has exactly one non-test caller in the tree, and the verb reaches it and executes from BOTH front doors: I ran the real binary end to end and watched a reading record land in the committed ledger, and the plugin markdown surface carries a full ingest page invoking the same verb. Five criteria across three shipped intents sat at MET_WITH_CONCERNS on this exact absence, and the record writer now has its intended caller.
    evidence: internal/core/reading/ingest.go:556 — "capture.IngestReading(capture.IngestReadingRequest{ — the ONLY non-test caller; a grep over internal/ and cmd/ finds no other"
    evidence: internal/surface/cli/reading.go:174 — "res, err := reading.Ingest(reading.IngestRequest{ — measured: `abcd reading ingest --output-json` wrote .abcd/work/issues/readings/rdg-.../rdi-....md"
    evidence: commands/reading.md:99 — "\"${CLAUDE_PLUGIN_ROOT}/abcd\" reading ingest \\ --output-json ./reading-output.json --json — the second front door, with the body-field table, the mint rule and the refusal granularity"
    evidence: .abcd/work/issues/resolved/iss-2608310912206941-the-reading-item-mint-has-no-caller-outside-its-own-package.md:13 — "resolution: \"The cold-reading ingest verb is that caller ... reachable from abcd reading ingest on the CLI and from the plugin markdown surface.\""
  - The host's headline question, answered empirically: NFKC is NOT producing false refusals of innocent prose. I A/B-ran the pre-NFKC binary (b54e8dc0) against the delivered one (8ec6f42f) over 34 constructed cases spanning every class the host named — ligatures in ordinary prose, superscripts and subscripts, vulgar fractions, Roman numerals, fullwidth CJK punctuation, non-breaking spaces in measurements and citations, soft hyphens at line breaks, combining accents in French/German/Polish, precomposed versus decomposed forms, mathematical alphanumerics, squared unit signs, the numero and trademark signs, halfwidth katakana, Kelvin/Angstrom/Ohm, and the registry's keywords inside longer words. NFKC changed the outcome in exactly 5 of the 34, and every one of the 5 is the registry's own phrase written in compatibility code points that render identically — fullwidth `Ｒａｔｅｄ　ｆｉｒｓｔ`, fullwidth `ｗｅ　ｒｅｃｏｍｍｅｎｄ`, the fi-ligature `the ﬁx is`, mathematical bold `𝐰𝐞 𝐫𝐞𝐜𝐨𝐦𝐦𝐞𝐧𝐝`, and a fullwidth colon after `disposition`. Not one innocent case was newly refused. The reason is structural rather than lucky: all four signatures are anchored on multi-word verb phrases, and no ordinary compatibility form collapses into one.
    evidence: internal/core/reading/ingest_regime.go:493 — "return norm.NFKC.String(folded) — measured A/B against b54e8dc0: 5 of 34 outcomes changed, all 5 the registry's own phrasing in compatibility code points, 0 innocent prose newly refused"
    evidence: internal/core/reading/ingest_regime.go:107 — "Pattern: regexp.MustCompile(`(?i)\\b(?:ranks?|ranked|rates?|rated|scores?|scored)\\s+...` — anchored on verb phrases, not bare words, which is why NFKC does not over-fire"
    evidence: internal/core/reading/ingest_regime_test.go:425 — "t.Run(\"innocent prose still lands\", func(t *testing.T) { — the builder's own 12-case battery; my 34 cases reach the same conclusion over a wider surface"
  - The record-size cap is now sound, and I could not defeat it. The decision sits on the assembled bytes in capture.IngestReading, and the upstream recordBytes filter is honestly documented as an estimate rather than the decision. I attacked all three lengthening paths the comment names. Escaping: 900 KB of double quotes and 700 KB of backslashes were refused at item level. Hidden-rune encoding: 300 KB of zero-width spaces estimated at 5,404,245 bytes and refused. Redaction, the path the comment says exceeds the doubling: 520 KB of a repeated IPv4 PASSED the estimate filter at 1,044,215 and was then caught by the exact count at 1,235,274. So an item can slip the filter, and when it does the capture-level decision catches it — which is precisely the division of labour the code claims.
    evidence: internal/core/capture/reading.go:227 — "if n := len(content); n > issueschema.RecordReadLimit { — measured: caught a 1,235,274-byte record whose upstream estimate was 1,044,215"
    evidence: internal/core/reading/ingest_regime.go:429 — "func recordBytes(fields map[string]string, bodyFields []string) int { — \"a cheap early FILTER, not the decision\"; measured bypassable exactly as documented"
    evidence: internal/core/reading/ingest_regime.go:228 — "Rule: \"record-too-large\" — measured: refuses at ITEM level, landing the rest, on the escaping and encoding paths"
  - All four declared builder deviations stand up under independent check. (1) Items are flat and `pattern` is still structurally distinguishable from a body field: it is in issueschema.ReadingRequired, the record ENVELOPE, and it appears in no position's body field set, so a payload cannot invert them — both sides are fixed key names. Ruling (18) is intact. (2) The stage holds a marker naming the run and the ids it reached, not rendered records, so the mint stays on the real ledger under the real lock. (3) ac-13 checks the mint's shape rather than an injected clock. (4) The residue text on BOTH itd-185 and spc-63 reads as OPEN, not as a closure, and I confirmed the class it names is genuinely uncaught: a Cyrillic homoglyph in `the fiх is` is accepted by the delivered binary.
    evidence: internal/core/issueschema/reading.go:124 — "ReadingRequired = []string{\"schema_version\", \"id\", \"run\", \"manifest\", \"position\", \"regime\", \"pattern\"} — pattern is envelope; no position's Fields list contains it"
    evidence: internal/core/reading/ingest.go:210 — "type stageMarker struct { Type string; RunID string; Records []string } — a marker, not rendered records"
    evidence: .abcd/development/specs/open/spc-63-one-ingest-verb-validates-every-cold-reading-output-includin.md:179 — "**What stays open, stated plainly because this section previously read as a closure:** a script-CONFUSABLE substitution ... it is open, which is why it is disclosed rather than filed under the calibration residue"
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:169 — "Closing that needs a confusables table, which is a new dependency and a maintainer's decision; until it is taken, this class is open. — measured: the Cyrillic homoglyph case is accepted, as disclosed"
  - Containment and trust-boundary handling are real, not asserted. Every path the verb reads, writes or DELETES inside the repository is resolved through an os.Root, the payload is read behind a single guarded open with a 4 MiB cap, the run id is the only payload value any path is built from and it is matched against the shared grammar before it becomes one, and a rerun of a run that already has an outcome — commit marker OR refusal record — is refused. I confirmed the rerun refusal at the CLI in both directions: after a successful run and after a refused one.
    evidence: internal/core/reading/ingest.go:315 — "root, err := os.OpenRoot(repoRoot) — every read, write and delete resolved through it; the sweep runs before the payload is even read"
    evidence: internal/core/reading/ingest.go:632 — "func refuseARerun(root *os.Root, runID string) error { for _, name := range []string{RunFileName, RefusalFileName} — measured at the CLI after both a committed and a refused run"
    evidence: internal/core/recordid/valid.go:36 — "func ValidReadingRunID(id string) bool { return readingRunIDRe.MatchString(id) } — one rule, refused before either side touches a path; measured: run_id \"../../etc\" refused"
  - The decision log records what the build settled, and each line is true of the code that shipped. All six 2026-08-31 entries check out against the tree at 1d84ce26: the flat payload item derived from issueschema.ReadingBodyFields, the stage as a marker with a filename-grammar-bounded rollback, the position/regime disagreement refused in LoadDefinition rather than cross-checked in the ingest verb, the size decision moved onto the assembled bytes in capture, and the fold with its confusable disclosure.
    evidence: .abcd/work/DECISIONS.md:2303 — "itd-185's payload item is FLAT (`pattern` plus the position's own body fields, read from `issueschema.ReadingBodyFields`) — true of internal/core/reading/ingest_regime.go:178-182"
    evidence: .abcd/work/DECISIONS.md:2317 — "The position/regime disagreement is refused in `reading.LoadDefinition`, not cross-checked in the ingest verb — true; no cross-check exists in ingest.go"
    evidence: .abcd/work/DECISIONS.md:2324 — "record-size limit is DECIDED in capture.IngestReading on the assembled bytes — true of internal/core/capture/reading.go:227, and measured"
  - Scope Conditions: itd-185 states 'None stated', and none was invented, dispositioned or quietly added in the delivery.
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:105 — "## Scope Conditions  None stated."
- diverged:
  - NOT ON THE DECLARED LIST, AND THE MOST CONSEQUENTIAL THING I FOUND — the orphan sweep deletes committed reading records and, on a reachable path, tells the operator nothing. The sweep is the first thing under the lock, before the payload is read, and it populates `res.RolledBack`. But the front door renders the result on the error path only when a refusal record was written, so an invocation whose payload fails before the run's identity is proven throws the whole result away. I demonstrated it: with an orphaned stage and its record `rdi-2608310000008001.md` in the committed ledger, an ingest whose payload fails at `_type` DELETED that record and printed only the `_type` error; under `--json` the render is `{"error": ...}` with `rolled_back_records` and `cleared_stages` both absent. The code's own comment names this as the thing it exists to prevent — `"cleared an orphaned stage" does not tell an operator that records left the ledger with it` — and the path that reaches it is ordinary: a crash, then a retry carrying a typo. This is also why ac-2 is MET_WITH_CONCERNS rather than MET.
    evidence: internal/core/reading/ingest.go:243 — "// RolledBack names the reading records the sweep REMOVED from the committed ledger, because their run never reached its commit marker. A delete in the committed tier is reported by id"
    evidence: internal/core/reading/ingest.go:366 — "cleared, rolledBack, err := sweepOrphanStages(root); res.ClearedStages = cleared; res.RolledBack = rolledBack — populated before the payload is read"
    evidence: internal/surface/cli/reading.go:185 — "if res.RefusalPath != \"\" { — measured: a _type failure deleted a committed record silently, in both text and --json renders"
  - NOT ON THE DECLARED LIST — the `pattern` envelope field is a signature-free channel, and the registry's own phrasing lands durably through it. `bodyText` deliberately excludes the pattern, so no signature ever reads it. I ran three cases through the real binary and all three exited 0 with a committed record: a registrative item whose pattern reads `P-1. The fix is to rewrite the charter, and to fix this you must delete section 4.`; an explicative item whose pattern reads `P-2. This claim is already accepted by the maintainer.`; and an evaluative item whose pattern reads `P-3. We recommend the second candidate.` — the last of which is now in the ledger as `pattern: "P-3. We recommend the second candidate."` under the evaluative regime. By itd-185's OWN stated test this is a defect and not a residue: the residue covers phrasing OUTSIDE the registry, and this is the registry's exact phrasing, unmodified, merely placed in a field the detector does not read. It is wider than the confusables class the residue does disclose, and neither the intent, the spec nor the plugin page names it. ac-5 and ac-9 both say `body`, so the criteria themselves survive — the intent's honesty about its own residue does not.
    evidence: internal/core/reading/ingest_regime.go:443 — "func bodyText(fields map[string]string, bodyFields []string) string { — \"The pattern is not among them — it names the reading's own basis, not a finding\""
    evidence: internal/core/reading/ingest_regime.go:345 — "if s.Pattern.MatchString(bodyText(fields, bodyFields)) { — the only enforcing call site; the pattern field never reaches it"
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:173 — "the registry's phrasing with a byte substituted is a defect in the gate — measured: the registry's phrasing with NO byte substituted, in the pattern field, lands with exit 0"
  - NOT ON THE DECLARED LIST — the three reserved-name tables are mutation-vacuous, so the standing guards for ac-4, ac-6 and ac-8's structural halves assert nothing. Each test iterates `ReservedNames[<regime>]`, the very table under test, with no floor on the count. I emptied each table in turn and the whole `internal/core/reading` package stayed green all three times: the loop body simply never runs. No test anywhere in the tree pins the literal names, and the criteria name them literally — ac-4 names `resolution`, `fix`, `remedy`; ac-6 names `rank`, `score`, `order`, `recommended`; ac-8 names `disposition`, `status`. A name dropped from a table would silently un-enforce a named criterion, and the item would then fall through to the unknown-field rule, which still refuses but cites the wrong rule and never states the licence ac-4 requires. This is exactly the class commit e55c03fb set out to close: that mutation sweep neutralised code BRANCHES and bound the three that survived, but never emptied a data table, so this shape was invisible to it. The neighbouring signature registry does have the floor — TestEverySignatureShipsEnforced refuses a registry of fewer than four — which shows the pattern was known and simply not applied here.
    evidence: internal/core/reading/ingest_regime_test.go:172 — "for _, field := range ReservedNames[RegimeRegistrative] { — measured: emptying the table leaves the package green"
    evidence: internal/core/reading/ingest_regime_test.go:127 — "for _, field := range ReservedNames[RegimeEvaluative] { — measured green when emptied"
    evidence: internal/core/reading/ingest_regime_test.go:224 — "for _, field := range ReservedNames[RegimeExplicative] { — measured green when emptied"
    evidence: internal/core/reading/ingest_regime_test.go:297 — "if len(Signatures) < 4 { t.Fatalf(\"the registry holds %d signature(s); spc-63 names four\" — the floor the reserved tables lack"
  - NOT ON THE DECLARED LIST — the signature registry refuses a great deal of prose a real cold reading would legitimately write, and the intent's Disclosed residue paragraph is one-directional about it. I ran 34 realistic reading outputs across all four positions and 14 were refused. The pattern in the failures is one thing: a reading QUOTING the document it read. An explicative reading exists to surface the claims a document makes, and `\bdisposition\s*[:=]` refuses any claim that quotes the token `disposition:` — a token this repository's own records carry everywhere — while `\balready\s+(?:accepted|rejected|settled|resolved)\b` refuses `Section 4 asserts the licensing question is already settled by adr-43` and `the minutes use the formula, it is resolved that the committee meets monthly`. A comparative reading is refused for `Clause 6 says the MIT licence should be adopted`, for `The cited paper closes with the sentence, we recommend further study`, and for `The suite scores below the threshold the charter names`. A detection reading is refused for `Section 3 says the fix is already merged while section 8 says it is pending` — the canonical shape of a detection. None of this is NFKC's doing; it is present in the pre-NFKC binary too. The intent DOES record `whether the signatures lint cleanly in practice` as its open question and reserves the degradation path for it, so the risk is disclosed at the Ruled and Out-of-Scope tiers. But the paragraph added under the criteria split and titled `Disclosed residue (ac-5 and ac-9)` speaks only of UNDER-catching and never of over-catching, which on this evidence is the larger and likelier failure. The builder's own innocent-prose battery is 12 cases at ONE position (`detection`/`why_a_tension`), so it exercises one of the four signatures.
    evidence: internal/core/reading/ingest_regime.go:133 — "|\\bdisposition\\s*[:=]` — measured: refuses `The schema lists disposition: accepted as an example row` and the French `la disposition : elle figure en annexe`"
    evidence: internal/core/reading/ingest_regime.go:110 — "|\\b(?:the\\s+)?(?:strongest|weakest|best|worst)\\s+(?:candidate|option|choice)\\b` — measured: refuses a characterisation that quotes the read document's own superlative"
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:158 — "The first is the one this intent has always disclosed: a fix proposal or a disposition **phrased** outside the registry's signatures is not caught. — the residue names only the under-catching direction"
    evidence: internal/core/reading/ingest_regime_test.go:438 — "{\"a ligature in ordinary prose\", \"the record de\\ufb01nes the constraint the tree ignores\"} — the 12-case battery, all at the detection position, exercising one signature of four"
  - NOT ON THE DECLARED LIST — `docs/reference/cli/commands.md` is wrong on three counts, and because the page is generated verbatim from the cobra `Long` string and pinned by a test, the defect is in the shipped help text an operator reads at the terminal. It says `Nothing durable is written until the whole payload validates`, which is false: a list-level refusal after the run's identity is proven writes `refusal.json` into the committed tier precisely BECAUSE the payload did not validate. It says an orphaned stage is `named and cleared by the next invocation` and never mentions that the sweep also deletes committed reading records — reproducing the exact wording ingest.go:243 warns against, and never naming `rolled_back_records`. And it says `each regime's reserved names are refused` and that the signature registry `catches` offending prose, both of which are false at the generative regime, which has no reserved-name row and where every signature hit raises a review flag and lands the item. The plugin page states all three correctly; the reference and the terminal help do not.
    evidence: docs/reference/cli/commands.md:831 — "Nothing durable is written until the whole payload validates — contradicted by internal/core/reading/ingest.go:669, which writes refusal.json under ReadingsRecordDir"
    evidence: docs/reference/cli/commands.md:833 — "an orphaned stage is named and cleared by the next invocation — omits the committed-tier delete at ingest.go:757"
    evidence: internal/core/reading/ingest_regime.go:48 — "// `generative` has no row and needs none — so \"each regime's reserved names\" is false for one of the four"
    evidence: internal/surface/cli/reading.go:143 — "Long: \"Validate the JSON a cold reading returned...\" — the single source the reference page and the terminal help are both generated from"
  - NOT ON THE DECLARED LIST — the brief carries the same three overclaims, and one of them is the very sentence an issue in this range corrected elsewhere. `.abcd/development/brief/04-surfaces/23-reading.md` states unconditionally that `a refused run still leaves a durable record`; iss-2608311234541392 corrected exactly that sentence in `commands/reading.md` by qualifying it with `once the run's identity is proven`, and the correction did not propagate to the brief. The brief also says `Each regime declares reserved names`, false at generative, and describes a two-layer gate at generative where — between the empty reserved table and the review-flag path — neither layer refuses anything.
    evidence: .abcd/development/brief/04-surfaces/23-reading.md:104 — "A list-level violation refuses the run, and a refused run still leaves a durable record — measured false on the LoadDefinition leg and on the write-phase leg"
    evidence: .abcd/work/issues/resolved/iss-2608311234541392-commands-reading-md-tells-a-host-that-every-list-level-refus.md:13 — "resolution: the page states that a refusal records only once the run's identity is proven — corrected in one surface of three"
    evidence: .abcd/development/brief/04-surfaces/23-reading.md:108 — "Each regime declares reserved names — false at generative (ingest_regime.go:51)"
  - NOT ON THE DECLARED LIST — iss-2608311145258479 is fixed by this range and left OPEN, while shipped source cites it by id as the failure it closes. Commit 7fa4bbbc implements the position/regime refusal and `internal/core/reading/definitions.go:130` names the issue as `exactly the failure shape the gate exists to close (iss-2608311145258479)`, but the record is still in `.abcd/work/issues/open/` at 1d84ce26 and the commit carries no `Resolves:` trailer. The commit message declares the reason honestly — the record lives in another checkout — so this is a disclosed constraint rather than an oversight, but the repository's own definition of done requires a fixing change to resolve its issue in the same diff, and a shipped comment citing an id that reads as open is the marker loss that rule exists to prevent. The sibling iss-2608311145286014 (the doubled error prefix) is likewise still open, and this range fixed half of it.
    evidence: internal/core/reading/definitions.go:130 — "which is exactly the failure shape the gate exists to close (iss-2608311145258479)"
    evidence: .abcd/work/issues/open/iss-2608311145258479-the-definition-locator-checks-that-a-regime-is-a-known-value.md:1 — "still in open/ at 1d84ce26; commit 7fa4bbbc says \"whose record lives in another checkout and is not resolved here\" and carries no Resolves: trailer"
  - NOT ON THE DECLARED LIST — a resolved issue record in this range asserts a property the very next commit repudiates. iss-2608311306536908's resolution says recordBytes counts every value double, `an exact bound rather than a guess`; commit 8ec6f42f then rewrote that function's doc to say it `is a cheap early FILTER, not the decision ... the ledger redactor exceeds that`, and moved the decision to capture. The record was superseded by iss-2608311351239562 and never amended, so the ledger now carries a resolved record telling a reader the estimate is exact when I measured it slipping by 191,059 bytes.
    evidence: .abcd/work/issues/resolved/iss-2608311306536908-recordbytes-under-estimates-the-reading-record-by-up-to-a-fa.md:11 — "resolution: \"recordBytes counts every value double, an exact bound rather than a guess\""
    evidence: internal/core/reading/ingest_regime.go:420 — "two attempts to make it one failed the same way ... the ledger redactor exceeds that — measured: an item estimated at 1,044,215 assembled to 1,235,274"
  - NOT ON THE DECLARED LIST — U+2800 BRAILLE PATTERN BLANK is accepted as a `pattern`, so the residue's claim that `every invisible rune is dropped` is an overclaim. U+2800 renders as nothing in every common font, but it is a graphic character: not Cf, not Other_Default_Ignorable_Code_Point, not a Variation_Selector, and unicode.IsSpace is false for it, so the fold leaves it standing and TrimSpace does not remove it. A record then asserts a provenance that renders as blank — the same failure mode the U+200B fix closed, one category further out. This does not breach ac-11, whose words are `empty or absent` and U+2800 is neither; it breaches the residue paragraph's own sentence about the fold. Every other blank-rendering form I could construct is caught, including the BOM and both Hangul fillers.
    evidence: internal/core/reading/ingest_regime.go:482 — "case unicode.Is(unicode.Cf, r), unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r), unicode.Is(unicode.Variation_Selector, r): return -1 — U+2800 is in none of the three"
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:166 — "every Unicode space folds to ASCII, every invisible rune is dropped, and NFKC folds the compatibility forms — measured: a pattern of one U+2800 is ACCEPTED, exit 0"
  - NOT ON THE DECLARED LIST — the durable refusal record renders the elision entry as `item 0`, which the code's own comment says must never happen. `boundedRefusals` deliberately gives the elision entry no ordinal `because it is not an item: rendering it as "item 0" would name a thing that does not exist`, and the terminal renderer honours that with an explicit branch. But `renderRefusals`, which composes the reason string written into refusal.json, formats every entry with `item %d` unconditionally. I ran a 100-item all-refused payload and the committed record's reason ends `item 0 (refusals-elided): and 80 more item(s) refused`. The fix was applied to one of the two surfaces, and it is the durable one that missed it.
    evidence: internal/core/reading/ingest_regime.go:267 — "// The elision entry carries NO ordinal, because it is not an item: rendering it as \"item 0\" would name a thing that does not exist."
    evidence: internal/core/reading/ingest_regime.go:546 — "parts = append(parts, fmt.Sprintf(\"item %d (%s): %s\", r.Ordinal, r.Rule, r.Detail)) — no branch; measured in refusal.json: \"item 0 (refusals-elided): and 80 more item(s) refused\""
    evidence: internal/surface/cli/reading.go:292 — "if r.Ordinal == 0 { fmt.Fprintf(w, \"                 (%s) %s\\n\", r.Rule, r.Detail); continue } — the branch the terminal has and the record writer lacks"
  - ac-1 and ac-10 are in literal tension, and the delivery resolves it on a boundary the criteria never state. ac-1 requires that a malformed output leave `no durable record for that run ... in the readings tree`; ac-10 requires that a list-level refusal leave a refusal record, and that record lives in the readings tree. The build draws the line at `the run's identity is proven` — a refusal before the parked manifest resolves writes nothing, one after it writes refusal.json. That is coherent and documented on the plugin page, but it is not in the criteria, and it is observable: an item carrying a non-string value is malformed by any ordinary reading of the word, and it produces `.abcd/development/readings/<run>/refusal.json`. I judged ac-1 MET because every whole-output malformation it plausibly names writes nothing anywhere, and reading ac-1 to forbid the refusal record would make ac-10 unsatisfiable.
    evidence: internal/core/reading/ingest.go:394 — "// The run's identity is proven from here ... A refusal below is therefore recordable against that run — the boundary, stated in the code and not in the criteria"
    evidence: commands/reading.md:141 — "**A refusal leaves a record once the run's identity is proven** ... A refusal reached BEFORE that point ... writes nothing anywhere"
    evidence: internal/core/reading/ingest_regime.go:192 — "Rule: \"item-shape\" — measured: `\"pattern\": 123` produced a refusal.json in the readings tree"
  - The manifest's `target_commit` reaches both durable records un-echoed, against the principle iss-2608311211235195 established in this same range. That record widened `echo` to cover the parked manifest's run id, position and assembler version, on the ground that a manifest is read back at the operator's word and is no more trusted for being a file on disk. `m.TargetCommit` is the one manifest-derived value that still goes into the run record and the refusal record raw. Low severity — a manifest this verb accepted has already had its content hash proven against the payload's citation — but the exemption is undeclared.
    evidence: internal/core/reading/ingest.go:591 — "Position: def.Position, Regime: def.Regime, TargetCommit: m.TargetCommit, — the run record"
    evidence: internal/core/reading/ingest.go:659 — "Position: def.Position, Regime: def.Regime, TargetCommit: m.TargetCommit, — the refusal record"
    evidence: internal/core/reading/ingest.go:530 — "// The manifest is a file on disk rather than the payload, but it is no more trusted for that ... its values reach the same terminal, so they are echoed under the same rule"
  - The bare `abcd reading` verb still prints a doubled prefix — `abcd: reading: reading: agents/cold-reading-detection.md states regime ...` — while `abcd reading ingest` prints it once. Commit 53b00afa fixed the stutter for the ingest verb only, and the code says why in as many words: the ingest message is load-bearing because six of thirteen criteria require a field, an ordinal or a signature id to be NAMED, and the assemble path was left alone as a change to a shipped verb rather than to this delivery. That is a defensible line; its residue is that the two verbs of one sub-tree now render refusals differently, and it rides on iss-2608311145286014, which is still open and whose title covers both paths.
    evidence: internal/surface/cli/reading.go:267 — "func trimCorePrefix(msg string) string { return strings.TrimPrefix(msg, \"reading: \") } — applied at line 190 for ingest and not at line 49 or 115"
    evidence: internal/surface/cli/reading.go:264 — "The `assemble` path above carries the same stutter and is deliberately left alone — measured: the bare verb prints \"abcd: reading: reading: ...\""
- missing:
  - Nothing has run a real reading through this verb. Every payload the delivery validates against — the fixtures, the builder's battery, and my own forty-plus probes — is synthetic. The intent's own recorded open question is whether the signatures lint cleanly in practice, and the delivery does not answer it, cannot answer it without a dispatched reading, and reserves the degradation path precisely because of it. My 34-case realistic-prose battery is the closest thing to an answer that exists, and it says the answer is probably no: 14 refusals, dominated by a reading quoting the document it read. The gate is therefore shipped enforced over a calibration that has never been taken, which is the standing tension the intent itself records against the widen-options `calibrated before it gates` clause.
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:93 — "Standing tension with the repo's widen-options promotion clause — \"calibrated before it gates\" — recorded as a standing tension; the ruled design governs the instrument meanwhile."
    evidence: .abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md:102 — "Whether the regime signatures lint cleanly — untested; the degradation path exists precisely because of it. — still untested at 1d84ce26"
  - A run refused inside the write phase leaves an empty directory standing in the COMMITTED tier. rollbackRun removes the ledger run directory only from inside the closure it runs when there were entries to remove, so a run that reached capture.IngestReading, had its directory created, and was then refused by the size invariant leaves `.abcd/work/issues/readings/<run>/` empty and permanent — the next invocation's sweep clears the stage and walks past it. Harmless in git, which does not track empty directories, but it is a durable artefact of a run the design says never happened, and it is the visible half of the same gap as the missing refusal record in ac-10.
    evidence: internal/core/reading/ingest.go:764 — "_ = root.Remove(ledgerRel) — inside unlink, which is only called when len(entries) > 0"
    evidence: internal/core/reading/ingest.go:767 — "if len(entries) > 0 { if err := underLedgerLock(root.Name(), unlink); err != nil { — measured: after the size-refused run and a subsequent sweep, the empty committed directory survives"