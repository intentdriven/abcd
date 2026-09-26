# `/abcd:docs` — Documentation Currency and Citations

Know, before anyone else reads them, whether this repo's docs still describe
the present: no prose narrating a change, no link that has stopped resolving,
no markdown stranded at the repo root, and no cited source that nobody has
checked in six months. The currency lint answers that in one read-only pass, so
it can run on every commit at no cost and gate a release without a network.

The citation sub-tree is the writing half, and it is the only place abcd reaches
the network on behalf of documentation. It runs when a maintainer asks, never
in a gate, which is what keeps the lint itself deterministic and offline.

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
| `cite` | — | shipped |
| `cite confirm` | — | shipped |
| `cite refresh` | — | shipped |


- **The lint** is the docs target of the one lint
  ([`16-lint.md`](16-lint.md), itd-2609212130136102); its contract stays in this
  chapter beside the citation baseline it enforces. It reports the findings and
  writes nothing. The lint's plugin command invokes its JSON form and summarises
  the result. The result carries
  `checks`, the number of banned tokens and enabled rules the configuration
  armed, and `documents`, the number of markdown documents its roots hold for
  the per-document rules to read. A lint that checked nothing (no rule armed,
  or roots that hold no document) says so loudly, with the reason, on the
  diagnostic stream in both renders and as `nothing_checked` and `warning` in
  the JSON, because "0 finding(s)" over a lint that read nothing is a green
  that means nothing (iss-2609150805167646). A configuration that arms no rule
  also prints "nothing was checked" in place of a finding count. The exit code
  is still 0 there (the product thinker's ruling of 2026-09-23 in the decision
  log): the configuration was read, and no rule it declares was broken, so a
  nonzero exit would turn red the CI of every repository prepared before its
  config carried rules. A rule name the lint does not run (a misspelling such
  as `links_reslove`) is refused when the configuration loads, enabled or not,
  so `checks` never counts a rule that checks nothing.
- **The citation refresh** fetches every cited URL once and rewrites the
  committed citation baseline. Each URL gets exactly one bounded attempt with no
  retries, and no response body is read: liveness is judged from the status
  line. Sources that refuse automated fetchers are printed as a manual checklist
  rather than recorded as broken.
- **The citation confirmation** records that a human verified a citation the fetcher
  could not read, either from named URLs or from a receipt file. Today the
  maintainer clears the printed checklist and names the URLs on the command line;
  the receipt form ships against a producer that does not exist yet, a generated
  checklist page that would hand the file back (a later rung of the same intent).
  Both forms write the same dated entry, so when the page arrives it is a second
  producer of one input rather than a second pathway. Only URLs the documentation
  actually cites can be confirmed.

Bare `abcd docs` prints command usage rather than a status board; the
[surfaces index](README.md) carries the one enumeration of where the
bare-status convention holds, and `docs` is not on it. The working verbs accept
the `docs-lint.json` to load and the repo to work over; the bare citation parent
only routes and takes neither. That is what makes the refresh fetch exactly the
set the gate demands receipts for.

**The lint's release-gate mode promotes an overdue citation from a warning to a
blocker, and arming it from a release is a design target.** The flag, not the
committed config, is the trust root: a repo must not be able to defang its own
release by editing `.abcd/docs-lint.json`, and an ordinary commit is never
blocked by the calendar. Nothing in the release machinery passes it. The release
workflow's docs-currency step, CI's, and the `docs-lint` make target each run
the lint in its plain mode, the scaffolded release template names the verb nowhere,
and `launch` computes its own citation preflight rather than shelling out. The
promotion is reachable only by a human typing the flag.

## What it checks

- **Change-narration.** Prose that narrates a change rather than describing
  present state. Unambiguous change-narration blocks; phrases that can also
  describe present state warn advisorily rather than block. Docs are present
  tense: what *is*, never what *was superseded*.
- **Broken relative links.** Every relative link resolves to a file in the tree.
  The rule's own `exempt` globs (repo-relative, `*` staying inside one
  directory) excuse a file from this check alone: a tool-mandated mirror of a
  root file, such as a byte-identical copy of `AGENTS.md` a tool reads from
  `.github/`, carries links that resolve from the root and not from the
  mirror's directory. The configuration's `exempt_paths` does not reach this
  rule, because it excuses how a record is written, never whether its links
  resolve.
