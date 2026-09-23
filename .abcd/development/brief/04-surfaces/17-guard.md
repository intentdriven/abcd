# `/abcd:guard` — Shell-Hazard Guard

A destructive shell command typed by accident costs a person hours: a force
push that erases a colleague's commits, an `rm -rf` with an unlucky glob, a
`git clean` over uncommitted work. `/abcd:guard` catches those before they run
and says what to run instead. It writes nothing, and it costs nothing to have
on: an unrecognised command passes through untouched.

Two things use it. A person or a script asks it about one command line. A
compatible agent harness asks it about every command the agent is about to run,
through a pre-tool-use hook. Both get the same answer from the same registry, so
what a person is taught and what an agent is stopped by cannot drift apart.

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
| `check` | gate | shipped |
| `hook` | — | shipped |

`check` takes the candidate on `--command` or on stdin, and stdin is the one to
prefer for a command line you did not type: the shell expands a double-quoted
`--command` argument before the guard starts, so a command substitution inside
it runs at check time, the one moment the check exists to prevent. `hook` reads
a host hook payload and maps the same decision onto the host's block/allow
protocol; it is wired from `hooks/hooks.json` rather than typed.

Bare `abcd guard` prints usage. Guard health lives where every other
install-state question is answered, on `abcd ahoy`.

## What a person gets back

Three verdicts, and they are the same decision reported two ways:

| Verdict | `guard check` | `guard hook` |
|---|---|---|
| `allow` | exit 0 | exit 0, silent |
| `warn` | exit 0, warning rendered | exit 1, warning on stderr |
| `block` | exit 1, why + successor rendered | exit 2, the host's blocking status, why + successor as the message |

A block names the entry, says in plain language what the command destroys, and
gives the safe command to run instead. That successor is what makes the guard
worth having rather than merely obstructive.

The exit codes are the contract, and the asymmetry in them is deliberate. On the
hook, only exit 2 stops anything; a warn exits 1 because a pre-tool-use hook that
exits 0 has its stderr discarded, so a warn returning 0 would run as if allowed
with nobody told (iss-231). A guard that cannot answer at all — an unparsable
command line, a registry with nothing left to check against, a registry switched
off — exits 1 on the hook and lets the command run, and exits 2 on `check` so
that a script never reads silence as clearance.

Either verb also speaks JSON, and that is the form the plugin page uses: a
verdict, and with it the entry that fired, its tier, why the command is
dangerous, and the safe successor. A `matches` list carries any further entries
the same line tripped, so a command hazardous in two ways reports both rather
than only the first; the rendered form says the same thing on an `also matched:`
line.

## Fail-open-loud

A broken guard never bricks a session and never silently stops protecting one.
The installed hook wraps the binary in a shim: the binary's own three statuses
pass through untouched, and anything else means the binary did not run at all,
which the shim reports as an unmissable `UNGUARDED` warning while letting the
command through.

The states that can independently be false are reported outside the session, on
`abcd ahoy`'s `guard:` line: whether the hook is installed, whether the binary it
calls is reachable, and whether a hazard registry is armed. A repo
`.abcd/guard.json` that will not load drops the repo's own overrides while the
bundled hazards stay armed, and that middle state is reported as itself rather
than folded into either extreme.

The two callers part company on exactly that file, deliberately. **On the hook,
the session keeps its protection:** the repo's overrides are dropped with a
notice on stderr, the bundled hazards still decide, and a hazardous command is
still stopped. Even an allow is made loud in that state, because a silent pass
would take the notice with it and nobody would learn the config is broken.
**On `check`, the verb refuses:** it names the file and the parse error, checks
nothing, and exits 2. The asymmetry follows from who is asking. The hook is
protecting a live session that will run the command either way, so protecting it
partly beats protecting it not at all; `check` is answering a person or a script
that asked a question, and a verdict drawn from half the registry they thought
they had is worse than being told the registry is broken.

## Turning it off is a diff

