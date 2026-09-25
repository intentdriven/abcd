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
| 7 | `/abcd:memory` | shipped | Curate what the project knows from outside sources, and query it | [`07-memory.md`](07-memory.md) |
| 8 | `/abcd` | shipped | Find out where you are, or what one record id is and what to do with it | [`08-abcd.md`](08-abcd.md) |
| 9 | `/abcd:reflect` | staged | Compose a phase retrospective from its audit receipt (design target, itd-24) | [`09-reflect.md`](09-reflect.md) |
| 10 | `/abcd:docs` | shipped | Find documentation that has gone stale, and maintain the citation baseline | [`10-docs.md`](10-docs.md) |
| 11 | `/abcd:history` | shipped | Keep session transcripts as a local, redacted corpus this project can study | [`11-history.md`](11-history.md) |
| 12 | `/abcd:version` | shipped | Know which abcd this is, how it was installed, and whether it is behind | [`12-version.md`](12-version.md) |
| 13 | `/abcd:consult` | shipped | Ask the local sources corpus what prior work says, and record what it changed | [`13-consult.md`](13-consult.md) |
| 14 | `/abcd:ingest` | shipped | Put a document or URL into the sources corpus with its reference metadata | [`14-ingest.md`](14-ingest.md) |
| 15 | `/abcd:prepare-this-repo` | shipped | Bring an owned repo up to abcd's conventions (interim bridge until abcd manages repos directly) | [`15-prepare-this-repo.md`](15-prepare-this-repo.md) |
| 16 | `/abcd:lint` | shipped | Check whether this repo still conforms to the working conventions | [`16-lint.md`](16-lint.md) |
| 17 | `/abcd:guard` | shipped | Find out whether a shell command is safe to run, and what to run instead | [`17-guard.md`](17-guard.md) |
| 18 | `/abcd:ideate` | shipped | Put a big, unproven idea through an admission gauntlet and record the verdict either way | [`18-ideate.md`](18-ideate.md) |
| 19 | `/abcd:identity` | shipped | Make every surface say the same thing about the project, and see the diff that would fix one | [`19-identity.md`](19-identity.md) |
| 20 | `/abcd:banlist` | shipped | Declare the names this repo must never publish, and stop them at commit time | [`20-banlist.md`](20-banlist.md) |
| 21 | `/abcd:update` | shipped | Complete a chosen update of the installed binary, verified and atomic | [`21-update.md`](21-update.md) |
| 22 | `/abcd:site` | shipped | Render the project website from the repository's own text, and gate what it publishes | [`22-site.md`](22-site.md) |
| 23 | `/abcd:reading` | shipped | Assemble what a cold reading may see, prove it, and validate what comes back | [`23-reading.md`](23-reading.md) |
| 24 | `/abcd:decide` | shipped | Mint a decision record with its id, date and skeleton, ready to write the decision into | [`24-decide.md`](24-decide.md) |
| 25 | `/abcd:worktree` | staged | Keep session and agent worktrees in a machine-scoped store rather than beside the checkout (design target — [itd-2609091014076309](../../intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)) | [`../05-internals/03-configuration.md` § The worktree store](../05-internals/03-configuration.md#the-worktree-store) |
| 26 | `/abcd:mode` | shipped | Say whose answer the agent loop is waiting on, so the status line and the board show it | [`08-abcd.md`](08-abcd.md) |
| 27 | `/abcd:implement` | shipped | Share one autonomous run between two sessions: claim a record before its lane, keep the second session inside its bounds, and compare the ways of dividing the work from the run log | [`27-implement.md`](27-implement.md) |
| 28 | `/abcd:peers` | shipped | See what the sibling worktrees and local branches hold before capturing, fixing or filing anything | [`08-abcd.md`](08-abcd.md) |
| 29 | `/abcd:report` | shipped | Tell abcd about a defect or propose an enhancement from a repository it manages, into an inbox in your own account | [`29-report.md`](29-report.md) |
| 30 | `/abcd:inbox` | shipped | Read the reports managed repositories filed, and promote one to a capture that names the sender only by its root-commit key | [`30-inbox.md`](30-inbox.md) |
| 31 | `/abcd:scribe` | shipped | Build the ledger scribe's context from the ledger alone, and ingest what it transcribed without letting it author anything | [`31-scribe.md`](31-scribe.md) |

## How much of this table a machine keeps honest

The **Status** column is machine-checked: the `surface_coverage` record-lint rule,
the row-level presence check over this index, asserts every `shipped` row has a backing surface (`commands/<name>.md` or
`skills/<name>/`) and every `staged` row has none — and, in reverse, that every
command file and skill directory has a row here. The bare `/abcd` top-level names
no sub-verb, so its command file is the rule's configured bare command and is
exempt from the file check.

The reverse sweep reaches those two directories and no others. The prompt files
under `agents/`, which a harness registers as invocable agents, are checked in
neither direction: nothing asserts that one has a row here, and nothing notices
when one is added or removed (iss-110).

The grain extends **inside** each row (spc-27, adr-40 decision 6): every surface
file in this directory carries a `## Sub-verbs` table recording, per verb, its
adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a non-assessment
verb) and whether it is `shipped` or `staged`. A verb with no sub-verb carries
the table with its header alone, so an empty table is a recorded fact rather than
a missing one, and a file without the table is a finding whatever its verb
registers. The rule checks each table against the committed command-tree
snapshot in both directions: a `shipped` row must be registered, a `staged` row
must not be, a registered sub-command must have a row, and a sub-command-bearing
verb cannot lack a file entirely. Host-delegated surfaces and the bare command are
exempt from that comparison by explicit configuration, never silently, and from
nothing else: their tables are still required and format-checked.
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
hand-written ([adr-2609231028044006](../../decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived-never-hand-authored.md),
invariant 18 in [`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)).
Every chapter in this directory ends with a generated appendix between two
marker comments, composed from the same walk of the command tree that builds the
compatibility snapshot. This register's **Command** and **File** columns are
what map a chapter to its commands, so a chapter with no row here, or a row
naming a chapter that does not exist, is refused by name, as is a chapter
without its markers. The generator still writes every other chapter and then
exits 1 naming each refusal, and the drift test fails the same way. A chapter whose
command the tree does not register — a staged design target, or a host-delegated
command with no verb — carries one sentence saying there is no shipped surface,
so no chapter lacks the block.

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


## Bare invocation

Typing a verb with no arguments is how a person finds out where they stand
without having to remember a sub-command name. Most verbs answer with a
read-only render of their own state, and close on the next move where there is
one to name.

It is a convention rather than a universal, and the exceptions are where the
tree does not yet meet its own discipline. Seven parents print usage with no state
at all: `disembark`, `docs`, `embark`, `guard`, `history`, `ideate`, and `scribe`. Bare
`abcd launch` refuses with a hint to pass `--dry-run`. Bare `abcd decide` refuses
because its one operand is the quoted title it mints a record from. And `abcd
update` is a mutating fetch-verify-swap rather than a render at all. This
paragraph is the one enumeration of the exceptions; the chapters point here
rather than restating it.

## Operator-internal verbs

Some verbs the binary registers carry no `commands/*.md` surface and no row
above, by design: the `surface_coverage` registry is the user-facing set, and
these are reached from the CLI or from the hook manifest alone. Each is listed
here with the record that delivered it, so the two tables together name every
verb the binary registers apart from the framework's own `help`.

| Verb | What it is | Delivered by |
|---|---|---|
| `changelog` | The deterministic, read-only emit of the next release cut — derived version, record set, guardrail, no prose. Nothing on the plugin surface runs it: `commands/launch.md`'s emit → compose → ingest orchestration runs `launch ship --json`, and names this verb only as the read-only preview of the same cut. `launch ship` is the write half. | itd-73 (derived versioning) and itd-67's changelog slice, both in `intents/planned/`; documented in [`04-launch.md`](04-launch.md) |
| `rules` | Renders the active rule set; a positional `DOMAIN` scopes to one. Read-only diagnostics over the hook-driven rule injection. | itd-3 (the modular rules loader); the loader it reports on is documented in [`05-internals/03-configuration.md`](../05-internals/03-configuration.md), which names no verb: the verb itself is documented only in the generated CLI reference and the repo's own conventions router |
| `spec` | The native spec store: bare invocation is a read-only status board, and `spec close` closes a spec and ships its linked intent (`planned/` → `shipped/`) only when no open spec is left naming it — an intent owns one or more specs, and `--remainder <slug>` mints the follow-on for a partial delivery in the same operation, carrying the closing spec's steps not marked landed. | itd-80 / spc-2 (intent lifecycle automation), adr-2609151513118583 (the 1:n relation); documented in [`05-intent.md`](05-intent.md) |
| `hook` | Hidden from `--help`: five host hook entrypoints, live-wired from `hooks/hooks.json`. `prompt-router` injects the rules a prompt matches and `prompt-router-reset` clears the per-session ledger so they inject again; `session-end` stages the session's own transcript, `subagent-stop` stages a finished sub-agent's transcript with its lineage, and `session-start` files both away and says how many reports wait in the inbox ([`29-report.md`](29-report.md)). The pre-tool-use adapter is `guard hook`, under `guard`. | itd-3 (the prompt router), itd-89 / spc-4 (the transcript clock), itd-103 / spc-16 (the guard hook); the transcript entrypoints are documented in [`11-history.md`](11-history.md), and the rule injection the router drives in [`05-internals/03-configuration.md`](../05-internals/03-configuration.md), which names no entrypoint of its own: the two router entrypoints have no documented home in this brief, and the generated CLI reference omits them by design |
| `completion` | The CLI framework's generated per-shell autocompletion scripts. | No record: generated by the CLI framework, not designed here |
| `statusline` | The harness-invoked status-line render: the harness runs it on every refresh with its payload on stdin, and it prints abcd's row in a managed repository or runs the user's recorded previous status command everywhere else. `ahoy install` wires it; no user invokes it. | itd-200 / spc-70 (the presence badge); the row and the state behind it are documented in [`08-abcd.md`](08-abcd.md) |

## The command files

The surface itself is [`commands/`](../../../../commands) at the repo root,
auto-loaded by compatible agent harnesses. Most markdown files there are slash
commands whose body instructs the host agent to invoke the `abcd` binary and
present the result: the markdown is the surface, the binary is the engine. Those
commands stay thin — they call `abcd <verb> --json` and format the result, and
never reimplement behaviour that belongs in the core.

The three host-delegated commands named below are the exception, and are
configured as such rather than being silently different: `/abcd:consult`,
`/abcd:ingest` and `/abcd:prepare-this-repo` back onto no verb of their own, so
`consult.md` and `ingest.md` invoke the binary nowhere at all and carry the
workflow itself, and `prepare-this-repo.md` calls other verbs on the way through.

The directory is **flat**, and that is load-bearing rather than tidiness. A
harness maps each `commands/` subdirectory to an extra namespace segment, so a
verb one level down in a subdirectory named after the plugin registers as
`/abcd:abcd:<verb>` — the plugin name twice — and every `/abcd:<verb>` this brief
documents is then an unknown command (iss-161). One file per verb, directly under
`commands/`:

<!-- index: commands -->
`abcd`, `ahoy`, `banlist`, `capture`, `consult`, `decide`, `disembark`, `docs`,
`embark`, `guard`, `history`, `ideate`, `identity`, `implement`, `inbox`,
`ingest`, `intent`, `launch`, `lint`, `memory`, `mode`, `peers`,
`prepare-this-repo`, `reading`, `report`, `scribe`, `site`, `update`,
`version`.
<!-- /index -->

`abcd.md` is the bare `/abcd` status board; every other file is `/abcd:<verb>`.
The listing is gated rather than trusted: the `index_drift` record-lint rule holds
the marked region to the contents of `commands/`, so a verb file added, renamed,
or removed without the same edit here fails the record gate.

**This documentation lives here rather than in `commands/README.md` because the
loader registers every markdown file under `commands/` as a slash command** — with
no frontmatter requirement and no name exemption, as `agents/README.md`
registering as an agent independently shows (iss-110). A readme beside the verbs
is therefore a spurious `/abcd:README` on every installed surface (iss-160), and
the only reliable fix is a home outside the auto-discovery root.

## No skills

**abcd ships zero skills** — the `/abcd:` surface is commands only, and there is
no `skills/` directory in the tree. `/abcd:consult`, `/abcd:ingest` and
`/abcd:prepare-this-repo` were once shipped as skills and are commands: each
mutates state — the sources corpus, its ledger, the target repo — which the
boundary rule makes command-shaped. They are **host-delegated** commands, with a
command page and no Go verb, so the workflow runs in the host agent. The
skill/command boundary is documented in
[`05-internals/08-skills.md`](../05-internals/08-skills.md).

`/abcd:grill` was likewise proposed as a skill. Its design target is promotion to
a sub-verb of `/abcd:intent`, since its mid-session glossary writes and
per-session output are command-shaped, and the `intent` parent ships no `grill`
sub-verb.

## Where to find related design

- **Plumbing internals**: [`05-internals/`](../05-internals)
- **Build sequence** (which surfaces ship in what order): [`06-delivery/01-build-sequence.md`](../06-delivery/01-build-sequence.md)
- **Verification matrix** (test coverage across surfaces): [`06-delivery/02-verification-matrix.md`](../06-delivery/02-verification-matrix.md)
