---
id: itd-183
slug: the-cold-reading-sees-exactly-what-the-assembler-passes-posi
spec_id: spc-61
kind: bundle-member
suggested_kind: bundle-member
reclassification_history: []
builds_on: [itd-86]
severity: major
impact: additive
---

# The cold reading sees exactly what the assembler passes — positive inclusion, field projection, and a per-run manifest

Typed links: consumes
[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md)
(every framing item on the include list cites the rule admitting it);
settles itd-86's central open question (blindness is structural, not
instructed).

## Press Release

> **Blindness becomes a property of the input, not a promise of the
> reader.** The location tiering is organisational, not an access control —
> nothing today prevents a reading reaching ledger content. The assembler
> closes that: it names what it includes (a record type added later,
> including one the instrument itself adds, is excluded by default rather
> than included by oversight), and it projects fields out of files rather
> than copying paths — a shipped intent holds both the claim record and its
> Audit Notes, and only the former may travel. Every run emits a manifest —
> what was passed, by path and field, hashed, with the run
> identifier — so a reader can judge contamination rather than accept a
> disclosure on trust. Invocation carries no free text: the operator
> supplies a position and a target state, and the reading's object and
> question come from its definition — there is no channel through which
> ledger content can travel in the framing of a request, because there is
> no prose input.

## What's In Scope

- **Include list:** the brief's `01-product/` and `02-constraints/`
  chapters — the framing section's construal statement included, the
  evidence chapter deliberately not (open questions and settled dead ends
  are deliberation, and a future brief-homed warm record must not walk in
  as "brief text"); `brief/glossary/`; `intents/shipped/` projected to
  press release, acceptance criteria, scope conditions, mechanism claim and
  `spec_id`; `intents/disciplines/`; `specs/`; the shipped tree (defined
  below). The chapter-level bound (03-evidence excluded as deliberation) was agreed at the maintainer review of 2026-08-28.
- **The include list is per-position** (maintainer readings design, 2026-08-28):
  assembly follows the invoked definition's object over the shared
  exclusion floor. The stated asymmetry: the widening reading excludes
  `intents/drafts/` and `intents/planned/` (they are the candidate set it
  is asked to widen); the entailment reading includes them (articulation
  precedes selection). The comparative reading's object is the widening
  reading's pre-admission output plus the selection-criteria discipline —
  within one cycle, before admission; a prior run's stored output stays
  read-blocked.
- Two assembler rules (ruled 2026-08-28): **no include may name a
  directory containing a record family** — "the shipped tree" is scoped
  to source, tests, documentation and configuration, `.abcd/` excluded
  wholesale, and record paths a reading legitimately needs named
  individually, so a family added later (the readings family included) is
  excluded by construction; and **a reading's object excludes the
  material whose state that reading exists to change** — the drafts
  asymmetry and the Audit-Notes exclusion are its instances, and the
  include list becomes derivable rather than remembered.
- **Exclude, and assert the exclusion:** `origin`; production mode; Audit
  Notes; scope-condition dispositions; `brief/03-evidence/`; `decisions/`;
  `roadmap/rfcs/`; `intents/superseded/`; `work/issues/` in every state,
  reading records and dispositions included; `plans/`; `research/notes/`;
  session transcripts; manifests; admission and selection grounds; the
  lapse log. Each exclusion item carries its detectable signal: frontmatter
  keys (`origin`, production mode) by key; `Audit Notes` by heading;
  directories by never appearing in the positive include walk; dispositions
  and grounds as record types in excluded paths, their fields dropped by
  projection. Prose-borne warmth inside an included chapter has no
  structural signal — that is what the chapter-level include bound and the
  glossary discipline carry, and it is disclosed as residue, not claimed as
  caught.
- **The manifest**, per run, on the render-without-writing idiom of
  `disembark plan` for dry runs; at ingest it is committed to the durable
  tier at `.abcd/development/readings/<run-id>/`, alongside the run
  record — a new record family (ruled 2026-08-28, superseding the earlier
  working-tier proposal: lifecycle selects the tier — a manifest is
  written once and never amended; a commit reference plus per-item hashes
  lets a third party re-run the assembler and diff the output, the
  strongest form of the checkability claim; and it carries paths, field
  names and hashes, never content, so nothing needs redaction). Manifests
  are warm on the next run: inside the read-block, excluded by rule.
- **The evals that guard this assembler** — the read-block eval
  (sentinel-planted warm content must never appear in output) and the
  amnesia eval (the same state assembled twice is byte-identical) — are
  separate instruments with their own claims and verdicts, split to their
  own drafts (the read-block eval and the amnesia eval) per the maintainer review of 2026-08-28.

## Resolutions implemented (2026-08-27; ratified 2026-08-28)

- **The reader-side boundary is owned here.** Passing curated input to a
  reader that can open the repository is instructed blindness with extra
  steps. The invocation constructs a context-isolated reading: the
  assembled input is the reading's working set and no repository path
  enters its context. This is the enforcement half of
  invariant 15 (adopted 2026-08-28); the definitions assert the posture, this
  intent builds the wall.
- **"The shipped tree" is defined**: the delivered artefact — source,
  tests, `docs/`, the root README — excluding `.abcd/**`, `agents/`,
  `evals/`, and the instrument's own package. The reading never receives
  its own definition, the eval that guards it, or the assembler's include
  table.