The hazard entries are bundled in the binary and merged with a repo's
`.abcd/guard.json`. That file is dedicated rather than a rules-loader domain, so
a rules kill switch can never silently disable a safety guard, and it must
declare `schema_version: 1` or the guard refuses to run at all rather than
running on a registry it cannot trust.

There is no flag, environment variable, or prompt that disarms the guard for a
session. The file is the only route, so switching the guard off lands in a diff
somebody reviews. What is not yet enforced is that the diff is *committed*: the
registry is read from the working tree, so an uncommitted edit takes effect on
the next command. The mitigation today is loudness rather than refusal — a
disabled registry makes every command it lets through carry an `UNGUARDED`
warning naming the file, and `abcd ahoy` reads `OFF`. Refusing a `disabled: true`
that is not in `HEAD` is a core-side change, tracked as an issue.

## What this guard is, and is not

The guard is a **mistake filter, not a security boundary**
([adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md)).
It catches a hazard typed by accident or reached through an ordinary wrapper. It
does not withstand an author trying to get a command past it.

That is a property of the layer rather than a gap in this implementation. The set
of programs that launch another program is open-ended: any binary that execs its
arguments is a wrapper, and a repository extends the set with one line in a
Makefile or a git alias. Three defects of exactly that shape have been filed
against three different enumerations in this package (gh-297, gh-299, iss-272),
which is the evidence rather than a coincidence.

So the guard is built to fail loud rather than to be complete. A hazard it cannot
resolve is warned about, not waved through. Anything needing an enforced boundary
needs a control at the execution layer — a sandbox, a permission system, a
restricted shell — with this guard in front of it to teach, never in place of it.
A missing wrapper name is a real defect in a mistake filter; it is not a silent
failure of a trust boundary, because there is no trust boundary here to fail.

## What an allow means

An allow means **no registry entry matched**, never that a command is safe.

The guard reads a command line the way bash does, so the obvious evasions are not
evasions. A hazard behind a wrapper the guard steps over is seen as itself. A
hazard behind a launcher it does not recognise is a **warn** naming the entry it
matched rather than an allow, because the guard cannot tell whether that program
runs the rest of the line. An unquoted glob is treated as producing whatever
literal it could produce, at every position an entry constrains, so a force push
spelled `git pus? --force` blocks. A command string handed to a shell is opened
and read. A git alias declared on the same command line is resolved, and the
command git would actually run is what gets checked. Where the reading is a
guess, over-blocking is the direction the guard takes.

What an allow still does not see is a hazard that never reaches command position
at all: one behind a wrapper flag the per-wrapper table does not name; a REST
path an entry names by its root segment when the host serves that API under a
prefix; a bare `$VAR` standing where the hazard would be inside a payload the
guard does read, because the guard sees the variable and not what the shell will
expand it to, and warning on every variable would bury the warnings that matter;
a payload inside a non-shell interpreter such as `python -c`, which is one
opaque token and today a silent allow; and any dangerous form no entry
describes. `abcd guard check --help` is the fuller statement of the same list,
kept beside the code that implements it, with a worked example for each and the
near-misses that *are* read spelled out beside them.

Coverage is what the registry names, and the registry grows from reality: someone
who sees something frightening captures it, and recurring captures are promoted
into the bundled defaults through the admission gate.

## References

- Plugin command: [`commands/guard.md`](../../../../commands/guard.md)
- Spec: [`spc-16`](../../specs/closed/spc-16-abcd-teaches-repo-agents-the-shell-commands-they-must-never.md)
- Intent: [`itd-103`](../../intents/shipped/itd-103-abcd-teaches-repo-agents-the-shell-commands-they-must-never.md)
- Install/health surface: [`01-ahoy.md`](01-ahoy.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd guard`

Sub-verbs: `abcd guard check`, `abcd guard hook`.

Flags: none.

### `abcd guard check`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--command` | string |

### `abcd guard hook`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
