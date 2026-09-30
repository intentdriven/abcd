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
with nobody told (iss-231). A guard that cannot answer at all — a registry with
nothing left to check against, a registry switched off — exits 1 on the hook and
lets the command run, and exits 2 on the check so that a script never reads
silence as clearance. A command line the guard cannot split is the exception on
the hook: it is blocked (`command-unparsable`), not let through, because a line
the guard misreads may be one bash runs, and a pass would carry every hazard in
it past the guard. On the check it exits 2, like the rest.

Either verb also speaks JSON, and that is the form the plugin page uses: a
verdict, and with it the entry that fired, its tier, why the command is
dangerous, and the safe successor. A `matches` list carries any further entries
the same line tripped, so a command hazardous in two ways reports both rather
than only the first; the rendered form says the same thing on an `also matched:`
line.

## Taught before it is refused

The same registry drives a second plane that needs no hook at all. The rules
loader bundles a `SHELL` domain generated from the registry: one rule per entry,
carrying the command it matches, whether the guard refuses or warns, the why and
the safe successor, recalled by the commands the registry names (`rm`,
`git push`, `pkill`, …) and by shell work in general. A prompt about shell-heavy
work injects those rules before the agent acts, so a host without hook support is
still taught the safe form and a host with hooks is taught it before the guard
would have to refuse. An entry added to the registry is taught and enforced from
the same release, with no second edit. The registry taught is the one the guard
enforces in the repository: an entry the repository adds in its
`.abcd/guard.json` is taught by the same generator as the bundled ones, its
rule marked `(repo)` after its entry id, and a guard file the guard refuses is
named on stderr and never taught. How the domain is recalled, overridden and
silenced is the rules loader's
([`05-internals/03-configuration.md`](../05-internals/03-configuration.md)).

## The question gate

The hook also answers for the host's question tool (itd-2609212130146198), and
there it consults the waiting-on state rather than the hazard registry. In a
repository abcd manages, a question to the human asked while `abcd mode` reads
managed is refused with the blocking status, and the one-line refusal names the
two settings, `abcd mode product-thinker` and `abcd mode facilitator`, so the
agent says whom it is asking before it asks and the status-line badge shows
it. Once the state names somebody the question runs, and the hook writes a
`question_open` marker in `.abcd/.work.local/`; the rules loader's prompt hook
resets the state to managed on the next human message and clears the marker.
Where the badge does not show, a question is not the gate's business and runs
silently. A state file or marker the gate cannot read or write is not a
decision: the question runs and the hook says so on exit 1, the same
fail-open-loud contract below. So is a tier `abcd mode` cannot write — a
read-only mount, a directory the session's user does not own: the refusal's
remedy could not run there, so the gate probes that the verb could set the
state before it refuses, and where it could not, the question runs on exit 1.
The probe is a file created in the tier and removed again; one the remove could
not reach, because the tier turned unwritable in between or the process died,
is swept by the next probe. The manifest's pre-tool-use matcher names the
shell tool and the question tool, and nothing else.

## Fail-open-loud

A broken guard never bricks a session and never silently stops protecting one.
The installed hook wraps the binary in a shim: the binary's own three statuses
pass through untouched, and anything else means the binary did not run at all,
which the shim reports as an unmissable `UNGUARDED` warning while letting the
call through. The warning names the tool whose call went through unchecked —
shell commands, or questions through the host's question tool — read from the
hook payload. The shim reads that payload once, through a command
substitution, and hands the binary what the substitution kept: the payload
less its trailing newlines, and, under the bash 3.2 that serves as `/bin/sh` on
macOS, less any raw NUL byte, which that shell drops silently. Neither changes
what a conformant host means: its JSON carries no raw NUL, since JSON escapes
one as `\u0000`, and a trailing newline is whitespace to JSON. A payload that
does carry a raw NUL reaches the binary with the byte gone rather than as an
unreadable payload, and one over the size cap only by trailing newlines comes
under it.

The states that can independently be false are reported outside the session, on
`abcd ahoy`'s `guard:` line: whether the hook is installed, whether the binary it
calls is reachable, and whether a hazard registry is armed. A repo
`.abcd/guard.json` that will not load drops the repo's own overrides while the
bundled hazards stay armed, and that middle state is reported as itself rather
than folded into either extreme. The three states — clean, repo layer dropped,
no registry at all — are decided once, in the core, and every caller formats
the same answer.

