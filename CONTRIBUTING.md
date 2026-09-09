# Contributing

abcd is a public project under active development. See [`AGENTS.md`](AGENTS.md)
for build/test/checks and working conventions, and
[`.abcd/development/`](.abcd/development/) for the design record.

## Licence

Contributions are accepted under the project's licence, inbound = outbound: by
submitting a change you agree it is licensed under the [MIT licence](LICENSE)
like the rest of the project, and that you are entitled to submit it under that
licence. There is no CLA and no `Signed-off-by:` requirement — a plain
inbound = outbound statement is the whole of it.

## How changes land

- **Issue first.** External contributions start from an accepted issue: open one
  (or pick an open one) and get a maintainer's nod before writing code. A pull
  request with no accepted issue behind it may be declined on scope alone —
  that is policy, not a judgement of the work.
- **Branch + PR** for substantive changes; CI gates the merge. Its `check` job
  builds, vets and tests (plain and race-enabled) on macOS + Linux, and on the
  Linux leg alone adds the `make fmt-check` format gate, the record-lint and
  docs-lint steps, and the site-render gate; separate jobs run the
  reviews-charter and issue-resolution checks (RS001–RS003), `gitleaks`,
  `zizmor`, dependency review, `govulncheck`, the smoke harness and the
  cold-reading evals (`make evals-cold-reading`, which runs on every event).
- **Merge queue.** Merging goes through the queue ("Merge when ready"): the
  required checks run against the actual merged result. The ruleset also
  requires a pull request to be current with `main` before its own checks
  count, deliberately (iss-172: it is what gates a duplicate record id minted
  by two checkouts), and auto-merge only enqueues a pull request whose checks
  count. So when `main` moves under an armed pull request it sits `BEHIND`
  outside the queue until its branch is updated. `scripts/pr-keep-current.sh`
  performs that update for every armed pull request (`--watch` repeats until
  none is armed); run it after arming auto-merge, and after every merge that
  moves `main`. A pull request confined to `docs/`,
  `.abcd/development/`, `.abcd/work/` and the root prose files stands the macOS
  leg, the race lane and the `zizmor`, `govulncheck` and smoke lanes down while
  it is in review; the queue run is not a pull-request event, so the full set
  gates the merge either way.
- **Publish surface reviews.** Paths listed in
  [`.github/CODEOWNERS`](.github/CODEOWNERS) ship behaviour to installed users
  (plugin hooks and commands, agent prompts, workflows, gates and build
  config). Changes there additionally require a code-owner review. The applied
  branch rulesets are mirrored under
  [`.abcd/work/rulesets/`](.abcd/work/rulesets/).
- **Volume cap.** At most three open pull requests per external author at a
  time — review attention is the scarce resource this protects.
- **Local gates.** `make preflight` runs the same build, vet, test and race
  steps locally, together with the lint-reviews, lint-issues, lint-decisions,
  record-lint,
  docs-lint and site-render gates and both tagged eval lanes (smoke,
  evals-cold-reading, about five seconds each — the untagged test step compiles
  neither) — but not the format gate, so run `make fmt-check` before pushing
  (it runs the gofmt from the toolchain `go.mod` declares, which is the one CI
  runs; `make fmt` applies it). The repository
  ships its hooks in [`.githooks/`](.githooks/); they are per-machine opt-in —
  run `git config core.hooksPath .githooks` once per clone to arm the
  pre-commit name guard and the pre-push preflight.
- **Conventional-commit prefixes** (`feat`/`fix`/`docs`/`chore`/`refactor`/`test`/`ci`),
  no scopes; short title, body explains why.
- A user-facing change **resolves its issue or ships its intent in the same diff**;
  the CHANGELOG is derived at release from those record transitions, and
  `## [Unreleased]` stays empty (a hand-written entry blocks the next cut).
- **Docs** are Diátaxis (one type per page, present tense); the design record lives
  under `.abcd/`, never in `docs/`. Prose follows the canonical
  [writing style guide](docs/reference/writing-style.md).
- **New dependencies need explicit maintainer sign-off** before they land in
  `go.mod`.

## AI assistance and authorship

