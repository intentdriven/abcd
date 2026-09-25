# `/abcd:site` — The Website as a Rendered Surface

A project website drifts from the project. The tagline on the landing page is a
version behind, the install snippet no longer matches the CLI, a statistic was
true last quarter. `/abcd:site` removes the drift by removing the second copy:
every sentence the site publishes is a span of a repository file, selected by path
and heading, and a gate refuses to publish text that is not
([adr-47](../../decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md)).

What that costs a maintainer: a sentence that would improve the site has to be
written into `docs/` or the record, where it must also read true on the forge.
What it buys: the site cannot say anything the repository does not, and nobody has
to remember to update it.

The bare form is **strictly read-only**: it reports what the repository has
declared and what the output directory holds. The build is the render, and it writes
only inside the directory it is given. The check that gates a rendered tree is the site
target of the one lint ([`16-lint.md`](16-lint.md), itd-2609212130136102; for
one release the retired spelling under this verb answers with it and exits
non-zero): it renders first when the directory holds no `index.html` — the one write path
besides the build, confined to the same directory. Bare `abcd lint` runs the
same gates as its `site` rule, over a render in a temporary directory outside
the repository.

It answers a different question from `/abcd:launch`: `launch` prepares what a
release ships to users who install the binary; `site` prepares what a reader sees
who never installs anything. The cadence that connects them is
[adr-48](../../decisions/adrs/0048-website-deploys-on-release-not-on-merge.md)'s:
production renders from the tag, with the released bytes.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `build` | — | shipped |

## The single-source rule

No text is written for the website. The only words the generator may add are the
interface strings in its own allowlist file, plus numbers, dates, file names and
asset names.

Two mechanisms keep that from being a promise nobody can check. Every rendered
block carries a `data-src` attribute naming the file and heading it came from, so
each block names its own source in the markup. And the interface-string file is
decoded against a closed struct with unknown fields refused, so a word added there
which no field reads fails the build rather than reaching a reader unreviewed.

Every picture is a committed asset under `docs/assets/img/`, referenced from a
docs page like any other image. SVGs are inlined so their colours follow the
reader's theme; rasters are copied verbatim. The build never draws.

## Behaviour

Bare, the verb reports what is declared and what the last build left. The build
renders into `./site`, and can stamp the render as an unreleased preview at this
commit; the lint's site target gates the rendered tree and exits 1 on findings.

Both write paths can be pointed at a different directory, and the bare board
reports on whichever directory it is pointed at.

The build stamp is injectable in all three of its parts — version, commit and date
— so a caller that knows better than the defaults can say so. That is what keeps
the render free of a clock read, and what lets a test pin the whole export byte for
byte. The preview stamp is for an untagged tree: it renders the word `unreleased`
with the commit in place of a version, the record export marks itself a preview,
and pinning a version alongside it refuses.

A build into a non-empty directory purges it only when the tree carries the build
marker a previous build of this repository wrote and git tracks nothing in it, and
refuses loudly otherwise. The build cannot remove a directory it did not write, so
a repository that commits its built site is refused and must be pointed at an
untracked output directory, and a tree with no root commit has no identity a marker
could name, so its non-empty output is emptied by hand. An output path with a
symlink at its leaf or at an ancestor inside the checkout, the repository root or a
directory containing it, and any directory holding `.git` are refused before
anything is read; the bare board reports the same refusal instead of counting
through it.

The build reads the repository and nothing else — no network at any point. Its
inputs are the composition declaration and the interface-string allowlist; the
record itself, read through the record-lint engine's own frontmatter scan so there
is one parser rather than two; the bibliography and the glossary through their own
parsers; one pass of git history; `CHANGELOG.md`; the two root prose files whose
text the site publishes, which are the acknowledgements behind the references page
and the authorship section of the contribution guide behind the contributors page;
and `docs/` with its committed assets. It writes the landing page, the record explorer, the machine-readable
record export, the install script from its committed template, the redirect and
header maps, the stylesheets and scripts, every referenced raster, and its own
build marker. Nothing else, nowhere else.

One input reaches past the durable record into the working tier, and it is off
unless a repository asks for it. The composition declaration carries an
issue-ledger switch: turned on, the explorer publishes the issue records
alongside the record families the site always reads, and the bare board reports
in a line of its own whether the ledger is published. Left alone, it is not, so a
repository publishes its working tier only by deciding to.

