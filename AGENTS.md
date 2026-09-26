# AGENTS.md

<!-- BEGIN ABCD -->
<!--
  Managed by abcd (Agent-Based Configuration for Development).
  Do NOT hand-edit content inside the abcd-managed fences — `/abcd:ahoy`
  silently overwrites this block on drift (per itd-3). Per-repo rule
  customisation goes in <repo>/.abcd/rules.json instead, and machine-wide
  customisation in ~/.abcd/rules.json.
-->

## abcd rule loader

This repository uses the abcd modular rules loader. On `UserPromptSubmit`, a hook
recall-matches the prompt against keyword triggers declared in the plugin-bundled
default domains, the machine's `~/.abcd/rules.json` and `<repo>/.abcd/rules.json`,
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
- Machine-wide overrides: `~/.abcd/rules.json` takes the same schema and holds
  the conventions shared by every repo on the machine. The layers apply in
  order — bundled defaults, then `~/.abcd/rules.json`, then
  `<repo>/.abcd/rules.json` — each replacing a field wholesale, so the repo wins
  a field both set and a repo `dormant` state or kill switch holds against a
  user layer. The file is read only when it is a regular file this account owns
  that no one else can write, within 256 KiB; a file failing that, or failing to
  parse or validate, fails the load loudly and the hook injects nothing. Absent,
  it costs nothing and nothing is created; a `HOME` or `~/.abcd` this account
  cannot search reads as absent. A dotfiles-symlinked `~/.abcd` can never host
  a `rules.json`: the file is refused behind a symlinked `~/.abcd`, and only a
  symlinked `~/.abcd` with no `rules.json` in it is spared, reading as absent.
- Provenance: a domain an override names (rules replaced, state changed, or a
  custom domain) renders as `## NAME (user override)` or
  `## NAME (repo override)`, after the last layer that named it, wherever it
  appears: the injected block, `abcd rules`, and the hook's diagnostic;
  `abcd rules --json` carries `"source": "user"` or `"source": "repo"` for it
  and `"source": "bundled"` for an untouched default.
- Kill switch: set `"disabled": true` at the top of `.abcd/rules.json`; at the
  top of `~/.abcd/rules.json` it silences every repo on the machine, and no repo
  file re-enables it.
- Foreign-uid roots: the loader and the shell guard read `.abcd/` from the
  repository root resolved for the session, never from a directory above the
  working tree. Where git cannot answer for that tree — a checkout owned by
  another uid, a container bind mount — the root is recovered from the `.git`
  marker instead, and a root the caller does not own is REFUSED: the session
  takes its own working directory as the root, nothing above it is read, and one
  line on stderr names what was refused. The refusal bounds the walk, not the
  working directory: a `.abcd/` there is still read, so a session started AT the
  refused root reads that root's configuration, over `~/.abcd/rules.json` for
  the rules. Re-admit such a checkout deliberately, from
  an account you control:
  `mkdir -p ~/.abcd && printf '%s\n' '<checkout>' >> ~/.abcd/trusted-roots`
  (one absolute path per line; `#` starts a comment). Only your home declares
  it — a file inside the checkout can never vouch for the checkout.
- Explicit activation: start a prompt with `*<DOMAIN>` (e.g. `*COMMITTING`,
  `*PII`) to inject that domain unconditionally — overrides a `dormant` state,
  but never the kill switch.

### Default domains

`COMMITTING`, `DOCUMENTATION`, `ROADMAP`, `ISSUES`, `INTENTS`, `LIFEBOAT`, `PII`,
`OPINIONS`, `LOAD`. Each carries recall keywords and its rules, bundled in the
abcd binary; a repo overrides them per-field via `.abcd/rules.json`. `OPINIONS`
points at the canonical conventions under `.abcd/development/principles/` rather
than copying them. `LOAD` carries the trust rule for load experiments: one owned
process group killed together through a re-checked handle and never by pattern,
clean proven by what is running, and explicit consent with a cap below the core
count on a live development machine.

### Reset triggers

