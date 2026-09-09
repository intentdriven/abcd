# `/abcd:guard` — Shell-Hazard Guard

`/abcd:guard` decides whether a shell command is safe to run, against abcd's
hazard registry. It is **strictly read-only** — it performs zero writes, it only
answers. It is the execution-time half of itd-103; the teaching half is the rules
loader injecting the same registry's entries before shell-heavy work.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `check` | gate | shipped |
| `hook` | — | shipped |


- **`/abcd:guard check`** — evaluate one candidate command line and report the
  decision. The candidate comes from `--command "<line>"` or from stdin. The
  plugin command (`commands/guard.md`) passes it on **stdin**, inside a
  quoted-delimiter heredoc: a candidate interpolated into a double-quoted
  `--command` argument is expanded by the shell before the guard starts, so a
  command substitution inside it would run at check time — the one moment the
  check exists to prevent. `--command` stays for literals a human typed.
- **`abcd guard hook`** — the host adapter. It reads a pre-tool-use hook payload
  on stdin, applies the same decision, and maps it onto the host's block/allow
  protocol. It is invoked by `hooks/hooks.json`, not by hand.

Bare `abcd guard` prints command usage — it does **not** render a status board;
guard health is reported by `abcd ahoy`, which is where install/health state
lives for every other part of abcd too.

## The decision

Core (`internal/core/guard`) returns one of three verdicts, and the front doors
only format it:

| Verdict | `guard check` | `guard hook` |
|---|---|---|
| `allow` | exit 0 | exit 0, silent |
| `warn` | exit 0, warning rendered | exit 1, warning on stderr |
| `block` | exit 1, why + successor rendered | exit 2, the host's blocking status, why + successor as the message |

Exit 1 on the hook is loud and non-blocking, and it is why a warn does not exit
0 there: a `PreToolUse` hook that exits 0 has its stderr DISCARDED, so a warn
returning 0 would run as if allowed with nobody told (iss-231). Exit 2 is the
one status that stops anything. Every non-decision path — an unreadable
payload, an unparsable command line, a registry that will not load — takes the
same exit 1, which is the fail-open-loud contract stated below.

The two front doors differ in how a verdict is REPORTED, never in the verdict
itself. `guard check` trims a trailing newline off a candidate read on stdin, so
the same command can reach the two doors one byte apart; a here-document left
open takes the same block either way, because that state is resolved when the
input ends and not only when it crosses a newline.

Two verdicts have no registry entry behind them and are the guard's own voice,
reported under reserved ids no entry may claim: a word the tokenizer cannot
expand (`brace-expansion-unexpanded`) and a here-document whose delimiter line
never comes (`heredoc-unterminated`). Both are **blocks**. They are tokenizer
states, but they are not faults: bash runs both lines, so an error — which the
hook maps to fail-open — would hand the command through unguarded, and a quiet
allow would trust a `<<` the classifier has misread before (iss-184). The same
reading gives a trailing backslash a verdict rather than an error: it is parsed
as bash 3.2 parses it (dropped), the reading under which a hazard executes.

