# AGENTS.md

Conventions-file check word: brindlewort. A session that can name it without
reading any file loaded this file as its instructions (itd-2610030814013772).

<!-- BEGIN ABCD -->
<!--
  Managed by abcd (Agent-Based Configuration for Development).
  Do NOT hand-edit content inside the abcd-managed fences — `/abcd:ahoy`
  silently overwrites this block on drift (per itd-3). Per-repo rule
  customisation goes in <repo>/.abcd/rules.json instead, and machine-wide
  customisation in ~/.abcd.noindex/rules.json.
-->

## abcd rule loader

This repository uses the abcd modular rules loader. On `UserPromptSubmit`, a hook
recall-matches the prompt against keyword triggers declared in the plugin-bundled
default domains, the machine's `~/.abcd.noindex/rules.json` and `<repo>/.abcd/rules.json`,
and injects only the matched domain rules into context — instead of
force-loading the full ruleset every turn.
A prompt that matches no domain injects nothing (zero added tokens).

- Inspect rules: `abcd rules` renders the active set; `abcd rules <DOMAIN>`
  (case-insensitive) scopes to one domain.
- Per-repo overrides: edit `<repo>/.abcd/rules.json`. It is
  `{"schema_version": 1, "disabled": false, "domains": {}}` — add a domain key to
  override a default per-field (e.g. `{"ROADMAP": {"state": "dormant"}}` silences
  it while keeping its rules) or to declare a custom domain
  (`{"recall": [...], "rules": [...]}`). A domain left with no rules at all
  (`{"rules": []}`, or a custom domain declared without any) is SKIPPED with a
  diagnostic on stderr naming it — it would otherwise inject a heading-only
  block, which reads as a domain that says nothing. The rest of the file still
  loads; `{"state": "dormant"}` is the way to silence a domain deliberately.
- Machine-wide overrides: `~/.abcd.noindex/rules.json` takes the same schema and holds
  the conventions shared by every repo on the machine. The layers apply in
  order — bundled defaults, then `~/.abcd.noindex/rules.json`, then
  `<repo>/.abcd/rules.json` — each replacing a field wholesale, so the repo wins
  a field both set and a repo `dormant` state or kill switch holds against a
  user layer. The file is read only when it is a regular file this account owns
  that no one else can write, within 256 KiB; a file failing that, or failing to
  parse or validate, fails the load loudly and the hook injects nothing. Absent,
  it costs nothing and nothing is created; a `HOME` or `~/.abcd.noindex` this account
  cannot search reads as absent. A dotfiles-symlinked `~/.abcd.noindex` can never host
  a `rules.json`: the file is refused behind a symlinked `~/.abcd.noindex`, and only a
  symlinked `~/.abcd.noindex` with no `rules.json` in it is spared, reading as absent.
- Provenance: a domain an override names (rules replaced, state changed, or a
  custom domain) renders as `## NAME (user override)` or
  `## NAME (repo override)`, after the last layer that named it, wherever it
  appears: the injected block, `abcd rules`, and the hook's diagnostic;
  `abcd rules --json` carries `"source": "user"` or `"source": "repo"` for it
  and `"source": "bundled"` for an untouched default.
- Kill switch: set `"disabled": true` at the top of `.abcd/rules.json`; at the
  top of `~/.abcd.noindex/rules.json` it silences every repo on the machine, and no repo
  file re-enables it.
- Foreign-uid roots: the loader and the shell guard read `.abcd/` from the
  repository root resolved for the session, never from a directory above the
  working tree. Where git cannot answer for that tree — a checkout owned by
  another uid, a container bind mount — the root is recovered from the `.git`
  marker instead, and a root the caller does not own is REFUSED: the session
  takes its own working directory as the root, nothing above it is read, and one
  line on stderr names what was refused. The refusal bounds the walk, not the
  working directory: a `.abcd/` there is still read, so a session started AT the
  refused root reads that root's configuration, over `~/.abcd.noindex/rules.json` for
  the rules. Re-admit such a checkout deliberately, from
  an account you control:
  `mkdir -p ~/.abcd.noindex && printf '%s\n' '<checkout>' >> ~/.abcd.noindex/trusted-roots`
  (one absolute path per line; `#` starts a comment). Only your home declares
  it — a file inside the checkout can never vouch for the checkout.
