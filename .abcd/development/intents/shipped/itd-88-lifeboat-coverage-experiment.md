---
id: itd-88
slug: lifeboat-coverage-experiment
spec_id: spc-3
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
related_adrs: [adr-35]
---

# Know What a Repo Can and Cannot Tell You — Before You Trust the Lifeboat

## Press Release

> **abcd ships `/abcd:disembark probe <repo>` — point it at any repository and get back an honest account of what can be rescued from it.** The probe reads a repo without touching it and reports, section by section, what a lifeboat could actually ground: which parts of the project's theory are written down somewhere, which are only half there, and which are simply absent. Every grounded claim cites the file it came from. Every blank carries what abcd searched and **the question a human has to answer instead**. Run it across several repos and `/abcd:disembark coverage` puts them in one table — so you can see, in a number, what keeping a record is worth.
>
> "I inherited a repository and three sentences of context," said Iris, product lead. "The probe gave me a list of exactly what the repo could not tell me — and a question for each one. That list *is* my first week. I stopped guessing what I didn't know."
>
> "I mine dead projects for a living, and the danger is always the confident summary that turns out to be invented," said Jack, consultant. "This one refuses to fill a section it can't cite. When it says blank, it means blank — and it shows me where it looked."

## Why This Matters

The lifeboat is what abcd's press release says abcd *is* — the surface that makes a project survivable. It is also the only major surface with **zero code**, and it has **no owning intent**: itd-7, itd-8, itd-9, itd-11, itd-16, itd-23 and itd-35 are all satellites orbiting an intent that was never written. This is it.

It matters *now*, and as an experiment rather than a feature, because **the brief's structure is an assumption nobody has tested**. The brief asserts that a project's theory decomposes into these sections. Build the packer first and you have assumed the answer — you will get a lifeboat shaped like the brief whether or not the brief is a shape reality fits.

So invert the build. **Probe before pack.** Run the same probe over a repo with a rich record and a repo with nothing but git, and the delta in section coverage is **what the record is worth**. That number is the point. If it is large, abcd's premise — that writing things down is the thing worth doing — survives contact with reality. If half the brief comes back permanently blank on every repository, **the structure is wrong, and we have learned that for the cost of one milestone instead of a whole phase.**

The honesty discipline is the other half. A rescue tool that invents a plausible section is worse than one that leaves it empty, because the reader cannot tell the difference. A blank is therefore a **first-class result**, not a failure: it names what is missing, what was searched for it, and the question a human must answer.

## What's In Scope

- **`abcd disembark probe <repo>`** — read-only, out-of-tree. Reports per-section coverage: `grounded` / `partial` / `blank`, with confidence, the tier it was grounded from, and the evidence cited.
- **Tiered source adapters that degrade**: Tier 0 git (every repo), Tier 1 conventions (`README`, `docs/`, `CHANGELOG`, `LICENSE`, `CONTRIBUTING`, ADRs wherever they live), Tier 2 abcd-native (`.abcd/`).
- **`abcd disembark coverage <coverage.json>...`** — the cross-repo aggregate: one table, section × repo. **This table is the artefact that answers whether the brief structure is sound.**
- **A stable, aggregatable coverage schema** — `schema_version`, per-section status, evidence, what was searched, and the question a blank raises.
- **The graveyard as a first-class section** — archaeology (Tier 0), recorded abandonment (Tier 1/2), then interpretation that **must cite the evidence beneath it or be dropped by the validator**.
- The packer (`disembark <repo> to <dest>`), embark, and the round-trip — built *after* the aggregate settles the section list.

## What's Out of Scope

