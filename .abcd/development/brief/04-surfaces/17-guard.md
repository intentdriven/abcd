# `/abcd:guard` — Shell-Hazard Guard

A destructive shell command typed by accident costs a person hours: a force
push that erases a colleague's commits, an `rm -rf` with an unlucky glob, a
`git clean` over uncommitted work. `/abcd:guard` catches those before they run
and says what to run instead. It writes nothing, and it costs nothing to have
on: an unrecognised command passes through untouched.

Two things use it. A person or a script asks it about one command line. A
compatible agent harness asks it about every command the agent is about to run,
through a pre-tool-use hook. For a command run in the directory it is asked
from, both get the same answer from the same registry, so what a person is
taught and what an agent is stopped by cannot drift apart. The one place they
part is a host's per-call working directory, described under "Where the command
runs".

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

The check takes the candidate as the value of its command flag or on stdin, and
stdin is the one to prefer for a command line you did not type: the shell
expands a double-quoted flag value before the guard starts, so a command
substitution inside it runs at check time, the one moment the check exists to
prevent. It takes no positional argument. The hook adapter reads a host hook
payload and maps the same decision onto the host's block/allow protocol; it is
wired from `hooks/hooks.json` rather than typed.

Bare `abcd guard` prints usage. Guard health lives where every other
install-state question is answered, on `abcd ahoy`.

## What a person gets back

Three verdicts, and they are the same decision reported two ways:

| Verdict | Asked by a person or a script | Asked by the host hook |
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
off — exits 1 on the hook and lets the command run, and exits 2 on the check so
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
than folded into either extreme. The three states — clean, repo layer dropped,
no registry at all — are decided once, in the core, and every caller formats
the same answer.

The two callers part company on exactly that file, deliberately. **On the hook,
the session keeps its protection:** the repo's overrides are dropped with a
notice on stderr, the bundled hazards still decide, and a hazardous command is
still stopped. Even an allow is made loud in that state, because a silent pass
would take the notice with it and nobody would learn the config is broken.
**On the check, the verb refuses:** it names the file and the parse error, checks
nothing, and exits 2. The asymmetry follows from who is asking. The hook is
protecting a live session that will run the command either way, so protecting it
partly beats protecting it not at all; the check is answering a person or a script
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
somebody reviews, and the diff must be *committed* before it counts. An edit
that weakens the registry — switching it off, or changing a blocker's tier or
pattern — is refused until `HEAD` carries it: the committed registry stays in
force, the hook announces the refused edit on every command, and the check
refuses to answer. Where git cannot say what `HEAD` carries, the edit is refused
too. An edit that only adds or tightens a hazard needs no commit. Once a
switch-off is committed, every command it lets through carries an `UNGUARDED`
warning naming the file, and `abcd ahoy` reads `OFF`.

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

## Where the command runs

A host whose shell tool takes a per-call working directory passes it as
`tool_input.workdir`, resolved against the session directory
(iss-2609212142557657). The command is checked against the registry of the
repository it runs in as well as the session's, and the stricter verdict wins,
so a workdir can add a hazard and never remove one. A workdir is never read as
a `cd`: a probe of the one host with the field found that a missing workdir
fails the call and runs nothing, so no failed-cd hazard exists. A malformed
workdir is refused with the blocking status.

The rule that keeps this sound is stated beside the code
(`internal/core/guard/workdir.go`): a host that falls back to the session
directory when the workdir cannot be entered must not pass the field at all, and
its adapter folds the workdir into the command as a `cd` instead, which the
guard already reads as the failed-cd hazard it then is. No adapter in this
repository sends the field yet, so the path is reached only by a host that does.

Here the two callers can give different answers. The check has no workdir: it
reads the registry of the directory it runs in and nothing else. The hook also
reads the registry of the repository the workdir names, so a command the check
allows in one repository can be warned about or blocked by the hook when its
workdir is another.

## What an allow means

An allow means **no registry entry matched**, never that a command is safe.

The guard reads a command line the way bash does, so the obvious evasions are not
evasions. A hazard behind a wrapper the guard steps over is seen as itself. A
hazard behind a launcher it does not recognise is a **warn** naming the entry it
matched rather than an allow, because the guard cannot tell whether that program
runs the rest of the line. An unquoted glob is treated as producing whatever
literal it could produce, at every position an entry constrains, so a force push
spelled `git pus? --force` blocks. A command or process substitution, unquoted
or inside double quotes, is followed into command position, and the words
written after one stay the
enclosing command's, so `rm $(true) -rf *` is read as `rm -rf *`. An unquoted
brace group is expanded as bash expands it and every word it produces is
checked, so `mkdir -p foo/{a,b}` passes and `git push {--force,} origin main`
blocks; a group past the expansion cap is refused rather than read in part. A
command string handed to a shell is opened and read. A git alias declared on the same command line is resolved, and the
command git would actually run is what gets checked. In a repository with more
than one worktree, a stash or pop that does not name its entry is warned about,
because the stash stack is shared across worktrees. Where the reading is a
guess, over-blocking is the direction the guard takes.

What an allow still does not see is a hazard that never reaches command position
at all: one behind a wrapper flag the per-wrapper table does not name; a REST
path an entry names by its root segment when the host serves that API under a
prefix; a bare `$VAR` standing where the hazard would be inside a payload the
guard does read, because the guard sees the variable and not what the shell will
expand it to, and warning on every variable would bury the warnings that matter;
a hazard nested more than eight double-quoted substitutions deep; a payload inside a non-shell interpreter such as `python -c`, which is one
opaque token and today a silent allow; and any dangerous form no entry
describes. The check's own help text is the fuller statement of the same list,
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

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

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
