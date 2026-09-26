---
name: guard
description: Check a shell command against abcd's hazard registry before it runs, by invoking the abcd binary. Read-only; performs zero writes.
argument-hint: "[check <command> | hook]"
---

# `/abcd:guard` shell-hazard check

Decide whether a shell command is safe to run, using abcd's hazard registry —
the bundled hazard entries merged with this repo's `.abcd/guard.json`. This
command performs **zero writes**.

## `check` — decide one command

Pass the candidate on **stdin**, inside a quoted-delimiter heredoc:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" guard check --json <<'ABCD_GUARD_EOF'
<the command line, verbatim, on one or more lines>
ABCD_GUARD_EOF
```

Never interpolate the candidate into `--command "…"`. The shell expands a
double-quoted argument before `abcd` ever starts, so a candidate containing
`$(…)` or a backtick would **execute at check time** — the exact moment the check
exists to prevent — and an embedded `"` would break the quoting outright. The
quoted delimiter (`<<'ABCD_GUARD_EOF'`, quotes included) switches expansion off,
so the candidate reaches the guard as written. Use `--command` only for a literal
you typed yourself.

Then report the JSON to the user:

- `verdict` — `allow`, `warn`, or `block`.
- `entry_id`, `tier` — which hazard matched, and how severe it is.
- `why` — the plain-language reason, written for a non-expert.
- `successor` — the safe form to run instead.
- `matches` — every entry the command tripped, blockers first.

Exit codes: `0` for allow and warn, `1` for a block, `2` when the guard could
not be evaluated at all (an unparsable command line, or a `.abcd/guard.json`
that does not load). Treat `2` as a fault to report, never as a clearance.
Unparsable means an unterminated quote in **command text**, which no shell runs
either; a quote inside a here-document **body** is document text and is not one.
Grammar a shell does run gets a verdict instead: a trailing backslash is read as
bash reads it, and a here-document whose delimiter line never comes is a
**block** (`heredoc-unterminated`), because the rest of the input may be commands
the guard did not check.

On a `block`, do not run the command. Tell the user the `why`, then run the
`successor` instead — the refusal is the lesson, so pass it on in full. On a
`warn`, the command may run; surface the warning first so the user can stop it.

The check reads the registry of the directory it runs in, and only that one.
Run it from the directory the command will run in: a command headed for another
repository can meet hazards that repository's `.abcd/guard.json` adds, and the
check run from here does not see them. The hook, given a per-call working
directory, reads both registries, so for such a command the two can answer
differently.

