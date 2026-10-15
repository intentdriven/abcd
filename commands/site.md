---
name: site
description: "Report what the website declares and what was built: Writes nothing; refuses any argument."
argument-hint: "[build|setup]"
block: agents
user-invocable: false
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
- `project` — whether `.abcd/site.json` holds a project site's settings
  (`"mode": "project"`) rather than a composition: then there are no chapters
  and no `ui_strings`, and the site builds with `--project` (below).
- `chapters` — how many chapters the manifest composes the landing page from.
- `issue_ledger` — whether the working-tier issue ledger is published. It is an
  explicit per-repo opt-in; the default renders the durable record only.
- `ui_strings` — whether the interface-string allowlist the manifest names is
  present. It is the complete list of words the generator may add.
- `baseline` and `baseline_entries` — the committed unresolved-reference
  ratchet and its size.
- `version`, `commit` — what a render would stamp the footer with.
- `out_dir`, `out_exists`, `out_files` — where a render writes (relative to the
  repository inside it, with the home directory as `~` outside it), and what is
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
`site-src/install.sh.tmpl`, the credit sources `.github/CONTRIBUTING.md` and
`ACKNOWLEDGEMENTS.md` (and the existence of `SECURITY.md` — in `.github/`, at
the root or in `docs/` — and `CITATION.cff` for the footer),
`.abcd/site-baseline.json` (the ratchet the health block counts against), and
`.claude-plugin/plugin.json` (the forge URL, licence and author the links and
footer use; without a forge URL there, the origin remote's github.com address)
— and writes the landing page, the record
export, the redirect and header maps, the stylesheet, the two scripts, the
`install.sh`, and every referenced raster into the output directory. The one
write outside it is `site-src/ui.json` itself, and only when the file lacks a
label abcd declares (a file written before that label existed): the build adds
each such label with abcd's default words, prints one stderr line per label
(`abcd site build: added the missing label status.target to site-src/ui.json
with its default words`), before the error when the build then fails, lists
them in `added_labels`, and changes nothing
else in the file. A label the file carries keeps its wording, a blank one is
still refused by name, and a key abcd does not declare is still refused. The
render `abcd lint site` makes of an empty output directory never completes the
file, so the gate refuses an incomplete one by name. It reaches no network. The default output directory is `site`,
which the repository does not track.

The last two are declared deviations from the generic input contract: a repo
without a package manifest renders without the forge links rather than failing,
and the baseline is per-repo site configuration on the same opt-in footing as
`.abcd/site.json` itself — the record data proper stays record-format, git and
`CHANGELOG.md`.

A site served below the root of its host takes `--base-path`: a project's
GitHub Pages address is `owner.github.io/<repo>/`, so its build passes
`--base-path /<repo>/` and every root-absolute link the build writes (the
stylesheet, the scripts, the pictures, the navigation and every record route)
is placed under that path. Give `abcd lint site` the same `--base-path`, so it
resolves what it reads against the same root. The default, `/`, writes exactly
the links a site served at the root of its domain needs.

`--project` builds a project site instead, from a `.abcd/site.json` that holds
the site's settings only (`"mode": "project"`, the `hosting` block and its
switch; `abcd site setup --provider github-pages` writes it, below). Nothing is
composed and nothing is read from `site-src/`: every page is built from what
abcd already writes to the repository, and the interface words, the
stylesheet, the scripts and the headers policy come from the binary. The about
page at `/` is, in this order and each only where the repository holds it: the
identity block `.abcd/positioning.json` points at (as the heading), the
brief's opening (the first section with text in
`.abcd/development/brief/01-product/01-press-release.md`), the press releases of
the three newest shipped intents whose audit was met (newest by id: a
timestamped id carries when abcd minted it, and an ordinal one such as `itd-7`
sorts before every timestamped one), and the release list in
`CHANGELOG.md`. Anything absent is left out, never filled; with no identity
block, the page is headed and titled by the repository's name on GitHub. With
neither an identity block nor a GitHub origin, the site has no title, and the
gate refuses it until one of the two is recorded. A
repository holding none of the four parts is refused before anything is
written, since the site would be empty. The repository's address and name
come from the origin remote alone: `.claude-plugin/plugin.json` is not read, so
no name, author or licence written there reaches a page. The record pages are
built as they always are, from the stores `.abcd/record-lint.json` names or,
where the repository carries no such file, from abcd's standard layout under
`.abcd/development/` (intents, specs, ADRs and principles), and the issue
ledger under `.abcd/work/issues`, so an intent's typed link to an issue
resolves; an issue is published only where the settings opt in to it, with
`"record": {"issue_ledger": true}` in `.abcd/site.json`. The
references page (which quotes `ACKNOWLEDGEMENTS.md`) and every picture (which
lives under `docs/`) are left out, and so is a hand-written `docs/` tree or
`README.md`: a project's own documentation is read on GitHub. A project site
also shows nothing read from git history, and the build does not read it: no
contributors page, no timeline, none of the health page's authorship checks
(the same-author pairs, the commits declaring nothing or more than one model,
and the share disclosing assistance), and no `authorship`, `history` or
per-record `dates` in `record.json`. A record is dated by the `date` in its
own frontmatter alone, and one without is undated.
A build whose `--project` and settings file disagree is refused, both ways.

