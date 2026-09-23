---
id: itd-178
slug: every-record-written-through-a-command-carries-its-origin-an
spec_id: spc-56
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Every record written through a command carries its origin and its production mode — stamped by the command, never typed by hand

## Press Release

> **A record now says where each of its items came from and how its text was
> produced.** `origin` names the arrival path — `researcher-authored`,
> `contributed-by-reading` (carrying the run and item identifiers that
> resolve to a reading record), or `extracted-from-record`. The production
> mode distinguishes `hand-written` from `dictated-and-formatted` from
> `scribe-transcribed`. Both are frontmatter keys written only by commands:
> no flag carries them as free text, hand editing is caught by the lint, and
> neither key touches authorship — disclosure at field granularity, on the
> same footing as the Assisted-by trailer at commit granularity.

## Alignment

- The three-term production-mode vocabulary is the ruled design's (the ruled authorship account); this intent ships the mechanism that stamps it, not
  the vocabulary decision.
- The attribution config seam to consult is itd-91's
  (`.abcd/config/identity.json`) — extend, never duplicate.
- Population is forward-only, per the ruled population properties: existing records are untouched, sparseness is information, and an
  absent stamp is never backfilled.

## What's In Scope

- The two keys in the record schemas, resolver support, and the
  command-side stamping on every write path that mints or mutates a record.
- The lint that reports a hand-edited record carrying either key.

## What's Out of Scope

- Passing either key to a reading — both are excluded by the input
  assembler's field projection, and the read-block eval asserts it.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a record written through a command, **when** it is committed,
  **then** both keys are present and neither was supplied as free text by
  the operator.
- **Given** a record with `origin: contributed-by-reading`, **when** it is
  read, **then** the item identifier and run identifier resolve to a
  reading record.
- **Given** a hand-edited record carrying either key, **when** the lint
  runs, **then** the lint reports it.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-789188ac284f -->