- Explicit activation: start a prompt with `*<DOMAIN>` (e.g. `*COMMITTING`,
  `*PII`) to inject that domain unconditionally — overrides a `dormant` state,
  but never the kill switch.

### Default domains

`COMMITTING`, `DOCUMENTATION`, `ROADMAP`, `ISSUES`, `INTENTS`, `LIFEBOAT`, `PII`,
`OPINIONS`, `LOAD`, `SHELL`, `GRILL`. Each carries recall keywords and its rules, bundled
in the abcd binary; a repo overrides them per-field via `.abcd/rules.json`.
`OPINIONS` points at the canonical conventions under
`.abcd/development/principles/` rather than copying them. `LOAD` carries the
trust rule for load experiments: one owned process group killed together through
a re-checked handle and never by pattern, clean proven by what is running, and
explicit consent with a cap below the core count on a live development machine.
`SHELL` is the teaching half of the shell-hazard guard: it is generated from the
same hazard registry `abcd guard` enforces in the repository, one rule per entry
(the command, why it is dangerous, and what to run instead; a rule from the
repository's own `.abcd/guard.json` is marked `(repo)`), and recalls on the
commands the registry names (`rm`, `git push`, `pkill`, …) and on shell work in
general, so an agent is taught the safe form before a host with hooks would
refuse the command and a host without hooks still teaches it.
`GRILL` carries the asking rules every abcd interview follows (one thing at a
time, the thing being decided quoted in full before the question, options that
widen rather than recommend); it is generated from the same limits the question
check in `abcd guard hook` enforces, and recalls on words of asking and
choosing, so it lands in most sessions. A repository that does not want it
silences it with `{"GRILL": {"state": "dormant"}}` in `.abcd/rules.json`.

### Reset triggers

`SessionStart` and `PreCompact` clear the per-session dedup ledger, so a matched
domain re-injects on the next prompt (the event-driven refresh that recovers
after compaction). Within a session a domain that stays in force is never
re-injected unchanged; one that leaves the active set (deleted, renamed, made
dormant, or a `*<DOMAIN>` activation the next prompt does not repeat) is
injected again when it returns.

For internals see `.abcd/development/brief/05-internals/03-configuration.md`.

<!-- END ABCD -->

abcd (Agent-Based Configuration for Development) is for people who know what they
want to build and need help shipping it: a host-agnostic **configuration layer
for intent-driven development**, delivered as a Go CLI and an agent-harness
plugin. A single `abcd` binary holds all behaviour in a transport-agnostic core;
the CLI, the markdown plugin surface, and (later) an MCP server are thin front
doors onto it.

Start with the plan and the design record:

- Design record (the specification): [`.abcd/development/`](.abcd/development/) —
  brief, roadmap, intents, decisions/adrs, research.
- Package map: [`internal/README.md`](internal/README.md).

## Build, test, and checks

Run from the repo root.

```bash
make preflight      # the pre-push gate: the load check first (load-check,
                    # a warning, never a failure), then fmt-check +
                    # lint-reviews +
                    # lint-issues + lint-decisions + record-lint +
                    # issue-drift + docs-lint + check-attribution +
                    # site-render +
                    # smoke + evals-cold-reading,
                    # then build + vet +
                    # test + race (internal), all on the go.mod toolchain
make build          # cross-compiles bin/abcd-<goos>-<arch> (there is no plain bin/abcd)
make fmt-check      # format gate, run through the go.mod toolchain's gofmt
make fmt            # rewrite what fmt-check names, with that same gofmt
go vet ./...        # static checks
go test ./...       # unit tests
go test ./internal/core/                 # a single package
go test -run TestStatus ./internal/core/ # a single test
```

**A push is gated before it connects.** The committed `.githooks/pre-push` hook
never runs the preflight: git opens a push's connection before it runs the hook,
and a preflight inside it outlasted the transport's idle timeout, so the push
reported success and moved nothing. `make preflight` ends by minting a receipt
for HEAD instead, and only when the working tree matched HEAD — nothing staged,
unstaged or untracked — both when the run began and when it ended, so the gates
read exactly the tree CI checks out. The hook refuses a push whose new commit has
no receipt. The sequence is: commit everything, `make preflight`, then a plain
`git push`. A receipt minted in any worktree of the checkout counts, and a commit
the remote already holds (a tag on a merged commit) needs none.

**In a source checkout of abcd, every abcd invocation is `go run ./cmd/abcd
<verb>` from the repo root** — never the plugin-root binary and never an `abcd`
on PATH. Both are whatever version was last published, and in this repository
that is the thing being developed, so they are stale by construction and fall
further behind with every commit: a verb, flag, schema field or refusal added
since the last release is unknown to them. The failure is not always a
refusal. `launch ship` refuses on its surface guard, loudly and correctly, but
`changelog --json` returned an empty cut against a tree holding 181 shipped
records, and `capture` would have written records through a schema the record
gates no longer accept — a plausible wrong answer, not an error. The plugin
command pages document a resolution ladder that reaches the plugin-root binary
first and falls through to `go run` only when nothing earlier resolves; in this
checkout the first rung exists and answers, so the fallback written for exactly
this case is never reached by following the ladder literally
(iss-2608230943088357 holds the surface half). The loader's `DOGFOODING` domain
in `.abcd/rules.json` injects this rule on a prompt that names `abcd` or any of
its top-level verbs (the `Available Commands` list of `go run ./cmd/abcd --help`).

CI (`.github/workflows/ci.yml`) runs its `check` job on macOS + Linux — build,
vet, test and the race-enabled internal tests on both, with the `make
fmt-check` format gate, the record-lint, issue-drift and docs-lint steps and the
site-render gate on the Linux leg alone. Separate jobs run the reviews-charter check
(`scripts/check-reviews.sh`) together with the issue-resolution gates
(RS001–RS006) and the decisions-append gate (DA001–DA003), full-history secret scanning (`gitleaks`), a workflow audit
(`zizmor`), dependency review, `govulncheck`, and the smoke harness
(`make smoke`). A
fail-closed classifier stands the macOS leg, the race lane and the `zizmor`,
`govulncheck` and smoke jobs down on a pull request confined to `docs/`,
`.abcd/development/`, `.abcd/work/`, the root prose files and the
community-health files in `.github/`; the Linux unit lane, the format gate and
the record gates always run, and every other event — the merge-queue entry that
gates the merge included — runs the lot.

## Working-tree layout (three tiers under `.abcd/`)

Development material lives under `.abcd/`; `docs/` is user-facing only.

- `.abcd/development/` — **durable record** (committed): brief, intents, ADRs,
  plans, research. In every repository checkout; not in the released binaries.
- `.abcd/work/` — **shared working** (committed): `CONTEXT.md` (current
  orientation) and `DECISIONS.md` (append-only decision log; architecture-shaping
  decisions graduate to ADRs under `.abcd/development/decisions/adrs/`), plus the
  issue ledger `issues/` (working-tier data per adr-32), the reviews charter
  `reviews/`, the branch-ruleset mirror `rulesets/`, and `intake.md`, the
  external-contribution runbook.
- `.abcd/.work.local/` — **local ephemeral** (gitignored): `NEXT.md` handover,
  `scratch/`, `logs/`, `reviews/` (intent-audit receipts), `private-names.txt`
  (per-machine banlist layer), and `transcripts/` when this checkout is declared
  in `~/.abcd.noindex/local-transcript-roots` (session transcripts default to the
  user-level `~/.abcd.noindex/transcripts/<root-sha>/records/` store, which creates
  itself; the per-repo location is an opt-in pull). Per-worktree, so it never
  merge-conflicts.

**Default to the local tier when in doubt.** Any artefact whose home is unclear —
tool exports, oracle/review output, traces, intermediate analysis — goes to
`.abcd/.work.local/scratch/` (or `logs/` for run output) **first**. Never to the
repo root, and never into a tracked directory on a guess. Promotion is cheap and
always available: an artefact that proves durable is moved up to `.abcd/work/` or
`.abcd/development/` later, in a change that says why. Demotion is not — an
artefact committed to the wrong tier is already in the history, and a stray
top-level directory is a `stray_root_docs` finding. **Guessing upward is
irreversible; guessing downward costs nothing.**

## Boundaries

- **Transport-agnostic core.** `internal/core` never writes to stdout or knows a
  transport; front doors under `internal/surface/*` format its results.
- **Wired or it isn't done.** Every verb is reachable from both the CLI and the
  plugin markdown surface and demonstrably executes there — no dead scaffolding.
- **Host-delegated by default.** LLM review/agent work is delegated to the host;
  native/CLI/API/MCP oracles are opt-in adapters.
- **Single repo, curated release.** `.abcd/**` stays in-tree and is present in
  every repository checkout — marketplace installs and release source archives
  included — never in the released binaries; the launch bundler denies the
  namespace structurally, though that filter has yet to run on a cut release.
  The repo is the plugin marketplace.
- **Never commit or push without being asked.** Substantive work goes on a branch
  and PR; new dependencies need explicit sign-off before `go get`.

## Concurrent sessions

Each rule below is the short form. The `CONCURRENCY` domain in
`.abcd/rules.json` carries the whole of each, and the rules loader injects it on
a prompt about worktrees, peers, sessions, releases or record ids (start a prompt
with `*CONCURRENCY` to force it).

- **The checkout is the unit of isolation, not the branch.** A second
  concurrent agent session works in its own `git worktree`: a branch switch
  swaps the working tree, HEAD and index under whoever else is using them.
- **That worktree goes in the machine-scoped store, and nowhere else:**
  `~/.abcd.noindex/worktrees/<root-sha>/<name>/`, keyed on the full object name
  of the repository's root commit. Never beside the checkout, in the directory
  the user keeps their projects in, or inside the working tree
  ([adr-2610031751065746](.abcd/development/decisions/adrs/2610031751065746-the-worktree-store-lives-under-the-renamed-home-abcd-noindex.md)).
  The store has no verbs yet: aim a plain `git worktree add` at the path, and
  retire it with `git worktree remove`.
- **Scan before mutating anything a peer reads or runs.** Check the harness's
  session listing and announce the mutation to any peer found; before
  capturing, resolving or picking a record, also run `go run ./cmd/abcd peers`.
- **A diff you did not make is a peer's work.** Never commit it, revert it, or
  stash it away silently; uncommitted peer work is untouchable.
- **A verifier works on a copy** (`git -C <wt> archive HEAD | tar -x -C
  <scratch>`), never on a live worktree, and a merge, commit, push or gate run
  proves the tree clean **immediately before the act** (itd-193). The copy goes
  outside every working tree: the session's scratchpad or a directory under
  `~/.abcd.noindex/`, never `.abcd/.work.local/scratch/`, where every verb run
  from it reads the worktree around it instead.
- **A session cutting a release has the final say on what merges before its
  tag.** A change ready while a peer is mid-cut is handed over as a pull
  request and merged or held on the cutting session's ruling.
- **Record ids need no coordination between checkouts.** Captures, intents,
  specs and ADRs mint timestamp-numeric ids through one allocator (adr-45):
  `abcd decide "<title>"` allocates `adr-<yymmddHHMMSS><rrrr>`, and the
  ordinals `0001`–`0058` keep their ids, so no record family needs a word
  first — mint the ADR.
- **A record is resolved on the branch that carries it**, never re-added to the
  default branch after a branch was cut from it: a record in two status folders
  at once makes every read of the ledger refuse (iss-2609100507430423).

## Definition of done

- `make preflight` is clean — the nine gates (`fmt-check`, `lint-reviews`,
  `lint-issues`, `lint-decisions`, `record-lint`, `issue-drift`, `docs-lint`,
  `check-attribution`, `site-render`), both tagged eval
  lanes (`smoke`, `evals-cold-reading`), plus `go build ./...`,
  `go vet ./...`, `go test ./...`, and
  `go test -race -timeout 20m ./internal/...`. The load
  check runs first (`load-check`, a warning, never a failure) and is not a gate:
  it exits 0 whatever it finds. The eval
  lanes are named separately because their files carry a build tag, so
  `go test ./...` compiles none of them; each costs about five seconds.
- **Preflight judges with CI's toolchain.** `make fmt-check`, the format gate
  CI's check job runs, is preflight's first gate, straight after the load
  check. It resolves gofmt from the toolchain `go.mod` declares rather than from
  PATH, because gofmt's rules move between releases and a bare `gofmt` on a
  newer machine names files CI considers correctly formatted
  (iss-2609081953452204); `make fmt` rewrites what it names, with that same
  binary. Every Go step preflight makes — build, vet, test, race, and each
  `go run` and `go test` of its gates — runs on that same declared toolchain,
  which is the one CI's `setup-go` installs, so a test that asserts
  standard-library wording cannot pass preflight on a newer local `go` and fail
  CI (iss-2609261850045839). One resolver, `scripts/pinned-toolchain.sh`, serves
  both; if the declared toolchain cannot be fetched it refuses and names the
  skew — it never falls back to the local `go`.
- Every new behaviour has a test watched fail before the change and pass after.
- **A user-facing change is accompanied by a RECORD, not by a hand-written
  CHANGELOG entry.** The changelog is derived: `launch ship` composes the dated
  section from the records that reached a terminal folder since the last tag, and
  `## [Unreleased]` must be EMPTY or the ingest refuses — a derived cut never
  folds hand-written prose into a generated section. So the way to announce a
  change is to resolve its issue or ship its intent in the same diff, which the
  point below already requires. Writing the entry by hand does not add a line: record-lint's
  `changelog_unreleased_empty` rule refuses it at the change, before it can block
  the next release.
- **A change that fixes a captured issue resolves it in the same change**, and
  says so with a `Resolves: iss-N` trailer. `lint-issues` (RS001) refuses a
  trailer whose record does not enter `.abcd/work/issues/resolved/` or
  `.abcd/work/issues/wontfix/` in the same diff — a bare delete of the open
  record satisfies nothing. Resolution is deliberately not a post-merge step: a step that happens
  after the merge is the one that gets forgotten, and a fixed-but-open issue
  leaves no marker to find it by. Resolving without a trailer stays legal — a
  stale issue closed on its own merits has no fixing commit to name. A `git
  revert` later in the same range withdraws a `Resolves:` only for a record its
  own diff takes back out of `resolved/` or `wontfix/`, a record the reverted
  commit itself moved in; a "This reverts commit" line over a commit that moves
  no record, or naming a commit that never moved that record, withdraws nothing.
- **A change that delivers a planned intent closes its spec in the same
  change**: `go run ./cmd/abcd spec close <spc-N>` moves the spec to `closed/`
  and, as its close-hook, the intent from `planned/` to `shipped/`. Nothing
  runs it for you, and without a declaration the omission is silent: `launch
  ship` composes the changelog from terminal folders only, so an intent whose
  code is on `main` with its spec still open ships with no changelog line and
  the cut exits 0. So the change says so with a `Delivers: itd-N` trailer, and
  `lint-issues` (RS005) refuses a trailer whose intent does not enter
  `.abcd/development/intents/shipped/` in the same diff, naming every spec
  still open that names it. The trailer means the change FINISHES the intent;
  a change that delivers part of one carries no trailer (a spec closed with
  `--remainder` leaves the intent planned). A change that declares no delivery
  is refused nothing, so the planned intents already sitting with open specs
  are out of the gate's reach; clearing them is a separate act.
  The intent's `impact` decides the derived version, so `shipped/` requires one
  and there is no default: a record that does not already declare it takes
  `--impact additive|breaking|fix` on the close, and a close with neither is
  refused before anything moves. Same shape as the issue rule above: the step
  that happens after the merge is the one that gets forgotten. A close that
  ships an intent (one without `--remainder`) also passes the doc-fidelity
  gate, and so does `launch ship` for every intent shipped since the last tag:
  each refuses until `go run ./cmd/abcd docs fidelity record` has saved a docs
  review for HEAD (`commands/docs.md` says how to run one). The review is
  labelled with the commit it read and kept in the checkout's local tier, so a
  cut made on `main` needs a review recorded there for the merge commit: the
  reviewer runs after the merge, not before it. A revert
  withdraws a `Delivers:` on the same terms, only for an intent its own diff
  takes back out of `shipped/`, an intent the reverted commit itself moved in.
- **A `resolved_by.commit` stamp names a commit that is actually reachable.**
  `abcd capture resolve --commit` is shape-checked only, so a wrong sha reads
  exactly like a right one; RS002/RS003 check reachability instead. Note the
  repository allows merge, squash and rebase merges, and the last two rewrite a
  cited branch sha out of existence — RS003 is what notices.
- **Pre-existing is not a defence.** A defect confirmed while doing other work
  is fixed, or deferred out loud as a recorded decision naming the finding and
  the reason. Capturing it and shipping past it is not the second option: filing
  is a decision to make no decision. The release cut enforces the consequential
  half — `changelog.GuardFindings` refuses a cut carrying a `major` or
  `critical` record that entered the ledger since the anchor tag and is still in
  `open/`, naming every one of them. Findings already in the ledger at the
  anchor are the standing backlog and do not trip it. The way past is to fix and
  resolve it, `wontfix` it with its reason, or add `deferred_after: <anchor
  tag>` and a `deferral_reason:` to the record, which is granted for that one
  cycle and lapses when the next release re-anchors. Full statement:
  [`.abcd/development/principles/pre-existing-is-not-a-defence.md`](.abcd/development/principles/pre-existing-is-not-a-defence.md).

## Attribution and acknowledgements

Each rule below is the short form. The `ATTRIBUTION` domain in
`.abcd/rules.json` carries the whole of each, and the rules loader injects it on
a prompt about commits, authors, trailers, reverts, bots or outward text (start
a prompt with `*ATTRIBUTION` to force it). `scripts/check-attribution.sh` is the
gate.

- **AI-assisted commits carry an `Assisted-by:` trailer**, kernel format
  (`Assisted-by: Claude:claude-opus-4-8`) — disclosure, not authorship. Never
  `Co-Authored-By:` for AI. There is no DCO, so no `Signed-off-by:` is required
  (adr-43). The human is the author of record, responsible for all AI-assisted
  output. See `.github/CONTRIBUTING.md`.
- **The trailer has exactly three accepted forms:** `<Vendor>:<model-version>`;
  `None`, which a human-only change declares, and which is a false disclosure
  for assisted work; and `abcd:<version>`, which only abcd writes, on a commit
  it composes from record facts.
- **Every commit is authored by a human, and the gate refuses a machine** in
  the author or committer role: an assistant vendor's name or mail domain, a
  `[bot]` name or mailbox, a trailing bot word, and, as author only, a
  `noreply@` mailbox. So a dependabot pull request is not mergeable as
  authored: `.github/workflows/dependency-reauthor.yml` re-authors a bump
  inside the declared bound, and a human lands every other bot change
  ([adr-2609292116133348](.abcd/development/decisions/adrs/2609292116133348-a-dependency-bump-inside-the-bound-is-re-authored-as-the.md)).
- **A revert or cherry-pick takes its trailer from one command:**
  `git -c abcd.assistedBy=<Vendor>:<model-version> revert|cherry-pick <sha>`.
  Never set the key as a standing `git config abcd.assistedBy`.
- **Naming a tool is confined to credit.** User-facing prose (`README.md`,
  `docs/`) stays host-agnostic — the `harness/*` docs-lint rules enforce it. The
  one sanctioned place to name a tool is attribution: the README badge and
  `ACKNOWLEDGEMENTS.md`, using the `<!-- docs-lint: allow -->` escape where a lint
  root is involved. Private, unpublished tool names never appear in any committed
  file.
- **Outward-facing text carries no session URL and no tool footer, and is
  re-read after it is created.** A pull-request body, an issue, a comment, a
  commit message and a release note are public the moment they exist. **After
  creating any PR, issue or comment, re-read what was actually created and
  strip either shape from it** — the harness appends them outside the model's
  own output. The commit-msg hook and the attribution gate judge commit
  messages (`abcd lint outbound`); nothing judges a forge artefact at posting
  time but that re-read.
- **`ACKNOWLEDGEMENTS.md`** credits ideas, tools, and writing in three parts —
  development, inspirations, references. Add an entry in the same change that lands
  it (adopts a pattern, cites a source in an ADR, integrates a tool), never later.