Every page opens with one line saying it is the record of an abcd-managed
repository, linking to that repository. Its words are the `managed_record`
interface string: on a composed site, the one in `site-src/ui.json`, declared
there like every other word the generator adds; on a project site, the one in
the interface words the binary carries, since a project site reads no
interface words from the repository.

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
unresolved-reference count above the baseline is worth naming to the technical
facilitator even though this verb does not gate on it.

A failure names its cause and its place: a markdown construct outside the
rendered subset is reported as `file:line`, and so is an image the page names
that the repository does not carry. Neither is a rendering bug to work around —
the fix is an edit to the page.

## `setup` — take a managed repository's site to a live address

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site setup --json
```

sets up the site of a repository abcd manages, in three stages, and emits
`{ "status": …, "files": […], "environments": […], "host": {…}, "remaining": […], "notes": […] }`
(with `added_labels` and `labels_file` when a label was added):

- `files` — the repository half, each `written`, `current`, `kept`, `removed`
  (a switched-off project site's workflow, below) or `refused`: `.abcd/site.json` (derived from the identity block and
  `docs/README.md`), the static inputs under `site-src/`, the workflow
  `.github/workflows/site.yml` (render on each published release with abcd's
  verified binary, deploy from the rendered archive) and `wrangler.jsonc`. The
  composition and the static inputs are the repository's own once they exist
  and are `kept`, with one exception: a `site-src/ui.json` lacking a label abcd
  declares gains it with abcd's default words and is `written`, each added
  label named on stderr and in `added_labels`, and nothing it already says is
  rewritten; a workflow or host configuration that differs from what setup
  writes is `refused`, the whole run writes nothing, and `--confirm` replaces it.
- `environments` — the forge's `site-render` and `site` deployment
  environments, each admitting only the default branch and tags `v*`, created
  through `gh` as you. When `gh` is missing, setup explains it and offers to
  install it, running the step only on a yes typed at a terminal; under
  `--yes` or a piped answer the offer is declined, the environments are not
  created, and `notes` carries the command to run. The default branch is the one the forge names; when the
  forge cannot answer, the checkout's branch stands in and `notes` says so. An existing environment is never rewritten (the forge's
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
when the technical facilitator has asked for the forge and host changes, in a
question headed with a `Tech` chip. A second run over an
unchanged repository reports `no_change` and writes nothing.

The first run names the host after the repository; `--name` and `--domain`
choose the host name and the custom domain, and are recorded in the
composition's `hosting` block. The composition's `pages` block switches pages of
the closed set (`landing`, `explorer`, `record_pages`, `graph`, `timeline`,
`glossary`, `status`) off; it cannot add one.

### A project's own GitHub Pages

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" site setup --provider github-pages --json
```

publishes the project's about page and development record on its own GitHub
Pages at each release, at `https://<owner>.github.io/<repo>/` (or at the root of
a custom domain given with `--domain`). It is the same verb with a second
provider, and these things differ:

