---
id: itd-184
slug: four-cold-reading-definitions-one-blindness-core-each-positi
spec_id: spc-62
kind: bundle-member
suggested_kind: bundle-member
reclassification_history: []
builds_on: [itd-86]
severity: major
impact: additive
---

# Four cold-reading definitions, one blindness core — each position licenses a different output, and none may hold another's licence

Typed links: `refines` [itd-86](../drafts/itd-86-cold-reading-surface.md) — the
single-document cold reading generalises to four positions, instances
within one detector context.

## Press Release

> **One definition with four objects cannot hold.** The prohibition against
> proposing is constitutive of the detection pass and would void the
> widening pass entirely — so there are four definitions, each holding its
> object, its question, and its regime value, with the blindness core
> byte-identical across all four: no project context · no ledger access ·
> no memory across runs · no ranking or prioritisation · no selection,
> explanation or commitment · named provenance on every item produced ·
> no passed input is authoritative.

| Definition | Regime value | Object |
| --- | --- | --- |
| Widening | `generative` | Brief current text incl. the construal statement; glossary; disciplines; specs; the shipped tree where one exists. Excludes `intents/drafts/` and `intents/planned/` |
| Entailment | `explicative` | The claim record — drafts and planned intents included — plus the constraint sources: disciplines, glossary, specs, brief current text |
| Comparative | `evaluative` | The candidate set (the widening reading's pre-admission output) against the declared selection criteria (the criteria discipline) |
| Detection | `registrative` | Shipped tree against the claim record |

## Ruled

- **Ruled (maintainer, 2026-08-28; decision log):** build four agent definitions, one per
  supply regime, as instances within one detector context — the count of
  definitions and the count of contexts are different countings and are
  compatible. This draft implements the ruling as stated.

## Per-instrument content (maintainer readings design, 2026-08-28)

Each definition holds five things and nothing else: its object, its
question, the blindness core verbatim, its regime value, and its item
shape.

Questions — widening: *given the situation as this design construes it,
what configurations does the construal admit that are not present in what
has been committed to?* Entailment: *what does this design commit to, by
being the kind of thing it is, that its articulation does not state?*
Comparative: *for each candidate and each declared criterion, how do
options of this shape ordinarily behave?* Detection: itd-86's question,
the shipped tree against the claim record.

Item shapes (validated per position by the output contract) — widening:
configuration · what admits it, and no third body field (no preference,
no comparison against what was built); entailment: claim surfaced · claim
type (criterion / causal / context) · what implies it — surfaced, never
dispositioned; comparative: one item per candidate-criterion pair —
candidate id · criterion · characterisation; detection: tension ·
constraint in play · why it is a tension. The pattern named is carried in
the record's envelope at every position, never in a body.

The object asymmetry over drafts is deliberate and is stated in the
assembler's include list: the widening reading must not see the candidate
set it is asked to widen; the entailment reading properly reads drafts,
since articulation precedes selection.

Widening's prohibitions are review-flags, not ingest refusals (the
generative licence is widest): a recommendation among configurations, or a
characterisation of one as better than another, flags for researcher
review — comparison belongs to the comparative position.

Adopted into the blindness core (2026-08-28; routed to the governing
document): **no passed input is authoritative** — no document passed to a reading
is designated the fixed side of any comparison; a discipline, a glossary
term or a declared criterion is as open to being named in an item as
anything else. The core carried verbatim in all four definitions includes
this condition, its seventh. Unlike the other six it is held by the
core's wording rather than by construction — the assembler cannot enforce
it — and it is disclosed as such.

## What's In Scope

- The four definition files under `agents/`, and the test holding the
  blindness core byte-identical across them.
- The regime value stated in each definition and not derivable from
  operator input.

## What's Out of Scope

- Enforcing the blindness — the assembler's job, checked by its evals.
- Validating what a reading produced against its regime — the output
  contract's supply-regime gate.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** the four definitions, **when** the delimited blindness-core span is
  extracted from each and compared, **then** the span is byte-identical across
  all four and carries the seven conditions in the fixed order.
- **Given** any definition, **when** its frontmatter is parsed, **then** it
  states a `regime` value drawn from the four, and the four definitions' values
  are distinct and resolvable from the position alone.
- **Given** the CLI command tree and the configuration schema, **when** every
  registered flag and every registered configuration key is enumerated
  programmatically and searched, **then** none of them sets or overrides a
  run's regime.