One limit is not among those states, because it is never false: the guard's
reach. The manifest's pre-tool-use matcher hands the hook the shell tool and
the question tool and nothing else, so a call through any other tool never
reaches the guard — a file the host's own tools write or edit, a command a tool
from another extension runs — and nothing warns about it, since nothing
failed. It is the guard's standing scope, not a degradation, and the `guard:`
line does not report it. Whether the guard should adjudicate more than the
shell is a separate question with a real cost: every further tool class needs
its own hazard vocabulary, and a guard that refuses a tool it cannot reason
about is worse than one that says plainly what it covers.

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
spelled `git pus? --force` blocks. A bracket expression is read exactly only
where it is one plain set (`--for[c]e`); a class (`[[:lower:]]`), a set that
opens with `!`, `^` or `]` or holds `-`, `[` or a backslash, and a pattern
with a `]` after the set closes (a second set among them) are read as any
run of characters, since the guard sees the
line with its backslashes already taken out and `r[m\]]` is the set of `m`
and `]`. So `r[[:lower:]] -rf /` and `git clea[!n] -fd` are read as `rm` and
`git clean`, an over-block where no escape was written. A git long flag written short of its full
name is read as git reads it, as the one option that prefix can mean. A command
or process substitution, unquoted
or inside double quotes, is followed into command position, and the words
written after one stay the
enclosing command's, so `rm $(true) -rf *` is read as `rm -rf *`. What a
substitution prints is not in the line, so a word holding one is unknown and
fails closed in every role it could play, read every way it can be at once: led
by a dash it is every flag it could become — one standing alone, one taking a
value, a shell's `-c` — before the command as well as after it; after a value
flag it is that flag's value; as an operand it is one operand; and in command
position it is any program its known text allows, a shell, a wrapper and git
among them; where the only entries that fire are ones such a name can be, the
block is reported as the substitution's (`program-name-unknown`), and its way
past is to spell the program's name. Every reader of a word goes through that one rule, and a test holds
the package to it. Text beside one in the same word is also read as bash leaves
it when the output is empty. One nested past the depth the guard reads, one
holding a case command, or more of them where the program name could be than
the guard follows, is refused rather than left unread. A parameter expansion
(`$VAR`, `$1`, `$@`, `${VAR:-git}`) prints a value the line does not hold, so it
is an unknown word by the same rule: `--$VAR` is every flag it could become, and
`$GIT` as the program is any program its known text allows. A `${…}` ends at its
own `}`, where a nested `"` opens a string of its own, and a substitution in its
text runs and is read. A variable that is the whole word is read as an operand,
as a wholly-substituted word is (`git push origin "$branch"`), and a string
handed to a shell carries its variables for that shell to expand, where they
are read by the same rule, the string's own quotes applying to the value. What a variable carries in from an earlier command
is not read: as a script it is not a stream, and as the program's name it is not
a `pkill` or `killall`, whose entries name only the program and an operand, nor a
bare interpreter inside a string, because a variable is how ordinary commands
carry a program or a path between commands. A here-document body is data, but the substitutions the shell
runs in a body whose delimiter is unquoted are read as commands, and such a body
is read by the lines bash compares with its delimiter, joined across a trailing
odd run of backslashes. A body begins on the line after the one that opened
it, and a command or process substitution still open at that line's end holds
it back: the substitution's own lines run as commands, and the body begins on
the line after it closes. A document a substitution opens and never reads is
pending after the close in bash 5 and dropped in bash 3.2, which runs the
lines it would cover, so that line is refused as an unterminated document. A backtick's text is read after bash's own pass over
it, which drops a backslash before `$`, a backtick or a backslash (and, directly
inside double quotes, a `"`), so an escaped substitution between backticks is
read as the one bash runs. A payload that is wholly a substitution printing a
here-document the shell does not change (`sh -c "$(cat <<'EOF' … EOF)"`, or the
backtick spelling where no backslash stands between the backticks) is also
read as that document's text, joined to any text beside it in the word. The same
substitution unquoted runs the words its document splits into, the first and
last joined to whatever is written against it in its word, and is read as those
words wherever it stands and at every payload layer, so a document whose text is another such substitution is read
too; the words are never read again as a command line, as bash never reads
them. On a line where another command names IFS such an output is refused,
because the guard splits on the default IFS only and does not work out which
assignment reaches which expansion; a prefix assignment, which does not reach
its own command's expansion, is read as the default split. The guard follows two execute-a-string layers and refuses a payload
nested deeper, whatever it holds, because it has stopped reading it. An ANSI-C string ends at its
closing quote, found before any escape is decoded, and at its first NUL, as bash
ends it. An arithmetic expansion is an expression, not commands. A shell reading its script from a pipe, a here-document or a
here-string is refused, because what it runs is text the guard read as data, and
so is one handed the stdin device behind a pipe or a process substitution as its
script, a `source` of one, and a line longer than the guard reads. An unquoted
brace group is expanded as bash expands it and every word it produces is
checked, so `mkdir -p foo/{a,b}` passes and `git push {--force,} origin main`
blocks; a group past the expansion cap is refused rather than read in part. A
command string handed to a shell is opened and read. A git alias declared on the same command line is resolved, and the
command git would actually run is what gets checked. A commit or push that
moves `core.hooksPath` for itself is read as skipping its hooks, which is what
it does. A delete chained after `pushd` or `popd` is read as one chained after
`cd`. A `kill` handed what a process search prints, in a substitution or
piped into `xargs kill`, is read as the kill by name it is — through a group
and into one, whose every command is read as handed what is piped into it,
through a shell string that runs the search, and into a shell string `xargs`
runs or a pipe or redirect feeds, whose every command is read as handed its
input, or whose positional parameters or own text hold the search's output; out
of an unquoted here-document's substitutions into the command that reads the
document and every command its output is piped on to; and into a substitution in a command that reads a pipe, which runs
with that pipe as its input — and a `pkill` or
`killall` selecting by user, group or terminal, its value written apart or
attached, as selecting every session under the account; `pkill`'s signal name
is read as a signal first, in any case. Flipping a source-ledger line to
public citation is blocked, because the ledger records that a person chose to
cite the source, and that choice is the person's to make. In a repository with more
than one worktree, a stash or pop that does not name its entry is warned about,
because the stash stack is shared across worktrees. Where the reading is a
guess, over-blocking is the direction the guard takes.

