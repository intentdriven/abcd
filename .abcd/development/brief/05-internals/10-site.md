# The Website — a Rendered Surface of the Record

> **One passage below is a design target; the rest describes what the binary
> and the workflows do.** `abcd site build` renders the whole site, `abcd
> lint site` gates it, and the deploy workflow rides the release chain — but
> abcdev.app still serves the MkDocs rendering of `docs/` at its root, and
> the first production deploy from a tag is what moves it. Both halves rest on
> [adr-47](../../decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md)
> and [adr-48](../../decisions/adrs/0048-website-deploys-on-release-not-on-merge.md),
> with the [investigation cluster](../../research/abcdev-site/plan.md) and the
> composition rules' executable spec beside them. A shipping change removes
> the mark it lands.

**abcdev.app is a surface of this repository and of nothing else.** `/` is a
landing page for product thinkers, `/docs/` the MkDocs rendering of `docs/`
(SSG-agnostic; replaceable by a later ADR), and `/record/…`,
`/contributors/`, `/references/` a record explorer — every page rendered at
build time from repository text, structured data the repository already
maintains, and committed assets, under adr-47's single-source rule: no text
is written for the website, and the build fails on a text node it cannot
source. The record is **never bundled, rendered read-only** — the site is
the third publication surface adr-47 adds to adr-30's two trees, and the
adr-28 launch boundary is unchanged.

**Plumbing** — per this directory's rule, plumbing lives here and not in
intents:

- **`abcd site build`** walks `.abcd/site.json`, composes the landing page
  from repository text, and emits `record.json`: the record graph with each
  mirrored typed link collapsed once, body mentions deduplicated against it,
  counts by store, lifecycle and status, releases, authorship, the unresolved
  references measured against `.abcd/site-baseline.json`, and the two
  precomputed chart arrangements. It
  inlines committed SVG assets, copies rasters verbatim, and writes into
  `site/` and nowhere else. The graph comes from the record-lint engine's own
  scan (`lint.LoadRecordGraph`) and the dates from one
  `git log --reverse --name-status` pass, so the record has one parser and
  history one read; the frontmatter-free principle store joins the graph from a
  directory read, because there is no frontmatter for the scan to see. The
  render is deterministic — sorted inputs, a fixed
  layout seed, coordinates published at drawing precision, no clock read
  beyond the injected build stamp — which is what lets `record.json` be an
  artifact nobody commits. Transport-agnostic core, front doors per adr-23;
  the composition rules' executable spec is `compose.py`/`build_data.py`
  under [`research/abcdev-site/`](../../research/abcdev-site/), ported rather
  than reinvented. Raster optimisation is a ledger-recorded dependency
  decision; the build never draws.
- **The explorer's pages** are rendered from that one export: `/record/` (stat
  tiles, a state bar per store, release cadence, latest decisions, record
  health, each visual with a table twin), `/record/<type>/<id>/` (frontmatter,
  the body verbatim, typed links phrased from that record's own side, and the
  forge links to the file and to its commit history), `/record/graph/` (the
  chart's stage and its list twin, driven by `site-src/record.js`, reading
  `?focus=<id>`), `/record/timeline/` (the five-lane genealogy as one static SVG
  emitted in Go), `/record/foundations/` (principles and disciplines as cards
  that list and link), `/contributors/` and `/references/`. The bibliography is
  rendered by a stdlib CSL-JSON formatter and numbered identically to
  `ACKNOWLEDGEMENTS.md`, with a build check that the two agree entry for entry.
  A reference whose target has left the tree renders as a dashed stub — on the
  record page, in record health and on the genealogy — never as a dead link and
  never as an arc to an invented position. A store that carries no frontmatter
  is marked `derived` in the export, so no page presents a file name or a git
  date as a field the record declared and no chart reads a lifecycle it never
  had. Every file the build serves matches a `site-src/headers` block — a
  document carrying a content policy, `nosniff` and a referrer policy, an asset
  carrying the two of those that govern a non-document — asserted over the whole
  emitted tree by a build test rather than by review. The Markdown subset carries what the
  record actually writes: nested lists, blockquotes with structure inside them,
  reference links, setext headings, rules, autolinks and CommonMark emphasis;
  anything still outside it is a build failure naming file and line.
- **`abcd lint site`** runs seven independent gates over a rendered tree —
  the provenance audit over every rendered text node, the hero against the
  Identity block, docs-lint's banned tokens over composed text (the verbatim
  record rendering under `/record/` exempt, the attribution escape a
  verification against the trailers git carries), CLI-snippet drift against
  the generated reference, the `.abcd/site-baseline.json` ratchet, the
  static mobile checks over every page, and the loop-figure labels — and
  reports every failure, not the first. The rendered-overflow screenshot
  audit is CI's optional, non-gating job; static and rendered gates are
  complementary, and the audit's first run caught an overflow the static
  gate cannot see.
- **The generic/specific boundary** of the verb family is governed by the
  itd-140 discipline: repo-agnostic input contract, genericity demonstrated
  on a sparse second instance before it is claimed, working-tier ledger
  publication opt-in only. This repository opts in through
  `.abcd/site.json`'s `record.issue_ledger`.
- **The README migration** — README's product narrative lives in
  `docs/explanation/{rationale,roles,artefacts,process}.md` and
  `docs/how-to/install.md`, and README is a contributor page keeping the
  universal install one-liner, with a test that the one-liner, the
  `install.sh` script and install.md's per-OS forms agree.

The user-facing capabilities ride on this plumbing as intents: itd-135 (the
landing page, umbrella), itd-136 (the record explorer pages), itd-137 (the
relationship chart and genealogy), itd-138 (`install.sh`), itd-139 (the
generic explorer demonstrated on a second instance — the gate adr-47 decision 6
puts in front of any repo-agnostic claim, and the reason none is made here).
Deploy cadence and
trigger are adr-48's as amended: production is a reusable workflow invoked
from the release chain as a separate, non-gating job after the release job
(a `release:` event never fires for releases the chain's own token creates),
checked out by resolved commit, rendered by the released, checksum-verified
and attestation-verified binary in a credential-free job, and deployed from
the attested artifact by a job that runs no build code; every push to main
deploys a source-built preview stamped `unreleased · <commit>`; emergencies
redeploy by `workflow_dispatch` from a tag, never from main. The build order
is fixed: `abcd site build` writes `site/` first and `mkdocs build` renders
into `site/docs/` second — the site build rewrites its output tree, so
reversing the order loses the docs render to the purge.
