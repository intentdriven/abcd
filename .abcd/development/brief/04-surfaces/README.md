# Surfaces — User-Facing Commands

This is the register of what a person can ask abcd to do. Each row names one
command, says in one line what someone gets from it, and points at the chapter
holding its contract. The one-liners are deliberately thin: a chapter is where a
claim about behaviour belongs, and a summary that repeats it is a second copy to
keep true.

Not every row ships. The **Status** column is machine-checked, and
[`06-delivery/`](../06-delivery) carries the delivery state in detail. Verbs that
are wiring rather than user-facing surface are listed separately under
[§ Operator-internal verbs](#operator-internal-verbs).

| # | Command | Status | Purpose | File |
|---|---|---|---|---|
| 1 | `/abcd:ahoy` | shipped | Install abcd into a project, and see what is installed and what is missing | [`01-ahoy.md`](01-ahoy.md) |
| 2 | `/abcd:disembark` | shipped | Carry the reasoning out of a project into a portable lifeboat, without writing to the project | [`02-disembark.md`](02-disembark.md) |
| 3 | `/abcd:embark` | shipped | Unpack a lifeboat's records into another repository, refusing the whole write on any conflict | [`03-embark.md`](03-embark.md) |
| 4 | `/abcd:launch` | shipped | Preview what a release would ship, and cut one by deriving its version and changelog from the record | [`04-launch.md`](04-launch.md) |
| 5 | `/abcd:intent` | shipped | Say what you want to build as a press release, and find out whether it is ready to implement | [`05-intent.md`](05-intent.md) |
| 6 | `/abcd:capture` | shipped | Get an observation out of your head and into a ledger in one line, and act on it later | [`06-capture.md`](06-capture.md) |
| 7 | `/abcd:library` | shipped | Keep the documents you consult and the cited pages distilled from what the project reads, record what each source changed, and ban the confidential ones' names at commit time | [`07-library.md`](07-library.md) |
| 8 | `/abcd` | shipped | Find out where you are, or what one record id is and what to do with it | [`08-abcd.md`](08-abcd.md) |
| 9 | `/abcd:reflect` | shipped | Look back on a cut release in a short interview, and file what it taught as its retrospective | [`09-reflect.md`](09-reflect.md) |
| 10 | `/abcd:docs` | shipped | Find documentation that has gone stale, and maintain the citation baseline | [`10-docs.md`](10-docs.md) |
| 11 | `/abcd:history` | shipped | Keep session transcripts as a local, redacted corpus this project can study | [`11-history.md`](11-history.md) |
| 16 | `/abcd:lint` | shipped | Check whether this repo still conforms to the working conventions | [`16-lint.md`](16-lint.md) |
| 17 | `/abcd:guard` | shipped | Find out whether a shell command is safe to run, and what to run instead | [`17-guard.md`](17-guard.md) |
| 18 | `/abcd:ideate` | shipped | Put a big, unproven idea through an admission gauntlet and record the verdict either way | [`18-ideate.md`](18-ideate.md) |
| 19 | `/abcd:identity` | shipped | Make every surface say the same thing about the project, and see the diff that would fix one | [`19-identity.md`](19-identity.md) |
| 20 | `/abcd:banlist` | shipped | Declare the names this repo must never publish, and stop them at commit time | [`20-banlist.md`](20-banlist.md) |
| 21 | `/abcd:update` | shipped | Complete a chosen update of the installed binary, verified and atomic | [`21-update.md`](21-update.md) |
| 22 | `/abcd:site` | shipped | Render the project website from the repository's own text, and gate what it publishes | [`22-site.md`](22-site.md) |
| 23 | `/abcd:reading` | shipped | Assemble what a cold reading may see, prove it, and validate what comes back | [`23-reading.md`](23-reading.md) |
| 24 | `/abcd:decide` | shipped | Mint a decision record with its id, date and skeleton, ready to write the decision into | [`24-decide.md`](24-decide.md) |
| 25 | `/abcd:worktree` | staged | Keep session and agent worktrees in a machine-scoped store rather than beside the checkout (design target — [itd-2609091014076309](../../intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md)) | [`../05-internals/03-configuration.md` § The worktree store](../05-internals/03-configuration.md#the-worktree-store) |
| 26 | `/abcd:mode` | shipped | Say whose answer the agent loop is waiting on, so the status line and the board show it | [`08-abcd.md`](08-abcd.md) |
| 27 | `/abcd:implement` | shipped | Share one autonomous run between two sessions: claim a record before its lane, keep the second session inside its bounds, and compare the ways of dividing the work from the run log | [`27-implement.md`](27-implement.md) |
| 28 | `/abcd:peers` | shipped | See what the sibling worktrees and local branches hold before capturing, fixing or filing anything | [`08-abcd.md`](08-abcd.md) |
| 29 | `/abcd:report` | shipped | Tell abcd about a defect or propose an enhancement from a repository it manages, into an inbox in your own account | [`29-report.md`](29-report.md) |
| 30 | `/abcd:inbox` | shipped | Read the reports managed repositories filed, and promote one to a capture that names the sender only by its root-commit key | [`30-inbox.md`](30-inbox.md) |
| 31 | `/abcd:lab` | shipped | Run a lab against a pinned snapshot of the repository and harvest what it found, with the evidence kept out of the repository | [`31-lab.md`](31-lab.md) |
| 32 | `/abcd:scribe` | shipped | Build the ledger scribe's context from the ledger alone, and ingest what it transcribed without letting it author anything | [`32-scribe.md`](32-scribe.md) |
| 34 | `/abcd:build` | shipped | Start the loop that takes one READY intent to delivered, refusing while a question is open or a peer holds it | [`34-build.md`](34-build.md) |
| 35 | `/abcd:drain` | shipped | Work the open ledger unattended: preview with `--dry-run` which open issues a machine may fix alone and in what order, then run it one lane at a time, routing every hand-back by its kind | [`35-drain.md`](35-drain.md) |
| 36 | `/abcd:dashboard` | shipped | Open the project from a phone or another computer on your own Tailscale network, through a server that answers only the devices Tailscale names | [`36-dashboard.md`](36-dashboard.md) |

## How much of this table a machine keeps honest

The **Status** column is machine-checked: the `surface_coverage` record-lint rule,
the row-level presence check over this index, asserts every `shipped` row has a backing surface (`commands/<name>.md` or
`skills/<name>/`) and every `staged` row has none — and, in reverse, that every
command file and skill directory has a row here. The bare `/abcd` top-level names
no sub-verb, so its command file is the rule's configured bare command and is
exempt from the file check.

The same rule sweeps a third directory, `agents/`, whose prompt files a harness
registers as invocable agents. It holds them to the table in
[§ The agent pages](#the-agent-pages) in both directions: every markdown file
directly under `agents/` has a row there, and every row there names a file that
exists, so an agent page added or removed without the same edit to the table
fails the record gate (iss-2610110456358519). A directory under `agents/` holds
an agent's fixtures and is not an agent.

The grain extends **inside** each row (spc-27, adr-40 decision 6): every surface
file in this directory carries a `## Sub-verbs` table recording, per verb, its
adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a non-assessment
verb) and whether it is `shipped` or `staged`. A verb with no sub-verb carries
the table with its header alone, so an empty table is a recorded fact rather than
a missing one, and a file without the table is a finding whatever its verb
registers. The rule checks each table against the committed command-tree
snapshot in both directions: a `shipped` row must be registered, a `staged` row
must not be, a registered sub-command must have a row unless it sits in a hidden
subtree (`launch smoke-pages` and `dashboard serve` today; `loadSnapshotSubVerbs`
in `internal/core/lint/subverbs.go`), and a sub-command-bearing verb cannot lack
a file entirely. The bare command, and any host-delegated
surface (a page with no verb; the list is empty today), are exempt from that
comparison by explicit configuration, never silently, and from nothing else:
their tables are still required and format-checked.
Operator-internal verbs are absent from this registry by design.

The surface-grain `Status` enum stays two-valued: there is no `partial`, because
the sub-verb rows carry that granularity, so a row may honestly read `shipped`
while its table shows which sub-verbs are still `staged`.

What none of this checks is prose. The tables pin rows and sub-verbs to the
command tree, and every `surface_coverage` finding opens by naming itself the
row-level presence check, so a green run is never read as a statement that a
chapter is correct (iss-246).

## The generated appendix

A chapter's shape — its verbs' flags and sub-verbs — is derived, never
hand-written ([adr-2609231028044006](../../decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived.md),
invariant 18 in [`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)).
Every chapter in this directory ends with a generated appendix between two
marker comments, composed from the same walk of the command tree that builds the
compatibility snapshot. This register's **Command** and **File** columns are
what map a chapter to its commands, so a chapter with no row here, or a row
naming a chapter that does not exist, is refused by name, as is a chapter
without its markers. The generator still writes every other chapter and then
exits 1 naming each refusal, and the drift test fails the same way. A chapter whose
command the tree does not register carries one sentence instead, so no chapter
lacks the block: a staged design target's says there is no shipped surface, and
a host-delegated command's, whose row here reads shipped, says it ships as a
command page with no verb, so the block never contradicts its own row.

Two tests in `internal/surface/cli` hold it, and both run in `go test ./...`,
so in `make preflight` and in CI. `TestSurfaceAppendicesMatchCommandTree`
regenerates every appendix and fails naming the chapter and each missing or
stale line. `TestSurfaceChapterProseStatesNoShape` fails, anywhere above the opening
marker, on a flag the command tree registers (long, or its single-dash
shorthand), on a sub-verb's command path written as an invocation (backticked,
fenced, or prefixed with `abcd ` or `/abcd:`), and on a backticked name of one
of the chapter's own sub-verbs. Another program's flag and the same words as
plain English are prose. Only the `## Sub-verbs` table and its standard note are exempt,
because the rule above checks them. Exit codes, output fields and what a verb
refuses stay prose and stay a review-grain claim.

**Adding a surface.** Add its row to the table above and its chapter to this
directory, end the chapter with the two marker lines (the generator's refusal
prints them), and run `go generate ./internal/surface/cli`. That regenerates the
snapshot and every appendix. Keep flags and sub-verb spellings out of the prose:
say what the surface is for, and let the appendix say how it is spelled.


## How the help lists the verbs

`abcd --help` lists the verbs a person runs under five labelled groups, with one
line above them saying that `abcd --help --agent` expands the list. With
`--agent` the help renders two blocks: the person's groups, then the verbs agents
and hosts call, each line naming the command page an agent reads next (itd-146).
Each block sizes its own name column, so the person's groups read the same in
both forms, and asking for the expanded list any other way, on the bare call or
through the help verb, is refused naming the one spelling that works.
Every other command's help is the framework's own, except that it opens with
the command's sentence (the section below).

| Block | Group | Verbs |
|---|---|---|
| people | set-up | `ahoy`, `dashboard`, `update`, and the framework's `help` and `completion` |
| people | records | `build`, `capture`, `decide`, `ideate`, `intent`, `library` |
| people | checks | `lab`, `reading` |
| people | portability | `disembark`, `embark` |
| people | release | `launch` |
| agents and hosts | — | `banlist`, `docs`, `drain`, `guard`, `guard hook`, `history`, `ideate record`, `identity`, `implement`, `inbox`, `intent audit ingest`, `intent consistency ingest`, `intent prepass`, `lint`, `mode`, `peers`, `reflect`, `report`, `rules`, `scribe`, `site`, `spec`, `statusline` |

The placement is presentation. No verb is hidden, renamed, moved or nested by
it, every verb runs the same whichever block lists it, and the group titles
carry no adr-40 bucket meaning. `version` is the root's `--version` flag
(itd-2609212130136102), not a verb, so it is in neither block, and bare `abcd`
shows the installed version as the board's last line (itd-2610090831227812).
`changelog` is in neither block either: it merged into `launch` as its preview,
and `abcd launch --dry-run` renders the cut it showed beside the bundle report
(itd-2610090831227812). The product
thinker placed every verb on 2026-10-09 by who types it: a command a person
needs to do their job is a person's, and only a command no person types is
listed with the agents (itd-2610090831227812; its spec's verb audit cites each
ruling). Neither the help nor the plugin menu caps the person's list, and the
two list the same commands: the command pages whose `block:` is `people` that
back a verb are the verbs this help lists under the person's groups. `rules`
and `spec` sit with the agents, their lines naming `commands/ahoy.md` and
`commands/intent.md`; the marker block's `abcd rules` still runs, because the
verb is listed, not removed.

It is gated like every other surface claim. The committed command-tree snapshot
records each visible top-level verb's group and each listed entry's block, so a
regroup shows in its diff; the drift test and the release gate's stale-surface
refusal name every verb whose placement moved without a regeneration. A test in
`internal/surface/cli` fails on a visible top-level verb registered with no
group, and another holds each command page's `block:` frontmatter to the tree.
Two more hold the plugin menu to the same placement: every page says `block:`
as `people` or `agents`, every agents page carries the host's
`user-invocable: false`, which leaves it off a person's `/abcd:` menu and keeps
its name, and no people page carries that key; and the people pages that back a
verb are exactly the verbs the person's groups list.
A regroup is not a break: the surface diff never reads the placement, because it
changes no invocation.

## The sentence every verb opens with

Every visible verb and sub-verb carries one sentence naming what it does, what
it writes (or that it writes nothing), and when it refuses, in that order: the
doing clause, a colon, a writing clause opening with "Writes", a semicolon, and
a refusing clause opening with "refuses" or "never refuses", at most 160
characters, under the [writing-style guide](../../../../docs/reference/writing-style.md)
(itd-2609212113220149). The sentence is declared once, in the surface manifest
(`internal/core/surface/sentences.go`), and rendered from there byte for byte:
it is the line every command list prints (a parent's list, the root's groups
and the agents block), the first line of the verb's own `--help`, and, for a
top-level verb with a plugin page, that page's `description:`. The committed
command-tree snapshot records it, `go generate ./internal/surface/cli` writes
the pages' descriptions from it, and the generated CLI reference carries it,
which puts it under the docs lint.

A page that carries a host workflow its bare verb does not describe adds one
"use when" clause after the sentence in its `description:` and nowhere else,
because the host chooses which page to load from that description. The clauses
are declared in one map in `internal/surface/cli/sentences.go`, keyed by page
name, each one line opening "Use when", at most 300 characters. Two pages
declare one: `library`, which carries the register and consult workflows, and
`ahoy`, whose `install` carries the repository preparation. A page outside the
map has the bare sentence as its description.

It is gated. A test in `internal/surface/cli` walks every visible command and
fails naming the verb and the defect when a sentence is missing, lacks a clause,
runs past the cap, or differs between the manifest, the command list, the help
and the page (the page's declared use-when clause aside); a synthetic tree
proves it names each defect. A reworded sentence
is not a break: the surface diff never reads it, because it changes no
invocation. Adding a verb therefore adds its sentence to the manifest in the
same change, and the regeneration carries it to every place it appears.

## Bare invocation

Typing a verb with no arguments is how a person finds out where they stand
without having to remember a sub-command name. Most verbs answer with a
read-only render of their own state, and close on the next move where there is
one to name.

It is a convention rather than a universal, and the exceptions are where the
tree does not yet meet its own discipline. Seven parents print usage with no state
at all: `disembark`, `docs`, `embark`, `guard`, `history`, `ideate`, and `scribe`. Bare
`abcd launch` refuses with a hint to pass `--dry-run`. Bare `abcd decide` refuses
because its one operand is the quoted title it mints a record from, and bare
`abcd build` refuses as a usage error because its one operand is the intent it
starts a run for; the run's state renders through `abcd implement status`. Bare
`abcd reflect` prints its usage, because its one operand is the cut release
whose retrospective seed it renders. Bare
`abcd drain` is the run itself, one move per invocation, so it is not a render;
what a drain would do renders through its dry run. Bare `abcd identity` and bare `abcd ahoy remote` list their sub-verbs,
because their reports are `abcd lint identity` and `abcd ahoy --remote`. Bare `abcd report` opens the editor on a
terminal and refuses anywhere else, because it files a report rather than
rendering one, and bare `abcd statusline` renders abcd's row only in a managed
repository, where it needs none of the payload the harness hands it on stdin,
and prints nothing of its own anywhere else. And `abcd
update` is a mutating fetch-verify-swap rather than a render at all. This
paragraph is the one enumeration of the exceptions; the chapters point here
rather than restating it. Each exception is also recorded with its reason in
the front door's exception table, and a test runs every other top-level verb
bare and fails when one renders no state, so a verb added later is either a
render or a recorded exception, and this paragraph must name every exception
the table holds.

## Operator-internal verbs

Some verbs the binary registers carry no `commands/*.md` surface and no row
above, by design: the `surface_coverage` registry is the user-facing set, and
these are reached from the CLI or from the hook manifest alone. Each is listed
here with the record that delivered it, so the two tables together name every
verb the binary registers apart from the framework's own `help`.

| Verb | What it is | Delivered by |
|---|---|---|
| `rules` | Renders the active rule set; a positional `DOMAIN` scopes to one. Read-only diagnostics over the hook-driven rule injection. | itd-3 (the modular rules loader); the loader it reports on is documented in [`05-internals/03-configuration.md`](../05-internals/03-configuration.md), and the verb in [The rules verb](#the-rules-verb) below |
| `spec` | The native spec store: bare invocation is a read-only status board, and `spec close` closes a spec and ships its linked intent (`planned/` → `shipped/`) only when no open spec is left naming it — an intent owns one or more specs, and `--remainder <slug>` mints the follow-on for a partial delivery in the same operation, carrying the closing spec's steps not marked landed. | itd-80 / spc-2 (intent lifecycle automation), adr-2609151513118583 (the 1:n relation); its lifecycle is documented in [`05-intent.md`](05-intent.md), and the verb in [The spec verb](#the-spec-verb) below |
| `hook` | Hidden from `--help`: five host hook entrypoints, live-wired from `hooks/hooks.json`. `prompt-router` injects the rules a prompt matches and `prompt-router-reset` clears the per-session ledger so they inject again; `session-end` stages the session's own transcript, `subagent-stop` stages a finished sub-agent's transcript with its lineage, and `session-start` files both away and says how many reports wait in the inbox ([`29-report.md`](29-report.md)). The pre-tool-use adapter is `guard hook`, under `guard`. | itd-3 (the prompt router), itd-89 / spc-4 (the transcript clock), itd-103 / spc-16 (the guard hook); the transcript entrypoints are documented in [`11-history.md`](11-history.md), and the rule injection the router drives, with `prompt-router`'s two outputs — the injected block and the `--json` envelope that names the active-domain set for a client that snapshots rules — in [`05-internals/03-configuration.md`](../05-internals/03-configuration.md#the-prompt-routers-output); the generated CLI reference omits both router entrypoints by design |
| `completion` | The CLI framework's generated per-shell autocompletion scripts. | No record: generated by the CLI framework, not designed here |
| `statusline` | The harness-invoked status-line render: the harness runs it on every refresh with its payload on stdin, and it prints abcd's row in a managed repository or runs the user's recorded previous status command everywhere else. `ahoy install` wires it; no user invokes it. | itd-200 / spc-70 (the presence badge); the row and the state behind it are documented in [`08-abcd.md`](08-abcd.md) |

### The rules verb

`abcd rules` renders the rule set the prompt router draws on, read-only. It loads
the bundled default domains, then the machine's `~/.abcd.noindex/rules.json`, then the
checkout's `.abcd/rules.json`, each layer replacing a field wholesale, and reads
the checkout from the root the loader resolves for the working directory.

- **Bare**, it renders every domain whose state is not dormant, under a
  `# abcd rules — N domain(s) active` heading, one `## NAME` block per domain. A
  domain an override names reads `## NAME (user override)` or
  `## NAME (repo override)`. With the kill switch set it prints one line naming
  every file that set it, and with no active domain it says so. The JSON form
  carries `disabled` and the `domains`, each with its `source` (`bundled`,
  `user` or `repo`), recall keywords and rules.
- **`abcd rules <DOMAIN>`** renders that one domain, matched case-insensitively,
  whatever its state and whatever the kill switch, so a dormant domain stays
  inspectable. An unknown name exits 2.
- **Diagnostics go to stderr, never stdout**, so the JSON form stays one
  document: a refused root, a domain skipped for carrying no rules, and each
  bundled entry of a guardrail domain an override's list leaves out.

### The spec verb

`abcd spec` addresses the spec store under `.abcd/development/specs/`, whose
lifecycle [`05-intent.md`](05-intent.md) describes. Both forms resolve the
checkout root first, so they reach the checkout's store from anywhere in the
tree, and outside a checkout they exit 2 with nothing read and nothing written.

- **`abcd spec`** is the read-only status board. It counts the open and the
  closed specs and lists each spec's id, status, slug and linked intent; the JSON
  form carries `open`, `closed` and the `specs` themselves. It takes no operand.
- **`abcd spec close <spc-N>`** moves the spec from `open/` to `closed/` and,
  when no open spec is left naming its intent, moves that intent from `planned/`
  to `shipped/`. A shared bundle spec reconciles every member it names, and the
  render reports each member's move. `--impact additive|breaking|fix` stamps an
  intent that declares no impact, and is accepted only at the close that ships
  it. `--remainder <slug>` mints the follow-on spec for the same intent in the
  same operation, carrying the steps not marked landed, so the intent stays
  planned; `--production-mode` stamps that minted remainder and is refused
  (exit 2, nothing written) without `--remainder`. A close the store refuses
  exits 2.
- **The close is gated first.** The close runs the doc-fidelity gate
  ([`10-docs.md`](10-docs.md)) over every intent it would ship, before anything
  moves, and a refusal exits 1 and names each reason; a `--remainder` close
  ships none, so it meets the coverage floor alone and needs no docs review.
  A close that ships an intent then makes its fidelity review owed: it mints
  an OWED receipt, parks its marker in the intent's Audit Notes and writes the
  review request under `.abcd/.work.local/reviews/`. A failed emit is a warning
  on stderr and the intent ships regardless. A re-run against an intent already
  shipped reports the receipt's state rather than a fresh obligation.

## The command files

The surface itself is [`commands/`](../../../../commands) at the repo root,
auto-loaded by compatible agent harnesses. Most markdown files there are slash
commands whose body instructs the host agent to invoke the `abcd` binary and
present the result: the markdown is the surface, the binary is the engine. Those
commands stay thin — they call `abcd <verb> --json` and format the result, and
never reimplement behaviour that belongs in the core.

Every page but the bare board backs a verb of its own name. Some carry a
workflow the host drives around the binary as well: the library page's
consulting and registering steps (search the corpus, read a document first,
choose its class and key) run in the host agent around the corpus verbs, and
the preparation workflow that brings an owned repository up to the conventions
runs from `ahoy.md`'s install section, around the binary's install. The two
host-delegated pages that once backed no verb, one for consulting the corpus
and one for registering a document, retired into the library page
(itd-2610090831227812).

The directory is **flat**, and that is load-bearing rather than tidiness. A
harness maps each `commands/` subdirectory to an extra namespace segment, so a
verb one level down in a subdirectory named after the plugin registers as
`/abcd:abcd:<verb>` — the plugin name twice — and every `/abcd:<verb>` this brief
documents is then an unknown command (iss-161). One file per verb, directly under
`commands/`:

<!-- index: commands -->
`abcd`, `ahoy`, `banlist`, `build`, `capture`, `dashboard`, `decide`,
`disembark`, `docs`, `drain`, `embark`, `guard`, `history`, `ideate`,
`identity`, `implement`, `inbox`, `intent`, `lab`, `launch`, `library`, `lint`,
`mode`, `peers`, `reading`, `reflect`, `report`, `scribe`, `site`, `update`.
<!-- /index -->

`abcd.md` is the bare `/abcd` status board; every other file is `/abcd:<verb>`.
The listing is generated, not hand-edited: `go generate ./internal/surface/cli`
writes the marked region from the contents of `commands/`, and the `index_drift`
record-lint rule fails while it differs, so a verb file added, renamed or
removed is followed by rerunning the generator, and a merge conflict inside the
markers is resolved the same way. Each entry is a file's name in `commands/`, so
a wrong entry is fixed by renaming the file, never by editing the list.

**This documentation lives here rather than in `commands/README.md` because the
loader registers every markdown file under `commands/` as a slash command** — with
no frontmatter requirement and no name exemption, exactly as the agent loader
treats `agents/` (iss-110). A readme beside the verbs
is therefore a spurious `/abcd:README` on every installed surface (iss-160), and
the only reliable fix is a home outside the auto-discovery root.

## No skills

**abcd ships zero skills** — the `/abcd:` surface is commands only, and there is
no `skills/` directory in the tree. Three workflows once shipped as skills, and
each mutates state, which the boundary rule makes command-shaped. Consulting
the sources corpus and registering a document in it write the corpus and its
ledger; they became host-delegated command pages and then retired into the
library page, whose verbs make every write (itd-2610090831227812). The third,
the preparation of a target repo, mutates that repo; its page retired into the
install section of `ahoy.md` (itd-2610090831227812). The
skill/command boundary is documented in
[`05-internals/08-skills.md`](../05-internals/08-skills.md).

`/abcd:grill` was likewise proposed as a skill. Its design target is promotion to
a sub-verb of `/abcd:intent`, since its mid-session glossary writes and
per-session output are command-shaped, and the `intent` parent ships no `grill`
sub-verb.

## The agent pages

Every markdown file directly under [`agents/`](../../../../agents) is a prompt a
harness registers as an invocable agent, and each has one row below. **Serves**
names the verb that builds the agent's input or validates what it returns, or
none when the host agent invokes it directly; **Documented in** is the chapter
or section holding its contract. The `surface_coverage` rule holds the rows to
the directory in both directions ([§ How much of this table a machine keeps
honest](#how-much-of-this-table-a-machine-keeps-honest)), so adding or removing
an agent page means adding or removing its row in the same change.

| Agent | Serves | Documented in | Purpose |
|---|---|---|---|
| [`cold-reading-comparative`](../../../../agents/cold-reading-comparative.md) | `reading` | [`23-reading.md`](23-reading.md) | Cold reading at the comparative position: how options of a candidate's shape ordinarily behave against each declared criterion |
| [`cold-reading-detection`](../../../../agents/cold-reading-detection.md) | `reading` | [`23-reading.md`](23-reading.md) | Cold reading at the detection position: where the shipped tree is in tension with the claim record |
| [`cold-reading-entailment`](../../../../agents/cold-reading-entailment.md) | `reading` | [`23-reading.md`](23-reading.md) | Cold reading at the entailment position: what a design commits to without stating it |
| [`cold-reading-widening`](../../../../agents/cold-reading-widening.md) | `reading` | [`23-reading.md`](23-reading.md) | Cold reading at the widening position: the configurations a design's construal admits that it has not committed to |
| [`docs-currency-reviewer`](../../../../agents/docs-currency-reviewer.md) | `docs` | [`10-docs.md`](10-docs.md) | Verifies each user-facing documentation claim against the code that implements it, as the release gate's semantic pass |
| [`graveyard-interpreter`](../../../../agents/graveyard-interpreter.md) | `disembark` | [`02-disembark.md`](02-disembark.md) | Says what a packed lifeboat's graveyard tried and why it was left behind |
| [`intent-auditor`](../../../../agents/intent-auditor.md) | `intent` | [`05-intent.md`](05-intent.md) | Audits a shipping intent's promise against the delivered code, or the brief and intents against each other |
| [`lifeboat-reviewer`](../../../../agents/lifeboat-reviewer.md) | `disembark` | [`02-disembark.md`](02-disembark.md) | Audits a packed lifeboat against its source repository and returns a verdict |
| [`planning-interviewer`](../../../../agents/planning-interviewer.md) | `intent` | [`05-intent.md`](05-intent.md) | Runs an intent's planning interview and edits the intent record after each answer |
| [`press-release-composer`](../../../../agents/press-release-composer.md) | `disembark` | [`02-disembark.md`](02-disembark.md) | Composes a lifeboat's press release from its packed brief, spine, and principles |
| [`principle-distiller`](../../../../agents/principle-distiller.md) | `disembark` | [`02-disembark.md`](02-disembark.md) | Distils durable principles from a packed lifeboat's decision record |
| [`question-drafter`](../../../../agents/question-drafter.md) | none | [§ Agents no verb dispatches](#agents-no-verb-dispatches) | Drafts one of abcd's own questions so it follows every asking rule and fits the rows limit |
| [`reflection-composer`](../../../../agents/reflection-composer.md) | `reflect` | [`09-reflect.md`](09-reflect.md) | Runs a cut release's retrospective interview and drafts its answers |
| [`release-changelog-composer`](../../../../agents/release-changelog-composer.md) | `launch` | [`04-launch.md`](04-launch.md) | Composes a release cut's changelog lines and release page from the records that shipped in it |
| [`ruthless-reviewer`](../../../../agents/ruthless-reviewer.md) | none | [§ Agents no verb dispatches](#agents-no-verb-dispatches) | Reviews a diff that builds and passes its checks, for correctness first |
| [`scribe`](../../../../agents/scribe.md) | `scribe` | [`32-scribe.md`](32-scribe.md) | Transcribes a returned reading and its dispositions into the ledger's declared shapes |
| [`security-reviewer`](../../../../agents/security-reviewer.md) | none | [§ Agents no verb dispatches](#agents-no-verb-dispatches) | Reviews a diff or design adversarially at a trust boundary |
| [`sota-researcher`](../../../../agents/sota-researcher.md) | none | [§ Agents no verb dispatches](#agents-no-verb-dispatches) | Researches the state of the art on one engineering question |

## Agents no verb dispatches

Most plugin agents under [`agents/`](../../../../agents) serve a verb, which
builds their input or validates what they return, and the chapter of that verb
documents them. Four serve no verb: the host agent invokes each directly, when
its `description`, a command page or abcd's asking rules say to, and reads the
report itself. Three of the four are called on by name. `build.md` and the
implement loop's validator roles dispatch the two reviewers, and `guard.md` and
`intent.md` send a question through the question drafter. Only
`sota-researcher` is named by no command page. In the binary they are roster entries: the bundled model-tier
proposal the `oracle-routing` offer of [`01-ahoy.md`](01-ahoy.md) renders places
the two reviewers at `frontier`, and the researcher and the question drafter at
`economy`. Each prompt declares `reads_untrusted_input: true`,
ships an injection canary under `agents/<name>/fixtures/`, and tells the agent
that everything it reads is data, never instruction.

- **`ruthless-reviewer`** reviews a diff that already builds and passes the
  project's checks, and stops to report a tree that does not. Its priorities,
  in order, are correctness, resource handling, error paths, API misuse and dead
  weight, then the project's own `AGENTS.md` rules. A finding is admissible only
  with a failure scenario, concrete inputs or state and the wrong result, and
  the verdict is SHIP or FIX FIRST.
- **`security-reviewer`** reviews a diff or a design adversarially at a trust
  boundary, starting from the boundaries `AGENTS.md` declares. A finding is
  admissible only with an attack path from an untrusted input to its
  consequence, and the verdict is APPROVE, BLOCK or NEEDS-INPUT, the last for
  what it could not establish. With no budget stated it reports within about
  twenty-five tool calls.
- **`sota-researcher`** researches the state of the art on one question, with
  web search and fetch among its tools. It anchors the date before it weighs
  recency, tiers every claim as evidence, consensus, contested, or anecdote and
  marketing, and cites only what it opened in the run. It returns a ranked list
  of recommendations, each with its source and tier, and a section on what it
  rejected.
- **`question-drafter`** drafts one of abcd's own questions before the agent
  running the interview asks it: handed the material to quote, the addressee,
  the decision and the defensible answers, it returns the host question tool's
  input and the rows each tab takes, counted as the question check in
  [`17-guard.md`](17-guard.md) counts them. Its asking rules and its row count
  are a block `cmd/asking-sync` renders from `question.Default`, so the
  prompt and the check read one statement of every limit. It puts nothing to
  the person itself and never marks an option recommended. The GRILL asking
  rules tell every agent running an abcd interview to draft through it.

`docs-currency-reviewer`, the reviewer the release gate runs as a semantic gate,
is documented in [`10-docs.md`](10-docs.md).

## Library and memory: two different jobs

abcd keeps two kinds of remembered material, and they do different jobs. The **library** is evidence you consult; **memory** is know-how that finds you.

| | Library (`/abcd:library`) | Memory (`/abcd:memory`) |
|---|---|---|
| What it holds | Outside material: papers, links, and the cited notes drawn from them | Short lessons learned while working, such as "this test is flaky under load" or "the person wants every question to carry an example" |
| Who adds to it | The person, deliberately, saying whether each item is confidential or public (abcd asks when they did not say) | Mostly agents, in the middle of a session |
| How it comes back | The person asks ("what do my sources say about pacing?") and gets answers that cite the item they came from | Brought into a session automatically when a prompt matches the words the note was saved with |
| Where it can go | Into a decision as a cited source; a confidential item is never named in anything public | Into the project's record, when a note is promoted so that every agent receives it |

A library answer can become a memory note ("source X says Y", with its citation), but the two are not one store with two names: the library is for what a decision may need to cite, and memory is for what an agent would otherwise relearn. The glossary entries [library](../glossary/core/library.md) and [memory note](../glossary/core/memory-note.md) define each term. The `library` command holds what the `source` and `memory` commands held, and the pages for consulting the corpus and registering a document folded into its page (itd-2610090831227812); memory notes are designed in itd-2610091918433290, and its open questions name where notes are stored and what an agent may write without the person's yes.

## Where to find related design

- **Plumbing internals**: [`05-internals/`](../05-internals)
- **Build sequence** (which surfaces ship in what order): [`06-delivery/01-build-sequence.md`](../06-delivery/01-build-sequence.md)
- **Verification matrix** (test coverage across surfaces): [`06-delivery/02-verification-matrix.md`](../06-delivery/02-verification-matrix.md)
