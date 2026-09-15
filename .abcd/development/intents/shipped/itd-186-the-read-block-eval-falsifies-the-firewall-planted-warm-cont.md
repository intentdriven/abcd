---
id: itd-186
slug: the-read-block-eval-falsifies-the-firewall-planted-warm-cont
spec_id: spc-64
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# The read-block eval falsifies the firewall — planted warm content that reaches a reading fails the build loudly

## Press Release

> **The only component capable of falsifying the blindfold rather than
> asserting it.** The eval plants sentinel warm content across every warm
> location class in a fixture repository state — a decision record, a
> wontfix reason, a framing trace, transcript-class text, an `origin`
> stamp on an included record type — and asserts its absence from the
> assembler's output. It asserts on fields, not paths: a warm field
> landing in a new place is exactly the failure a path assertion misses.
> Its oracle is independent of the assembler's include table — an eval
> that read the same table could only assert the table, never falsify it.

## What's In Scope

- The sentinel fixture state and the planted warm content, one plant per
  warm location class, maintained in the eval and never derived from the
  assembler's configuration.
- The field-level assertion over the assembler's output, wired into CI so
  the firewall is checked on every change, not per case run.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** the fixture state, **when** the eval runs, **then** it passes only if
  the assembler output contains no planted warm content and no field on the
  exclusion list.
- **Given** a ledger path moved to a new location holding a plant, **when** the
  eval runs, **then** it fails with a non-zero exit and a message naming the
  leaked sentinel's class token and the position at which it leaked.
- **Given** a warm field introduced on a record type already on the include
  list, **when** the eval runs, **then** it fails.
- **Given** a repository state containing manifests and reading records from
  prior runs, **when** the eval runs, **then** none of them appears in the
  assembler's output — the instrument's own exhaust is tested against its own
  read-block (added 2026-08-28; nothing else tests it).
- **Given** the eval package, **when** every Go file under `evals/` is parsed,
  **then** no import path names the assembler's package or its include list, so
  the oracle's exclusion table is transcribed rather than derived.
- **Given** the materialised fixture, **when** the eval runs, **then** every
  declared sentinel class is present the declared number of times, so a corpus
  that lost a plant fails rather than passing silently.

**Disclosed residue (ac-5).** The import guard holds the oracle structurally
independent of the assembler, but a hand-transcribed table can still fall behind
the exclusion list it mirrors. That staleness is bounded by the holed variant
and by ac-6, never by the import check. Prose-borne warmth inside an included
chapter carries no structural signal and stays the residue itd-183 records.


## Grounds