- **Manifest warmth has one ground**: the manifest's *content* is cold (it
  enumerates passed cold items only, hashed, no timestamps), which is why
  committing it for reader audit is safe; the manifest as *evidence* is
  warm (it reveals run timing and target selection), which is why it sits
  inside the read-block and no future reading receives it. Both halves are
  stated, neither is traded away.
- **Comparative candidate ids — superseded by M8 (the maintainer rulings of 2026-08-28):** the
  ruled interface carries position and target state only. The ids-only
  candidate list proposed 2026-08-27 is retained solely as the recorded fallback, if testing against the four definitions shows the evaluative
  position needs candidates at invocation: restricted to the intent
  family, shape-validated, never prose.

## What's Out of Scope

- The reading definitions and the output contract — sibling intents.
- Preventing timing and target-selection leakage: the operator still
  chooses when to run a reading and what to point it at, which can carry
  information. The manifest and session records make this visible after the
  fact; they do not prevent it. Disclosed residue, not a gap to be closed
  here.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a repository state, **when** the assembler runs, **then** its
  output contains no field on the exclusion list.
- **Given** a new record type added under `.abcd/development/`, **when**
  the assembler runs, **then** that type is absent from the output without
  any change to the assembler.
- **Given** the invocation interface, **when** it is inspected, **then**
  it accepts a position and a target state, and nothing else — no
  free-text argument anywhere (ruled, maintainer 2026-08-28). Whether the evaluative position needs its candidates or criteria at
  invocation is tested against the four definitions before the interface
  freezes; the anticipated revision, if the need is shown, is a
  shape-validated record-id list, never prose.
- **Given** an assembler run, **when** the manifest is emitted, **then**
  every item passed appears with its path, its field where projection
  occurred, and a hash — and a reader can determine that a
  named excluded field was not passed.
- **Given** a reading invocation, **when** its context is constructed,
  **then** it contains the assembled input and no repository path — the
  blindness is enforced by construction, not instructed.

## Rulings (2026-08-27, as revised 2026-08-28)

- **Manifest home: the durable tier** (revised by the maintainer rulings
  of 2026-08-28, decision log). Written at ingest to
  `.abcd/development/readings/<run-id>/`, alongside the run record — a
  new record family, excluded by this assembler like every record
  family; the local render remains for
  dry runs. Grounds: the manifest's purpose is that a reader can judge
  contamination rather than accept a disclosure on trust, and a gitignored
  manifest delivers that only on the machine that ran it. It is safe to
  commit because it enumerates the passed (cold) items only.
- **No free text at any position — ruled as M8 (the maintainer rulings of 2026-08-28),
  stricter than proposed:** position and target state only. The ids-only
  comparative argument proposed here is superseded; it survives only as
  the recorded fallback if the evaluative position proves to need candidates
  at invocation.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-139d9b40f0e8 -->