**Disclosed residue (ac-3).** The file enumeration is programmatic — every
tracked JSON under `.abcd/`, read from the git index — so it cannot fall behind
the surface it guards. Two written lists survive inside it, and they fail in
opposite directions. The reading verb's pinned operand set fails CLOSED: any
addition to that verb turns the guard red, so it is a tripwire rather than an
enumeration. The generated-baseline exclusion fails OPEN: a regime knob written
into a machine-written baseline is skipped, which is tolerable only because
nothing reads a baseline as configuration, and that exclusion is where it would
stop being tolerable. Outside both, a channel that was never registered — an
environment variable read ad hoc, say — remains beyond the check.


## Grounds

- pursued: This conjecture is pursued now because one definition with four objects cannot hold: the prohibition against proposing is constitutive of the detection pass and would void the widening pass entirely, so the count of definitions has to be settled before anything is written that a reading would read against. The byte-identical core is the cheap guarantee that separating them by licence does not also separate them by blindness.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-426034d44293 -->
Fidelity review — receipt rcp-426034d44293 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:1ce226bf70dd91d2ad0cad86c22e82ae1516311d65766e4d22e85f38e5374c1d
Input attestations: diff:2fd11881..99d87ced on build/itd-184, merged as 78f6bd1d@sha256:37cd9e3ff743fff3bd7f1a049d69425b8d5bb7dc23136d76d8a64178a0eb5cb5;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: I extracted the delimited span from all four files myself rather than trusting the test's name: each file carries exactly one `blindness-core:begin` and one `blindness-core:end`, the span between them is 2917 bytes in all four, and its sha256 is 9a7cb91b7c4a27b9268b75406917a2a6333b4d8163c0ab333d635bd7cc05c5d5 four times over — byte-identical, established independently of the Go test. Reading the span, it carries the seven conditions numbered 1-7 in exactly the order itd-184's press release states them: no project context, no ledger access, no memory across runs, no ranking or prioritisation, no selection/explanation/commitment, named provenance, no passed input is authoritative. The seventh discloses in its own wording that it is held by the wording alone and not by any mechanism, which is what the intent said it would do. Both gates are proved capable of failing by mutation, not by passing: changing one full stop to an exclamation mark inside the detection copy turns TestBlindnessCoreIsByteIdenticalAcrossDefinitions red, and swapping conditions 4 and 5 in ALL FOUR copies (so byte-identity still holds) turns TestBlindnessCoreCarriesSevenConditions red naming condition 5 out of the fixed order. The order check is therefore independent of the identity check, which is what the criterion's two clauses require. No caveat: both clauses are observably realised and each is held by a gate I watched fail.
  evidence: agents/cold-reading-widening.md:47 — "< !-- blindness-core:begin -- >  — span 2917 bytes, sha256 9a7cb91b7c4a27b9268b75406917a2a6333b4d8163c0ab333d635bd7cc05c5d5"
  evidence: agents/cold-reading-entailment.md:47 — "< !-- blindness-core:begin -- >  — same 2917-byte span, same sha256"
  evidence: agents/cold-reading-comparative.md:48 — "< !-- blindness-core:begin -- >  — same 2917-byte span, same sha256"
  evidence: agents/cold-reading-detection.md:44 — "< !-- blindness-core:begin -- >  — same 2917-byte span, same sha256"
  evidence: internal/core/reading/definitions_test.go:34 — "var blindnessConditions = []string{ \"**No project context.**\", ... \"**No passed input is authoritative.**\" }"
  evidence: internal/core/reading/definitions_test.go:147 — "func TestBlindnessCoreIsByteIdenticalAcrossDefinitions(t *testing.T) {  — measured: one-character edit to the detection copy turns it red"
  evidence: internal/core/reading/definitions_test.go:171 — "func TestBlindnessCoreCarriesSevenConditions(t *testing.T) {  — measured: swapping conditions 4 and 5 in all four copies turns it red naming condition 5 out of order"
