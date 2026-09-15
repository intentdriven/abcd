# Meta

This file holds **stable conventions** for the brief itself: directory structure, file-naming rules, and the brief↔lifeboat shape contract. The brief reflects the project's *current* state at all times — not versioned, not archived in this directory; see [adr-5](../decisions/adrs/0005-brief-is-current-state.md). History lives in `git log`; inflection-point rationale lives in [`../decisions/adrs/`](../decisions/adrs).

## The truth rule

Everything in the brief is decided. A section may describe machinery that is
**shipped** or machinery that is **staged** — a design target a future change
builds toward — but never an undecided option: the brief records no proposals,
no alternatives still under consideration, no aspirational maybes. The only
variance a reader must resolve is shipped vs staged, and a staged claim says so
explicitly. Unsettled questions are recorded *as
questions*, never as claims — registered in
[`03-evidence/03-open-questions.md`](03-evidence/03-open-questions.md), or
carried inline as an explicit `Open question (adr-35)` note where one bears
directly on a passage (DECISIONS.md, 2026-07-14). This follows from
[adr-5](../decisions/adrs/0005-brief-is-current-state.md): the brief is the
project's current state, and that state includes what the project has decided
to build but not yet built.

## Directory structure

The brief is split across numbered folders rather than a single `README.md`. Reasons:

1. **Concurrent editing** — multiple agents can work on different sections without serialising on one file.
2. **Diff legibility** — `git log brief/04-surfaces/02-disembark.md` tracks the evolution of one command's design, not a whole-brief blob.
3. **Agent context budget** — agents that need only one section can pull just that file (relevant to the [`05-internals/03-configuration.md`](05-internals/03-configuration.md) `maxAgentTokens` budget).
4. **Reusable shape** — the same numbered-folder layout serves as a template for future projects (the lifeboat output shape mirrors this skeleton, see [`04-surfaces/02-disembark.md § 5`](04-surfaces/02-disembark.md#5-output-shape)).

## Naming convention

- **Numeric prefixes** use two-digit zero-padding (`01-`, `02-`, …, `10-`, …) so sort order survives past 9 entries.
- **Hyphens** (not underscores) separate words, matching the kebab-case norm elsewhere in the project (`spc-1-add-oauth`, `itd-N-<slug>`, `adr-N-<slug>`).
- **`README.md`** (the index) deliberately has *no* numeric prefix because it's the entry point, not a section. GitHub renders it automatically when browsing the folder.

## Brief vs lifeboat

The brief skeleton is, deliberately, the same shape as a populated lifeboat (see [`04-surfaces/02-disembark.md § 5`](04-surfaces/02-disembark.md#5-output-shape)). This is a design contract:

- **`/abcd:ahoy`** scaffolds a managed repo's config (`.abcd/config.json` + `.abcd/rules.json` — see [`04-surfaces/01-ahoy.md`](04-surfaces/01-ahoy.md)), never this skeleton. The **design target** (staged, not shipped) for a fresh repo is the **all-blank coverage state**: every section of this skeleton present as a blank to be filled deliberately, measured by the same three-valued coverage accounting a lifeboat reports (`grounded` / `partial` / `blank`) — not a copied file tree.
- **`/abcd:disembark`** uses this skeleton's *shape* as a target schema for what lifeboat-composer agents must produce.
- **`/abcd:embark`** reads a lifeboat (which is structurally a populated brief + audit/provenance extras) and writes a new repo's brief by walking the brief↔lifeboat mapping in reverse — the amended press release becomes the new repo's initial brief.

There is one canonical skeleton, used three ways. The mapping table below is that contract; round-trip tests catch divergence.

## The brief ↔ lifeboat mapping

This table is **generated from `internal/core/lifeboat/mapping.go`**, which is its single source of truth. A test (`TestBriefCarriesTheRenderedMappingTable`) asserts the two agree, so the document cannot drift from the code — edit the Go table, not this block.

It is a **hypothesis, not a measurement.** Each row states the best status a lifeboat could reach for one brief section given the source material a repository actually has. `abcd disembark probe` measures the same sections against real repositories and reports the same three-valued status (`grounded` / `partial` / `blank`), so prediction and evidence are directly comparable — and where they disagree, this table is what loses.

**Tiers are cumulative**: a repository with conventions still has git, so the status quoted at a tier is what is achievable using that tier *and every poorer one together*. A richer tier can therefore never ground a section worse than a poorer one.

- **Tier 0 — git**: commit history, authors, branches, reverts, file lifespans, tags, dependency churn. Present in **every** repository.
- **Tier 1 — conventions**: `README`, `docs/`, `CHANGELOG`, `LICENSE`, `CONTRIBUTING`, issue exports, and ADRs wherever they live. Present in most.
- **Tier 2 — abcd-native**: the `.abcd/` record — decisions, intents, specs, brief, roadmap, issues, reviews, memory. Present only where abcd manages the repo.

<!-- BEGIN GENERATED: brief-lifeboat-mapping -->
| Brief section | Lifeboat path | Tier 0 git | Tier 1 conventions | Tier 2 abcd-native | Reads |
|---|---|---|---|---|---|
| `product/press-release` | `brief/01-product/01-press-release.md` | blank | partial | grounded | README lede, docs/, shipped intents' press releases |
| `product/context` | `brief/01-product/02-context.md` | partial | grounded | grounded | README, docs/, CONTRIBUTING, commit subjects |
| `product/mental-model` | `brief/01-product/03-mental-model.md` | blank | partial | grounded | docs/ explanation pages, ADR context sections, the brief |
| `product/scope` | `brief/01-product/04-scope.md` | partial | partial | grounded | README features, intents' in-scope/out-of-scope sections, the code's own surface |
| `product/personas` | `brief/01-product/05-personas.md` | blank | blank | partial | personas registry, press-release quote attributions |
| `product/framing` | `brief/01-product/06-framing.md` | blank | partial | grounded | the brief's framing section, the brief-creation interview's committed framing products |
| `constraints/platform` | `brief/02-constraints/01-platform.md` | partial | grounded | grounded | build manifests, CI workflows, README requirements |
| `constraints/dependencies` | `brief/02-constraints/02-dependencies.md` | partial | grounded | grounded | add/remove churn from git history; the authoritative list needs the manifest and lockfile |
| `constraints/invariants` | `brief/02-constraints/03-invariants.md` | blank | partial | grounded | CONTRIBUTING, agent-conventions router, lint configs, ADR consequences |
| `constraints/naming` | `brief/02-constraints/04-naming.md` | blank | partial | grounded | glossary, naming registry, reserved-vocabulary tables |
| `evidence/what-worked` | `brief/03-evidence/01-what-worked.md` | partial | partial | grounded | CHANGELOG, code that survived, reviews and retrospectives |
| `evidence/what-didnt` | `brief/03-evidence/02-what-didnt.md` | partial | partial | grounded | the graveyard beneath it — reverts, dead branches, superseded records |
| `evidence/open-questions` | `brief/03-evidence/03-open-questions.md` | blank | partial | grounded | TODO and FIXME markers, open issues, intents' open-questions sections |
| `evidence/tradeoffs` | `brief/03-evidence/04-tradeoffs.md` | blank | partial | grounded | ADR alternatives-considered sections wherever ADRs live |
| `surfaces` | `brief/04-surfaces/` | partial | grounded | grounded | CLI entrypoints and help text, README usage, per-surface design files |
| `internals` | `brief/05-internals/` | partial | partial | grounded | package layout, architecture docs, the internals chapters |
| `delivery/build-sequence` | `brief/06-delivery/01-build-sequence.md` | partial | partial | grounded | tags and release history, CHANGELOG, roadmap phases |
| `delivery/verification-matrix` | `brief/06-delivery/02-verification-matrix.md` | partial | partial | grounded | test files and what they target, CI workflow steps; which check covers which surface is authored judgement |
| `delivery/out-of-scope` | `brief/06-delivery/03-out-of-scope.md` | blank | partial | grounded | README non-goals, intents' out-of-scope sections |
| `glossary` | `brief/glossary/` | blank | partial | grounded | docs glossary pages, the bounded-context glossary |
| `graveyard` | `graveyard/` | grounded | grounded | grounded | reverted commits, branches abandoned unmerged, files deleted after substantial history, dependencies added then removed; then superseded records |
| `rescue/spine` | `rescue/` | partial | partial | grounded | commit history as a spine where no record exists; the intent corpus where one does |
| `docs/adrs` | `docs/adrs/` | blank | grounded | grounded | ADRs wherever they live, copied verbatim |
| `activity/issues` | `activity/issues/` | blank | partial | grounded | issue exports, the capture ledger |
<!-- END GENERATED: brief-lifeboat-mapping -->

Two rows carry most of the thesis:

- **`graveyard` is the only section grounded at Tier 0.** What a project abandoned is written into its git history whether or not anyone wrote it down — which is why the graveyard earns a section of its own rather than a footnote inside `evidence/what-didnt`.
- **`product/personas` is blank below abcd-native, and only partial there.** If the probe confirms that across the corpus, personas are not derivable from a repository at all: the section is a question for a human, not an extraction, and it should not be in a lifeboat's brief.

`coverage.{json,md}` and `_provenance.json` are not rows here. They are the lifeboat's own machinery — the report of what could *not* be filled, and the record of how it was produced — and they are always written, whatever the tiers.

## No archive directory

The brief does not maintain an `archive/<NN>/` directory of prior iterations. Per [adr-5](../decisions/adrs/0005-brief-is-current-state.md), the live `brief/` directory IS the brief; `git log` covers history; ADRs cover inflection-point rationale. Disembark snapshots (per [adr-35](../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)) provide the audit-traceable history chain when a forensic snapshot is needed.