- **Greenfield scaffolding** — starting a new project with abcd conventions and no lifeboat to embark from is **itd-21**'s (`/abcd:init-project scaffold`). This intent never scaffolds; it only reads existing repositories.
- **Mining chat transcripts for unrecorded rationale (Pass B)** — there is no corpus. It ships as a **declared exemption** in `_provenance.json`, never a silent gap, until the transcript store has data.
- The hostile-lifeboat threat battery, the Merkle chain (itd-16), `--with-code` (itd-8), schema migrators beyond the version stamp (itd-9), RepoPrompt portability (itd-7), and backgrounded execution — all deliberately deferred.
- Writing anything into the source repository, ever.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a repository with no abcd record at all (git and nothing else), **when** the user runs `abcd disembark probe <repo>`, **then** a coverage report is produced naming **every** brief section it cannot fill, each blank carrying **what was searched** and **the question a human must answer** — and the run never fails merely because the repo is poor.
- **Given** any repository, **when** `probe` runs against it, **then** the source tree is **byte-identical afterwards** — a test hashes it before and after — and abcd writes nothing inside it.
- **Given** a section reported `grounded`, **when** the report is read, **then** every claim in it **cites the source file it came from**; a claim with no citation is a defect, not a stylistic lapse.
- **Given** coverage reports from several repositories of mixed record quality, **when** the user runs `abcd disembark coverage <coverage.json>...`, **then** one table renders section × repo, and the **delta between a rich-record repo and a git-only repo is legible as a number**.
- **Given** the same repository probed twice with no changes between runs, **when** the two reports are compared, **then** they are identical — the probe is deterministic, so a delta in the aggregate means a delta in the repos, never in the tool.
- **Given** a repository whose richest tier is Tier 0, **when** it is probed, **then** the `graveyard` section is still `grounded` — what a project abandoned is recoverable from git history alone (reverts, branches abandoned unmerged, files deleted after substantial history, dependencies added then removed).
- **Given** an interpreted graveyard entry that cites no layer-1 or layer-2 evidence id, **when** the Go validator runs, **then** the entry is **dropped** — cite-or-be-dropped is enforced by code, not by the model's good intentions.
- **Given** the packer exists (a later milestone), **when** `dry-run` and `to <dest>` are compared, **then** they report the same planned writes — one code path, so **`dry-run` cannot lie about what `to` will do**, and a test asserts exactly that.

## Prior Art