## `hook` — the host adapter

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" guard hook
```

Reads a host pre-tool-use hook payload on stdin and applies the check's
decision before a shell command executes, adding the registry of a per-call
working directory's repository when the host names one (below). It is invoked by the plugin's hook manifest,
not by hand; a blocker returns the host's blocking status with the successor and
the why as the message, and a warn or an allow lets the command run.

Anything the adapter cannot turn into a decision — an unreadable payload, a tool
call that is not a shell command, a registry that does not load — allows the
command and warns loudly. A guard that cannot answer never stops a session, and
is never silently absent. A command line is never in that set: one the guard
cannot split is a **block** (`command-unparsable`), because a line the guard
misreads may be one bash runs, and letting it through would pass every hazard in
it; a trailing backslash and an unterminated here-document are decided too. A
here-document body is read as data, even when the line that opened it ends in
`&&`, and the command substitutions an unquoted delimiter lets the shell run in
it are read as commands.

A host whose shell tool takes a per-call working directory passes it beside the
command as `tool_input.workdir`. The adapter resolves it against the session
directory. When it names an existing directory in another repository, the
command is checked against that repository's registry as well as the session's,
and the stricter verdict wins. So a workdir can add a hazard and can never take
one away. A workdir is not a `cd`: the one host that has the field fails the
call when the directory is missing, so no failed-cd hazard exists. A workdir that
no directory could be named by is refused with the blocking status and the
reason: a value that is not a string, holds a NUL byte, a control character or
invalid UTF-8, or is over 4096 bytes.

## Registry and overrides

The bundled hazards ship inside the binary. A repo overrides them in one file in
the repository, `.abcd/guard.json`: add an entry key to change one field (for
example `{"tier": "warn"}`) or to declare a new hazard, and set
`{"disabled": true}` to switch the guard off entirely.

There is no flag, environment variable, or prompt that turns the guard off for a
session — the file is the only route, so the change lands in a diff someone
reviews. Two things follow, and both must be said plainly if a user asks. An
edit that weakens the guard — `"disabled": true`, or a blocker retiered or its
pattern changed — takes effect only once it is **committed**: until `HEAD`
carries it, the edit is refused, the committed registry stays in force, the hook
says so on every command, and the check exits `2` naming the edit. An edit that
only adds or tightens a hazard takes effect at once. And a repo whose guard is
switched off is an unguarded session: every command it lets through carries an
UNGUARDED warning naming the file, so the state cannot pass unnoticed.

**Never write `.abcd/guard.json` on your own initiative.** Disabling or
retiering a hazard is the user's decision to make and to review.

### What this guard is

The guard is a **mistake filter, not a security boundary**. It catches a hazard
typed by accident or reached through an ordinary wrapper — the cases that
actually cost people work. It does not withstand an author trying to get a
command past it, and it does not claim to: the set of programs that launch
another program is open-ended, so no list shipped inside the binary can
enumerate it, and a repository extends that set with one line in a Makefile.

Say this plainly if a user asks whether the guard makes a session safe. It does
not. Anything that needs an enforced boundary needs a control at the **execution
layer** — a sandbox, a permission system, a restricted shell — with this guard
in front of it to teach, never in place of it.

### What an allow does and does not mean

An allow means **no registry entry matched**. It is never a statement that a
command is safe.

A hazard behind a launcher the guard does not recognise is no longer silent: it
is a **warn** naming the entry it matched, because the guard cannot tell whether
that program runs the rest of the line. Report it like any other warn — the
command may run, and the user decides.

An unquoted glob (`*`, `?`, `[…]`) at a position an entry constrains is read as
the pattern it is: bash expands it against the working directory before the
command runs, so a spelling the pattern *can* produce (`git pus? --force`,
`git push --forc?`) is treated as produced and blocks. A glob anywhere else
(`ls *`, `git add *.md`) changes nothing, and a quoted one is literal.

A long flag of `git push` or `git commit` written short of its full name is read
the way git reads it: git accepts any prefix no other option of the subcommand
shares, so `git push --force-w` is `--force-with-lease` and `git commit
--no-veri` is `--no-verify`, and both block. A prefix only blocked options share
(`--forc`) blocks too, although git refuses it as ambiguous.

A command or process substitution (`$(…)`, a backtick pair, `<(…)`, `>(…)`),
unquoted or inside double quotes, runs its own command, which is checked like
any other, and the words
written after it still belong to the command it sits in: `rm $(true) -rf *` is
read as `rm -rf *`, and `git push >(cat) --force` as a force push. What a
command substitution prints is not in the command line, so a word holding one is
an unknown word, and it fails closed in every role it could play, read every
way it can be read at once: written with a leading dash (`--$(…)`, `-r"$(…)"`)
it is every flag it could still become — one that stands alone, one that takes
the next word as its value, a shell's `-c` — wherever it stands, before the
command as well as after it (`sudo -$(…) root <hazard>`, `git -$(…) /tmp push
--force`, `bash -$(…) '<hazard>'` all block); after a value flag (`git -C $(pwd)
push`) it is that flag's value, never the word after it; as an operand it counts
as one. In command position it is any program its known text still allows:
`$(echo git) push --force`, `"$(which git)" push --force` and `sudo $(echo git)
push --force` block, and so does an unknown name followed by a shell's `-c`
string, a wrapper's options or an alias, because the name can be the shell, the
wrapper or git. Where the only entries that fire are ones the unknown name can
be, the block is reported as `program-name-unknown`, with the entries the line
reads as among its matches, and its way past is to spell the program's name. A
program name nothing fixes can be `pkill` or `killall`, so an unknown name
followed by any operand (`"$(which python3)" script.py`, `$(date) x`) blocks —
an accepted over-block, answered the same way. So is a here-document whose
delimiter is a substitution (`cat <<$(echo EOF)`): the guard does not run it to
learn the delimiter, so no line ends the document and it blocks as
`heredoc-unterminated`; write the delimiter out.
A command with more than eight substitutions where its program name could be is
a **block** (`substitution-unread`), because the guard stops following them.
Text written beside one in the same word is also read as bash leaves it when the
output is empty, so a flag glued to one is still the flag. A word that is wholly
a substitution is read as an operand, not as a flag: that is how a commit
message or a branch name is spelled every day (`git commit -m "$(cat msg)"`), so
`git push $(printf -- --force)` is not seen. A parameter expansion holding a
substitution (`${X:-$(…)}`) prints that substitution's output, so its word is
unknown from the `${` on: `--${X:-$(…)}` is every long flag, and a wholly
`${…}` word is read as a wholly-substituted one is. Inside double quotes a
`${…}` ends at its own `}`, and a `"` in it opens a nested string rather than
closing the outer one, so `echo "${MSG:-"don't"}"` is one word and runs. A
here-document body is data,
but where its delimiter is unquoted (`<<EOF`, not `<<'EOF'`, `<<"EOF"` or
`<<\EOF`) the shell runs the command substitutions in it, and each is read as a
command; such a body is read by the lines bash compares with the delimiter, so a
line ending in an odd number of backslashes joins the next one before the
compare, and `x\` followed by `EOF` does not end the document. Between
backticks bash drops a backslash before `$`, a backtick or a backslash before it
reads the command, and directly inside double quotes one before a `"` too, so a
backtick's text is read after that pass: an escaped `\$(…)` or an escaped
backtick pair there is the substitution bash runs, in a word or in an unquoted
here-document body inside the backticks. A `"$(cat <<'EOF' … EOF)"` whose
document the shell does not change, or its backtick spelling with no backslash
between the backticks, is also read as that document's text, joined to any text
written beside it in the same word, where it is a payload, so `sh -c` or `eval`
handed one reads the document as the command it runs. Unquoted, the same
substitution runs the words its document splits into on blanks and newlines, the
first and last joined to whatever is written against the substitution in its
word (`$(…)x`, or a second such substitution), and those words are read as bash
builds them: as the command in command position, as operands after it, and at
every payload layer the guard follows, so a document whose own text is
`$(cat <<'F' … F)` is read too. Text written in the word is not split, as bash
does not split it, and an assignment's value is not split either. On a command
line where any other command names IFS (`IFS=x;`, `export IFS=x`), an unquoted
fixed output is a **block** (`ifs-split-unread`): the guard splits on the default
IFS only, and refuses rather than work out which assignment reaches which
expansion. A prefix assignment (`IFS=x $(…)`) does not reach its own command's
expansion, and is read as the default split. The words are never
read again as a command line, as bash never reads them, so a `;` or a `$(` in
them stays a word. The guard follows two execute-a-string layers, an `sh -c` or
`eval` inside another; a payload nested deeper is a **block**
(`execute-string-uninspectable`) whatever it holds, because the guard has
stopped reading it. A substitution nested more than
eight double-quoted substitutions deep, or one holding a case command, is a
**block** (`substitution-unread`), because the guard has stopped reading it and
its command runs all the same. An arithmetic expansion `$(( … ))` is read as an
expression, not as commands; a command substitution inside it is followed. An
ANSI-C string ends at its closing quote, found before any escape is decoded, so
`$'\c'` is closed and the command after it is read; and it ends at its first
NUL byte (`$'\x00'`, `$'\0'`), as bash ends it, so `$'\x00'git` is `git`.