- ac-2 — MET: The four `regime:` values exist in frontmatter and are distinct: generative, explicative, evaluative, registrative — I read them out of the files, not out of a test. They are resolvable from the position alone BY CONSTRUCTION rather than by lookup: DefinitionPath derives the filename from the position (`agents/cold-reading-<position>.md`), so a position resolves to its file with no table in between, and LoadDefinition then refuses a file whose stated `position:` disagrees with the filename it was reached by. Membership in the closed set is checked, duplicated `regime:`/`position:` keys are refused rather than silently resolved, and the file's stated value is returned verbatim — the locator deliberately does not substitute the position table's value, which is what makes a drift detectable at all. On the two-tables question the host raised: `issueschema.ReadingPositions` (pre-existing, from spc-58 — it is present at 2fd11881, not added here) is a second hardcoded position-to-regime table, and it AGREES with all four definitions. Its existence does not undermine ac-2's 'stated in the definition', because TestRegimeValuesAreTheFourAndDistinct cross-checks every definition's frontmatter value against issueschema.ReadingRegime(position), so the two tables cannot drift apart unnoticed in CI. I proved that gate can fail: setting widening's frontmatter to `regime: evaluative` produces three findings at once — the table disagreement, the duplicate regime against comparative, and 3 distinct regimes across 4 positions. No path decides a regime anywhere other than the definition file: the only two non-test readers of a regime are the locator (frontmatter) and capture/reading.go:505 (the pre-existing table, used to validate a record's declared regime), and a repository-wide sweep found no environment variable and no flag or key that carries one.
  evidence: agents/cold-reading-widening.md:14 — "regime: generative"
  evidence: agents/cold-reading-entailment.md:14 — "regime: explicative"
  evidence: agents/cold-reading-comparative.md:13 — "regime: evaluative"
  evidence: agents/cold-reading-detection.md:13 — "regime: registrative"
  evidence: internal/core/reading/definitions.go:50 — "func DefinitionPath(p Position) string { return DefinitionsDir + \"/\" + definitionPrefix + string(p) + \".md\" }"
  evidence: internal/core/reading/definitions.go:95 — "if stated != string(pos) { ... which is not the position its filename holds"
  evidence: internal/core/reading/definitions.go:81 — "for _, dup := range frontmatter.Duplicates(lines) { if dup.Key == \"position\" || dup.Key == \"regime\" {"
  evidence: internal/core/issueschema/reading.go:107 — "{Position: PositionWidening, Regime: \"generative\", ...}  — the second table; present at 2fd11881, agrees with all four definitions"
  evidence: internal/core/reading/definitions_test.go:239 — "if want := issueschema.ReadingRegime(string(p)); def.Regime != want {  — measured: setting widening to evaluative turns this red on three counts"
- ac-3 — MET: Both halves of the enumeration are programmatic and I proved the guard capable of failing five separate ways rather than reading its header and believing it. Command tree: walked through commandSurface, the repository's one canonical cobra recursion, which descends into hidden commands and records each command's own and own-persistent flags. Mutations — (1) a `--regime` flag on `reading assemble` turns it red twice, on the name check and on the pinned operand set; (2) an innocuously named `--supply-licence` operand on the same verb turns it red on the pinned set alone, which is the builder's fails-CLOSED claim demonstrated; (3) a `--regime` PERSISTENT flag on the ROOT command is caught at `abcd`; (4) an `--override-regime` flag on a HIDDEN command is caught at `abcd mutant`. Configuration: enumerated from the git index, every tracked `.json` under `.abcd/`, plus reflection over the two largest schema types. Mutation (5): a `default_regime` key written into `.abcd/work/rulesets/main-protection.json` — a file that sits outside the two directories the pre-99d87ced glob covered — turns it red, so the index enumeration genuinely reaches what the old written list missed. Three anti-vacuity floors (>=10 commands, >=20 flags, >=40 files, >=500 keys) and a not-a-repository fatal stop the guard from passing by seeing nothing; I confirmed the last of those by running it in a non-git tree, where it fails closed. On the criterion's own words: nothing registered sets or overrides a regime, and I established the underlying fact by sweep as well as by guard — the only non-test readers of a regime are the definition locator and capture/reading.go:505, and no environment variable named or shaped like a regime exists anywhere in the tree. The one exclusion, `*-baseline.json`, I verified fails OPEN by mutation (6): a `default_regime` key in `.abcd/citations-baseline.json` leaves the guard green. That exclusion covers two machine-written caches that nothing reads as configuration, so it removes a false red rather than a registered key — which is why this is MET rather than narrowed. The gap it leaves is real and I have recorded it, together with the fact that itd-184's own residue paragraph does not disclose it, under diverged.
  evidence: internal/surface/cli/regime_surface_test.go:227 — "func TestNoOperatorSurfaceSetsARegime(t *testing.T) {  — measured red on 5 distinct mutations, green on the baseline mutation"
  evidence: internal/surface/cli/surface.go:87 — "func commandSurface(cmd *cobra.Command) []surface.Command {  — recurses into hidden children; measured: `abcd mutant --override-regime` caught"
  evidence: internal/surface/cli/surface.go:104 — "cmd.LocalFlags().VisitAll(  — own plus own-persistent; measured: a root persistent --regime caught at \"abcd\""
  evidence: internal/surface/cli/regime_surface_test.go:141 — "listed, err := gitutil.Run(repoRoot, \"ls-files\", \"-z\", \"--\", \".abcd\")  — measured: a key in .abcd/work/rulesets/main-protection.json turns it red"
  evidence: internal/surface/cli/regime_surface_test.go:76 — "var readingOperands = map[string][]string{ \"abcd reading\": {}, \"abcd reading assemble\": {\"dry-run\", \"out\", \"position\", \"target\"} }  — measured: --supply-licence turns it red, so it fails CLOSED as declared"
  evidence: internal/surface/cli/regime_surface_test.go:96 — "const generatedBaselineSuffix = \"-baseline.json\"  — measured: a default_regime key in .abcd/citations-baseline.json leaves the guard GREEN, so it fails OPEN as declared"
  evidence: internal/surface/cli/regime_surface_test.go:138 — "if !gitutil.InRepo(repoRoot) { t.Fatal(  — measured: fails closed in a non-repository tree"
  evidence: internal/surface/cli/regime_surface_test.go:171 — "if files < 40 { ... if belowTheOldGlobs == 0 { ... if excluded > 3 {  — three anti-vacuity floors, plus >=10 commands, >=20 flags, >=500 keys"
  evidence: internal/core/capture/reading.go:505 — "if regime := fm[\"regime\"].(string); regime != issueschema.ReadingRegime(position) {  — one of only two non-test regime readers; no env var or operand is a third"

