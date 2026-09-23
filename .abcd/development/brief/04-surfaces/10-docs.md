# `/abcd:docs` — Documentation Currency and Citations

Know, before anyone else reads them, whether this repo's docs still describe
the present: no prose narrating a change, no link that has stopped resolving,
no markdown stranded at the repo root, and no cited source that nobody has
checked in six months. `abcd docs lint` answers that in one read-only pass, so
it can run on every commit at no cost and gate a release without a network.

The `cite` sub-tree is the writing half, and it is the only place abcd reaches
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
| `lint` | lint | shipped |


- **`docs lint`** reports the findings and writes nothing. The plugin command
  invokes it with `--json` and summarises the result. The result carries
  `checks`, the number of banned tokens and enabled rules the configuration
  armed. A configuration that arms none runs nothing, and the verb says so
  ("nothing was checked") in place of a finding count, because "0 finding(s)"
  over a lint that ran no rule is a green that means nothing
  (iss-2609150805167646). The exit code is still 0 there: the configuration was
  read, and no rule it declares was broken.
- **`docs cite refresh`** fetches every cited URL once and rewrites the
  committed citation baseline. Each URL gets exactly one bounded attempt with no
  retries, and no response body is read: liveness is judged from the status
  line. Sources that refuse automated fetchers are printed as a manual checklist
  rather than recorded as broken.
- **`docs cite confirm`** records that a human verified a citation the fetcher
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
`--config` (the `docs-lint.json` to load) and `--root` (the repo to work over);
the bare `cite` parent only routes and takes neither. That is what makes the
refresh fetch exactly the set the gate demands receipts for.

**`docs lint --release-gate` promotes an overdue citation from a warning to a
blocker, and arming it from a release is a design target.** The flag, not the
committed config, is the trust root: a repo must not be able to defang its own
release by editing `.abcd/docs-lint.json`, and an ordinary commit is never
blocked by the calendar. Nothing in the release machinery passes it. The release
workflow's docs-currency step, CI's, and the `docs-lint` make target each run a
bare `abcd docs lint`, the scaffolded release template names the verb nowhere,
and `launch` computes its own citation preflight rather than shelling out. The
promotion is reachable only by a human typing the flag.

## What it checks

- **Change-narration.** Prose that narrates a change rather than describing
  present state. Unambiguous change-narration blocks; phrases that can also
  describe present state warn advisorily rather than block. Docs are present
  tense: what *is*, never what *was superseded*.
- **Broken relative links.** Every relative link resolves to a file in the tree.
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
  files; the fetching lives in `cite refresh`.
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

## Output

The `--json` payload carries `blockers` (a count) and `findings` (each with
`File`, `Line`, `RuleID`, `Severity`, `Message`). A `blockers` value of zero
means the docs are currency-clean. The command exits non-zero when a blocker is
present, so it composes directly into CI and the release gate.

## Composition

`docs lint` is the deterministic, fast, always-runnable currency check. The
`docs-currency-reviewer` agent is its semantic complement: it verifies that
every user-facing claim still matches the code, which a structural lint cannot.
The release gate runs both.

`cite refresh` composes with the gate by separation: the gate stays
deterministic because the fetching happens elsewhere and arrives as a committed
record a reviewer reads in a diff. The baseline's age surfaces at `abcd ahoy`
and in the `abcd launch --dry-run` preflight, which names what the citation
baseline would stop a release on while a release still cuts.

## References

- Plugin command: [`commands/docs.md`](../../../../commands/docs.md)
- Lint engine: `internal/core/lint`
- The documentation invariants it enforces: [`../02-constraints`](../02-constraints)
