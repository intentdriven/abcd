---
name: site
description: "Report what the website declares and what was built: Writes nothing; refuses any argument."
argument-hint: "[build|setup]"
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

## `setup` — take a managed repository's site to a live address

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site setup --json
```

sets up the site of a repository abcd manages, in three stages, and emits
`{ "status": …, "files": […], "environments": […], "host": {…}, "remaining": […], "notes": […] }`:

- `files` — the repository half, each `written`, `current`, `kept` or
  `refused`: `.abcd/site.json` (derived from the identity block and
  `docs/README.md`), the static inputs under `site-src/`, the workflow
  `.github/workflows/site.yml` (render on each published release with abcd's
  verified binary, deploy from the rendered archive) and `wrangler.jsonc`. The
  composition and the static inputs are the repository's own once they exist
  and are `kept`; a workflow or host configuration that differs from what setup
  writes is `refused`, the whole run writes nothing, and `--confirm` replaces it.
- `environments` — the forge's `site-render` and `site` deployment
  environments, each admitting only the default branch and tags `v*`, created
  through `gh` as you. An existing environment is never rewritten (the forge's
  write would replace its required reviewers): one on named rules and no rule
  beyond those two gains the rules it lacks, and one that admits more, through
  its protection mode or a rule of its own such as branch `*`, is
  `unrestricted`, with restricting it listed in `remaining`.
- `host` — with a hosting credential stored on this machine, the host is
  created and the domain routed to it, and `address` is the live address.
  Without one, `status` is `no_credential` and nothing is contacted.
- `remaining` — the exact steps left for you: committing the written files,
  storing the credential, and one `gh secret set … --env site` command for
  each deploy secret the environment does not hold yet. abcd never reads or
  sets a secret's value.

Both remote stages ask before they write, naming each change. An unanswered
run declines them and exits `1`; `--yes` confirms in advance — pass it only
when the user has asked for the forge and host changes. A second run over an
unchanged repository reports `no_change` and writes nothing.

The first run names the host after the repository; `--name` and `--domain`
choose the host name and the custom domain, and are recorded in the
composition's `hosting` block. The composition's `pages` block switches pages of
the closed set (`landing`, `explorer`, `record_pages`, `graph`, `timeline`,
`glossary`, `status`) off; it cannot add one.

Report each stage's outcome, then the remaining steps in order. Never paste a
credential into the conversation: the store is the file `~/.abcd/credentials.json`
(mode `0600`), which the user writes themselves.

## The gate over what was rendered

Whether a build may be published is `/abcd:lint site` — run `abcd lint site
--out site`. It runs the site's gates over the built output directory and names
every finding at its source.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