Gap audit:
- honoured:
  - Four definitions, one per supply regime, each holding its object, its question, the blindness core verbatim, its regime value and its item shape — and nothing else.
    evidence: agents/cold-reading-widening.md:20 — "## Object / ## Question / ## The blindness core / ## Regime / ## Item shape — the five headings, and no sixth"
    evidence: internal/core/reading/definitions_test.go:252 — "func TestDefinitionHoldsItsFiveParts(t *testing.T) {"
  - The item shapes match the bodies spc-63 will validate, and no definition names another position's body field — so a licence cannot be smuggled in as a field.
    evidence: internal/core/reading/definitions_test.go:35 — "for _, field := range issueschema.ReadingBodyFields[string(p)] {  — and lines 43-57 refuse every OTHER position's field"
    evidence: agents/cold-reading-widening.md:109 — "Two body fields and no third: `configuration` and `what_admits_it`. — matches ReadingPositions[widening].Fields exactly"
    evidence: internal/core/issueschema/reading.go:107 — "Fields: []string{\"configuration\", \"what_admits_it\"}"
  - The work is WIRED: the definition locator is reachable and demonstrably executing from the production entry point, not only from tests.
    evidence: internal/surface/cli/reading.go:47 — "status, err := reading.Describe(captureRoot(mustCwd()))  — I ran `abcd reading` from a checkout of 78f6bd1d: it lists the four definitions"
    evidence: internal/core/reading/status.go:56 — "defs, err := LoadDefinitions(repoRoot)  — replaces the pre-range os.ReadDir listing"
    evidence: internal/core/reading/definitions.go:99 — "states no 'regime' in its frontmatter  — measured: stripping `regime:` from the entailment definition makes the REAL BINARY exit 2 on the bare verb, in both text and --json renders"
  - Builder deviation 1 stands up: the `task_classes` closed enum is genuinely unenforceable in Go, and no enforcement site was missed.
    evidence: internal/core/lint/agentcontract.go:198 — "if strings.TrimSpace(p.scope[\"task_classes\"]) == \"\" {  — the ONLY check: non-empty, no membership test"
    evidence: .abcd/development/brief/02-constraints/04-naming.md:91 — "This table is the source of truth today: the binary carries no `task_classes` schema and no cross-check test reads the field (iss-265). — measured: replacing [cold_reading] with [utter_nonsense_not_in_any_enum] leaves TestColdReadingDefinitionsSatisfyTheAgentContract green"
    evidence: agents/README.md:55 — "`cross_document_audit`, `cold_reading`). — the token added to both prose surfaces, correctly, and to no Go enum"
  - Builder deviation 2 stands up: spc-62's locator sentence now describes what is actually there, and the capture's own premise was indeed half wrong.
    evidence: .abcd/development/specs/closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md:201 — "the definition LOCATOR lives in that package and is spc-62's own delivery: `LoadDefinition(repoRoot, position)` resolves a position ... returns the file's sha256 — every clause verified by running it"
    evidence: .abcd/work/issues/resolved/iss-2608310954048572-spc-62-states-that-internal-core-reading-already-locates-the.md:18 — "no non-test file in that package references agents/ at all — FALSE at 2fd11881: status.go:16 declared `const DefinitionsDir = \"agents\"` and status.go:51 read that directory"
  - All three issues the host named as resolved in this range are genuinely closed, in resolved/, each carrying a resolution line that matches what the code now does.
    evidence: .abcd/work/issues/resolved/iss-2608310954048572-spc-62-states-that-internal-core-reading-already-locates-the.md:13 — "resolution: the locator is now built in internal/core/reading and the sentence corrected to describe it"
    evidence: .abcd/work/issues/resolved/iss-2608311100440709-the-ac-3-operator-surface-guard-walks-the-generated-baseline.md:13 — "resolution: the configuration walk skips the machine-written baselines by their -baseline.json shape — verified by mutation"
    evidence: .abcd/work/issues/resolved/iss-2608311100496798-the-ac-3-operator-surface-guard-s-configuration-walk-is-glob.md:13 — "resolution: the configuration walk enumerates every tracked .abcd json from the git index — verified by mutation against a file outside the old globs"
  - Scope Conditions: itd-184 states 'None stated', and none was invented or dispositioned in the delivery.
    evidence: .abcd/development/intents/shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md:97 — "## Scope Conditions  None stated."
  - Builder deviation 4 stands up: nothing was written to .abcd/work/DECISIONS.md in the range, and the two rulings the delivery implements are cited to the intent and spec rather than re-minted.
    evidence: .abcd/development/specs/closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md:1 — "Ruling (8) settles it — cited, not re-decided; `git diff --name-status 2fd11881..99d87ced` touches no DECISIONS.md"
