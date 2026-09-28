---
id: itd-136
slug: alice-opens-the-record-explorer-and-reads-the-whole-developm
spec_id: spc-38
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-135]
severity: minor
impact: additive
---

# Alice opens the record explorer and reads the whole development record — every decision, intent, spec and issue as a page, with contributors and references — without reading YAML on GitHub

## Press Release

> **Alice opens the record explorer and reads the whole development record
> without reading YAML on GitHub.** Alice evaluates developer tooling for
> their team, and abcd's strongest evidence — thirty-eight ratified
> decisions, a hundred and thirty press-release intents, a structured issue
> ledger, an audited trail of what shipped — was invisible unless they were
> willing to read frontmatter in a code browser. Now `/record/` opens on a
> dashboard: stat tiles for releases, decisions, intents, specs, issues and
> principles, lifecycle bars, the release cadence, the latest decisions, and
> record health as a committed baseline that can only shrink. Every record in
> the tree is a page of its own — frontmatter, the body rendered from its
> Markdown, its inbound and outbound typed links, and the link out to GitHub.
> `/contributors/` names the humans as authors of record and presents the
> `Assisted-by:` trailer tallies as what they are — disclosure, not
> authorship — next to the policy that requires them. `/references/` renders
> the bibliography the research directory already keeps, numbered exactly as
> ACKNOWLEDGEMENTS.md numbers it. All of it is rendered at build time from
> one export of the record — no API calls, no hand-written summaries. "I
> stopped taking the project's word for it," said Alice. "The record showed
> me what was decided, when, and what it cost — from the same files the tool
> itself works from."

## Why This Matters

The record is abcd's product argument: intent-driven development leaves an
inspectable trail. A rendered explorer makes that argument self-demonstrating
— the binary rendering its own record is the strongest possible proof that
the record is machine-readable — and it arms a second detector: the build
fails on a cross-reference the tree cannot resolve, which turns record drift
from an invisible debt into a visible gate (the ratchet baseline is seeded
with the dangling references known today and can only shrink). The
contributors page makes the attribution convention legible to outsiders: the
human is the author of record; the trailer is disclosure.

## Acceptance Criteria

- Given `/record/`, then the dashboard's counts are derived at build time
  from the tree into `record.json` — a pure build artifact, never
  committed: determinism is asserted by a double-build diff in CI, and the
  published data cannot drift from the tree because production is rendered
  from the tag by the released binary (adr-48) — and every visual has a
  table twin for assistive tech.
- Given any record in the tree (`adr-N`, `itd-N`, `spc-N`, `iss-N`, a
  principle), then `/record/<type>/<id>/` renders its frontmatter, its body,
  its inbound and outbound typed links, and an open-on-GitHub link.
- Given a typed cross-reference whose target is not in the tree and not in
  the committed baseline `.abcd/site-baseline.json`, when the site builds,
  then the build fails naming the reference; given one in the baseline, then
  fixing it shrinks the baseline and a build that grows the baseline fails.
- Given the spec link recorded from both ends (intent `spec_id` ↔ spec
  `implements`) and `related` pairs listed in both files, then the build
  collapses mirrored references so each distinct link renders once.
- Given `/contributors/`, then authors of record come from `git shortlog`
  through `.mailmap`, bot and tool authors sit on a separate labelled row,
  the `Assisted-by:` share and per-model tallies are presented as disclosure
  with `CONTRIBUTING.md` linked, and model names are confined to this page
  under the sanctioned attribution escape.
- Given the repo declares principles (`principles/`) or active disciplines
  (`intents/disciplines/`), then a foundations page lists each as a card
  linking its record page — it lists and links, never explains (context
  belongs in `docs/` and is selected from there); given neither directory
  exists, then the page and its navigation entry are omitted.
- Given `/references/` ships, then the bibliography renders from
  `.abcd/development/research/references.csl.json`, numbered identically to
  `ACKNOWLEDGEMENTS.md`, with DOIs linked and the attribution line the CSL
  style requires; given no renderer compatible with adr-47's no-Node and
  no-committed-HTML constraints exists at build time, then the page and its
  navigation entry are omitted — never a broken or half-rendered page.