- **[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)** ratifies this intent's shape: probe before pack, read-only and out-of-tree, coverage as a first-class output, the graveyard as a new section, `voyage/` at the operator level (superseding adr-4).
- **Satellites of this intent, all pre-existing**: itd-11 (Pass B pitfall mitigation — its low-confidence-quarantine mechanism becomes the graveyard's cite-or-be-dropped validator), itd-35 (lifeboat integrity audit), itd-16 (hash-chain/Merkle audit), itd-8 (`--with-code`), itd-9 (schema migrators), itd-7 (RepoPrompt workspace portability), itd-23 (Spec Kit interop). Each assumed an owning intent that did not exist.
- **itd-21 (`no-lifeboat-scaffolding`)** was read first to check for overlap: it owns the **greenfield** path (scaffold a *new* project with no lifeboat to embark from). It does not overlap — this intent only ever reads repositories that already exist. The boundary is recorded in What's Out of Scope.
- **Naur (1985), *Programming as Theory Building*** — the recovery-humility frame the lifeboat already cites: the theory lives in the people, and an artefact is a proxy for it. This intent takes that seriously enough to *measure* the proxy's coverage rather than assert it.
- **`repolinter` and Conftest** are the closest external analogues already adopted in this repo (by itd-85's audit engine): a rule set producing a per-rule verdict over a repository. Coverage differs in that a failing rule there is a defect, whereas a blank here is **information** — the question a human must answer.

## Open Questions

- Which brief sections survive the cross-repo aggregate? `product/personas` is predicted blank below abcd-native and only partial there; if that holds across the corpus, the section is not derivable from a repository at all and should not be in a lifeboat's brief. **This is the question the experiment exists to answer, and it is deliberately open.**
- How many repositories does the aggregate need before the delta is trustworthy rather than anecdotal?
- Does `partial` earn its place as a status, or does it become a bucket where every hard call goes to die?

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4d07032fc6ab -->
Fidelity review — receipt rcp-4d07032fc6ab (verifier abcd:intent-fidelity-reviewer claude-fable-5).

Provenance: abcd:intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:bda482993615f6ee00d06f9649bff9c9bc8f22c989683386437eea4db28369b2 · prompt_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86
Input attestations: diff:docs/itd-88-coverage-experiment @ 7b24b98befbd20440b335a96d0154bd33cbcec6e: internal/core/lifeboat/ (probe/coverage/plan/pack/graveyard), internal/surface/cli/cli.go disembark wiring, commands/abcd/disembark.md, .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md@-; rubric:.abcd/.work.local/reviews/rcp-4d07032fc6ab.request.md@sha256:bda482993615f6ee00d06f9649bff9c9bc8f22c989683386437eea4db28369b2;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Live probe of a git-only fixture (git init, one revert, nothing else) exits 0 and renders all 23 sections with every blank carrying its searched list and a human question; completeness and question-on-every-blank are test-enforced.
  evidence: cmd: go run ./cmd/abcd disembark probe <git-only fixture> (exit 0) — "grounded 1 · partial 4 · blank 18  (of 23 sections) … searched: glossary, naming registry, reserved-vocabulary tables / ? Nothing probed grounds constraints/naming; a human must supply it."
  evidence: internal/core/lifeboat/probe_test.go:353 — "func TestProbeNeverBlankSectionCarriesAQuestion"
  evidence: internal/core/lifeboat/probe_test.go:322 — "func TestProbeReportsEverySection"
- ac-2 — MET: TestProbeLeavesEveryFileByteIdentical sha256-hashes every file before and after a probe and fails on any rewrite/create/remove; it and TestProbeNeverMutatesTheSource ran green (go test -run …, PASS), and all reads go through a contained read-only os.Root.
  evidence: internal/core/lifeboat/probe_test.go:254 — "func TestProbeLeavesEveryFileByteIdentical … t.Errorf(\"probe rewrote %s (sha256 %s -> %s)\""
  evidence: cmd: go test -run 'TestProbeLeavesEveryFileByteIdentical|TestProbeNeverMutatesTheSource' ./internal/core/lifeboat/ — "--- PASS: TestProbeLeavesEveryFileByteIdentical / --- PASS: TestProbeNeverMutatesTheSource"
  evidence: internal/core/lifeboat/probe.go:102 — "Probe must be side-effect-free and must never write to the source repository"
- ac-3 — MET: The anti-fiction rule is test-enforced (every non-blank row must cite evidence) and a live JSON sweep over abcd-cli, cobra, and requests found zero grounded/partial rows without a citation.
  evidence: internal/core/lifeboat/grounding_test.go:105-108 — "if s.Status != StatusBlank && len(s.Evidence) == 0 { t.Errorf(\"section %s is %s but cites no evidence\""
  evidence: cmd: disembark probe --json over abcd-cli/cobra/requests, checked for evidence-less non-blank rows — "non-blank rows missing evidence: NONE (all three repos)"
- ac-4 — MET: Live `disembark coverage self.json cobra.json requests.json` renders one 23-section × 3-repo status table, the per-repo probe summaries put the delta in numbers (grounded 21 vs 4 vs 3), and the research note states it as 17–18 grounded sections.
  evidence: cmd: go run ./cmd/abcd disembark coverage <3 reports> (exit 0) — "brief section  abcd-cli  cobra  requests  verdict … 0 of 23 sections are blank in every probed repo."
  evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:39-42 — "Keeping an abcd-native record is worth roughly **17–18 grounded sections**"
  evidence: internal/core/lifeboat/coverage.go:203 — "func Aggregate(covs []Coverage) AggregateReport"
- ac-5 — MET: TestProbeIsDeterministic requires byte-identical JSON across two probes and ran green, and a live double probe of this repository produced byte-identical output under cmp.
  evidence: internal/core/lifeboat/probe_test.go:385 — "func TestProbeIsDeterministic … if string(ja) != string(jb)"
  evidence: cmd: disembark probe . --json twice, cmp — "DETERMINISTIC: byte-identical"
- ac-6 — MET_WITH_CONCERNS: A live git-only fixture grounds graveyard at (git, high) with only the git tier present, and TestProbeGraveyardFromGitAlone enforces it — but grounded status requires a revert: deletions alone yield partial (this repository itself scored partial), and the parenthetical's unmerged-branches and removed-dependencies signals live in the packer's archaeology layer, not the probe's grounding decision.
  evidence: cmd: disembark probe <git-only fixture> — "tiers present: git … + graveyard grounded (git, high) / evidence: 1 files deleted (e.g. f.txt), 1 reverted commits"
  evidence: internal/core/lifeboat/grounding_test.go:115 — "func TestProbeGraveyardFromGitAlone"
  evidence: internal/core/lifeboat/sources_git.go:81-89 — "deletions without any revert are only partial evidence, not a grounded graveyard"
  evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:118-120 — "only `partial (git, medium)` on this repository (40 deletions, no reverts detected)"
- ac-7 — MET: IngestLessons drops any lesson whose evidence refs resolve to no layer-1/2 finding id ("no valid evidence refs") and TestIngestLessonsCiteOrDropped asserts the uncited lesson is dropped while the cited one is written; the test ran green.
  evidence: internal/core/lifeboat/graveyard_lessons.go:116 — "drop(\"no valid evidence refs\")"
  evidence: internal/core/lifeboat/graveyard_lessons_test.go:96 — "func TestIngestLessonsCiteOrDropped … Evidence: []string{\"no-such-id\"} … res.Dropped != 1"
  evidence: cmd: go test -run TestIngestLessonsCiteOrDropped ./internal/core/lifeboat/ — "--- PASS: TestIngestLessonsCiteOrDropped"
- ac-8 — MET_WITH_CONCERNS: One code path holds — Pack calls the same Plan the dry-run renders (pack.go:84) — and parity is test-enforced as a hash chain (dry-run manifest hash = ManifestSHA256(planned files) = provenance hash = independent re-hash of the written tree), all green; the caveats are that the shipped verbs are `plan`/`pack <repo> <dest>` rather than the promised `dry-run`/`to <dest>`, and the parity assertion spans TestPlanManifestReportsHashAndTotals plus TestPackProvenanceHashVerifies rather than one direct plan-vs-written-files comparison.
  evidence: internal/core/lifeboat/pack.go:84 — "lb, err := Plan(repoAbs)"
  evidence: internal/core/lifeboat/pack_test.go:96 — "func TestPackProvenanceHashVerifies … does not verify against the written tree"
  evidence: internal/core/lifeboat/plan_test.go:741-743 — "if m.ManifestSHA256 != ManifestSHA256(lb.Files) { t.Error(\"manifest hash disagrees with the file set\")"
  evidence: internal/surface/cli/cli.go:387-388 — "Use: \"plan [repo]\" … without writing anything (dry run)"

Gap audit:
- honoured:
  - Probe before pack, read-only and out-of-tree — reads a repo without touching it
    evidence: internal/core/lifeboat/probe_test.go:254 — "TestProbeLeavesEveryFileByteIdentical"
    evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:24-25 — "Both were byte-identical after probing"
  - A blank is a first-class result: what was searched and the question a human must answer
    evidence: internal/core/lifeboat/coverage.go:112-121 — "searched: … / ? %s"
    evidence: cmd: disembark probe <git-only fixture> — "every one of 18 blanks carries searched + question"
  - The cross-repo table answers what keeping a record is worth, in a number
    evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:39-42 — "grounds **21 of 23** … the two git-plus-conventions repositories ground **4** and **3**"
  - Stable, aggregatable coverage schema with schema_version, enforced at the aggregate
    evidence: internal/core/lifeboat/coverage.go:16 — "SchemaVersion int `json:\"schema_version\"`"
    evidence: internal/surface/cli/cli.go:369-375 — "missing schema_version … upgrade abcd"
  - Graveyard as a first-class section with a code-enforced cite-or-be-dropped validator
    evidence: internal/core/lifeboat/graveyard_lessons.go:116 — "drop(\"no valid evidence refs\")"
  - Packer built after the aggregate settled the section list
    evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:77-78 — "the packer is built to all 23 brief sections"
- diverged:
  - Packer surface promised as `disembark <repo> to <dest>` with `dry-run`; shipped as `disembark plan [repo]` and `disembark pack <repo> <dest>`
    evidence: .abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41 — "The packer (`disembark <repo> to <dest>`)"
    evidence: internal/surface/cli/cli.go:413 — "Use: \"pack <repo> <dest>\""
  - Graveyard grounding is narrower than the four-signal promise: only reverts ground; deletions alone are partial; unmerged branches and removed dependencies feed the pack-path archaeology, not the probe's grounding
    evidence: internal/core/lifeboat/sources_git.go:81-89 — "deletions without any revert are only partial evidence"
    evidence: internal/core/lifeboat/graveyard_archaeology_test.go:146 — "TestArchUnmergedBranchesOrderedByDivergence"
  - Press release headlines `/abcd:disembark probe` and `/abcd:disembark coverage`; the markdown command surface documents only plan and pack — probe/coverage ship as CLI verbs only
    evidence: commands/abcd/disembark.md:4 — "argument-hint: \"<source-repo> <dest> | plan <source-repo>\""
    evidence: internal/surface/cli/cli.go:306 — "Use: \"probe [repo]\""
  - The recorded delta compares record-rich against git+conventions repos, not a strictly git-only repo; the corpus is two foreign repositories
    evidence: .abcd/development/research/notes/2026-07-26-itd-88-coverage-experiment.md:134-137 — "suggestive but not yet a trustworthy population — the finding here is a first reading"
- missing:
  - Pass B ships as a declared exemption in `_provenance.json`, never a silent gap — no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct
    evidence: internal/core/lifeboat/plan.go:65-80 — "type Provenance struct { SchemaVersion … Omissions } — no exemption field; grep 'exemption' across internal/core/lifeboat/ returns nothing"