- diverged:
  - NOT ON THE DECLARED LIST — itd-184's own Disclosed residue paragraph is now narrower than the guard it describes, and was never corrected. It says the enumeration is 'programmatic over the registered surfaces rather than a written list'; the delivered guard's header says in capitals that TWO WRITTEN LISTS SURVIVE. More consequentially, the residue names only 'a channel that was never registered' as what the walk cannot see, and omits the fails-OPEN `*-baseline.json` exclusion — which I proved is a real blind spot: a `default_regime` key written into `.abcd/citations-baseline.json` leaves the guard green. iss-2608311100496798 found exactly this class of false sentence and corrected the guard header and spc-62; the intent's residue paragraph was left as it was (the intent file is untouched across 2fd11881..99d87ced). spc-62 does disclose both edges correctly, so the design record is honest at the spec tier and under-discloses at the intent tier.
    evidence: .abcd/development/intents/shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md:113 — "The enumeration is programmatic over the registered surfaces rather than a written list, per itd-195, so it cannot fall behind the command tree it guards."
    evidence: internal/surface/cli/regime_surface_test.go:25 — "TWO WRITTEN LISTS SURVIVE, and both are stated rather than implied"
    evidence: .abcd/development/specs/closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md:186 — "A knob written into one of the machine-written baselines (`*-baseline.json`) is skipped. — stated here, and only here"
  - NOT ON THE DECLARED LIST — a position/regime drift is caught in CI but NOT at runtime, and the bare verb reports the drifted definition as healthy. definitions.go's own header argues that answering from the position table 'would make the frontmatter decorative and a drift between the two undetectable' — but knownRegime checks MEMBERSHIP only, never agreement with the position. Measured: setting `regime: evaluative` under `position: widening` makes the real binary print all four definitions and exit 0, while the Go test goes red. spc-63 will read a run's regime out of LoadDefinition, so a drifted definition would today hand the ingest verb the wrong licence with no refusal from the locator; the only backstop is the separate check at capture/reading.go:505 against the pre-existing table, which is a different comparison. The design intent is sound; the enforcement sits one tier further out than the comment implies.
    evidence: internal/core/reading/definitions.go:10 — "a locator that answered from the table would make the frontmatter decorative and a drift between the two undetectable"
    evidence: internal/core/reading/definitions.go:156 — "func knownRegime(token string) bool { for _, p := range issueschema.ReadingPositions { if p.Regime == token {  — membership, not agreement with p.Position"
    evidence: internal/core/reading/definitions_test.go:239 — "if want := issueschema.ReadingRegime(string(p)); def.Regime != want {  — the ONLY place the agreement is enforced; measured: binary exits 0 on the same drift"
  - NOT ON THE DECLARED LIST — the locator's refusal messages reach the operator with a doubled prefix. Both the CLI wrapper and every error definitions.go returns prepend 'reading: ', so the bare verb prints `abcd: reading: reading: agents/cold-reading-entailment.md states no 'regime' ...`, and the --json render carries the same doubling inside its error string. User-visible on the exact refusal path spc-62 introduced as the proof of wiring.
    evidence: internal/surface/cli/reading.go:49 — "return &exitError{Code: 2, Msg: \"reading: \" + scrubPaths(err)}"
    evidence: internal/core/reading/definitions.go:100 — "return Definition{}, fmt.Errorf(\"reading: %s states no 'regime' in its frontmatter; ...\"  — measured output: \"abcd: reading: reading: agents/...\""
  - NOT ON THE DECLARED LIST — the host's brief names five build commits; the range 2fd11881..99d87ced carries SIX. The unlisted one, a1046515, is not cosmetic: it introduced the `scalar` helper after the locator's first cut refused well-formed definitions by comparing a quoted value against itself, and that helper is the THIRD copy of the strip-then-Unquote idiom in the tree. Captured, correctly, and left open.
    evidence: internal/core/reading/definitions.go:139 — "This is the THIRD copy of the strip-then-decode idiom — capture's reader and record-lint's schema gate hold the other two"
    evidence: .abcd/work/issues/open/iss-2608311039531552-the-strip-a-matched-quote-pair-then-frontmatter-unquote-idio.md:16 — "The idiom belongs beside Unquote in internal/core/frontmatter, with the three call sites moved onto it. — OPEN"
  - The ac-3 search is a NAME-substring match for the token 'regime' plus a pinned-operand tripwire on the two reading commands. A configuration key or a flag on some other command that set a regime under a different name would pass the search. Adequate today and only today: I verified by sweep that no such channel exists, and the tripwire makes any new operand on either reading command go red — but the guarantee rests on the tripwire's two-command scope, not on the search.
    evidence: internal/surface/cli/regime_surface_test.go:65 — "const regimeToken = \"regime\"  — matched case-insensitively against flag names, shorthands and configuration keys"
    evidence: internal/surface/cli/regime_surface_test.go:76 — "var readingOperands = map[string][]string{ \"abcd reading\": {}, \"abcd reading assemble\": {...} }  — the tripwire covers these two commands and no others"
