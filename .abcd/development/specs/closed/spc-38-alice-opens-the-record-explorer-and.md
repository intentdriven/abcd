---
id: spc-38
slug: alice-opens-the-record-explorer-and
intent: itd-136
---
# alice-opens-the-record-explorer-and-reads-the-whole-developm

## Summary

spc-38 delivers itd-136's record explorer: one `record.json` export of the
record graph, and the pages rendered from it — the dashboard, a page per
record with its full body, contributors, a foundations page, and a
references page rendered by a small in-repo CSL formatter. On the itd-140
boundary every page here is **generic-side**: input is the record format
(`.abcd/development/**` frontmatter and lifecycle directories, plus the
opted-in issue ledger), git history, and `CHANGELOG.md` dated headings —
nothing else — and a missing optional source omits the dependent page and
its navigation entry while the build succeeds. No prose anywhere describes
the generator as repo-agnostic (itd-140 rule 3).

## Settled constraints

- **`record.json` is a pure build artifact, never committed** (interview
  ruling): determinism is asserted by a double-build diff in CI, and
  production cannot drift from the tree because it is rendered from the tag
  by the released binary (adr-48).
- **Full bodies at `/record/<type>/<id>/`** (interview ruling); the issue
  ledger is published because this repo's `.abcd/site.json` opts in
  (adr-32 tiering; itd-140 rule 4).
- **Verbatim record rendering at `/record/**` is exempt from the
  banned-token gate** (adr-47 decision 3); the contributors page prints
  model names under the declared attribution escape.
- **The baseline is a ratchet** seeded with the eight dangling references
  the in-repo re-count finds (adr-22→adr-14/adr-15/adr-17, adr-25→adr-8,
  adr-27→adr-16, adr-28→adr-18, adr-35→adr-4, itd-3→spc-1) — fixing one
  shrinks it, growing it fails the build.
- **One canonical primitive**: the graph comes from the record-lint
  engine's existing typed-reference scan, exported — never a third parser.

## Mechanism

### `record.json` (in `abcd site build`)

- A new exported loader on the lint engine's record scan (the
  `record_schema` walker already parses every store, block-sequence
  spellings included) yields nodes and typed references; `internal/core/site`
  consumes it. Landed as a separate behaviour-preserving refactor commit
  before the feature commit.
- Nodes: id, type, lifecycle from directory, title, path, dates; edges:
  typed links with mirrored pairs collapsed (intent `spec_id` ↔ spec
  `implements`; `related` recorded in both files) so each distinct link
  renders once; body mentions deduplicated against typed edges.
- Dates from one `git log --reverse --name-status` pass through the
  repo's isolated-git helpers: created, entered-current-directory, last
  touched, for every record file; frontmatter `date` wins where a record
  carries one.
- Releases from `CHANGELOG.md` dated headings via the changelog package's
  own matcher; contributors from `git shortlog` folded through `.mailmap`;
  assistance tallies from `Assisted-by:` trailers; counts by type and
  lifecycle; the two precomputed arrangements (spc-39's layouts) ride in
  the same file; `health` carries the unresolved references measured
  against the baseline.
- `schema_version: 1`; sorted inputs, fixed seeds, no timestamps beyond
  the build stamp; a golden-file test over an in-process fixture repo pins
  the export, and CI builds twice and diffs.

### Pages

- **Dashboard `/record/`**: stat tiles (releases, decisions, intents,
  specs, issues, principles), lifecycle bars, release cadence, latest
  decisions, record health — counts, dates, ids and titles only. A visual
  whose labels and numbers already render as text (a bar with its legend, a
  chart with labelled ticks) carries no table twin; a twin accompanies only
  a visual that is the sole carrier of its numbers. The cadence chart
  labels every tick with version and date, and below 700px a plain list
  stands in for the strip — exactly one of the two renders at any width.
- **Per-record pages**: frontmatter, the body rendered verbatim through the
  Markdown renderer, inbound and outbound typed links, an open-on-GitHub
  link. Retired ADR ids render as dashed baseline stubs — no tombstone
  files (interview ruling).
- **Contributors `/contributors/`**: authors of record from `git shortlog`
  through `.mailmap`; a separate labelled bots-and-tools row driven by an
  author-pattern rule (bot-suffixed authors and the mailmap-canonicalised
  pre-policy commit); the `Assisted-by:` share and per-model tallies
  presented as disclosure with `CONTRIBUTING.md` linked.
- **Foundations**: cards for `principles/` and `intents/disciplines/`
  entries that list and link, never explain; if neither directory exists
  the page and its navigation entry are omitted.
- **References `/references/`**: rendered from
  `.abcd/development/research/references.csl.json` by a small in-repo CSL
  formatter (stdlib-only — the renderer that satisfies the no-Node,
  no-committed-HTML gate), numbered identically to `ACKNOWLEDGEMENTS.md`
  (a check asserts the numbering agrees), DOIs linked, the attribution
  line the style requires; absent a compatible renderer the page and nav
  entry are omitted — never a broken page.
- Graceful absence throughout: a missing bibliography, Identity block or
  CHANGELOG omits the dependent page and nav entry and the build succeeds.

### Checks (in `abcd site check`)

- Unresolved typed cross-references outside `.abcd/site-baseline.json`
  fail the build naming the reference; the baseline only shrinks.
- The 390 px static checks run over every explorer route.

## Acceptance-criteria mapping

- AC 1 (counts derived at build into an uncommitted `record.json`;
  double-build diff; visuals carry their numbers as text) → export +
  determinism gate + dashboard.
- AC 2 (every record a page with frontmatter, body, typed links, GitHub
  link) → per-record pages.
- AC 3 (baseline ratchet) → baseline check, seeded with the eight.
- AC 4 (mirrored references collapse) → edge deduplication.
- AC 5 (contributors: mailmap authors, bots row, disclosure framing,
  model names confined under the escape) → contributors page.
- AC 6 (foundations lists-and-links or is omitted) → foundations page.
- AC 7 (references numbered like ACKNOWLEDGEMENTS or omitted) → CSL
  formatter + numbering check.
- AC 8 (390 px, no horizontal scroll) → static checks + CI screenshots.

## Out of scope

- The relationship chart and genealogy behaviours (spc-39; their layouts
  are computed here only as `record.json` fields).
- Any second-instance fixture demonstration (itd-139, drafts).
- Search across the record (plan phase 4).