- pursued: Every other component in the workstream asserts the blindfold; this one is the only component capable of falsifying it, and an eval that read the assembler's own include table could only ever confirm the table rather than test the property. It is pursued now because the assembler exists to hold a firewall whose failure is otherwise silent: warm content that reaches a reading contaminates the reading invisibly, and nothing else in the cycle would notice.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e3f354adbbee -->
Fidelity review — receipt rcp-e3f354adbbee (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:758e88f4da2a169d389d5194db11ac0c0d576de3347cb0ae5fb8e0ff2ef49b66
Input attestations: diff:2fd11881..b52a766b on build/itd-186, merged as 446c607f@sha256:6204639e5a9bf73ac56b9da390578deba5148c9b78944ea1e7b450229b966be5;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: I ran the lane on an isolated extract of 446c607f rather than reading it: `go test -tags coldreading ./evals/...` is green, and the three assertions run at all four positions, comparative included. The pass is not vacuous, and I established that by mutation rather than by reading the floors' headers. Running TestReadBlockBaselineIsClean against the holed corpus turns it red at ALL FOUR positions, each finding naming its class; deleting the `origin` exclusion row leaks 3 findings per position; deleting the `Audit Notes` row leaks 2; adding an include row for `.abcd/development/plans` produces exit 2 rather than a leak, which is the builder's declared deviation 3 (two independent mechanisms) demonstrated rather than asserted. The anti-vacuity architecture holds: requireCarriers fails on a missing carrier PATH, on a missing projected FIELD off the manifest's field column, and on missing carrier BYTES, and requireOracleTables pins all eight table sizes to literals including coverage at 50. On the criterion's own words — 'given the fixture state' — the eval passes exactly when the assembler leaks no planted content and no excluded field FROM THAT STATE, and I confirmed the fixture's records carry no shape the oracle cannot see. The corpus-bounded reach of that guarantee is real and substantial and I have recorded it under diverged and missing, but it does not contradict this criterion, which is scoped to the fixture state.
  evidence: evals/coldreading_test.go:31 — "func TestReadBlockBaselineIsClean(t *testing.T) {  — measured green; measured RED at all four positions when pointed at variantHoled, reporting LEDGER-FRAMING and TRANSCRIPT"
  evidence: evals/coldreading_oracle_test.go:289 — "func requireCarriers(t *testing.T, a assembled) {  — path, then manifest field, then bundle bytes; three distinct faults, three distinct messages"
  evidence: evals/coldreading_oracle_test.go:366 — "{\"coverage\", len(coverage), 50},  — the count pin is duplicated rather than derived, and gates TestEveryAssemblerRuleHasAFalsifier via requireOracleTables"
  evidence: evals/coldreading_fixture_test.go:47 — "var fullyAsserted = everyPosition  — all four positions, not the three spc-64 asked for"
  evidence: internal/core/reading/include.go:276 — "{Rule: \"field projection\", Signal: \"frontmatter key\", Detail: \"origin\"},  — measured: deleting this row turns the baseline red 3 times per position"
- ac-2 — MET: I checked the message the operator actually sees rather than the test's claim about it. The holed variant relocates the LEDGER-FRAMING plant out of `.abcd/.work.local/ledger/` into an included brief chapter — literally a ledger path moved to a new location holding a plant — and I ran TestReadBlockBaselineIsClean over that corpus: `go test` reports FAIL (non-zero exit) and every finding line reads `[widening] sentinel absence: ABCD-EVAL-SENTINEL-LEDGER-FRAMING leaked (found in the assembled input's raw serialisation)`. Both required tokens are there: the class token in full and the position, at all four positions. The check is not only in the failure path — TestReadBlockCatchesAHoledFirewall asserts the exact class SET (so a detector that stopped detecting fails rather than passing quietly) and then re-reads each violation's rendered string for the class token and the position. The negative control also refuses an empty holes table outright, so the control cannot be emptied into a no-op.
  evidence: evals/coldreading_oracle_test.go:197 — "head = fmt.Sprintf(\"[%s] %s: %s leaked (%s)\", v.Position, v.Rule, sentinelPrefix+v.Class, v.Detail)  — measured output: \"[widening] sentinel absence: ABCD-EVAL-SENTINEL-LEDGER-FRAMING leaked\", go test exit non-zero"
  evidence: evals/coldreading_test.go:87 — "if !strings.Contains(msg, sentinelPrefix+v.Class) || !strings.Contains(msg, position) {"
  evidence: evals/coldreading_test.go:52 — "if len(holes) == 0 { t.Fatal(\"the holes table is empty, so the negative control controls nothing\""
  evidence: evals/coldreading_fixture_test.go:261 — "Class: \"LEDGER-FRAMING\", From: \".abcd/.work.local/ledger/2026-08-30-declined-construals.md\", To: \".abcd/development/brief/01-product/06-framing.md\""
- ac-3 — MET: Watched red twice, on both halves of the class, against the real binary. Deleting the `origin` frontmatter-key exclusion from the assembler's Exclusions turns TestReadBlockCatchesWarmFieldsOnIncludedTypes red at every position with 3 findings each; deleting the `Audit Notes` heading exclusion turns it red with 2, naming `item itm-0005 carries the heading "Audit Notes"`. Both plants sit on record types the include table admits — `origin` on a spec and a discipline that travel WHOLE, `production_mode` on a brief chapter — which is what the criterion asks for. The corpus is built deliberately so this is falsifiable at all: each of the four excluded headings has a home on a record type that travels whole, because on a PROJECTED type the projection keeps the heading out whatever the floor says, so deleting the heading's exclusion there would leak nothing and the rule could not be falsified. That reasoning is stated in the corpus and I confirmed it by running the mutations against the whole-travelling homes.
  evidence: evals/coldreading_test.go:103 — "func TestReadBlockCatchesWarmFieldsOnIncludedTypes(t *testing.T) {  — measured red on two separate Exclusions deletions"
  evidence: evals/coldreading_fixture_test.go:146 — "Every one of the four excluded headings has a home on a record type that travels WHOLE ... because on a PROJECTED record type the projection keeps the heading out whatever the floor says"
  evidence: evals/testdata/cold-reading/baseline/.abcd/development/specs/open/spc-1-a-design-record.md:4 — "origin: ABCD-EVAL-SENTINEL-WARM-KEY  — on a spec, which travels whole"
  evidence: internal/core/reading/include.go:278 — "{Rule: \"field projection\", Signal: \"heading\", Detail: \"Audit Notes\"},  — measured: deleting it leaks the heading into item itm-0005"
- ac-4 — MET: The exhaust class is genuinely planted and genuinely blocked, and I proved both by mutation. The corpus carries a prior manifest under `.abcd/development/readings/<run-id>/`, a prior reading record under `.abcd/work/issues/readings/` and a prior disposition — three plants, whose presence is independently pinned by the ac-6 guard. Admitting the readings family and deleting its Exclusions row makes the assembler REFUSE (exit 2, refuseOwnArtefact catching its own tag), which the eval reports as a failed assembly; admitting the issue ledger's readings bucket and deleting the `.abcd/work/issues` Exclusions row leaks ABCD-EVAL-SENTINEL-EXHAUST and turns the test red with 3 findings per position. The criterion's parenthetical is honoured too: this runs at all four positions including comparative, where spc-64 asked only for the exhaust read-block — so the position most likely to be handed the instrument's own output by accident is asserted.
  evidence: evals/coldreading_test.go:129 — "func TestPriorRunExhaustNeverReaches(t *testing.T) {  — measured: exit 2 under one mutation, EXHAUST leaked ×3 per position under the other"
  evidence: evals/coldreading_fixture_test.go:179 — "Name: \"EXHAUST\", Homes: [readings/rdg-.../manifest.json, work/issues/readings/rdi-1..., work/issues/dispositions/dsp-1...], Count: 3"
  evidence: internal/core/reading/assemble.go:587 — "func refuseOwnArtefact(rel string, raw []byte) error {  — measured: turns the readings-family mutation into a refusal rather than a leak, exactly as the matrix row declares"
  evidence: evals/testdata/cold-reading/baseline/.abcd/development/readings/rdg-2608300900000001/manifest.json:1 — "the prior-run manifest the fixture plants"
- ac-5 — MET: Both halves of the builder's claim verified independently of the test. `go list -tags coldreading -deps -test ./evals/` reports exactly three of this module's packages — `evals`, `internal/gitutil`, `internal/gittest` — so the assembler is not linked into the cold-reading test binary at all, transitively or otherwise, which is strictly stronger than the criterion's direct-import wording. The same command under `-tags smoke` pulls in 46 module packages including `internal/core/reading`, so the builder's disclosure that the SMOKE lane does reach the assembler transitively via the CLI surface is accurate and is stated in the test's own doc comment rather than hidden. The guard itself walks every `.go` file under `evals/` with go/parser and refuses a vacuous pass on an empty file set; it parses 10 files, the four fixture Go files under testdata included, so it is broader than the criterion asks. The `internal/gittest` import is not a loophole: it is required by this repository's own iss-28 hermeticity gate, TestTestGitCallsAreHermetic, which fails any test file spawning git without it.
  evidence: evals/coldreading_test.go:172 — "func TestOracleImportsNothingFromTheAssembler(t *testing.T) {  — measured: parses 10 files; `go list -tags coldreading -deps -test ./evals/` yields only evals, gitutil, gittest"
  evidence: evals/coldreading_test.go:157 — "var bannedImports = []string{\"internal/core/reading\", \"internal/core/launch\"}"
  evidence: evals/coldreading_test.go:186 — "if len(files) == 0 { t.Fatal(\"found no Go files under evals/ — the import guard would pass vacuously\") }"
  evidence: internal/gittest/hermetic_git_test.go:43 — "TestTestGitCallsAreHermetic is the enforcement half of iss-28  — measured: this is why the eval imports gittest at coldreading_fixture_test.go:22"
- ac-6 — MET: I deleted a plant and watched the guard redden, twice, in two different ways. Deleting the whole `adm-2-selection-grounds.md` plant file makes TestEverySentinelIsPlanted report `ABCD-EVAL-SENTINEL-GROUNDS is planted 1 time(s), want 2`, plus a homes mismatch, plus the tracked-set failure — three independent messages for one lost plant. Blanking a token in place while leaving the file present (`ABCD-EVAL-SENTINEL-DEFINITION` → harmless text) reddens it on both the baseline and the holed variant, which is the harder case a file-existence check would miss. The counts are declared literals rather than derived from a walk, so a corpus that lost a plant fails rather than agreeing with itself, and the guard additionally checks that every repo-side plant is TRACKED — a plant git refused to track is one the assembler walking the tracked set could never have leaked. The class count itself is pinned at 14 by requireOracleTables, so the table cannot shrink under the guard.
  evidence: evals/coldreading_test.go:219 — "func TestEverySentinelIsPlanted(t *testing.T) {  — measured red on a deleted plant file and on a blanked token, both variants"
  evidence: evals/coldreading_test.go:248 — "if !tracked[rel] { t.Errorf(\"%s is planted in %s, which the fixture repository does not track\"  — measured: fires on the deleted plant"
  evidence: evals/coldreading_fixture_test.go:66 — "// Count is how many times the token appears across the materialised fixture. It is declared rather than counted so a corpus that lost a plant fails."
  evidence: evals/coldreading_oracle_test.go:359 — "{\"sentinelClasses\", len(sentinelClasses), 14},  — the plants table cannot shrink behind the guard"

Gap audit:
- honoured:
  - The press release's central claim — an oracle independent of the assembler's include table — is delivered structurally, not promised. The eval never links the assembler: it invokes the built binary out of process, its exclusion table is hand-transcribed with the record source on every row, and the dependency closure of the dedicated lane proves the assembler is absent from the test binary entirely.
    evidence: evals/coldreading_oracle_test.go:246 — "func assemble(t *testing.T, f fixture, position string) assembled {  — out of process, through the built binary; measured closure: evals, gitutil, gittest only"
    evidence: evals/coldreading_oracle_test.go:88 — "{Path: \".abcd/development/brief/03-evidence\", Source: \"itd-183: the evidence chapter is deliberation\"}  — every oracle row carries the record source it was transcribed from"
  - 'It asserts on fields, not paths' is true and is the reason the negative control works. Nothing in checkSentinelAbsence mentions where a plant was, which is why relocating LEDGER-FRAMING into an included brief chapter is caught by content rather than missed by path.
    evidence: evals/coldreading_oracle_test.go:394 — "func checkSentinelAbsence(a assembled) []violation {  — matches over a.BundleRaw, no path mentioned"
    evidence: evals/coldreading_oracle_test.go:439 — "func checkFieldAbsence(a assembled) []violation {  — recursive JSON key walk at any depth, plus a per-item text scan"
  - The anti-vacuity work that took three review rounds is real and each mechanism reaches a distinct failure. The carrier floor pins BYTES, not paths, with markers drawn from distinct sections; and the one contracted field no marker can reach — spec_id, whose projected text is the bare string `spc-1` that also travels inside the whole spec — is pinned off the manifest's own `field` column instead. The two checks reach different halves of the projection contract.
    evidence: evals/coldreading_fixture_test.go:313 — "// It exists because one of the five contracted fields cannot be pinned by a marker AT ALL, and no number of markers could have found that."
    evidence: evals/coldreading_fixture_test.go:378 — "Fields: []string{\"Press Release\", \"Acceptance Criteria\", \"Scope Conditions\", \"Mechanism\", \"spec_id\"},  — all five contracted fields pinned on itd-1"
    evidence: evals/coldreading_oracle_test.go:320 — "for _, field := range c.Fields { if projected[c.Path][field] { continue }"
  - Builder deviation 1 stands up and is strictly stronger than spc-64. The comparative position is asserted in FULL rather than for the exhaust read-block alone, and the reason given — leaving it out left six of the ten sentinel classes unasserted there and made the oracle's own drafts-at-comparative exclusion a row that could never fire — is accurate. I ran all four positions and comparative carries the full three assertions.
    evidence: evals/coldreading_fixture_test.go:43 — "Leaving comparative out left six of the ten sentinel classes unasserted there, and left the oracle's own drafts-at-comparative exclusion a row that could never fire"
    evidence: .abcd/development/specs/closed/spc-64-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md:132 — "At the comparative position the eval asserts only the read-block over the instrument's own stored prior outputs  — delivered wider than specified"
  - Builder deviations 2, 3 and 4 all stand up under measurement. `holed/` is genuinely a baseline plus a two-entry overlay applied at materialisation, so the variants cannot drift; the two-line watched-red run is a real property of the assembler, not a shortcut — I added an include row for `.abcd/development/plans` WITHOUT deleting its Exclusions row and the assembler exited 2 rather than leaking, so exclusion by absence from the positive walk and the fail-closed assertExclusions gate are indeed two independent mechanisms; and the `internal/gittest` import is compelled by iss-28's own enforcement test.
    evidence: evals/coldreading_fixture_test.go:494 — "copyTree(t, baselineDir, f.Root); if variant == variantHoled { for _, h := range holes { os.Remove(...); copyFile(...) } }"
    evidence: internal/core/reading/assemble.go:480 — "func assertExclusions(cands []candidate, exclusions []Exclusion) error {  — measured: an include row added alone yields exit 2, never a leak"
    evidence: evals/coldreading_fixture_test.go:534 — "That helper reaches internal/gitutil and nothing else, so it costs the oracle none of its independence from the assembler."
  - iss-2608311238236490 CONFIRMED, not overturned. `internal/core/reading`'s Exclusions carries no entry for `.abcd/.work.local`, and I read the manifest an actual run writes: its 23 asserted exclusions never mention the tier. The tier is nonetheless unreachable — `.abcd` is a denySegment pruning every root row, and no include row names it — and the baseline run leaks neither LEDGER-FRAMING nor TRANSCRIPT. So it is a disclosure gap against brief invariant 16, not a leak, exactly as classified. The eval's own oracle is the stricter of the two, and the matrix's two local-tier rows correctly need no Exclusions row deleted, which is how the asymmetry surfaced.
    evidence: internal/core/reading/include.go:275 — "var Exclusions = []Exclusion{  — 23 entries, none naming .abcd/.work.local; measured in a written manifest, `local tier mentioned: False`"
    evidence: internal/core/reading/deny.go:24 — "var denySegments = []string{\".git\", \".abcd\", \"agents\", \"evals\"}  — why the tier is unreachable despite the silence"
    evidence: evals/coldreading_oracle_test.go:104 — "{Path: \".abcd/.work.local\", Source: \"brief invariant 14: the local ledger side, which no reading consumes\"}  — the oracle names what the assembler does not"
  - Scope Conditions: itd-186 states 'None stated', and the delivery neither invented nor dispositioned one.
    evidence: .abcd/development/intents/shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md:35 — "## Scope Conditions  None stated."
  - The CI job promised by the 2026-08-30 maintainer ruling exists and carries no `inert` condition, so it genuinely runs on a record-only pull request — the diff most able to introduce warm content into included material — where the `smoke` job stands down.
    evidence: .github/workflows/ci.yml:480 — "cold-reading-evals:  — no `if:` guard on the job; steps run `make evals-cold-reading`"
    evidence: Makefile:57 — "evals-cold-reading: \tgo test -tags coldreading ./evals/...  — selection is by BUILD TAG, so a later eval joins with no workflow edit"
- diverged:
  - THE FOURTH WAY — the coverage matrix does not name the assembler's Match rule at all, and does not declare it a gap either. `Row.Match` is the file grain of positive inclusion — include.go's own comment calls it 'positive at every grain' — and `matches` is what keeps a file inside an admitted Source from travelling when its basename is not named. No coverage row names it; no row's Falsifier touches it. I made it a total no-op (`matches` always true) and the assembly's manifests came out BYTE-IDENTICAL at widening and entailment, 17 and 22 items, and the whole lane went green. It is not harmless: on a corpus carrying one unmatched-extension file inside an admitted Source and one at the root, the same mutation admits two extra items and leaks both of their tokens. This is precisely the class the matrix's own limit paragraph says it cannot check — 'a row rewritten to a rule the assembler does not have ... needs a human reading the include table against this list' — except worse, because the rule is not rewritten, it is simply absent, so the declared-gap discipline never engaged.
    evidence: internal/core/reading/deny.go:65 — "func (r Row) matches(base string) bool {  — measured: forcing `return true` leaves the coldreading lane GREEN and the fixture manifests byte-identical"
    evidence: internal/core/reading/include.go:105 — "// every file, which no row uses — inclusion is positive at every grain."
    evidence: evals/coldreading_coverage_test.go:92 — "var coverage = []coverageRow{  — 50 rows; none names Match, and none declares it a gap"
    evidence: evals/coldreading_coverage_test.go:440 — "What it does not and cannot check is that the matrix names every rule the assembler HAS  — the disclosed limit, and this is a live instance of it"
  - A SECOND AND LARGER OMISSION — `verifyRedaction` is not named by any coverage row. It is roughly 130 lines and six distinct refusal paths (duplicated excluded key, unresolvable YAML shape, excluded key in the first block at any indent or flow depth, indented ATX heading, raw-HTML heading, setext-underlined heading), and it is the fail-closed half of the exclusion floor's key-and-heading side — the assembler's own comment calls a floor without it 'a disclosure, not a gate'. I deleted the call outright: the coldreading lane stayed GREEN. It is not unfalsifiable in principle, only against this corpus: with a setext-underlined `Audit Notes` in a whole-travelling glossary record, the shipped assembler refuses with exit 2 naming the heading, while the same assembler minus verifyRedaction exits 0 and carries the excluded section's token into the bundle. The matrix touches this seam in exactly one row, and that row is about the ORACLE's own `excludedKeyLine` regex rather than an assembler rule, in a matrix whose declared subject is the assembler's contract.
    evidence: internal/core/reading/project.go:127 — "if err := verifyRedaction(rel, doc, out, keys, headings); err != nil {  — measured: deleting this call leaves the lane GREEN"
    evidence: internal/core/reading/project.go:323 — "func verifyRedaction(rel, original, redacted string, keys, headings map[string]bool) error {  — measured leak: clean binary exit 2 'underlines the excluded heading \"Audit Notes\"'; no-verify binary exit 0, token in bundle"
    evidence: evals/coldreading_coverage_test.go:424 — "Rule: \"an excluded frontmatter key is refused at any indentation, not only at column 0\"  — the one row at this seam, and it describes the eval's own regex, not the assembler's"
  - FOUR MORE assembler mechanisms are invisible to the corpus and unnamed by the matrix, each verified by mutation to leave the lane green: sectionSpan's 'ends at the next heading of the same level or SHALLOWER' rule, whose own comment says a `###` under a projected `##` would otherwise travel short and diverge from the redactor; redactExcluded's block-scalar continuation drop, without which a block-valued `origin:` leaves its value behind in the frontmatter; segmentDenied's case-INSENSITIVE component match, without which a differently-cased `.abcd`/`agents`/`evals` segment escapes the structural deny; and sameRendering, without which `## **Audit Notes**` is no longer the excluded heading. The fixture's records are deliberately flat and plainly spelled, so none of these can fire against it. Each is arguably a declarable gap; none is declared.
    evidence: internal/core/reading/project.go:1077 — "func sectionSpan(sections []site.Section, i, total int) (int, int) {  — measured: level-agnostic mutation leaves the lane GREEN"
    evidence: internal/core/reading/deny.go:34 — "if strings.EqualFold(seg, denied) {  — measured: `seg == denied` leaves the lane GREEN"
    evidence: internal/core/reading/project.go:89 — "for j := i + 1; j < len(lines); j++ {  — the block-scalar run; measured: disabling it leaves the lane GREEN"
    evidence: internal/core/reading/project.go:231 — "func sameRendering(a, b string) bool {  — measured: forcing false leaves the lane GREEN"
  - NOT ON THE DECLARED LIST — the same seam is open on the ORACLE's side too, so the two blind spots compound. The eval's heading check matches only ATX headings (`atxHeading`, `^[ ]{0,3}#{1,6}`). A setext-underlined or raw-HTML excluded heading arriving in a bundle item's text would satisfy `checkFieldAbsence` completely. Today that state is unreachable because the assembler refuses it — but the assembler's refusal is the mechanism the matrix does not name and no plant exercises, so the property rests entirely on an unfalsified guard, with no independent oracle behind it. This is the shape ac-1's 'no field on the exclusion list' would still pass over.
    evidence: evals/coldreading_oracle_test.go:429 — "var atxHeading = regexp.MustCompile(`^[ ]{0,3}#{1,6}\\s+(.*?)\\s*#*\\s*$`)  — ATX only; no setext and no raw-HTML reading"
    evidence: internal/core/reading/project.go:415 — "the setext refusal inside verifyRedaction — the only thing standing between that spelling and the bundle"
  - The intent's scope claim 'wired into CI so the firewall is checked on every change' is delivered as a job that RUNS on every change but GATES nothing. `.abcd/work/rulesets/main-protection.json` lists eight required contexts — attribution, check (macos-latest), check (ubuntu-latest), external-review, gitleaks, record-lint, smoke, zizmor — and `cold-reading-evals` is not among them. spc-64's ruled purpose was that a record-only pull request is the diff most able to introduce warm content and a stood-down job reports green for work that did not happen; a job whose red conclusion cannot block a merge is one step short of that. The literal words 'checked on every change' are satisfied; the protection the ruling was reaching for is not. Correctly captured and left open rather than papered over.
    evidence: .abcd/work/rulesets/main-protection.json:43 — "\"context\": \"smoke\"  — the last of eight required contexts; cold-reading-evals absent"
    evidence: .abcd/work/issues/open/iss-2608311051046981-the-new-cold-reading-evals-ci-job-is-not-a-required-status-c.md:12 — "the job spc-64 lands reports a conclusion without gating the merge ... an unrequired check does not stop one — OPEN"
    evidence: .abcd/development/specs/closed/spc-64-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md:154 — "A stood-down job still reports its check context green, which is a green for work that did not happen."
  - NOT ON THE DECLARED LIST, and minor — the matrix's row count is pinned in a DIFFERENT file from the matrix, inside requireOracleTables alongside seven oracle tables, while the gap count is pinned beside the matrix. Both do gate (TestEveryAssemblerRuleHasAFalsifier calls requireOracleTables first), so this is not a hole; but a reader of coldreading_coverage_test.go alone sees only `declaredGaps = 7` and would reasonably conclude the row count is unpinned. The file's own header says 'this table is the eval's real claim', and half the pin on that claim lives elsewhere.
    evidence: evals/coldreading_coverage_test.go:511 — "const declaredGaps = 7  — the gap pin, beside the matrix"
    evidence: evals/coldreading_oracle_test.go:366 — "{\"coverage\", len(coverage), 50},  — the row pin, in the oracle file"
    evidence: evals/coldreading_coverage_test.go:444 — "requireOracleTables(t)  — which is what makes the row pin gate this test"