- missing:
  - The plugin markdown surface still describes the pre-change behaviour. commands/reading.md calls the bare verb's `definitions` field 'the reading definitions present under agents/', which is what Describe did BEFORE this range; it now reports what the locator RESOLVES, and a malformed definition refuses the whole verb with exit 2 — which I confirmed by running the binary. commands/ is untouched across 2fd11881..99d87ced. Under this repository's own 'wired or it isn't done' boundary the markdown surface is one of the two front doors, so a delivered behaviour change is documented on one door and not the other. Captured as a minor observation and explicitly held outside the builder's lane rather than taken.
    evidence: commands/reading.md:33 — "(the reading definitions present under `agents/`)"
    evidence: .abcd/work/issues/open/iss-2608311039586922-commands-reading-md-describes-the-bare-verb-s-definitions-fi.md:14 — "The plugin surface should say resolved rather than present, and say that a malformed definition is a refusal. The itd-184 builder's lane did not include commands/, so the correction was captured rather than taken. — OPEN"
  - Nothing dispatches a reading. The four definitions ship unrun for the whole cycle by spc-62's own declaration, and the regime a definition states is never rendered to an operator: Describe reports only the definition NAMES, not the regime or the file hash the locator computes. Not a criterion gap — every criterion is about the definitions and the surfaces, not about a run — but it is the bound on what any of this has been measured against, and the instruments sit at prompt_version 0.1.0, declared shipped and honestly unmeasured.
    evidence: internal/core/reading/status.go:31 — "Definitions []string `json:\"definitions\"`  — names only; Regime and SHA256 resolved at status.go:56 are discarded"
    evidence: .abcd/development/specs/closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md:246 — "**Running a reading.** The instrument ships unrun for the whole cycle: the definitions are written, linted and tested, and none is dispatched."