Development of abcd is assisted by an AI coding assistant. The convention
follows the Linux kernel's `Documentation/process/coding-assistants.rst`, which
independently reached the same design: an `Assisted-by:` trailer for
disclosure, and never an authorship assertion for a tool. The rules:

- **Human author of record.** The human contributor is the author of every change
  they submit and is responsible for all AI-assisted output — its correctness, its
  licensing, and its fit for the project. AI assistance never transfers that
  responsibility.
- **Commit as yourself.** The gate reads the git author AND committer of every
  commit in a pull request, merge commits included, and refuses a machine
  identity on any of five signals. Four apply to both roles: an assistant
  vendor's name standing alone (`Claude`, `Copilot`, `Gemini` and their kin,
  matched whole, so a human named Claudette passes); an assistant vendor's mail
  domain (`@anthropic.com`, `@openai.com`); the forge's own `[bot]` name suffix;
  and a bot mailbox (`NNNN+name[bot]@users.noreply.github.com`, or
  `@dependabot.com`). The fifth applies to the AUTHOR role only: **any** address
  whose mailbox begins `noreply@` or `donotreply@`, with or without hyphens and
  whatever the host, not just a vendor's. Your forge privacy address
  (`1234+you@users.noreply.github.com`) is yours and passes — the `[bot]` marker
  in the mailbox is what marks a machine, not the `users.noreply.github.com`
  host — and the forge's own `GitHub <noreply@github.com>` committer stamp on a
  web-UI merge passes too, which is why the fifth signal is author-only. Set
  `user.name` and `user.email` to a human before you commit; the assistant
  belongs in the trailer, never in the identity fields the contributor graph
  reads. An automated dependency bump is therefore landed by a human rather than
  merged as the bot authored it.
- **Disclosure by trailer, not co-authorship.** AI-assisted commits carry an
  `Assisted-by: <Agent>:<model-version>` trailer (the kernel format) —
  disclosure only. abcd never uses `Co-Authored-By:` for AI: it asserts an
  authorship the tool does not hold and inflates the contributor graph.
- **The trailer stands on its own line.** The gate matches the trailer anchored
  to a whole line, in the commit message and in the pull-request body alike — a
  trailer buried mid-sentence does not count as disclosure.
- **Wrote it yourself? Say so.** A change no AI assisted carries
  `Assisted-by: None` on its commit and in its pull-request body. Disclosure runs
  both ways: the gate refuses a silent omission because it cannot tell "no AI was
  involved" from "I forgot the trailer", so the human-only case states itself
  rather than being inferred. Never write a vendor trailer for work a model did
  not touch — a false disclosure is worse than none, and the reviewer reading the
  diff is the check on it.
- **No tool footer, and quote it in a fence when you need to.** A "generated with
  <tool>" footer names a tool outside the two credit surfaces this project
  sanctions (the README badge and `ACKNOWLEDGEMENTS.md`), so the gate refuses one
  in a commit message or a pull-request body — including the italic and bold forms
  a host appends by default. Writing *about* the rule is expressly allowed. In a
  pull-request body — which a forge renders as markdown — the check reads the
  document's own voice, so a banned form quoted inside a fenced code block is an
  example rather than a violation, and, the same way, a trailer that appears only
  inside a fence there does not count as your disclosure. A commit message is never
  rendered as markdown, so the gate reads it verbatim: a fenced footer in a commit
  message is refused exactly like a real one — quote the banned form mid-sentence
  instead of fencing it. Close your fences: in a pull-request body an unbalanced one
  is checked as ordinary prose, so a stray ``` cannot switch the gate off for
  everything below it.

## Security

Report vulnerabilities privately — see [`SECURITY.md`](SECURITY.md). Never open
a public issue for a security finding.

## Acknowledgements

[`ACKNOWLEDGEMENTS.md`](ACKNOWLEDGEMENTS.md) credits the ideas, tools, and writing
behind abcd in three parts — development, inspirations, and references. Add an entry
**in the same change that lands it**: the PR that adopts an external pattern, cites
a source in an ADR, or integrates a tool. Adding it at the moment it lands is what
keeps the file from going stale — it is never reconstructed after the fact.