The explorer includes a **glossary page set** and the term links that reach it: the
first use of a glossary term on a record page is a link to that term's entry, and
only the first. A use inside a code span, a heading, a link already there, or on
the term's own entry page is left exactly as the record wrote it. Both halves are
graceful absences: a repository that keeps no glossary gets no pages, no navigation
entry and no links.

## The gates

The lint's site target runs seven independent gates over a rendered tree and reports
every failure rather than the first: provenance, hero drift against the identity
block, banned tokens over composed text, `abcd …` snippets against the generated
CLI reference, the unresolved-reference ratchet, the static mobile checks, and the
loop-figure labels. The seven are named once in the code that runs them, and the
check's report prints each name as it runs it, passing or failing.

Scope follows adr-47 decision 3 exactly. Composed surfaces are the landing page and
every manifest-selected span; the verbatim record rendering is exempt; and the
attribution escape is a verification, so a name on the contributors page must match
a trailer or contributor git actually carries. The externally-generated docs tree
is dropped from the page walk before any gate sees it, the mobile checks included,
so they say nothing about that subtree: its words are gated at the source by
docs-lint instead. In production the two trees share one output directory, so
"every page" means every page this build wrote.

The rendered-overflow screenshot audit is CI's optional, non-gating job. The static
gates here are what a browserless binary can assert, and the two are complementary
by design.

Two inputs are **declared deviations** from the generic input contract, recorded
here rather than argued away. The plugin manifest is this repository's package
manifest; a repo without one renders without the forge links and the copyright line
rather than failing. The reference baseline is per-repo site configuration that the
health block counts against, which is the same opt-in shape as the composition
declaration itself: the record data proper stays record-format plus git plus the
changelog, and only the health measurement consults a configured ratchet.

The rendered `<title>` and `<meta name="description">` carry identity-block text
with no provenance attribute, because neither element holds visible text a
provenance walk would reach. The check special-cases both.

The render is **deterministic**: sorted inputs, a fixed layout seed, coordinates
published at the precision the chart draws them, and no clock read beyond the build
stamp the caller injects. Two builds of one tree are byte-identical, which is what
lets the record export be a build artefact nobody commits.

Three things are derived rather than decided. The featured quotation is the newest
shipped intent whose audit rollup records met criteria and none unmet, dated by the
day its file entered `shipped/`, with the id descending as the tie-break; an intent
whose press release is still, in its entirety, the minted seed template is skipped,
because the site would otherwise quote the placeholder back at the reader as the
project's own words. The Beta badge renders while the newest dated changelog
version's major is 0. The footer's version and commit are the build stamp.

Graceful absence throughout: no changelog omits the release badge, the release pill
and the releases list, and the build succeeds; no identity block omits the hero's
eyebrow, tagline and pitch, and the headline and lede that carry the page stay.

## The markdown subset

The renderer carries what the record actually writes: ATX and setext headings,
paragraphs, CommonMark emphasis and code spans, links, images, fenced code, pipe
tables, thematic breaks, nested lists, and blockquotes with structure inside them.
Anything else is a build error naming file and line. Passing an unknown construct
through unrendered publishes raw markdown to readers; dropping it publishes a hole.
A build that stops and says which line is the only outcome anybody can act on.

## The record export

The build derives one machine-readable file holding the record graph: nodes with
their lifecycle, title, dates and degree; typed links with each mirrored pair
collapsed once; body mentions deduplicated against those links; counts by store,
lifecycle and status; releases; authorship and assistance tallies; the unresolved
references measured against the committed baseline; the precomputed chart
arrangements; and a summary of the git walk the dates came from. It is a build
artefact and is never committed.

## References

- Plugin command: [`commands/site.md`](../../../../commands/site.md)
- Decisions: [`adr-47`](../../decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md), [`adr-48`](../../decisions/adrs/0048-website-deploys-on-release-not-on-merge.md)
- Internals: [`05-internals/10-site.md`](../05-internals/10-site.md)
- Composition rules: [`research/abcdev-site/`](../../research/abcdev-site/)
- Release surface: [`04-launch.md`](04-launch.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd site`

Sub-verbs: `abcd site build`.

| Flag | Type |
|---|---|
| `--out` | string |

### `abcd site build`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--commit` | string |
| `--date` | string |
| `--out` | string |
| `--preview` | bool |
| `--version` | string |

<!-- surface-appendix:end -->
