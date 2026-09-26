---
name: lint
description: "Check this repository against the conventions, every target included: Writes nothing; refuses with exit 2 on an error finding and exit 1 on warnings alone."
argument-hint: "[docs | outbound | site | identity]"
block: people
---

# `/abcd:lint` repo-conformance check

Run the abcd binary's read-only conformance lint for the current repo and
present the result. Bare `lint` and every target but one perform **zero
writes**; `lint site` renders the site into its `--out` directory (default
`./site`, under the working directory) when that directory holds no
`index.html`, and leaves the render there. It reports gaps, it never fixes
them (remediation stays with `/abcd:prepare-this-repo`).

`lint` is the one check, with targets. Bare, it runs every target that judges
the repository: the working conventions, the docs (`docs-currency`), the identity
(`identity-positioning`), the outbound policy over every committed file
(`privacy-hygiene`) and, in a repository that declares a site, the site's gates
(`site-gates`, rendered into a temporary directory outside the repository and
removed).
Each target also runs on its own, with its own report: `lint docs`, `lint
outbound`, `lint site` and `lint identity`, below. When the user names a target,
run that one.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint --json
```

Then summarise the JSON for the user. Its shape is `{ "findings": [ … ],
"skipped": [ … ] }`:

- `findings` — each has a stable `ruleId`, a `severity` (`error` or `warn`), a
  `file` (omitted where a `site-gates` finding names no source span), a `line` on
  content-scanning findings only (`docs-currency`, `privacy-hygiene`,
  `identity-positioning` — path-presence findings omit the key entirely), a `message`, and a `fix`. Group them by severity: report
  `error` findings first (these fail conformance), then `warn` findings
  (advisory). For each, give the `file` (with `:line` when present), the
  `message`, and the `fix`.
- `skipped` — rule ids that did not apply to this repo (e.g. `docs-currency`
  when there is no `docs/`). Mention them as "not applicable", not as failures.

State the outcome plainly: if there are no findings the repo conforms; otherwise
lead with how many errors and warnings there are. The process exit code is the
Conftest tri-state — `0` clean, `1` warnings only, `2` any error — so
`abcd lint` can also gate a repo's CI.

## `lint outbound` — judge one piece of outbound text

The sub-verb judges a single artefact rather than the repo: a commit message, a
pull-request body, an issue, a comment, a release note. It applies abcd's
outbound policy — never a live agent-session URL, never a tool's own attribution
footer — and it **reports and refuses; it never rewrites the text**, because the
text belongs to whoever wrote it.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint outbound --label pr-body ./body.md
```

It reads the file named as the positional, or standard input when there is none
(or when it is `-`). `--label` names the artefact in the report; `--root` picks
the repo whose `.abcd/config/pii.json` configures the scan (default: the current
directory).

The exit code is the verdict, and it is **not** the parent verb's Conftest
tri-state: `0` the artefact is clean, `1` the artefact is refused, `2` the check
could not run (an unreadable or empty artefact, a degraded scanner config). Both
patterns are hard-fail, so the tri-state's advisory middle rung has no meaning
here; a caller that branches on non-zero is right either way, and one that
distinguishes must not read "the gate was broken" as a verdict on the text.

With `--json` it emits one document: `label`, `findings` (each with a `kind` of
`harness:session_url` or `harness:attribution_footer`, a `line`, a `column` and a
`suggested_fix`), and the `policy` text. The matched span is masked in both
renderings — a CI log on a public repository is public text, so the gate must not
republish the leak it is reporting. A refusal arrives as the exit status alone;
there is no second error envelope on top of the report.

This is the check abcd's own CI runs over every commit message in a pull
request's range and over the pull-request body
(`scripts/check-attribution.sh`).

## `lint docs` — the docs-currency gate (zero writes, zero network)

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint docs --json
```

Then summarise the JSON for the user:

- `nothing_checked` and `warning` — when `nothing_checked` is `true` the lint
  checked nothing and still exited 0: relay `warning` (it says why), tell the
  user nothing was checked, never that the docs are clean, and point them at
  `.abcd/docs-lint.json`. The same warning is printed on stderr.
- `checks` — how many checks the configuration armed (banned tokens plus enabled
  rules). `0` means no rule ran.
- `documents` — how many markdown documents the configured roots hold for the
  per-document rules. `0` means those rules read nothing: the roots are empty or
  hold no markdown.
- `blockers` — how many blocker findings exist; any blocker fails the gate.
- `findings` — for each, its `File`, `Line`, `RuleID`, `Severity`, and
  `Message`; group them so the user sees what to fix.

The lint enforces present-tense docs: unambiguous change-narration (`previously`,
`formerly`, `renamed from`, `has been replaced`, `we switched`, `to be
implemented`) blocks, while phrases that also describe present state
(`deprecated`, `no longer`, `migrated from`) warn advisorily rather than block.
It also checks that relative links resolve and that no stray markdown sits at the
repo root (it belongs under `docs/`). Point the user at the offending file and
line for each finding, and note whether it is a blocker or a warning.

Where a repo arms them, the citation rules add: footnote markers and definitions
in bijection, every crosswalk table row carrying a footnote, well-formed URLs and
DOIs, refused source domains, and the committed baseline — no cited URL without a
receipt, none recorded broken, none whose recorded final address has drifted from
what the page cites, and a staleness warning past 180 days. Every one of these
reads committed files only; nothing dials out.

`--release-gate` runs the same lint with one difference: a citation past the
365-day threshold blocks instead of warning. It is for release machinery only —
an ordinary commit is never blocked by the calendar.

If `nothing_checked` is `false` and `blockers` is zero, the docs are
currency-clean.

## `lint site` — say whether what the site build rendered may be published

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint site --out site
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

## `lint identity` — what this repo says about itself

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint identity --json
```

emits `{ "block": …, "severity": …, "surfaces": [ … ] }`:

- `block` — the canonical `title`, `tagline`, and optional `pitch`, plus the
  `file` and `line` they are recorded at.
- `severity` — the family's weight, `warn` (the default) or `blocker`.
- `surfaces` — one entry per registered surface: its `id`, the `file` and `line`
  checked, and a `status` of `ok`, `drifted`, `absent` (no such file in this
  repo — not a fault), or `unlocatable` (the file is there but the locator
  matches nothing, so drift would go unseen). A `drifted` entry carries `found`
  (the exact text the surface says), `missing` (which block fields it no longer
  carries), and `canonical` (what the block says it should).

Report the block first, then any surface that is not `ok`, naming the file, the
line, what it says, and what the block says. It exits `0` even when it reports
drift: this form is a status render, and the gate is bare `abcd lint`. The
proposed correction and the recording of the block are `/abcd:identity`.

A `privacy-hygiene` finding on a deliberately illustrative line can be waived by
adding `abcd-lint:allow` on that line (the earlier `abcd-audit:allow` spelling is
honoured too). No other rule honours that marker: a `docs-currency` finding takes
the docs-lint engine's own `<!-- docs-lint: allow -->` escape, and the remaining
rules have no line waiver — resolve what they report.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