A shell reading its script from a pipe, a here-document or a here-string
(`curl … | sh`, `bash <<'EOF'`) is a **block** (`interpreter-reads-stream`):
what it runs is text the guard read as data. So is a shell handed the stdin
device behind a pipe (`curl … | bash /dev/stdin`, `/dev/fd/0`), and a shell or
`source` handed a process substitution as its script (`bash <(curl …)`, `bash <
<(curl …)`, `source <(curl …)`). A shell handed a script file (`bash
script.sh`, `bash script.sh < input`) is not. A command line longer than 64 KiB
is a **block** (`command-too-long`), because the guard does not read it.

An unquoted brace group is expanded the way bash expands it, and every word it
produces is checked: `mkdir -p foo/{a,b}` is allowed, `git push {--force,} origin
main` is a force push. A group that would expand past 4096 words on one command
line is not expanded and is a **block** (`brace-expansion-unexpanded`), because
the words it would pass are ones the guard has not read.

A git alias declared in the command line is expanded before the match, because
git resolves it before it runs: `git -c alias.p='push --force' p origin main` is
a force push, and so are its `--config-env`, `GIT_CONFIG_KEY_n`/`VALUE_n` and
`GIT_CONFIG_PARAMETERS` spellings. A `!` body is read as a shell command, and
an alias that command declares is resolved in turn, two `!` bodies deep; an alias
nested deeper than that is a **block**, because the guard has stopped following
it. Two
consequences to report accurately: an alias that shadows a git builtin
(`-c alias.push='push --force' push`) is refused even though git would ignore
it — an accepted over-block — and configuration delivered from a FILE
(`GIT_CONFIG_GLOBAL`, `-c include.path=`, an `--config-env` variable the line
does not set) is a **warn** under `git-config-rewrite-unread`, because the
directive is visible and its body is not.