What stays unparsable — and so still fails open loudly on the hook — is exactly
one class: an unterminated quote (`'`, `"`, `$'…'`, or one in a here-document
delimiter word) in **command text**, which no shell runs either. Text inside a
here-document **body** is not command text, so an apostrophe in a document is
data. Getting that boundary right is what the class rests on: a body starts on
the line after the redirection even when that line ends in `&&` or `|` (bash
collects the bodies at the end of the physical line), and whether a `<<` opens a
document at all is decided by what ENCLOSES it. Inside a `$(( … ))` expansion or
a bare `(( … ))` command there is no redirection, so a `<<` there is a shift;
anywhere else a delimiter-shaped word opens a document, a `( cmd <<EOF )`
subshell included. Reading the bytes after the delimiter word instead — "does a
paren pair close right here?" — sees only the flattest shift: `$(( (1 << n) + 1
))` closes its sub-expression with a single `)`. Read either the other way and
document text lands in command position, where one apostrophe becomes an error
the hook hands through unguarded.

The delimiter is a **word**, not an identifier: `cat <<20`, `cat <<$D` and the
`<<20` of `let mask=1<<20` are all here-documents, because the shell tokenises
the redirection before any builtin sees an arithmetic expression. The word set
read is the conservative superset — a letter, a digit, `_`, or a `$`-led word,
whose value the guard cannot know, so the document never terminates and the
fail-closed block answers. What is left out is the residual: a delimiter bash
allows but this set does not (`<<!`, `<</tmp/x`), and a `<<` in an arithmetic
context the tokenizer does not model — bash's deprecated `$[ … ]` — where a
delimiter-shaped operand is read as a document and over-blocks. In both the
body claim above is the thing that gives: for a delimiter the reader does not
recognise, the lines below it are read as command text after all.

A third reserved id is a **warn**: `git-config-rewrite-unread`, raised when a
git command points at configuration whose body the guard cannot read —
`GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM`, `-c include.path=`, an `includeIf`,
or a `--config-env` naming a variable the command line does not set. git
rewrites its own subcommand from an alias found there, so what runs need not be
what is written; the directive is visible even though the body is not, and a
warn is what a visible directive with an unreadable value is worth.

A guard that cannot be **evaluated** — an unparsable command line, a registry
that will not load, a candidate too long to read, a registry switched off — is a
fourth case, and the two front doors part company on it deliberately:

- `guard check` exits **2**. A script that asked the guard a question must never
  read silence as clearance.
- `guard hook` exits **1** and warns loudly. Exit 1 rather than 0 is the whole
  point: a pre-tool-use hook that exits 0 has its stderr discarded, so the
  warning would exist and nobody would ever see it. Exit 1 is non-blocking, so
  the command still runs — a guard that cannot answer never stops a session.

## Fail-open-loud

The installed hook entry wraps the binary call in a shim. The binary's own three
statuses pass through untouched — 0 (allow), 1 (ran, could not decide, allowed
loudly), 2 (block). Anything else means the binary did not run at all: missing,
not executable, crashed, killed. The shim then allows the command and emits an
unmissable `UNGUARDED` warning of its own. A session is therefore never bricked
by a broken guard, and never silently unprotected either — and a binary that ran
and reported is never described as having failed to run.

The outside-the-session half is `abcd ahoy`'s `guard:` line, which reports the
things that can independently be false: whether the hook is installed, whether
the binary it calls is reachable, and whether a hazard registry is armed. The
registry check distinguishes two states, because `guard.Load` fails safe: a
repo `.abcd/guard.json` that does not load drops only the repo's overrides
while the bundled hazards stay armed — reported as `repo_overrides_dropped`, a
mild but loud state — whereas `registry_loadable: false` means no registry at
all, the only genuinely-unguarded registry state, which the embedded defaults
make unreachable in practice. Each fault also surfaces as a gap
(`guard.hook_missing`, `guard.binary_unreachable`, `guard.registry_unloadable`
for the dropped repo layer, `guard.registry_empty` for an absent registry).

## Registry and overrides

The hazard entries are bundled in the binary and merged with a repo's
`.abcd/guard.json` — a dedicated file rather than a rules-loader domain, so a
rules kill switch can never silently disable a safety guard. The file declares
its schema: `guard.Load` requires an explicit `schema_version` of 1 and refuses
a file without one, so `guard check` on a registry missing the field answers
`guard: unsupported schema_version: .abcd/guard.json must declare
schema_version 1, got 0` and exits 2. Past that, an entry key overrides one
field or declares a new hazard, and `{"schema_version": 1, "disabled": true}`
switches the guard off. The kill switch is the whole file's two lines, not the
`disabled` key alone: a bare `{"disabled": true}` is a refused registry, not a
disarmed one.

There is **no flag, environment variable, or prompt** that disarms the guard for
a session: the file is the only route, so the change lands in a diff. What the
file is *not*, today, is verified as committed — `guard.Load` reads the working
tree, so an edit takes effect on the next command, before review. The mitigation
is loudness rather than enforcement: a disabled registry makes every command it
lets through carry an UNGUARDED warning naming the file, and `abcd ahoy` reports
`OFF`. Closing the gap properly (refusing a `disabled: true` that is not in
`HEAD`) is a core-side change to `guard.Load`, tracked as an issue.

## What this guard is

The guard is a **mistake filter, not a security boundary** ([adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md)).
It catches a hazard typed by accident or reached through an ordinary wrapper —
the cases that actually cost people work. It does not withstand an author trying
to get a command past it.

That is a property of the layer, not a gap in this implementation. The set of
programs that launch another program is open-ended: any binary that execs its
arguments is a wrapper, GTFOBins catalogues hundreds for privilege escalation
alone and has no reason to list the ones that matter here (`nice`, `setsid`,
`stdbuf` grant no privilege and hide a hazard perfectly), and a repository
extends the set with one line — `make deploy`, `npm run release`, a git alias.
Three defects of exactly this shape have been filed against three different
enumerations in this package (gh-297 the interpreter set, gh-299 git's global
value flags, iss-272 the wrapper set), which is the evidence, not a coincidence.

So the guard is built to **fail loud rather than to be complete**: a hazard it
cannot resolve is warned about, not waved through. Anything that needs an
enforced boundary needs a control at the **execution layer** — a sandbox, a
permission system, a restricted shell, sudo's `NOEXEC` — with this guard in
front of it to teach, never in place of it. A missing wrapper name is a real
defect in a mistake filter; it is not a silent failure of a trust boundary,
because there is no trust boundary here to fail.

## What an allow means

An allow means **no registry entry matched** — never that a command is safe.

Since [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md)
a hazard behind a launcher the wrapper table does not name is **not** an allow:
when no entry matches a segment, the matcher re-runs from each later
command-position token, and a hazard found there is a loud warn naming the entry
it matched. Never a block — the guard cannot tell whether an unrecognised program
runs the rest of the line, and `rg git push --force docs/` is a search. That is
the fail-safe; the wrapper names below are what upgrade a warn to a precise
refusal.

A word bash rewrites before exec is read as bash reads it. An unquoted glob
(`*`, `?`, `[…]`) is a pattern bash expands against the working directory, so
at every position an entry constrains — the command name, the subcommand, a
flag or flag-value alternative — a literal the pattern *can* produce is treated
as produced: `git pus? --force origin main` is the force push whenever a file
called `push` exists, in a directory the guard cannot see, so it blocks. An
unconstrained position is never compared, which is why `ls *` and `git add
*.md` do not change; behind zsh's `noglob` nothing expands and the compare is
literal. Negation is bash's spelling, not Go's: `git clea[!x] -fd` is read as
the same `git clean -fd`, and lands on the `git-clean` entry, whose tier is
`warn` — so it warns rather than blocking, as every spelling of `git clean -fd`
does. What the negation glob decides is which entry the line reaches, not the
verdict that entry carries. At a FLAG position the pattern must also be
flag-shaped — its own literal prefix begins with `-` — and the scan stops at a
`--` operand terminator, because a flag constraint is offered every argument
rather than one position: without that, a bare `*` matched every long
alternative at once and `rm *`, `git push origin *` and `git commit -m msg *`
became blocks. `--forc?` and `--force*` spell the dash and still fire. The
compare is a floor: a glob inside an attached short value (`-XDELET?`), a glob
that hides the leading dash itself (`[-]-force`), extended globs (`?(…)`,
`*(…)`, `+(…)`, `@(…)`, `!(…)`),
POSIX classes (`[[:alpha:]]`) and a glob resolved by a directory tree
(`repos/o/*` for an API path) are not modelled, and a globbed *wrapper* name
(`sud? rm -rf *`) reaches only the fail-safe's warn. A globbed interpreter name
(`s? -c '…'`) is opened, because a miss there would be silent — and opened *in
addition to* that warn, never instead of it, since which program a pattern
expands to is a guess.

The same reading extends past the shell to the one program that rewrites its own
argv from its own arguments: git resolves an alias, and an alias can be declared
in the command line itself (`-c alias.p='push --force'`, `--config-env`, the
`GIT_CONFIG_COUNT`/`KEY_n`/`VALUE_n` triple, `GIT_CONFIG_PARAMETERS`). Those are
exactly the values the operand walk steps over to find the subcommand, so the
alias NAME reached the compare and its body did not. The values are now read:
when operand 0 names an alias the same command line declares, the command git
would actually run is checked against the ordinary entries, following a bounded
chain of nested aliases, and a `!`-prefixed body — which git hands to a shell —
is read as an execute-a-string payload. Two costs are taken deliberately. git
ignores an alias that shadows a builtin, so `git -c alias.push='push --force'
push` is a plain push that the rewrite refuses: an accepted false positive,
over-blocking being the fail-safe direction, and cheaper than a builtin table
that only a probe of the installed git could keep true. And the alias body is
split on whitespace where git uses its own quote-aware splitter. Configuration
delivered from a FILE stays unreadable — that is the warn above, not a match.

What an allow still does not see is a hazard that never reaches command position
at all:

- a command string handed to an interpreter (`eval`, `sh -c`) — read, in fact,
  along with the `su -c` / `runuser -c` / `script -c` / `flock -c` family; what
  is not read is a payload the tokenizer cannot resolve, which warns;
- a hazard inside a NON-shell interpreter's payload (`python -c`, `perl -e`) — one
  opaque token the tokenizer cannot read: a silent allow, not a warn. The silent
  allow is the recorded posture (iss-315, resolved) and no open record tracks a
  warn: warning on every such payload is the storm adr-42 avoids;
- one launched through a known wrapper carrying a value-taking flag the matcher's
  per-wrapper table does not name: `sudo -u bob <hazard>` is seen, the bundled
  short form `sudo -Hu bob <hazard>` reaches only the fail-safe's warn, not the
  entry that names it (`bob` is read as the command). The miss is a non-match,
  never a false block;
- an API path an entry names by its ROOT segment when the host serves that API
  under a prefix: a GitHub Enterprise Server install mounts the same endpoints
  under `/api/v3/`, so `gh api -X DELETE https://ghe.example/api/v3/repos/o/r`
  is not seen. The `https://api.github.com/…` URL form *is* — an operand is
  normalised to its path (scheme, host, query, fragment dropped) before the
  depth check. Matching a root segment wherever it appeared would close the
  remainder and falsely refuse `DELETE /teams/{id}/repos/{owner}/{repo}`, which
  removes a repository from a team and destroys nothing;
- one whose enclosing command carries flags AFTER a substitution — `$(…)` and
  backticks are both followed into command position now, but a substitution
  written before an enclosing command's trailing flags truncates them, so
  `cd s && rm $(true) -rf *` reads as neither form's hazard (the deep gap
  iss-148 still tracks);
- a dangerous form no entry describes.

A wrapper's own arguments are stepped over with it, including the mandatory
operand in `timeout DURATION COMMAND` (iss-148) and in `chrt PRIORITY`,
`taskset MASK`, `flock FILE`, `chroot DIR`. The set is `sudo doas command env
nohup time xargs timeout exec nice setsid stdbuf ionice eatmydata proxychains
chrt taskset unshare nsenter flock chroot runuser busybox noglob nocorrect` —
the last two being zsh's precommand modifiers, which take no options of their
own, so the token after either is command position; missing from the set, a
Tier-1 blocker behind `noglob rm -rf *` never reached command position and
evaded to a mere warn (iss-2608270655497992). The set is an
upgrade, not the safety property: every per-wrapper flag list is derived by
probing the installed binary rather than by reading its `--help`, which
contradicts its own parser often enough to have cost a live bypass (gh-299).

Coverage is what the registry names. The registry grows from reality: a
facilitator who sees something scary captures it, and recurring captures are
promoted into the bundled defaults through the admission gate.

## References

- Plugin command: [`commands/guard.md`](../../../../commands/guard.md)
- Spec: [`spc-16`](../../specs/closed/spc-16-abcd-teaches-repo-agents-the-shell-commands-they-must-never.md)
- Intent: [`itd-103`](../../intents/shipped/itd-103-abcd-teaches-repo-agents-the-shell-commands-they-must-never.md)
- Install/health surface: [`01-ahoy.md`](01-ahoy.md)