A recursive delete is read by what it deletes. Of the filesystem root or the
home directory (`/`, `/*`, `~`, `$HOME`, `${HOME}`, each also with a trailing
`/`, `/*` or `/*/`, and the home's dotfiles `~/.*`, `$HOME/.*`, `${HOME}/.*`,
each also with a trailing `/`) it is a block wherever it stands, with or without `-f`. Of the directory the shell is
in or the one above it (`*`, `*/`, `.`, `..`, `./*`, `./*/`, `../*`, `.*`,
`./.*`, and `$PWD` or `${PWD}`, each also with `/*`) it is a warn, graded like
`git clean`, because that directory is usually the repository and emptying a
build directory the same way is ordinary work. Chained after a `cd` any
recursive forced delete blocks, as above. The target is compared as written,
before the shell expands it, so `$HOME` and `$PWD` are seen as those words. It
is first read the way bash reads its text: a backslash-newline inside a name
is dropped (`$HO\⏎ME` is `$HOME`); each word a brace group makes keeps the
variables its text holds, and a name runs on into the letters a list or a
sequence places after it (`{$HOME,x}`, `$HOME/{.*,}`, `$HO{ME,}`,
`$HO{M..M}E`); an expansion whose operator can leave the value as it is reads
as the variable itself — a default, an assignment or an error message
(`${HOME:-x}`), where the colon forms of a single parameter never print an
empty value (`${1:-dist}/` is `${1}/` or `dist/`), while those of `$@` and
`$*` test the parameter count and can (`${@:-x}/` is `/` after
`set -- "" ""`), and an empty word prints the empty text
(`${X:-}/` is also `/`), a trim or a pattern replacement (`${HOME%/}`, `${HOME#x}`,
`${HOME/x/y}`), any substring of a variable, which also reads as the root
and as nothing (`${X:1}`, `${PWD:0:1}`), while a slice of the positional
parameters or a part of one reads as those parameters (`"${@:2}"` and
`"${1:2}"` as `"$2"`), a case change or a transform (`${X^}`, `${X@P}`), and
a subscript read to its matching `]` with any text after it (`${HOME[x[0]]}`,
`${HOME[0]]}`, which the bash 3.2 of macOS prints as the value), the last
three also as nothing (`${A[0]}/` is also `/`); and an alternative, which prints its
word or nothing, reads as that word as written (`${X:+$HOME}`, `${X:+/}`,
`${X:+$HOME/*}`), including one the bash 3.2 of macOS reads at the first
operator after a subscript (`${X[0]]:+$HOME}`). Unquoted, the alternative's
word is split on whitespace and a substitution in it that prints nothing
drops out, as bash splits and drops them (`${X:+$HOME }`,
`${X:+$(true)$HOME}`). A word written as an ANSI-C or a locale string reads
as the text it decodes to (`${X:-$'\x2f'}`, `${X:-$"/"}`); a positional, a
special or an indirect parameter takes the same operators (`${1:-/}`,
`${#:+/}`, `${!X:-/}`, the last read as every target past its operator,
since the variable it names is not in the line); a parameter that can print
nothing at the top of a fresh shell — `$!` before any job runs in the
background, `$@`, `$*` and a positional one with no argument, `$_` after
`x=`, and `$-` under dash — also reads as the text beside it (`$!/`,
`"${1}"/` and `/$!` are `/`), and `$!` in a pattern as text of any length
(`${PWD%%$!*}` is `${PWD%%*}`); a replacement's pattern is
read both where bash 3.2 ends it and where bash 5 does, at a quoted `/`
(`${X/"/"*/$HOME}`); and on a line that names IFS — in any word or
anywhere in its text (`: $((IFS=1))`), or through an assignment target
that holds an expansion — an unquoted default's or
alternative's word, and an unquoted home, reads as every target, since the
fields bash splits it into rest on that IFS (`IFS=x; rm -rf ${U:-x/x}`,
`IFS=Uv; rm -rf $HOME/x`). bash sets the variable a target's value names,
so the rule reads whether a target holds an expansion (`$name`, `${…}`
with its case changes and transforms, `$(…)`, a backtick substitution),
never which bytes are written beside it: `(( ${a}${b} = 1 ))` names IFS
with a=I and b=FS. A target is the word before an assignment operator
(`=` or a compound one, spaced or not) or beside a `++` or `--`, read in
the raw text of each layer, so it is found in every position bash assigns
through one: an assignment word's name, a declaration's or `env`'s operand,
an eval'd string, every arithmetic context (`(( ))`, `$(( ))`, `$[ ]`, a
`for (( ))` header, a subscript, a substring offset, `[[ -eq ]]`), a string
an arithmetic context later reads (`let "$x = 1"`, an integer variable's
value), and `${!x:=1}`, which sets the name x holds. A subscript is not
the name (`a[$i]=x` names `a`); its body is arithmetic and read as such.
The builtins that take a name as an operand (`read "$x"`,
`printf -v "$x" 1`, `mapfile`, `getopts`, `wait -p`) are read over their
words, and so is a nameref's declaration (`declare -n r=$x`,
`local -n r`), whose value is the name a later plain assignment sets. The
rule stays off a test's comparison (`[ $a = b ]`), a word-leading `--`
(`git log --$fmt`) and the `--` that ends options, and over-reads on the
refusing side: `IFS=x rm -rf ${U:-x/x}`, and `read -p "$prompt" f;`,
`echo "$k = $v";` or `[ ! $a = b ];` before `rm -rf ${U:-x/x}`. A run of `/` written before the home names the
home (`/$HOME`). A trim that leaves the path above the home
(`${HOME%/*}`) blocks as the home does. Each target is also compared as a path
with its redundant separators taken out, since the kernel reads a run of
slashes as one, a `.` segment as the directory itself and the root as its own
parent (`//*`, `$HOME//`, `/./*`, `/../*`, `.//*`). A target that begins at
the root or the home has each `..` folded into the directory before it, as
the path reads lexically: `/tmp/../*` and `/tmp/x/../..` are the root, and a
`..` past the home climbs to a directory that holds the home, so `~/..`,
`~/../*` and `$HOME/../../*` read as the home and `~/../*/*` as `~/*`, while
`~/../x` stays a sibling. The kernel reads a `..` otherwise only after a
symlink, and the lexical reading is the one that blocks; a trailing `..` is
folded too, though rm refuses it. A target that begins at the working
directory is folded the same way: a `..` past `$PWD` or `${PWD}`, or past the
start of a relative path, is the directory above it, so `$PWD/../*`,
`./../*` and `x/../../*` warn as `../*` does, and `$PWD/x/../*` as `$PWD/*`;
a relative path whose `..` stays inside the working directory is compared as
written. A trailing `.` after such a `..` (`../.`) warns too, though rm
refuses it. Each target is also read the way its glob can expand: a run of
`*` is one `*`, which is what every shell without globstar expands `**` to
(with globstar it matches more), so `/**`, `~/**` and `~/../**` block as
`/*`, `~/*` and `~/../*` do; and a segment written with a leading `.` whose
glob can match the name `..` (`.?`, `.*`, `.[.]`, `.[!x]`, `..*`) reads as
`..` where a further segment follows it, since the bash 3.2 and `/bin/sh` of
macOS have no globskipdots and expand `~/.?/*` and `~/.*/*` to include
`~/../*`, and `/.?/*` to the root's entries. A bracket expression in such a
segment is read as the glob compare above reads it, so one it cannot decide
reads as able to match `..`: `~/.[[:punct:]]/*`, `~/.[].]/*`, `~/.[--.]/*` and
`~/.[\!.]/*`, which the bash 3.2 and `/bin/sh` of macOS expand to the
directory holding the home, block, and so does `~/.[a-z]/*`, which no shell
expands that far. That reading is an over-block on bash 5.3, which leaves
such a segment unexpanded, and on a directory of that name. The guard sees an
escaped `*`, `?` or `[` as the glob it would be unescaped, so `~/*\*` and
`~/.\?/*` block as `~/*` and `~/.?/*` do. A glob whose leading `.` is not written (`??`, `?.`, `[.]?`) never
matches a dot name and is compared as written, and a dot-glob as the final
segment (`~/.?`) is not read as `..`, since rm refuses an operand whose last
segment is one; `~/.*` blocks as the home's dotfiles. A glob or a `..` inside
quotes is read the same way, so `'/**'` and `"~/.?"/*` block as `"/*"` and
`'~/*'` do, over-blocking a name that holds those characters.