- **Broken heading anchors.** A link's `#fragment` names a heading or an
  explicit HTML anchor of the markdown page it resolves to, or of the linking
  page for a bare `#fragment`: the `link_anchors` rule slugs the target's ATX
  headings as the forge renders them (a repeated heading suffixed `-1`, `-2`)
  and reports a fragment that names none. It is its own rule so it lands at
  warning beside the blocking file check, and reads the same `exempt` globs.
- **Stray root markdown.** Markdown at the repo root belongs under `docs/`
  unless it is one of the allowlisted files. A root markdown **symlink** is
  judged by its resolved target's stem rather than by its own name, which is
  what lets the bridge files pass while appearing in no allowlist. The tradeoff
  is known and accepted: creating such a symlink is as deliberate an act as
  adding the allowlisted file. A symlink whose target does not resolve is itself
  a finding.
- **Citations**, where a repo arms the rules: footnote markers and definitions
  in bijection, every crosswalk row carrying a footnote, well-formed URLs and
  DOIs, refused source domains, and the committed baseline at
  `.abcd/citations-baseline.json` (no cited URL without a receipt, none recorded
  broken, none whose recorded final address has drifted from what the page
  cites, and a staleness warning past 180 days). All of it reads committed
  files; the fetching lives in the citation refresh.
- **Host-agnostic prose.** User-facing docs must not name a specific agent
  harness or bundled tool. This repo's config defines a family of `harness/*`
  banned tokens, each a blocker, so the published surface stays host-agnostic;
  the `<!-- docs-lint: allow -->` escape covers the sanctioned exception,
  attribution.
- **Harness leak**, a separate rule from those tokens and armed here as a
  blocker, refusing the two shapes a harness stamps onto text the repository did
  not ask it to stamp: a live agent-session URL, and a tool's own "generated
  with" attribution footer. The class is defined once in the scanner's canonical
  pattern set and consulted from here, so the outbound scrub and this lint
  cannot disagree about what a leak is. Two escapes: a fenced block is quoted
  material a page must be able to show, and a line carrying the
  `abcd-lint:allow` waiver is deliberately illustrative.

A gitignored path under a root is not the repository's documentation, so the
walk prunes it: the lint asks git once per root which untracked paths it
ignores and reads none of them, the way a cached clone a fetch script writes
under a root would otherwise be linted file by file. A committed file is never
pruned, because git ignores no tracked file, and outside a repository nothing
is. The lint names what it pruned, so a smaller tree is never read as a clean
one; `record-lint` prunes the same way and names the paths on stderr.

## Output

The JSON payload carries `blockers` (a count) and `findings` (each with
`File`, `Line`, `RuleID`, `Severity`, `Message`), and `pruned`, the gitignored
paths the walk skipped (a wholly ignored directory once, with its trailing
slash), absent when it skipped none; the text render names them on one line. A `blockers` value of zero
means the docs are currency-clean. The command exits non-zero when a blocker is
present, so it composes directly into CI and the release gate.

## Composition

The currency lint is the deterministic, fast, always-runnable currency check. The
`docs-currency-reviewer` agent is its semantic complement: it verifies that
every user-facing claim still matches the code, which a structural lint cannot.
The release gate runs both.

The citation refresh composes with the gate by separation: the gate stays
deterministic because the fetching happens elsewhere and arrives as a committed
record a reviewer reads in a diff. The baseline's age surfaces at `abcd ahoy`
and in the launch preview's preflight, which names what the citation
baseline would stop a release on while a release still cuts.

## References

- Plugin command: [`commands/docs.md`](../../../../commands/docs.md)
- Lint engine: `internal/core/lint`
- The documentation invariants it enforces: [`../02-constraints`](../02-constraints)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd docs`

Sub-verbs: `abcd docs cite`.

Flags: none.

### `abcd docs cite`

Sub-verbs: `abcd docs cite confirm`, `abcd docs cite refresh`.

Flags: none.

### `abcd docs cite confirm`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--config` | string |
| `--receipt` | string |
| `--root` | string |

### `abcd docs cite refresh`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--config` | string |
| `--root` | string |

<!-- surface-appendix:end -->