Fidelity review — receipt rcp-139d9b40f0e8 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:10d41a4cb84629aa2560c55940c136223bd9fffea6fd7106b65bc7d6e26dfcad
Input attestations: diff:b6ccb701..build/itd-183 (3e03dfa2)@sha256:8df3bfd0b13fdb84675400b159397e584a8a44903934b179c01624e08d986555;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The floor is real, layered and fail-closed, and it holds on the delivered repository state. Inclusion is positive at field granularity (the shipped-intent row projects five named fields and nothing else); redactExcluded strips key- and heading-signalled material from every admitted markdown file; verifyRedaction then REFUSES the run when an excluded key or heading survives redaction, which is the difference between a disclosure and a gate; assertExclusions refuses any item under a path-shaped exclusion; and the `.abcd` deny is structural. I did not take that on trust — I ran `abcd reading assemble --position widening --target HEAD` against this checkout at db3291f8 and audited all 907 items: no committed record's Audit Notes, `origin` or `production_mode` reached the bundle, and every `.abcd` path in the manifest is one of the six admitted sources. CONCERN, three-part, and it is the consequential half of this verdict. (i) The criterion is a UNIVERSAL and there are three VERIFIED counterexamples, all open, all recorded as leaking rather than theorised: a fence delimiter inside a frontmatter block scalar toggles fenceMask and switches off the excluded-key scan for the rest of the block (iss-2608301350533102, 'Verified leaking on this round and on the parent'); a newline between an attribute's `=` and its opening quote declines the markup mask on both readings so a `>` inside the value truncates a heading opener (iss-2608301350534164, 'Verified leaking'); and the raw-heading soft bound `\n[ \t]*\n` does not match a CRLF blank line, so an unclosed `<h2>` in a CRLF file reads its title over the rest of the document and travels (iss-2608301421380392, 'Verified admitted'). Six further floor records are open beside them. (ii) A residue I found on the LIVE corpus, not on exotic input and not among the nine records: the heading and key signals are scoped to markdown (`if !strings.EqualFold(path.Ext(rel), ".md") { return doc, nil }`), so `internal/core/site/fixture_test.go` carries three literal `## Audit Notes` sections with their rollup lines into the bundle at itm-0736 while the manifest asserts `Audit Notes` refused. The content is synthetic fixture text rather than a real audit, and the bound is disclosed in the charter, but it is an on-corpus instance of an excluded heading in the output. (iii) I record my reading rather than bury it. I considered NOT_MET and rejected it: NOT_MET is for a promise contradicted at its core or absent, and this floor is present, wired, fail-closed, exercised by roughly seventy spelling tests that pass, and correct over every record this repository actually holds; the counterexamples are signal-RECOGNITION gaps on shapes no record in the corpus writes. What is delivered is narrower than the universal, which is what MET_WITH_CONCERNS names. A maintainer reading ac-1 as a SECURITY guarantee rather than a mechanism claim should read this as NOT_MET instead, and I flag the choice rather than make it silently. Separately: 'no reading runs this cycle' bears on the RISK and not on this verdict. The criterion's trigger is 'when the assembler runs', which happened; whether a reader consumes the bundle changes the blast radius to zero and changes nothing about what the output contains.
  evidence: internal/core/reading/include.go:147 — "var intentProjection = []string{"
  evidence: internal/core/reading/include.go:206 — "Fields:    intentProjection,"
  evidence: internal/core/reading/project.go:50 — "func redactExcluded(rel, doc string, exclusions []Exclusion) (string, error) {"
  evidence: internal/core/reading/project.go:57 — "if !strings.EqualFold(path.Ext(rel), \".md\") {"
  evidence: internal/core/reading/project.go:323 — "func verifyRedaction(rel, original, redacted string, keys, headings map[string]bool) error {"
  evidence: internal/core/reading/project.go:320 — "which is the one thing the manifest exists not to do. A floor a file can"
  evidence: internal/core/reading/assemble.go:480 — "func assertExclusions(cands []candidate, exclusions []Exclusion) error {"
  evidence: internal/core/reading/assemble_test.go:15 — "func TestExcludedFieldsNeverReachTheBundle(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:81 — "func TestShippedIntentProjectsFiveFieldsOnly(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:510 — "func TestExcludedKeySurvivingRedactionRefusesTheFile(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2608301350533102-fencemask-spans-the-frontmatter-so-a-fence-delimiter-inside.md:37 — "that exists to catch exactly that gap is switched off. Verified leaking on this"
  evidence: .abcd/work/issues/open/iss-2608301350534164-a-newline-between-an-attribute-name-s-equals-and-its-opening.md:30 — "Every renderer reads that as the excluded heading. Verified leaking on this"
  evidence: .abcd/work/issues/open/iss-2608301421380392-the-raw-heading-soft-bound-does-not-match-a-crlf-blank-line.md:30 — "Verified admitted; the byte-identical document with LF line endings refuses."
  evidence: internal/core/site/fixture_test.go:665 — "## Audit Notes"
  evidence: .abcd/development/readings/README.md:73 — "The heading signal is scoped to markdown."
- ac-2 — MET: The absence is structural rather than remembered, and this delivery is its own strongest witness. `.abcd` is a denied path COMPONENT, case-insensitively, on every segment, and the deny is measured from an include row's own Source downward — so the three root rows that walk the repository for `.go`, `.md` and configuration cannot reach `.abcd` at any depth, while a record path a reading legitimately needs escapes the deny only because it is named individually at its own leaf. A record family invented after the table was written is therefore absent by construction, with no edit to the assembler. Two tests hold it: rule 1 as an executable property of the TABLE, and a fixture that plants `.abcd/development/inventions/inv-1-a-new-record.md` with a sentinel and asserts its absence from both the bundle text and the manifest at all four positions. The decisive evidence is on the live corpus rather than the fixture: this very delivery ADDS a record family, `.abcd/development/readings/`, and my run over 907 items at db3291f8 shows not one item from it, nor from `agents/`, `evals/`, `internal/core/reading/` or `.abcd/work/` — the include table did not have to be told.
  evidence: internal/core/reading/deny.go:24 — "var denySegments = []string{\".git\", \".abcd\", \"agents\", \"evals\"}"
  evidence: internal/core/reading/deny.go:100 — "if pathContainsDeniedSegment(sub) || prefixDenied(rel) {"
  evidence: internal/core/reading/deny.go:12 — "The one difference from launch's is where the deny is measured FROM. Here it"
  evidence: internal/core/reading/include_test.go:95 — "func TestNoIncludeNamesARecordFamilyDirectory(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:59 — "func TestNewRecordFamilyIsAbsentWithoutTableChange(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:72 — "if strings.HasPrefix(path, \".abcd/development/inventions/\") {"
  evidence: internal/core/reading/include_test.go:115 — "func TestTheAssemblersOwnOutputIsNeverItsInput(t *testing.T) {"
  evidence: .abcd/development/readings/README.md:1 — "# Readings"
- ac-3 — MET_WITH_CONCERNS: The channel the criterion exists to close IS closed, and closed at both front doors. A positional argument of any kind is refused before the run with exit 2 and a message that says why; `--position` resolves against a four-token closed set and is refused by name otherwise; `--target` accepts `HEAD` or `^[0-9a-f]{7,40}$` and nothing else, refusing branch names and tags as mutable; neither operand is defaulted, because a defaulted position would pick a reading's object on the operator's behalf. The plugin page carries the same two operands and the same claim. CONCERN one, and it is the direct answer to the question put: `--out` IS a free-text string operand on the invocation, so the criterion's emphatic 'no free-text argument ANYWHERE' is not literally kept. I judge it outside the interface the criterion guards, and I judge it on the code rather than on intent: `--out` is a destination, and the delivery structurally prevents it from becoming a channel — neither artefact carries an output path at all, the core is handed the resolved path while the operator is shown only the string they typed, and a directory the include table could reach is refused outright so a run cannot be written where a later run would read it. `--dry-run` is a boolean and is not free text on any reading. But the criterion's own words are 'anywhere', and a reader checking the flag set will find a string flag; that gap between the ruled words and the shipped surface is a maintainer's to close. CONCERN two: the criterion's second sentence requires that whether the evaluative position needs candidates at invocation 'is tested against the four definitions BEFORE the interface freezes'. That test has not happened — the four definitions are spc-62/itd-184 and are not in this delivery — while the interface has shipped, versioned and pinned by a golden hash of the rendered table. spc-61 scopes the fallback out ('stays recorded and unbuilt unless the four definitions show the evaluative position needs it'), so the sequencing is disclosed rather than overlooked, but the criterion's precondition is owed rather than discharged.
  evidence: internal/surface/cli/reading.go:71 — "Args: func(_ *cobra.Command, args []string) error {"
  evidence: internal/surface/cli/reading.go:73 — "reading assemble: this verb takes no positional argument"
  evidence: internal/surface/cli/reading.go:83 — "if position == \"\" {"
  evidence: internal/core/reading/include.go:51 — "func ParsePosition(s string) (Position, error) {"
  evidence: internal/core/reading/assemble.go:99 — "targetRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)"
  evidence: internal/core/reading/assemble.go:375 — "reading: target %q is neither HEAD nor a hexadecimal commit sha of 7 to 40 digits"
  evidence: internal/surface/cli/reading.go:133 — "assembleCmd.Flags().StringVar(&outDir, \"out\", \"\","
  evidence: internal/core/reading/assemble.go:57 — "Neither ARTEFACT carries an output path at all."
  evidence: internal/surface/cli/reading.go:121 — "if outDir != \"\" {"
  evidence: internal/core/reading/assemble.go:310 — "func refuseSelfAdmittingOutDir(repoRoot, outDir, label string) error {"
  evidence: internal/surface/cli/reading_surface_test.go:144 — "func TestAssembleRefusesFreeTextOperands(t *testing.T) {"
  evidence: internal/surface/cli/reading_surface_test.go:179 — "func TestTargetRefusesBranchAndTag(t *testing.T) {"
  evidence: commands/reading.md:18 — "it carries no free text at any position — the operator supplies a"
  evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:277 — "The comparative fallback interface (a shape-validated record-id list at"
- ac-4 — MET_WITH_CONCERNS: The criterion has two conjuncts and they land differently. The FIRST is fully realised and I verified it on the live corpus rather than by fixture: one manifest item per bundle item, each carrying `path`, `sha256`, and `field` where and only where projection occurred. My run at db3291f8 produced 907 bundle items and 907 manifest items, every one with a non-empty path and hash, the item keys matching one-for-one, and the distinct field values being exactly the projection's own names — `Press Release`, `Acceptance Criteria`, `Scope Conditions`, `spec_id`, and empty for a whole-file item. The SECOND conjunct — 'a reader can determine that a named excluded field was not passed' — survives the floor's fallibility, and I say so having tested it rather than reasoned about it. Every exclusion is asserted into the manifest with the signal by which a reader detects it (23 entries at the widening position), and an assertion is a FALSIFIABLE claim: holding the bundle, I checked the assertion `{rule: field projection, signal: heading, detail: Audit Notes}` against the 907 item texts and refuted it at itm-0736. The determination mechanism worked; it is what caught the divergence. That is exactly the press release's claim — 'a reader can judge contamination rather than accept a disclosure on trust' — and it is why a floor that CAN miss does not sink this criterion: the criterion asks for determinability, and determinability is what remains true when the floor is wrong. CONCERN, two-part. (i) The determination requires the BUNDLE, and the delivery deliberately separates the two artefacts so the bundle goes to the reader while the manifest stays with the auditor; an auditor holding the manifest ALONE cannot determine anything about heading or key content, because the manifest carries paths and hashes and never content. On the stricter reading — the manifest by itself is sufficient evidence of exclusion — this criterion is NOT_MET, and I name that reading rather than hide behind the one I took. (ii) The manifest's exclusion entries carry no scope qualifier, so `Audit Notes | heading | field projection` reads as unconditional; the narrowing that makes it true (the signal is scoped to markdown) lives in the charter, which a third party holding only a run's two artefacts does not have. That is the shape of the on-corpus divergence I found, and of the three recorded exotic ones: the manifest can assert something untrue, and it does not carry the qualifier that would make it true.
  evidence: internal/core/reading/manifest.go:46 — "type ManifestItem struct {"
  evidence: internal/core/reading/manifest.go:49 — "Field   string `json:\"field,omitempty\"`"
  evidence: internal/core/reading/assemble.go:177 — "manifest.Items = append(manifest.Items, ManifestItem{"
  evidence: internal/core/reading/assemble.go:178 — "ItemKey: key, Path: c.path, Field: c.field, SHA256: sha256Hex([]byte(c.text)),"
  evidence: internal/core/reading/assemble.go:172 — "Exclusions:       exclusions,"
  evidence: internal/core/reading/include.go:267 — "Exclusions is the exclusion floor: every field, heading and directory the"
  evidence: internal/core/reading/manifest_test.go:12 — "func TestManifestCoversEveryBundleItem(t *testing.T) {"
  evidence: internal/core/reading/manifest_test.go:40 — "func TestManifestNamesTheFieldWhereProjectionOccurred(t *testing.T) {"
  evidence: internal/core/reading/manifest_test.go:67 — "func TestManifestAssertsNamedExclusions(t *testing.T) {"
  evidence: internal/core/reading/manifest.go:54 — "and asserts what it refused. It carries no item content, so committing it"
  evidence: internal/core/reading/assemble.go:74 — "separate files, so the assembled input can go to a reader while the manifest"
  evidence: internal/core/site/fixture_test.go:665 — "## Audit Notes"
  evidence: .abcd/development/readings/README.md:73 — "The heading signal is scoped to markdown."
- ac-5 — NOT_MET: The criterion's subject is a READING INVOCATION and its constructed context, and this delivery constructs none: spc-61 scopes running a reading out for the whole cycle in terms that leave no room ('no reading is commissioned, no bundle is dispatched'), and the CLI's own header states that dispatching is host work. Applying the itd-192 discipline the host directs — a criterion whose producer does not exist is judged by whether THIS PHASE wires it — the producer is not wired here and will not be wired by a later phase either, because the enforcing half is permanently outside the binary. That is the divergence, and the delivery states it plainly rather than hiding it: the spec discloses the host obligation 'never claimed as an enforcement this binary performs', and the plugin page discharges it by INSTRUCTION — 'The other half is yours. When you dispatch an assembled input to a reader, grant that reader no repository access.' The criterion's own operative clause is 'enforced by construction, NOT INSTRUCTED', and half of it is instructed by the delivery's own text. A second, independent ground: 'contains ... no repository path' is false of the delivered artefact on any reading but the structural one. The bundle's SHAPE is genuinely pathless — item key is an ordinal `itm-NNNN`, kind is a closed class vocabulary naming no location, and the path mapping lives only in the manifest — and the test asserts exactly that, over a skeleton with every text blanked. But item TEXT is repository prose: in my run at db3291f8, 558 of 907 items quote a repository path, and 4 quote their own. The test's own comment concedes the point ('Item text is necessarily prose and source that may quote paths of its own, so the claim is made where it is a claim'), which makes this a disclosed narrowing rather than a hidden one. WHAT IS ACTUALLY DELIVERED, and it is not nothing: the half of the isolation that a binary CAN enforce is enforced by construction, tested, and reachable from both front doors. I record NOT_MET on the criterion as written rather than downgrade the criterion to the half it can have, because the two halves it conjoins are not both achievable by any binary and the record should say so once, here, rather than let a future audit re-litigate it. itd-184 (the four definitions, which each restate the host obligation) is the intent that carries the remaining half, and it carries it as a statement, not as an enforcement.
  evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:268 — "**Running a reading.** The instrument ships unrun for the whole cycle: no"
  evidence: internal/surface/cli/reading.go:13 — "Nothing here runs a reading. The verb produces the input a reading would be"
  evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:187 — "the plugin surface section, disclosed as a host obligation and never claimed"
  evidence: commands/reading.md:90 — "The other half is yours. When you dispatch an assembled input to a reader,"
  evidence: commands/reading.md:91 — "grant that reader **no repository access** — no file tools, no path, no working"
  evidence: internal/core/reading/manifest.go:22 — "It carries NO repository path, by construction — the key"
  evidence: internal/core/reading/include.go:65 — "Kind is a bundle item's material class. It names WHAT a passed item is and"
  evidence: internal/core/reading/assemble_test.go:103 — "func TestBundleCarriesNoRepositoryPath(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:108 — "skeleton.Items = nil"
  evidence: internal/core/reading/assemble_test.go:101 — "necessarily prose and source that may quote paths of its own, so the claim is"
  evidence: .abcd/development/readings/README.md:152 — "remaining half of the isolation, that a dispatching host grants the reader no"

Gap audit:
- honoured:
  - The include table is one Go source rendered into the readings charter under an anti-drift test, so the charter cannot claim a bound the walk does not keep — and the assembler version is pinned to a golden hash of that render, so the table cannot move without the version moving with it
    evidence: internal/core/reading/include.go:165 — "var Table = []Row{"
    evidence: internal/core/reading/include_test.go:46 — "func TestReadingsCharterCarriesTheRenderedIncludeTable(t *testing.T) {"
    evidence: internal/core/reading/include_test.go:230 — "const includeTableDigest = \"7f19b6953a098544b519447db82e0da930b41f6546cf6436143e2d9871181d24\""
    evidence: .abcd/development/readings/README.md:101 — "< !-- BEGIN GENERATED: reading-include-table -- >"
  - Wired on both front doors and demonstrably executing: the verb is registered in the command tree, the plugin page carries the same two operands, and the binary runs end to end on this repository — 907 items, a manifest hash, exit 0
    evidence: internal/surface/cli/cli.go:340 — "root.AddCommand(newReadingCommand(&asJSON))"
    evidence: internal/surface/cli/reading_surface_test.go:89 — "func TestReadingVerbReachesBothPlanes(t *testing.T) {"
    evidence: commands/reading.md:39 — "\"${CLAUDE_PLUGIN_ROOT}/abcd\" reading assemble \\"
    evidence: docs/reference/cli/commands.md:776 — "#### `abcd reading assemble`"
    evidence: .abcd/development/release/surface.json:1031 — "\"path\": \"abcd reading assemble\""
  - The core is transport-agnostic as adr-23 requires: the reading package takes a structured request and returns a structured result, and every byte of rendering lives behind the CLI front door
    evidence: internal/core/reading/include.go:8 — "The package is cobra-free and stdout-free like every sibling under"
    evidence: internal/core/reading/assemble.go:124 — "func Assemble(req AssembleRequest) (AssembleResult, error) {"
    evidence: internal/surface/cli/reading.go:183 — "func renderAssembleResult(w io.Writer, res reading.AssembleResult) {"
  - Re-runnability is enforced rather than promised: a target that is not the commit in front of the assembler refuses, any uncommitted included path refuses, a deleted or renamed included path refuses through the include table's own admissibility, and the walk is intersected with git's tracked set so a gitignored or submodule file can never enter a bundle an auditor could not reproduce in a clean clone
    evidence: internal/core/reading/assemble.go:386 — "if target != \"HEAD\" && !strings.HasPrefix(head, target) {"
    evidence: internal/core/reading/assemble.go:403 — "func refuseDirtyIncludedPaths(repoRoot string, position Position, cands []candidate) error {"
    evidence: internal/core/reading/assemble.go:665 — "func trackedSet(repoRoot string) (map[string]bool, error) {"
    evidence: internal/core/reading/assemble_test.go:377 — "func TestAssembleRefusesADeletedIncludedPath(t *testing.T) {"
    evidence: internal/core/reading/assemble_test.go:399 — "func TestIgnoredFilesNeverEnterTheAssembly(t *testing.T) {"
  - The instrument's own output never becomes its input, and the promise is kept twice over: the path deny makes a run directory unreachable, an output directory the table could reach is refused at the moment it is named, and any admitted file carrying an artefact tag in its BYTES is refused whatever its filename or nesting
    evidence: internal/core/reading/assemble.go:587 — "func refuseOwnArtefact(rel string, raw []byte) error {"
    evidence: internal/core/reading/assemble.go:572 — "The check is CONTENT-SIGNED, not extension- and parse-shaped. An earlier form"
    evidence: internal/core/reading/assemble.go:339 — "reading: the output directory %s is inside the include table's reach"
    evidence: internal/core/reading/assemble_test.go:683 — "func TestOwnArtefactRefusalIsContentSigned(t *testing.T) {"
  - Record enumeration reuses lint.LoadRecordGraph rather than growing a second parser of the record's shape, and a configuration that is absent, silent about a named store, or pointing a store at a non-directory refuses the run instead of enumerating nothing and reporting clean — the no-false-green rule applied where it is easiest to break
    evidence: internal/core/reading/assemble.go:705 — "graph, err := lint.LoadRecordGraph(cfg, repoRoot)"
    evidence: internal/core/reading/assemble.go:620 — "func requireConfiguredStores(repoRoot string, cfg lint.Config) error {"
    evidence: internal/core/reading/assemble.go:694 — "is absent, so the record scan enumerates"
    evidence: internal/core/reading/assemble.go:745 — "info, err := os.Lstat(base)"
    evidence: internal/core/reading/assemble_test.go:863 — "func TestAbsentWalkSourceRefuses(t *testing.T) {"
  - The widening/entailment asymmetry the intent states is in the table rather than remembered, and holds on the live corpus: no draft or planned intent appears among the 907 items of a widening run
    evidence: internal/core/reading/include.go:220 — "Positions: []Position{PositionEntailment},"
    evidence: internal/core/reading/include.go:301 — "Positions: []Position{PositionWidening, PositionComparative, PositionDetection},"
    evidence: internal/core/reading/include_test.go:187 — "func TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem(t *testing.T) {"
    evidence: internal/core/reading/assemble_test.go:44 — "func TestDraftBodyIsColdAtEntailmentAndWarmElsewhere(t *testing.T) {"
  - Determinism is delivered as the charter states it, and the manifest's own doc comment refuses to overclaim it: the bundle carries no run id and no timestamp so two assemblies of one state are byte-identical, while the manifest differs in the run id alone and is honestly described as not timestamp-free
    evidence: internal/core/reading/manifest.go:32 — "It carries no run identifier and no timestamp, so two assemblies of one"
    evidence: internal/core/reading/manifest.go:57 — "It carries no timestamp FIELD, but it is not timestamp-free and must not be"
    evidence: internal/core/reading/manifest.go:79 — "func encode(v any) ([]byte, error) {"
    evidence: internal/core/reading/assemble_test.go:132 — "func TestWalkIsLexicographicAndByteStable(t *testing.T) {"
    evidence: .abcd/development/readings/README.md:11 — "It carries no timestamp field, but it is not timestamp-free: the run identifier"
  - The key-and-heading half of the floor fails CLOSED like its path half, which is the design decision that separates this delivery from a disclosure: a duplicated excluded key, an unresolvable YAML construction, a surviving key, an indented ATX heading, a raw HTML heading and a setext underline each refuse the run rather than travel under a manifest asserting the opposite
    evidence: internal/core/reading/project.go:330 — "declares the excluded key %q more than once (line %d);"
    evidence: internal/core/reading/project.go:335 — "uses the YAML construction %q at line %d in its frontmatter,"
    evidence: internal/core/reading/project.go:389 — "indents the excluded heading %q at line %d; the floor"
    evidence: internal/core/reading/project.go:442 — "underlines the excluded heading %q at line %d; the floor"
    evidence: internal/core/reading/assemble_test.go:1074 — "func TestUnresolvableFrontmatterShapesRefuse(t *testing.T) {"
- diverged:
  - The exclusion floor's key and heading signals are scoped to MARKDOWN, so a record-shaped document embedded in a source file travels whole — this is not hypothetical: a widening run over this repository at db3291f8 passes three literal `## Audit Notes` sections with their acceptance rollups from a Go test fixture, while the same run's manifest asserts `Audit Notes` was refused. Disclosed in the charter, absent from the manifest a third party actually receives, and absent from the nine records the branch ships open
    evidence: internal/core/reading/project.go:57 — "if !strings.EqualFold(path.Ext(rel), \".md\") {"
    evidence: internal/core/site/fixture_test.go:665 — "## Audit Notes"
    evidence: internal/core/site/fixture_test.go:667 — "Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0"
    evidence: .abcd/development/readings/README.md:73 — "The heading signal is scoped to markdown."
    evidence: internal/core/reading/include.go:278 — "{Rule: \"field projection\", Signal: \"heading\", Detail: \"Audit Notes\"},"
  - 'Blindness is enforced by construction, not instructed' is delivered as one enforced half and one instructed half. The bundle's structure is pathless by construction; the reader's repository access is a host obligation discharged by a paragraph in the plugin page, which is instruction — the delivery says so in its own words rather than claiming otherwise
    evidence: commands/reading.md:84 — "### The host obligation this binary cannot discharge"
    evidence: commands/reading.md:90 — "The other half is yours. When you dispatch an assembled input to a reader,"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:187 — "the plugin surface section, disclosed as a host obligation and never claimed"
    evidence: .abcd/development/intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:100 — "The invocation constructs a context-isolated reading: the"
  - 'The bundle carries no repository path' is a claim about the bundle's SHAPE, not its content: 558 of 907 items in a live widening run quote a repository path in their text and 4 quote their own, so a reader can learn the repository's layout from the material even though no item is mapped to a location. Disclosed in the test's own comment, and nowhere in the intent, the spec's summary or the charter's closing section, each of which states the pathlessness unqualified
    evidence: internal/core/reading/assemble_test.go:101 — "necessarily prose and source that may quote paths of its own, so the claim is"
    evidence: internal/core/reading/assemble_test.go:118 — "for _, fragment := range []string{\"/\", \"\\\\\", \".abcd\", \"internal\", \"docs\", root} {"
    evidence: .abcd/development/readings/README.md:150 — "The bundle is pathless by construction: each item is a key, a material class"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:177 — "make the bundle itself pathless"
  - The invocation carries a third operand the ruled interface does not name. `--out` is a free-text string flag against a criterion that says 'no free-text argument anywhere'; it is structurally prevented from becoming a framing channel, which is why I do not read it as a breach of the threat model, but the shipped flag set and the ruled sentence do not agree
    evidence: internal/surface/cli/reading.go:133 — "assembleCmd.Flags().StringVar(&outDir, \"out\", \"\","
    evidence: internal/core/reading/assemble.go:57 — "Neither ARTEFACT carries an output path at all."
    evidence: .abcd/development/intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:35 — "question come from its definition — there is no channel through which"
  - `--dry-run` is delivered as 'writes nothing into the repository's own tiers' rather than the spec's 'renders and writes nothing': with `--out` the two artefacts still land. The narrowing is stated in the flag help and the plugin page, so it is loud rather than silent, but the closed spec's own sentence was never corrected to match
    evidence: internal/surface/cli/reading.go:136 — "assembleCmd.Flags().BoolVar(&dryRun, \"dry-run\", false,"
    evidence: internal/core/reading/assemble.go:42 — "DryRun writes nothing into the repository's own tiers. With OutDir set the"
    evidence: commands/reading.md:66 — "Without it, they land in the local-tier run directory"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:89 — "`--dry-run` renders and writes nothing, on the render-without-writing idiom"
  - Assembler rule 1 is derivable in the DENY but remembered in its test: the executable property is stated against a hand-maintained list of five known family roots, and the deny is measured from a row's own Source downward, so a family placed inside an already-admitted walk directory (a brief chapter, the glossary) would be admitted with no assembler change. Every family this repository actually holds is closed structurally, the readings family this delivery itself adds included
    evidence: internal/core/reading/include_test.go:35 — "var recordFamilyRoots = []string{"
    evidence: internal/core/reading/deny.go:12 — "The one difference from launch's is where the deny is measured FROM. Here it"
    evidence: internal/core/reading/deny.go:92 — "sub := rel"
    evidence: .abcd/development/intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:63 — "excluded by construction; and **a reading's object excludes the"
  - The bundle's readings family is declared as a durable record family by this intent, but the family's own record schema and its `run.json` are not in this delivery — the charter and the `.abcd/development/readings/` directory exist carrying only the include table; `ingest` and the run record are spc-63's, and the durable-tier promotion the ruling names has no producer this cycle
    evidence: .abcd/development/readings/README.md:6 — "A run's artefacts live at `.abcd/development/readings/<run-id>/`, where"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:87 — "`ingest` promotes `manifest.json` and writes `run.json`"
    evidence: internal/core/reading/manifest.go:100 — "It has NO front door yet, and that is deliberate rather than an oversight"
    evidence: internal/core/reading/assemble.go:82 — "const DefaultRunDir = \".abcd/.work.local/scratch/reading-runs\""
- missing:
  - The exclusion floor's universal is not achieved and the branch ships knowing it: three VERIFIED leaks on exotic input stay open by maintainer ruling — a fence inside a frontmatter block scalar switching off the excluded-key scan, a newline between an attribute's `=` and its quote hiding a heading opener, and a CRLF blank line the raw-heading soft bound does not match. All three pre-date round 10 and none is closed in this delivery
    evidence: .abcd/work/issues/open/iss-2608301350533102-fencemask-spans-the-frontmatter-so-a-fence-delimiter-inside.md:12 — "fenceMask spans the frontmatter so a fence delimiter inside a block scalar switches off the excluded key scan for the rest of the block"
    evidence: .abcd/work/issues/open/iss-2608301350534164-a-newline-between-an-attribute-name-s-equals-and-its-opening.md:30 — "Every renderer reads that as the excluded heading. Verified leaking on this"
    evidence: .abcd/work/issues/open/iss-2608301421380392-the-raw-heading-soft-bound-does-not-match-a-crlf-blank-line.md:12 — "the raw heading soft bound does not match a CRLF blank line so an unclosed heading in a CRLF file reads its title over the rest of the document and travels"
    evidence: internal/core/reading/project.go:294 — "func fenceMask(lines []string) []bool {"
    evidence: internal/core/reading/project.go:197 — "rawHeadingBoundRe = regexp.MustCompile(`(?is)</([a-z][a-z0-9-]*)\\s*>|<h[1-6](?:\\s[^>]*)?/?>|\\n[ \\t]*\\n`)"
  - Six further exclusion-floor records ship open beside the three verified leaks: the line-0 frontmatter confinement trade, the compact block-sequence mapping, the flow-context explicit key, the duplicated tag definition, a quadratic opener shape and an over-asserting refusal message. Nine open records against one floor is the arms-race the maintainer ruling names, and the floor's own intent is owed
    evidence: .abcd/work/issues/open/iss-2608301237456350-confining-the-frontmatter-block-to-line-0-silently-admits-a.md:8 — "found_during: \"itd-183-round-9-security\""
    evidence: .abcd/work/issues/open/iss-2608301237450573-pre-existing-on-the-itd-183-branch-a-compact-nested-mapping.md:8 — "found_during: \"itd-183-round-9-security\""
    evidence: .abcd/work/issues/open/iss-2608301251398360-a-flow-context-explicit-key-leaks-an-excluded-field-and-want.md:8 — "found_during: \"itd-183-round-9-ruthless\""
    evidence: .abcd/work/issues/open/iss-2608301251394412-openstag-restates-htmltagre-s-rule-in-hand-written-code-givi.md:8 — "found_during: \"itd-183-round-9-ruthless\""
    evidence: .abcd/work/issues/open/iss-2608301421382564-a-raw-heading-opener-with-no-hard-bound-renders-its-title-ov.md:8 — "found_during: \"itd-183-round-10-ruthless\""
    evidence: .abcd/work/issues/open/iss-2608301421381157-the-escaped-key-refusal-returns-through-an-error-message-ass.md:8 — "found_during: \"itd-183-round-10-ruthless\""
  - 'The invocation constructs a context-isolated reading' has no producer in this delivery and no path to one inside the binary: nothing dispatches a bundle, the four definitions are itd-184's, and the enforcing half of the isolation is permanently a host obligation. The intent's own resolution — 'the definitions assert the posture, this intent builds the wall' — is delivered as half a wall
    evidence: .abcd/development/intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:103 — "invariant 15 (adopted 2026-08-28); the definitions assert the posture, this"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:269 — "reading is commissioned, no bundle is dispatched, and no reading record is"
    evidence: internal/surface/cli/reading.go:14 — "given and the manifest an auditor checks it by; dispatching it to a reader is"
    evidence: .abcd/development/intents/planned/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md:4 — "spec_id: spc-62"
  - The comparative fallback's precondition is owed rather than discharged: ac-3 requires the evaluative position's need for candidates at invocation to be tested against the four definitions BEFORE the interface freezes, and the interface has shipped — versioned, golden-hashed and documented on both front doors — with the four definitions still unwritten
    evidence: .abcd/development/intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:145 — "invocation is tested against the four definitions before the interface"
    evidence: .abcd/development/specs/closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md:278 — "invocation) stays recorded and unbuilt unless the four definitions show the"
    evidence: internal/core/reading/include.go:24 — "const AssemblerVersion = \"1.0.0\""
    evidence: internal/core/reading/status.go:16 — "const DefinitionsDir = \"agents\""

## Grounds

- pursued: This conjecture is pursued now because the location tiering the repository already has is organisational rather than an access control: nothing today prevents a reading reaching ledger content, so every claim the instrument makes about what it saw rests on a disclosure taken on trust. The assembler is the cheapest point at which that becomes checkable, and the two evals and the output contract that follow have nothing to guard until it exists.