- missing:
  - The matrix's headline claim — 'for every rule the assembler's contract carries, the mutation that removes it' — is not delivered for at least two rule families and four further mechanisms. The 50 rows do NOT honestly correspond to the assembler's actual rules: `Row.Match` and `verifyRedaction` are absent from the matrix entirely, neither caught nor declared a gap, and both are demonstrably consequential read-block rules whose complete removal leaves the delivered lane green. `evals/README.md` repeats the claim to the reader in the same terms. The matrix's discipline is that an unfalsifiable rule 'carries its reason in Gap rather than being quietly omitted'; these were quietly omitted, which is the one outcome that discipline exists to prevent. The fix is additive — six more rows, at least two of them Gap rows — and nothing already in the matrix is wrong.
    evidence: evals/coldreading_coverage_test.go:5 — "// The coverage matrix: for every rule the assembler's contract carries, the mutation that removes it and the plant that dies when it does."
    evidence: evals/coldreading_coverage_test.go:19 — "And a row that no mutation can falsify carries its reason in Gap rather than being quietly omitted"
    evidence: evals/README.md:82 — "`coldreading_coverage_test.go` is the matrix: one row per rule, the mutation that removes it, and the plants that die.  — the same claim, to the reader"
  - The always-run lane is not a merge gate. Adding `cold-reading-evals` to the branch ruleset and to its committed mirror is the step that turns the delivered job into the protection spc-64's ruling describes, and it has not been taken in this range.
    evidence: .abcd/work/rulesets/main-protection.json:23 — "\"required_status_checks\": [  — eight contexts follow; cold-reading-evals is not one of them"
    evidence: .abcd/work/issues/open/iss-2608311051046981-the-new-cold-reading-evals-ci-job-is-not-a-required-status-c.md:14 — "Add cold-reading-evals to the ruleset and to its committed mirror. — OPEN"