- **Only what abcd writes.** A project site renders nothing written for the
  website: its pages are built from the identity block, the brief, the
  development record and `CHANGELOG.md` (see `build --project` above), and the
  project's own docs are read on GitHub. A repository holding none of the
  about page's four parts yet is refused before anything is asked or written,
  since the site would be empty.
- **A release of abcd that builds project sites.** The workflow downloads
  abcd's latest release and builds with it, so its build fails until abcd
  publishes a release whose `site build` takes `--project`. Setup does not ask
  the network which release is latest, so it says this in a note on every run
  for a project site.
- **The switch.** Before anything is written, setup asks whether to switch the
  project site on, naming the address. A yes writes `.abcd/site.json` holding
  the project site's settings (`"mode": "project"` and the `hosting` block with
  `"publish": true`, and no composition) and adds `.github/workflows/site.yml`,
  which builds and gates with `--project`; nothing else in the repository
  changes, and nothing is written under `site-src/`. A no, or an unanswered
  run, writes nothing and changes nothing remote, and the status is
  `declined`. Settings already written with the switch on, but no workflow,
  are asked the switch again before the workflow is added. A composition that
  already exists with no `hosting` block is not rewritten: setup refuses and
  names the block to add, and once it is added the next run still asks the
  switch before it adds the workflow (that repository keeps its composed
  site).
- **An existing Pages site.** Setup reads the repository's Pages site before it
  asks or writes. One the repository already publishes, built by a workflow, is
  named by its address and build type in the switch question and in the host
  outcome. `--yes` never answers it: a run under `--yes` refuses before
  anything is asked or written, naming the site, unless `--replace-existing` is
  passed too, which publishes over it without asking. Pass
  `--replace-existing` only after a question naming the live site's address
  was answered yes. A site published from a branch, or whose build type GitHub
  does not report, one already served at a custom domain when `--domain` names
  another or none (the refusal says to declare that domain: `--domain` on the
  run that writes the `hosting` block, its `domain` field once the block
  exists), and one GitHub serves below another path than the workflow builds
  for are refused before anything is written.
- **HTTPS on a custom domain.** GitHub serves a new custom domain over http
  until it has issued the domain's certificate, which can take up to a day
  after the domain resolves. The address is still reported as `https://…`, and
  the host outcome's `detail` says HTTPS is not enforced yet.
- **Public repositories only.** Setup reads whether the repository is public
  from the forge first, and refuses a private one (or one whose visibility it
  cannot read) in one line, before it asks or writes anything. The workflow
  checks again at each release, so a repository made private later publishes
  nothing.
- **No secret and no credential.** The deploy job runs in the `github-pages`
  environment with the run's own short-lived token (`contents: read`,
  `pages: write`, `id-token: write`) and nothing else; there is no host
  configuration file and no `gh secret set` step. The host stage creates the
  Pages site (built by the workflow), sets the custom domain only when one was
  given, and reports the address, acting through `gh` as you, the way the forge
  stage does. Environments are `site-render` and `github-pages`.
- **The base path.** The workflow builds and gates the site with
  `--base-path /<repo>/`, the path GitHub serves it below (the root for a custom
  domain or an `<owner>.github.io` repository).
- **Switching it off.** Set `"publish"` to `false` in the `hosting` block of
  `.abcd/site.json`. The workflow reads the switch both at the released commit and
  at the head of the default branch, so the next release publishes nothing, and
  neither does a later push or dispatch that renders an older tag. A later `abcd site setup` touches nothing
  remote and offers to remove the workflow it added, `removed` on a yes; a
  workflow edited by hand, or whose removal was declined, is `kept`, with the
  reason in its `detail`. Pages already published stay up until they
  are unpublished in the repository's settings.

Report each stage's outcome, then the remaining steps in order. Never paste a
credential into the conversation: the user stores it themselves with `abcd ahoy
credential hosting.cloudflare` (see `/abcd:ahoy`), which explains where it can
live and verifies it before keeping it; `host.credential` names the credential,
never its value.

## The gate over what was rendered

Whether a build may be published is `/abcd:lint site` — run `abcd lint site
--out site` (with `--project` for a project site). It runs the site's gates
over the built output directory and names every finding at its source.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
