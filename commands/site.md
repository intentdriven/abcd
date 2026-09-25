---
name: site
description: Render this repository's website — the landing page composed from repository text under the single-source rule, and the record export derived from the record, git history and the changelog — by invoking the abcd binary. The bare form performs zero writes; build and check write only inside the output directory (check renders the site first when the directory has no index.html).
argument-hint: "[build|check]"
block: agents
---

# `/abcd:site` the website as a surface of the record

A project's website drifts from the project. Copy is written for the site,
lives only there, and slowly stops being true; the record that says what the
project actually decided stays invisible in frontmatter nobody reads. This
command renders a site that cannot drift, because it contains no words of its
own: every sentence is a span of a file in this repository, selected by path and
heading through `.abcd/site.json`.

## Bare — what this repo has declared

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site --json
```

emits `{ "manifest": …, "ui_strings": …, "baseline": …, "out_dir": … }`:

- `manifest` — whether `.abcd/site.json` is present. Absent means this repo
  declares no composition, and there is nothing to render.
- `chapters` — how many chapters the manifest composes the landing page from.
- `issue_ledger` — whether the working-tier issue ledger is published. It is an
  explicit per-repo opt-in; the default renders the durable record only.
- `ui_strings` — whether the interface-string allowlist the manifest names is
  present. It is the complete list of words the generator may add.
- `baseline` and `baseline_entries` — the committed unresolved-reference
  ratchet and its size.
- `version`, `commit` — what a render would stamp the footer with.
- `out_dir`, `out_exists`, `out_files` — where a render writes, and what is
  there now.

Report the declared inputs first, then the output directory's state. It writes
nothing and exits `0` whatever it finds.

## `build` — render the site

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site build --out site
```

reads exactly this set — `.abcd/site.json`, `site-src/ui.json`,
`.abcd/record-lint.json` (where the record stores are), the record under
`.abcd/development/` and the opted-in issue ledger, git history,
`CHANGELOG.md`, the composed pages and assets under `docs/`, the static inputs
`site-src/{site.css,site.js,record.js,redirects,headers}` and the served
`site-src/install.sh.tmpl`, the credit sources `CONTRIBUTING.md` and
`ACKNOWLEDGEMENTS.md` (and the existence of `SECURITY.md` and `CITATION.cff`
for the footer), `.abcd/site-baseline.json` (the ratchet the health block
counts against), and `.claude-plugin/plugin.json` (the forge URL, licence and
author the links and footer use) — and writes the landing page, the record
export, the redirect and header maps, the stylesheet, the two scripts, the
`install.sh`, and every referenced raster into the output directory, and
nowhere else. It reaches no network. The default output directory is `site`,
which the repository does not track.

The last two are declared deviations from the generic input contract: a repo
without a package manifest renders without the forge links rather than failing,
and the baseline is per-repo site configuration on the same opt-in footing as
`.abcd/site.json` itself — the record data proper stays record-format, git and
`CHANGELOG.md`.

Four flags exist so a build can pin what the footer says rather than reading it
from the working tree: `--version`, `--commit`, `--date` and `--preview` (stamp
the build as unreleased at this commit, for a preview deployment of an untagged
tree). `--preview` and `--version` are mutually exclusive: a preview build is
stamped unreleased, so pinning a version contradicts it. Left unset, the version
and date come from the newest dated `CHANGELOG.md` heading and the commit from
git `HEAD`.

Report the files written, then the five measurements the render prints: the
page count rendered from the record, the
record's size (records, links, mentions), the unresolved references against the
committed baseline, the chart packing's overlap count (which is zero or the
picture is wrong), and the version and commit stamped into the footer. An
unresolved-reference count above the baseline is worth naming to the maintainer
even though this verb does not gate on it.

A failure names its cause and its place: a markdown construct outside the
rendered subset is reported as `file:line`, and so is an image the page names
that the repository does not carry. Neither is a rendering bug to work around —
the fix is an edit to the page.

## `check` — say whether what was rendered may be published

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site check --out site
```

reads the built output directory and the repository it came from, and runs seven
independent gates. Each reports EVERY failure it finds rather than the first, so
one run is one review round. An output directory with no `index.html` is
rendered first: a caller who has not built yet is asking the same question as
one who has. It writes nothing except that render, and reaches no network.

- `provenance` — every visible word on a composed surface sits inside an element
  whose `data-src` names a repository span that RESOLVES (the file exists, and
  the heading anchor exists in it), or is an interface string, a number, a date,
  a file name or an asset name. The `<title>` and `<meta name="description">`
  carry Identity text with no attribute to name it, and are checked against the
  Identity block rather than skipped; a page that names itself reads
  `<page> · <project>`, and both halves are held up.
- `hero` — the rendered hero's eyebrow, tagline and pitch equal the Identity
  block, read through the same parser the positioning surfaces use.
- `banned-tokens` — the documentation lint's banned tokens, over the text every
  composed surface publishes, whichever tree the span came from. The escape is
  read source-side: a token is exempt where the source line it was selected from
  declares it legitimate. Credit is the one place naming a tool is the sanctioned
  use, so a span selected from the acknowledgement file, and the attribution
  page's own authorship data, are exempt from the naming bans — never from
  provenance, which still matches every name against the history that carries it.
- `snippets` — every `abcd …` command the site shows names a command the
  generated CLI reference documents, with flags that reference documents for it.
- `baseline` — an unresolved cross-reference outside the committed ratchet
  fails; a ratchet entry whose reference now resolves is reported as shrinkable
  and fails nothing. Growing is refused, shrinking is invited.
- `mobile` — over every page this build writes, the record rendering included
  (the `/docs/` tree is the documentation generator's own output and is dropped
  before any gate walks it, so no gate here examines it): the viewport
  meta, an overflow container above every table and command block, a max-width
  rule for images in the linked stylesheet (resolved from the served root, which
  is where a root-absolute href points), no picture wider than the content
  column, and no inline fixed width above 390 px. The rendered-overflow audit
  needs a browser and runs in CI.
- `figure-labels` — every text label in a lifted diagram is a phrase on the page
  it illustrates, where the manifest asks for it.

Report the gates that passed, then every finding under the gate that raised it,
each with the file and the `data-src` span a reader's next edit goes to. Exit is
`0` when nothing failed, `1` when something did, and `2` when the check could
not run at all (no composition manifest, an unreadable input). A shrinkable
baseline entry is printed as a note and does not change the exit code.

The fix for a finding is always at its SOURCE — the page, the record, the
manifest or the stylesheet — never in the generated output, which the next build
overwrites.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