Fidelity review — receipt rcp-789188ac284f (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:4580e6eb3969c14bbc7f22e0643f0a01de84574330dc7a2580c47952962b3004
Input attestations: diff:932629f9..build/itd-178@sha256:3aa9a34db399a6c64025795e929d7b83ac14c9c75202b1bf6b4d575e0d6592e3;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: All four mint paths named by spc-56 stamp both keys together — CreateDraft validates the pair at the one draft-mint primitive and seedDraft writes both lines, renderSpec writes both, commitCapture writes both, and Promote derives extracted-from-record from what it did — and the operator supplies neither as free text: `origin` has no flag anywhere on the surface, while `--production-mode` is a closed choice refused at the door (resolveProductionMode) and again in the core (NewStamp/ModeOrDefault). The strongest evidence is end-to-end through the front door: the CLI tests file a real record through cobra and read the disclosure pair off the bytes that landed, including the pin-derived default and the refusal that writes nothing. Concern one — the criterion's universal is narrower than delivered: `capture disposition` mints a dsp record through a command with neither key, and the delta was closed by NARROWING the plugin prose to 'intent, spec and issue records' rather than by widening the code (iss-2608300931349990, resolved in range). Concern two — not one committed record in this corpus carries either key: a grep over every record store matches only spc-56's own illustrative code fence, so 'when it is committed' is proved in temp directories and never yet on a committed record.
  evidence: internal/core/intent/create.go:145 — "stamp, err := provenance.NewStamp(kind, opts.ProductionMode)"
  evidence: internal/core/intent/create.go:314 — "b.WriteString(provenance.KeyOrigin + \": \" + stamp.OriginValue() + \"\\n\")"
  evidence: internal/core/spec/spec.go:197 — "fmt.Fprintf(&b, \"%s: %s\\n\", provenance.KeyOrigin, stamp.OriginValue())"
  evidence: internal/core/capture/workflow.go:136 — "stamp, err := provenance.NewStamp(provenance.KindResearcherAuthored, req.ProductionMode)"
  evidence: internal/core/capture/promote.go:139 — "Origin:         provenance.KindExtractedFromRecord,"
  evidence: internal/surface/cli/cli.go:1708 — "func resolveProductionMode(cwd, flag string) (string, error) {"
  evidence: internal/core/issueschema/issueschema.go:89 — "\"origin\": true, \"production_mode\": true,"
  evidence: internal/surface/cli/capture_surface_test.go:660 — "func TestCaptureProductionModeFlag(t *testing.T) {"
  evidence: internal/surface/cli/capture_surface_test.go:690 — "func TestCaptureProductionModeDefaultsToThePin(t *testing.T) {"
  evidence: internal/surface/cli/intent_cli_test.go:570 — "func TestProductionModeFlagRefusesFreeText(t *testing.T) {"
  evidence: internal/core/intent/create_test.go:220 — "func TestSeedDraftStampsProvenance(t *testing.T) {"
  evidence: internal/core/spec/store_test.go:312 — "func TestSpecCreateStampsProvenance(t *testing.T) {"
  evidence: internal/core/capture/serialize_test.go:264 — "func TestCommitCaptureStampsProvenance(t *testing.T) {"
  evidence: internal/core/capture/reading.go:335 — "return fsutil.WriteFileAtomic(path, []byte(content), 0o644)"
  evidence: commands/capture.md:61 — "Records of other families — a disposition, for one — carry neither."
  evidence: .abcd/work/issues/resolved/iss-2608300931349990-provenance-plugin-pages-over-claim.md:16 — "capture disposition writes a dsp record with neither"
- ac-2 — MET_WITH_CONCERNS: The resolution machinery exists and executes: ParseOrigin is the one predicate that reads the value and refuses a pointer that is not spelled <rdg-N>/<rdi-N>, and the lint resolves the PAIR in one lookup — the item's bucket IS its run — reporting both a dangling item and an item that sits in a different run. The gate is live rather than fixture-only in its plumbing: record_provenance is registered in LintAt, armed as a blocker in this repo's own record-lint.json (pinned by TestRecordProvenanceIsArmedInThisRepo), and `go run ./cmd/record-lint` walks the corpus and emits zero record_provenance findings, so the rule runs on real bytes. CONCERN, and it is the consequential one: the criterion's Given clause has NO PRODUCER in this cycle — NewStamp refuses contributed-by-reading outright, so nothing in this repository can mint the record the criterion presupposes, and the reading-ingest door (spc-63) has not landed. The check is therefore proved against a fixture reading store only, and 'when it is read' is realised solely inside the lint: no record-reader path resolves the pointer. Two readings are available and I state mine — I take the DECLARED-NARROWING reading (spc-56 discloses the sequencing up front as a stated fact, and scopes minting the reading store and the reading-ingest verb out, which is the signed-off narrower scope MET_WITH_CONCERNS exists for), NOT the strict 'wired or it isn't done' reading, which would make the capability unreachable from the production entry point and draw NOT_MET. I flag the choice rather than bury it: the resolution half IS on the live path and would fire today on a hand-typed pointer; only the mint half is absent, and its absence is scoped, dated and disclosed. A second, smaller caveat: resolution is item-to-directory-bucket, so a pointer resolves against the item's run DIRECTORY and no rdg run record is required to exist.
  evidence: internal/core/provenance/provenance.go:86 — "func ParseOrigin(v string) (Origin, error) {"
  evidence: internal/core/provenance/provenance.go:98 — "if !hasSlash || !readingRunRe.MatchString(run) || !readingItemRe.MatchString(item) {"
  evidence: internal/core/lint/provenance.go:140 — "if run, ok := runOf[o.Item]; !ok {"
  evidence: internal/core/lint/provenance.go:78 — "runOf := map[string]string{}"
  evidence: internal/core/provenance/provenance.go:179 — "case KindContributedByReading:"
  evidence: internal/core/provenance/provenance.go:181 — "origin %s is minted only by the reading-ingest verb, which carries the run and item identifiers; no write path in this repository can supply them"
  evidence: internal/core/provenance/provenance_test.go:89 — "func TestNewStampRefusesTheUnmintableOrigin(t *testing.T) {"
  evidence: internal/core/provenance/provenance_test.go:47 — "func TestParseOriginReadingPointer(t *testing.T) {"
  evidence: internal/core/lint/schema_test.go:1087 — "func TestRecordProvenanceReportsUnresolvableReading(t *testing.T) {"
  evidence: internal/core/lint/schema_test.go:1123 — "// The fixture reading item: run rdg-3 holds item rdi-17."
  evidence: .abcd/development/specs/closed/spc-56-every-record-written-through-a-command-carries-its-origin-an.md:130 — "Reading-record resolution ships armed, exercised by fixture."
  evidence: .abcd/development/intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md:19 — "`contributed-by-reading` (carrying the run and item identifiers that"
- ac-3 — MET_WITH_CONCERNS: The lint exists, is wired, and discriminates: record_provenance reports four states no write path produces — an out-of-vocabulary value, a lone key, extracted-from-record with no promoted_from back-edge, and an unresolvable reading pointer — each with its own table test, and it is registered in LintAt and armed as a blocker in .abcd/record-lint.json rather than merely available. The single-key case matters most for this criterion: any hand edit that adds ONE of the two keys is always reported, which is the commonest shape of the hand edit the criterion names. CONCERN, three-part. (i) The criterion's literal universal is not achieved and the delivery says so in the user-visible message: a hand edit typing a legal value in a legal combination is byte-identical to a command's write and is silent; spc-56 restates the honest bound as 'in every shape a command could not have written'. (ii) The rule was armed as a BLOCKER with no staging period and zero corpus findings — I ran the gate and record_provenance reports nothing over the whole corpus — so it has never discriminated on live data; the armed test asserts configuration, not outcome. The no-false-green requirement is met in prose (the rule's own comment says why silence is correct here) but not by any assertion over the real corpus. (iii) The still-open nit records that a restamp can silently repair a lone-origin record — itself a blocker state — into a clean pair, so a command can launder away exactly the finding this criterion depends on.
  evidence: internal/core/lint/provenance.go:57 — "func checkRecordProvenance(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {"
  evidence: internal/core/lint/provenance.go:110 — "case !hasMode:"
  evidence: internal/core/lint/provenance.go:43 — "const handEditResidual = \" (this rule reports what no write path could have produced; \""
  evidence: internal/core/lint/lint.go:545 — "if rpCfg, ok := cfg.Rules[ruleRecordProvenance]; ok && rpCfg.Enabled {"
  evidence: .abcd/record-lint.json:267 — "\"record_provenance\": {"
  evidence: internal/core/lint/schema_test.go:1017 — "func TestRecordProvenanceLoneKey(t *testing.T) {"
  evidence: internal/core/lint/schema_test.go:1040 — "func TestRecordProvenanceOutOfVocabulary(t *testing.T) {"
  evidence: internal/core/lint/schema_test.go:1064 — "func TestRecordProvenanceExtractedWithoutPromotedFrom(t *testing.T) {"
  evidence: internal/core/lint/schema_test.go:1124 — "func TestRecordProvenanceIsArmedInThisRepo(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-56-every-record-written-through-a-command-carries-its-origin-an.md:141 — "The honest claim this spec makes is \"the lint reports a hand-edited record\""
  evidence: .abcd/work/issues/open/iss-2608300941548519-itd-178-second-round-observations.md:11 — "a record carrying a lone origin, itself a blocker, is silently repaired into a clean pair by a restamp"

Gap audit:
- honoured:
  - The vocabulary is one leaf package holding both closed sets and the one parser, standard library only, read by the writers and the gate alike — no second copy to drift
    evidence: internal/core/provenance/provenance.go:14 — "package provenance"
    evidence: internal/core/provenance/provenance.go:143 — "func ModeOrDefault(v string) (Mode, error) {"
    evidence: internal/README.md:28 — "**`core/provenance/`** — the record's disclosure vocabulary: where an item came"
  - Both keys are written together on every mint path spc-56 named, which is what makes a lone key a state no command produced
    evidence: internal/core/intent/create.go:315 — "b.WriteString(provenance.KeyProductionMode + \": \" + stamp.ModeValue() + \"\\n\")"
    evidence: internal/core/spec/store.go:259 — "renderSpec(id, slug, intentID, stamp)"
    evidence: internal/core/capture/workflow.go:136 — "provenance.NewStamp(provenance.KindResearcherAuthored, req.ProductionMode)"
  - An enum flag is not free text: `origin` carries no flag at all and `--production-mode` is refused outright outside the closed set, before anything is written
    evidence: internal/surface/cli/cli.go:1725 — "var productionModeFlagHelp = \"how this record's text was produced: \" + provenance.ModeList() +"
    evidence: internal/surface/cli/cli.go:1530 — "intentCmd.Flags().StringVar(&intentProductionMode, \"production-mode\", \"\", productionModeFlagHelp)"
    evidence: internal/core/provenance/provenance.go:130 — "func ParseMode(v string) (Mode, error) {"
  - The itd-91 attribution seam is extended, not duplicated: the repo default rides the existing identity pin as an optional member, validated at the boundary, with one resolution of 'absent means hand-written'
    evidence: internal/core/identity/identity.go:43 — "ProductionMode string `json:\"production_mode,omitempty\"`"
    evidence: internal/core/identity/identity.go:138 — "func DeclaredProductionMode(root string) (provenance.Mode, error) {"
    evidence: internal/core/identity/identity_test.go:315 — "func TestLoadPinAbsentMemberDefaultsHandWritten(t *testing.T) {"
  - The issue schema knows both keys, so a stamped record is not dropped as malformed by capture's reader — the failure mode spc-56 named explicitly
    evidence: internal/core/issueschema/issueschema.go:89 — "\"origin\": true, \"production_mode\": true,"
    evidence: internal/core/capture/serialize_test.go:236 — "func TestCaptureReaderAcceptsProvenanceKeys(t *testing.T) {"
    evidence: internal/core/provenance/provenance_test.go:116 — "func TestKeysAreKnownToTheIssueSchema(t *testing.T) {"
  - Forward-only population is honoured in the gate: a record carrying neither key is not a finding, so no wall of blockers and nothing to backfill
    evidence: internal/core/lint/provenance.go:96 — "if !hasOrigin && !hasMode {"
    evidence: internal/core/lint/schema_test.go:1002 — "func TestRecordProvenanceSilentOnUnstampedRecord(t *testing.T) {"
  - Wired on both front doors: the flag reaches every record-minting verb on the CLI, and both plugin pages carry a disclosure section stating the derivation, the closed choice, the restamp rule and the residual
    evidence: internal/surface/cli/cli.go:2569 — "captureCmd.Flags().StringVar(&captureProductionMode, \"production-mode\", \"\", productionModeFlagHelp)"
    evidence: internal/surface/cli/cli.go:2701 — "promoteCmd.Flags().StringVar(&promoteProductionMode, \"production-mode\", \"\", productionModeFlagHelp)"
    evidence: commands/capture.md:58 — "## Disclosure: where a record came from and how its text was produced"
    evidence: commands/intent.md:78 — "## Disclosure: where a record came from and how its text was produced"
    evidence: .abcd/development/brief/04-surfaces/05-intent.md:230 — "origin: researcher-authored   # arrival path (itd-178), DERIVED from which command ran and carried by no flag:"
  - The reviewer finding that the restamp would trip the branch's own blocker is genuinely closed: a restamp of a record carrying no origin is refused before any write, and an unstamped record still resolves when no mode is declared
    evidence: internal/core/capture/workflow.go:382 — "func restampField(fm map[string]any, issID, mode string) ([]kv, error) {"
    evidence: internal/core/capture/workflow.go:388 — "so it predates disclosure and there is nothing to restamp"
    evidence: internal/core/capture/workflow_test.go:927 — "func TestTransitionRefusesRestampOnUnstampedRecord(t *testing.T) {"
    evidence: .abcd/work/issues/resolved/iss-2608300925423309-restamp-on-an-unstamped-record-trips-the-provenance-blocker.md:12 — "A restamp of a record carrying no origin is refused before any write"
- diverged:
  - 'Every record written through a command' is delivered as 'intent, spec and issue records': `capture disposition` mints a dsp record through a command with neither key, and the delta was closed by narrowing the user-facing prose rather than by widening the code
    evidence: internal/core/capture/reading.go:260 — "func Disposition(req DispositionRequest) (DispositionResult, error) {"
    evidence: internal/core/capture/reading.go:335 — "return fsutil.WriteFileAtomic(path, []byte(content), 0o644)"
    evidence: commands/capture.md:60 — "Intent, spec and issue records carry two frontmatter keys, written by the"
    evidence: .abcd/work/issues/resolved/iss-2608300931349990-provenance-plugin-pages-over-claim.md:12 — "The two plugin pages narrow their claim to intent, spec and issue records"
  - The lint reads the record_schema scan (scanRecordStores) rather than LoadRecordGraph, because the graph carries neither frontmatter keys nor line numbers — a justified substitution, but spc-56's own text and its acceptance-mapping table still name LoadRecordGraph, so the closed spec now describes a reader the code does not use
    evidence: internal/core/lint/provenance.go:69 — "records, _, err := scanRecordStores(repoRoot, scanCfg)"
    evidence: internal/core/lint/provenance.go:50 — "because it needs"
    evidence: .abcd/development/specs/closed/spc-56-every-record-written-through-a-command-carries-its-origin-an.md:110 — "walks the record stores through `lint.LoadRecordGraph` (the one canonical record"
    evidence: .abcd/development/specs/closed/spc-56-every-record-written-through-a-command-carries-its-origin-an.md:147 — "The four `record_provenance` states above, over `LoadRecordGraph`"
  - Promote stamps through an added DraftOptions member rather than a common stamping seam, so `origin` is an optional field each future mint path must remember to pass; a caller that forgets is silently defaulted to researcher-authored rather than refused
    evidence: internal/core/intent/create.go:131 — "// Origin is the draft's arrival path (itd-178). It is DERIVED from which"
    evidence: internal/core/intent/create.go:151 — "kind := opts.Origin"
    evidence: internal/core/capture/promote.go:139 — "Origin:         provenance.KindExtractedFromRecord,"
  - record_provenance ships armed as a blocker with no staging period and zero findings over the live corpus, so a gate that has never discriminated on real data now gates every preflight; its silence is argued in a comment, not asserted by any test over the corpus
    evidence: .abcd/record-lint.json:268 — "\"enabled\": true,"
    evidence: .abcd/record-lint.json:269 — "\"severity\": \"blocker\""
    evidence: internal/core/lint/provenance.go:27 — "never backfilled — which is also why this rule can ship armed as a blocker over"
    evidence: internal/core/lint/schema_test.go:1124 — "func TestRecordProvenanceIsArmedInThisRepo(t *testing.T) {"
  - The restamp gate tests origin PRESENCE, not validity: a record with an out-of-vocabulary origin still accepts a restamp, and a lone-origin record — itself a record_provenance blocker — is silently repaired into a clean pair by a legal command write; left open by decision as a nit
    evidence: internal/core/capture/workflow.go:392 — "if asString(fm[provenance.KeyOrigin]) == \"\" {"
    evidence: .abcd/work/issues/open/iss-2608300941548519-itd-178-second-round-observations.md:11 — "the restamp gate tests origin presence, not validity"
- missing:
  - The press release's third arrival path has no producer: `contributed-by-reading` carrying the run and item identifiers is refused by the constructor, so nothing in this cycle can mint it and the resolution check ships with no production input at all
    evidence: .abcd/development/intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md:19 — "`contributed-by-reading` (carrying the run and item identifiers that"
    evidence: internal/core/provenance/provenance.go:181 — "no write path in this repository can supply them"
    evidence: .abcd/development/specs/closed/spc-56-every-record-written-through-a-command-carries-its-origin-an.md:135 — "no command in this repository can mint `contributed-by-reading`"
  - 'A record now says where each of its items came from and how its text was produced' is true of no record in this repository: a scan of every configured record store finds not one committed record carrying either key, so the delivery is demonstrated in temp directories and by fixture, never on the corpus the gate walks
    evidence: .abcd/development/intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md:17 — "**A record now says where each of its items came from and how its text was"
    evidence: .abcd/record-lint.json:257 — "\"record_stores\": {"
    evidence: internal/core/lint/provenance.go:25 — "Population is forward-only (the ruled population property): a record carrying"

## Grounds

- pursued: Pursued now because a record says nothing about where its items came from or how its text was produced, and the reading workstream is about to start contributing items that a reader must be able to tell apart from researcher-authored ones. The keys have to exist before the first contributed item lands, or the distinction is unrecoverable.