What an allow still does not see is a hazard that never reaches command position
at all: a word that is wholly a command substitution or a variable standing
where a flag would be, which is read as an operand because that is how a commit
message or a branch name is spelled every day; a delete target printed whole by a
substitution (`rm -rf $(echo /)`), which is read by its known text because that
is how an everyday delete names what it removes (`rm -rf $(find . -name
'*.pyc')`); a target spelled any other way than the words above (`rm -rf
"$DIR"/*` with `DIR` unset, `rm -rf /?*`), a default's own word, which bash
prints only when the variable is unset (`rm -rf ${DIR:-$HOME}`, and
`${X[0]]-$HOME}`, which the bash 3.2 of macOS reads as a default after the
subscript), a `..` after a symlink, which is read past lexically (a link to
the root under a named directory), or after a segment holding a variable,
which is not folded (`/tmp/$X/../../*` is the root with `X` unset), a `..`
past the home followed by a glob other than `*` (`~/../?*`, as `/?*`), a
relative `..` that stays inside the working directory (`x/../*`, the
directory `*` names), a `..` after a `~user` home, whose depth is not known
(`~root/../../*`), an
alternative nested more than three deep, and a substring of `$PWD` that
prints the root (`${PWD:0:1}`), which warns as `$PWD` does; one behind a wrapper flag the per-wrapper
table does not name; a REST
path an entry names by its root segment when the host serves that API under a
prefix; an IFS the shell already holds when the line starts, or gains during the line
through a name the guard does not read (a sourced file, a nameref set before
the line, or an operator the line does not write: a value built from
expansions or a command's output that an arithmetic context evaluates, as in
`x=$(cmd); : $((x))`, or a decrement written as its own word in such a
string, as in `n="1 + --$x"` for an integer `n`),
since every line is read from the default IFS; a pid list a kill reads through a variable or a file, or from a `ps |
grep` chain;
a payload inside a non-shell interpreter such as `python -c`, which is
one opaque token and today a silent allow; and any dangerous form no entry
describes. Nor does an allow see what a variable carries in from an earlier
command: a pid list (`p=$(pgrep make); kill $p`), a stream path handed to a
shell, shell text run through `eval "$X"` or placed in a string a shell runs,
or `pkill` or `killall` as a variable's value standing as the program with an
operand (`$P make`), because reading each would refuse the ordinary commands a
variable carries a value for; whether to is an open ruling. Nor does an allow see what a
lone substitution prints when it stands as the whole command (`$(cat msg.txt)`,
`$(date)`): such a name can be any program, but with no operand after it no
entry matches, so it allows by the posture above, where an allow means no entry
matched. What a substitution prints is read only for the exact shape `cat
<<DELIM`, a newline, the body, the delimiter line and blanks. `/bin/cat`,
`command cat`, `cat -`, a redirection after the delimiter word, a
backslash-newline before the `<<`, a command after the document, a
backslash-newline after a backtick's document, and an output inside a `${…}`
all leave it unknown, so each of those standing alone as the command allows,
though bash runs what it prints; handed to `sh -c` as its string, each warns. The check's own help text is the fuller statement of the same list,
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