A `git commit` or `git push` that points `core.hooksPath` somewhere else for
itself — through `-c`, `--config-env` or the `GIT_CONFIG_*` environment — skips
the repository's hooks exactly as `--no-verify` does, and blocks under the same
entries whatever the value, because the guard cannot tell a directory of real
hooks from an empty one. Setting the key with `git config` is not refused.

What an allow still does not see is a hazard that never reaches command position
at all: one launched through a known wrapper carrying a value-taking flag the
guard does not name (`sudo -u bob <hazard>` is seen; the bundled short form
`sudo -Hu bob <hazard>` reaches only the warn, not the entry that names it), one
whose API path an entry names by its ROOT
segment but the host serves under a prefix (a GitHub Enterprise Server install
mounts the same endpoints under `/api/v3/`; the `https://api.github.com/…` URL
form **is** read), a parameter expansion that carries no substitution (`$VAR`,
`${VAR:-git}`) wherever it stands — as the program's name, as a flag
(`--$VAR`), or inside a payload the guard reads — because the guard sees the
variable, not what the shell expands it to, an IFS the shell already holds when
the line starts (every line is read from the default IFS), a hazard inside a non-shell interpreter's payload (`python -c`,
`perl -e`) — one opaque token the tokenizer cannot read, today a silent allow, not
a warn (a warn for it is a recorded design target, not yet implemented), or a
dangerous form no entry describes.
Coverage is what the registry names. Say exactly this if a user asks about
coverage — never that the guard cleared the command.

In a repository with more than one worktree, a `git stash` or `git stash pop`
that does not name its entry is a **warn** (`git-stash-shared-stack`): git keeps
one stash stack for the whole repository, so a bare pop can take another
worktree's work. A stash with a message and a pop by entry (`stash@{N}`) are not
warned about, and neither is anything in a single-worktree clone.

A candidate too long to read is refused (exit 2), not answered on the part that
fitted.

To check whether the guard is actually armed in this repo, run
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy` and read its `guard:` line: it reports
whether the hook is installed, whether the binary is reachable, and whether the
registry loads.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

Whichever rung fires, the candidate still goes in on stdin via the
quoted-delimiter heredoc shown above.

**User input:** $ARGUMENTS