- Given a 390 px viewport, then every explorer route renders with no
  horizontal scroll, verified by the static checks in `abcd site check`
  plus the screenshot audit in CI.

## Open Questions

- Whether `/record/<type>/<id>/` pages render full bodies or summaries plus
  GitHub links — full bodies are the plan, and they make the working-tier
  issue ledger's wording more visible than the tree already is on GitHub.
- Whether retired ADR ids get tombstone files or stay baseline entries
  rendered as dashed stubs (the plan's §7; the genealogy in itd-137 hangs on
  the same answer).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-5a9275d115bd -->
Fidelity review — receipt rcp-5a9275d115bd (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:d5f0a5c6380f27b2d6b50190f4ccebb2223dab1a47f54a5182e1c05f40642de5
Input attestations: diff:tree at 7c476185 (main lineage, itd-136 shipped)@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the build writes record.json into the output directory, which .gitignore excludes, TestBuildRecordExportShape pins its shape and TestDashboardVisualsCarryTheirNumbersAsText proves every visual carries its numbers as text; determinism is asserted by TestBuildIsDeterministic, which builds twice and diffs in the Go test lane CI runs, rather than by a distinct double-build step in a workflow — the assertion exists, its home differs from the criterion's letter
  evidence: internal/core/site/build.go:518 — "if err := write("record.json", recordJSON); err != nil {"
  evidence: .gitignore:68 — "/site/"
  evidence: internal/core/site/build_test.go:696 — "func TestBuildIsDeterministic(t *testing.T) {"
  evidence: internal/core/site/build_test.go:721 — "func TestBuildRecordExportShape(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:324 — "func TestDashboardVisualsCarryTheirNumbersAsText(t *testing.T) {"
- ac-2 — MET: TestExplorerCoversEveryRecord walks every record in the tree to its /record/< type>/< id>/ page and TestRecordPageRendersItsBodyAndLinks checks frontmatter, rendered body, typed links both ways and the forge links recordpage.go's fileLinks adds; both green at BASE
  evidence: internal/core/site/explorer_test.go:63 — "func TestExplorerCoversEveryRecord(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:104 — "func TestRecordPageRendersItsBodyAndLinks(t *testing.T) {"
  evidence: internal/core/site/recordpage.go:213 — "// fileLinks names the record's file and offers the two views of it the forge"
  evidence: internal/core/site/explorer_test.go:140 — "func TestBlockedByReadsBothWays(t *testing.T) {"
- ac-3 — MET: the baseline check refuses an unresolved reference outside .abcd/site-baseline.json naming it, refuses a grown baseline and invites a shrink when a baselined target arrives; TestCheckRefusesADanglingSpecTarget, TestCheckRefusesAGrownBaseline, TestCheckInvitesAShrinkingBaseline and TestHealthUnresolvedListsTheDanglingReference are green
  evidence: internal/core/site/check.go:20 — "// 5. Baseline ratchet — unresolved references outside the committed baseline"
  evidence: internal/core/site/check.go:150 — "CheckBaseline = "baseline""
  evidence: internal/core/site/check_test.go:674 — "func TestCheckRefusesADanglingSpecTarget(t *testing.T) {"
  evidence: internal/core/site/check_test.go:561 — "func TestCheckRefusesAGrownBaseline(t *testing.T) {"
  evidence: internal/core/site/check_test.go:570 — "func TestCheckInvitesAShrinkingBaseline(t *testing.T) {"
- ac-4 — MET: collapseEdges normalises each typed reference to one direction and drops the mirrored duplicate; TestBuildRecordExportShape asserts the implements pair collapses to one edge
  evidence: internal/core/site/recordjson.go:288 — "// collapseEdges normalises each typed reference to one direction and drops the"
  evidence: internal/core/site/recordjson.go:224 — "edges, typedPairs := collapseEdges(graph.Edges, index)"
  evidence: internal/core/site/build_test.go:778 — "t.Errorf("implements edges: %d, want 1 (the mirrored pair collapses)", implements)"
- ac-5 — MET: contributors.go folds git shortlog through .mailmap, derives the bots-and-tools row from the [bot] suffix and the tool vocabulary, tallies Assisted-by values as disclosure, and the page carries the CONTRIBUTING.md attribution policy and a link to it; TestContributorsSeparatesAuthorshipFromDisclosure and TestContributorsRefuseWithoutTheirPolicy are green
  evidence: internal/core/site/contributors.go:6 — "// `git shortlog` folded through `.mailmap` — the authors of record, humans"
  evidence: internal/core/site/contributors.go:146 — "// Bots are the forge bots and tool-authored commits, kept in a separate row"
  evidence: internal/core/site/contributors.go:126 — "// ModelTally is one distinct `Assisted-by:` value and how often it appears."
  evidence: internal/core/site/explorer_test.go:391 — "func TestContributorsSeparatesAuthorshipFromDisclosure(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:403 — "`data-src="CONTRIBUTING.md#attribution"`,"
- ac-6 — MET: the foundations page lists each principle and discipline as a card linking its record page, and is omitted with its navigation entry when neither directory exists; TestFoundationsListsAndLinks and TestBuildWithoutFoundations are green
  evidence: internal/core/site/explorer_test.go:361 — "func TestFoundationsListsAndLinks(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:941 — "func TestBuildWithoutFoundations(t *testing.T) {"
- ac-7 — MET: csl.go renders the bibliography from the CSL JSON and holds its numbering to ACKNOWLEDGEMENTS.md's numbered list; TestReferencesRenderFromCSL and TestBuildWithoutBibliography (page and navigation omitted) are green
  evidence: internal/core/site/csl.go:11 — "// The load-bearing part is not the formatting, it is the NUMBERING. The record"
  evidence: internal/core/site/csl.go:32 — "const AcknowledgementsRelPath = "ACKNOWLEDGEMENTS.md""
  evidence: internal/core/site/explorer_test.go:780 — "func TestReferencesRenderFromCSL(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:916 — "func TestBuildWithoutBibliography(t *testing.T) {"
- ac-8 — MET: the static mobile gate refuses a missing viewport, an unwrapped table and an inline width over 390 px, and the CI screenshot audit measures overflow at 390 among its widths; TestCheckRefusesAMissingViewport and TestEveryTableScrollsInsideItsOwnBox are green
  evidence: internal/core/site/check.go:1427 — "const maxInlineWidthPx = 390"
  evidence: internal/core/site/check_test.go:817 — "func TestCheckRefusesAMissingViewport(t *testing.T) {"
  evidence: internal/core/site/explorer_test.go:466 — "func TestEveryTableScrollsInsideItsOwnBox(t *testing.T) {"
  evidence: site-src/audit/overflow-audit.js:40 — "const WIDTHS = [360, 390, 768, 1360];"
  evidence: .github/workflows/site-screenshots.yml:162 — "node site-src/audit/overflow-audit.js "$base" "$RUNNER_TEMP/screenshots""

Gap audit:
- honoured:
  - rendered at build time from one export of the record — no API calls, no hand-written summaries
    evidence: internal/core/site/recordjson.go:3 — "// record.json — the whole development record as one machine-readable file."
  - the build fails on a cross-reference the tree cannot resolve, and the baseline can only shrink
    evidence: internal/core/site/check_test.go:561 — "func TestCheckRefusesAGrownBaseline(t *testing.T) {"
  - a retired target in the baseline renders as a stub rather than a broken link
    evidence: internal/core/site/explorer_test.go:267 — "func TestRecordPageRendersARetiredTargetAsAStub(t *testing.T) {"
- diverged:
  - the determinism assertion lives in the Go test lane CI runs, not in a separate double-build workflow step as the criterion's letter says
    evidence: internal/core/site/build_test.go:696 — "func TestBuildIsDeterministic(t *testing.T) {"
    evidence: .github/workflows/ci.yml:392 — "go run ./cmd/abcd site build --out "$RUNNER_TEMP/site-render-check""
  - the criterion names `abcd site check`; the static checks answer to `abcd lint site` after itd-2609212130136102, with `site check` a stub for one release
    evidence: .github/workflows/ci.yml:393 — "go run ./cmd/abcd lint site --out "$RUNNER_TEMP/site-render-check""
- missing: (none)