`SessionStart` and `PreCompact` clear the per-session dedup ledger, so a matched
domain re-injects on the next prompt (the event-driven refresh that recovers
after compaction). Within a session the hook does not re-inject unchanged rules.

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
                    # a warning, never a failure), then lint-reviews +
                    # lint-issues + lint-decisions + record-lint +
                    # issue-drift + docs-lint + site-render +
                    # smoke + evals-cold-reading,
                    # then build + vet +
                    # test + race (internal)
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
`.abcd/development/`, `.abcd/work/` and the root prose files; the Linux unit
lane, the format gate and the record gates always run, and every other event —
the merge-queue entry that gates the merge included — runs the lot.

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
  in `~/.abcd/local-transcript-roots` (session transcripts default to the
  user-level `~/.abcd/transcripts/<root-sha>/records/` store, which creates
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

- **The checkout is the unit of isolation, not the branch.** A second
  concurrent agent session works in its own `git worktree`: one checkout has
  one working tree, one HEAD, and one index, and a branch switch swaps all
  three under whoever else is using them. The lint gates read the whole tree,
  so foreign work-in-progress fails them in both directions.
- **That worktree goes in the machine-scoped store, and nowhere else.** A
  session's own checkout lives at `~/.abcd/worktrees/<root-sha>/<name>/`, keyed
  on the repository's root commit the way the history, transcript and voyage
  stores already are — a checkout moves, is renamed and is cloned twice on one
  machine, while its root commit does none of that. Not beside the checkout,
  not in the directory the user keeps their projects in, and not inside the
  working tree, which every tree scan walks. A tool never creates a directory
  in space the user did not hand it, and beside a checkout there is no declared
  tier at all:
  [adr-2609091248200336](.abcd/development/decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
  is the rule and
  [`the-users-directory-is-theirs`](.abcd/development/principles/the-users-directory-is-theirs.md)
  is the stance. **The store has no verbs yet.** Aim a plain `git worktree add`
  at the path and create the lane by hand; the store's own `add`, its listing
  and its reclaim are
  [itd-2609091014076309](.abcd/development/intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md),
  in `drafts/`, so until it ships nothing enumerates the lane or prunes a spent
  worktree for you, and a worktree in the store is retired with
  `git worktree remove` like any other.
- **Scan before mutating git state.** Before a commit, branch switch, stash,
  rebase, or `git worktree add`/`remove` in a checkout that might be shared,
  check for peer sessions via the harness's session listing, and announce the
  mutation to any peer found. Before capturing, resolving or picking a record,
  also run `go run ./cmd/abcd peers` (`--json` for a machine reader): it lists
  what every sibling worktree and local branch of this checkout holds that this
  tree does not, uncommitted captures included, and writes nothing. It sees
  records, not sessions: an empty listing says no peer holds anything that
  differs, not that no peer is running. A worktree counts even though it leaves
  HEAD alone. The sharpest case is a worktree created *inside* the checkout — the
  shape the store above exists to keep out — which churns the tree a peer's
  scan walks, so a concurrent `make preflight` can fail
  `TestPayloadTreeImplementationsResolveIdentically` with `the payload carries
  N rejected file(s)` while the directory populates. That signature, during
  another session's worktree churn, is a retry rather than a bisect — the
  steady-state worktree is harmless, and only the creation window flakes
  (iss-2608261331317889).
- **A diff you did not make is a peer's work.** Never commit it, revert it,
  or stash it away silently — coordinate with the session that made it;
  uncommitted peer work is untouchable. (Mechanical presence detection is
  seeded as iss-2608220750029993; until it ships, this convention is the
  gate.)
- **A verifier works on a copy.** An agent that mutates code to see whether a
  test catches the mutation, or patches or instruments a tree to probe it, does
  that on a scratch copy (`git -C <wt> archive HEAD | tar -x -C <scratch>`),
  never on a live worktree, and proves `git status --porcelain` empty before
  reporting. The hazard is the window, not the intent: while a mutation is
  applied, a gate reports on code nobody wrote and a merge can take it into a
  branch, and the restore step is itself fallible. It also breaks the rule
  above from the other end — a peer seeing the modification cannot tell a
  mutation from real work. Correspondingly, a merge, commit, push or gate run
  proves the tree clean **immediately before the act**, never inheriting an
  emptiness check from earlier in the sequence (itd-193).
- **A session cutting a release has the final say on what merges before its
  tag.** A change ready to land while a peer is mid-cut is handed over as a
  pull request, announced to the cutting session with the records it would
  move into the cut, and merged or held on that session's ruling, never on the
  landing session's. The cutting session reads the diff for anything its
  release gate's semantic review will see and answers with one of two
  orderings: merge now and sync into the release branch, so the records join
  this cut, or hold until the tag, so they fall into the next. A record closed
  for work an earlier release already carried is stamped `shipped_in:` with
  that release, whichever ordering is chosen, or it lands in the changelog as
  if it shipped today. The rule holds even when the change is orthogonal to
  the release gates: orthogonal to the gates is not orthogonal to the release
  branch, the cutting session is the one that can see the overlap, and a
  full-tier release receipt names the commit its reviewers read, so nothing
  merges between the CHANGELOG roll and the tag (iss-2609091037191879 seeds
  the mechanical form).
- **Record ids need no coordination between checkouts.** Captures, intents
  and specs mint timestamp-numeric ids through one allocator that reads no
  maximum (adr-45), so two current checkouts minting in the same window
  allocate distinct ids unless they share the same second and the same
  four-digit draw, a coincidence the armed uniqueness detectors assert against;
  the per-checkout mint lock only serialises minters inside one checkout. ADRs
  mint through that same seam: `abcd decide "<title>"` allocates
  `adr-<yymmddHHMMSS><rrrr>` and files it as `<stamp>-<slug>.md` (the 2026-09-01
  ruling in `.abcd/work/DECISIONS.md`, the turn adr-45 ruling 3 deferred), so two
  checkouts deciding in the same window cannot allocate one number either. The
  ordinals `0001`–`0058` keep their ids and their filenames and every reader
  admits both vintages through one derivation, so no record family needs a word
  first — mint the ADR.
- **A record is resolved on the branch that carries it**, never re-added to the
  default branch after a branch was cut from it. The ledger's status signal IS
  folder membership, so the one state it cannot represent is a record in two
  status folders at once — and that is what the re-add produces: the branch moves
  the record `open/` → `resolved/` while the default branch adds it back into
  `open/`, git pairs an add on one side with a delete-plus-add on the other, and
  the integration tree carries both copies with no status at all. Every read of
  the ledger now refuses on it and names both files (iss-2609100507430423), so
  the remedy is to move or remove one; the convention is what stops it arising.

## Definition of done

- `make preflight` is clean — the seven gates (`lint-reviews`, `lint-issues`,
  `lint-decisions`, `record-lint`, `issue-drift`, `docs-lint`, `site-render`),
  both tagged eval
  lanes (`smoke`, `evals-cold-reading`), plus `go build ./...`,
  `go vet ./...`, `go test ./...`, and
  `go test -race -timeout 20m ./internal/...`. The load
  check runs first (`load-check`, a warning, never a failure) and is not a gate:
  it exits 0 whatever it finds. The eval
  lanes are named separately because their files carry a build tag, so
  `go test ./...` compiles none of them; each costs about five seconds.
- `make fmt-check` reports nothing. The format gate is CI's own step, outside
  `make preflight`, so run it before pushing. It resolves gofmt from the
  toolchain `go.mod` declares rather than from PATH, because gofmt's rules move
  between releases and a bare `gofmt` on a newer machine names files CI
  considers correctly formatted (iss-2609081953452204); `make fmt` rewrites what
  it names, with that same binary. If the pinned toolchain cannot be fetched the
  target refuses and names the skew — it never falls back to the local gofmt.
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
  stale issue closed on its own merits has no fixing commit to name.
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
  that happens after the merge is the one that gets forgotten.
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

- **AI-assisted commits carry an `Assisted-by:` trailer**, kernel format
  (`Assisted-by: Claude:claude-opus-4-8`) — disclosure, not authorship. Never
  `Co-Authored-By:` for AI (it asserts an authorship the tool does not hold and
  inflates the contributor graph). There is no DCO: contributions are inbound =
  outbound MIT, so no `Signed-off-by:` is required (adr-43). The human is the
  author of record, responsible for all AI-assisted output. See `CONTRIBUTING.md`.
- **Every commit is authored by a human, and the gate refuses a machine.** The
  contributor graph is built from the author and committer fields, so a machine
  there asserts an authorship it does not hold — and a squash merge re-appends a
  mis-identified branch author as a co-author, inflating the graph again on every
  squash. `scripts/check-attribution.sh commits` reads the identity of every
  commit in a range, merge commits included, and refuses one on any of five
  signals. Four are checked in both roles: an assistant vendor's name standing
  alone as the identity name (`Claude`, `Copilot`, `Gemini` and their kin,
  matched whole so a human named Claudette passes); an assistant vendor's mail
  domain (`@anthropic.com`, `@openai.com`); the forge's own `[bot]` name suffix;
  and a bot mailbox (`NNNN+name[bot]@users.noreply.github.com`, or
  `@dependabot.com`). The last two are structural rather than nominal, which is
  why a second automation lands in the right place with no edit to the list. The
  fifth signal is checked in the AUTHOR role only: **any** address whose mailbox
  begins `noreply@` or `donotreply@` (with or without hyphens), whatever the host — it
  is not scoped to a vendor, because an address named for not being read names
  no person in the role that claims authorship. It is refuse-machines, not an
  allowlist of names: this repository takes outside contributions
  (`.abcd/work/intake.md`), and a person's forge privacy address
  (`1234+name@users.noreply.github.com`) is a human's and passes — the `[bot]`
  marker in the mailbox is the discriminator, never the
  `users.noreply.github.com` host. The role asymmetry is what keeps the history
  green: the forge as COMMITTER (`GitHub <noreply@github.com>`) is how every
  web-UI merge and squash is stamped on a human's click, and passes in that role
  alone. **The consequence is deliberate: a dependabot pull request is not
  mergeable as authored, so a dependency bump is landed by a human.**
- **A human-only change declares itself: `Assisted-by: None`.** The convention is
  disclosure, and work no AI touched has nothing to disclose — but silence cannot
  say so, because an absent trailer and a forgotten one are the same bytes. The
  declaration is the positive form, and it is the only accepted non-vendor value:
  a free-text escape would reopen the omission it closes. Claiming it for assisted
  work is a false disclosure, which is the thing this convention exists to prevent.
- **Naming a tool is confined to credit.** User-facing prose (`README.md`,
  `docs/`) stays host-agnostic — the `harness/*` docs-lint rules enforce it. The
  one sanctioned place to name a tool is attribution: the README badge and
  `ACKNOWLEDGEMENTS.md`, using the `<!-- docs-lint: allow -->` escape where a lint
  root is involved. Private, unpublished tool names never appear in any committed
  file.
- **Outward-facing text carries no session URL and no tool footer, and is
  re-read after it is created.** A pull-request body, an issue, a comment, a
  commit message and a release note are public the moment they exist, and a forge
  keeps the pre-edit revision of whatever was posted — so scrubbing later is not a
  remedy. Two things leak here: a live agent-session URL, and a tool's own
  "generated with" attribution footer, which overrides the `Assisted-by:`
  convention above. Both are banned in public text. **After creating any PR, issue
  or comment, re-read what was actually created and strip either shape from it**
  — the harness appends them outside the model's own output, so text that left
  clean can arrive dirty. The policy is a value, not just this paragraph
  (`scanner.OutboundPolicy`): the scanner's canonical pattern set carries the
  class into every store-before-commit redactor, `abcd lint`'s privacy rule
  refuses either shape in any committed file, and the `harness_leak` lint rule
  refuses it in the record and the docs. One definition, three wired surfaces
  (itd-152). A commit message is judged by the check-direction front door onto
  the same policy, `abcd lint outbound`, twice: the committed
  `.githooks/commit-msg` hook refuses either shape before the commit exists, and
  the attribution gate in CI judges every commit message of a pull request and
  its body again. A fourth exists as a primitive with no front door:
  `scanner.ScrubOutbound` sanitises one outbound artefact and is covered by
  tests, but no command or plugin verb calls it, because `spc-45` deliberately
  scopes a forge client out. The three wired surfaces judge text that is already
  committed or already stored, so none of them reaches a forge artefact: until
  the primitive is wired, the protection at posting time is the re-read-and-strip
  step above, performed by whoever posts.
- **`ACKNOWLEDGEMENTS.md`** credits ideas, tools, and writing in three parts —
  development, inspirations, references. Add an entry in the same change that lands
  it (adopts a pattern, cites a source in an ADR, integrates a tool), never later.
