---
name: guard
description: "Judge a shell command against the hazard registry before it runs: Writes branch records and a question's mode; refuses a hazard and an unknown sub-verb."
argument-hint: "[check [--command <command>] | hook | approve --dir <dir> | allow --once --command <line>]"
block: agents
user-invocable: false
---

# `/abcd:guard` shell-hazard check

Decide whether a shell command is safe to run, using abcd's hazard registry —
the bundled hazard entries merged with this repo's `.abcd/guard.json`. The
check performs **zero writes**. The hook writes the record of a branch the
session creates (below), and refuses nothing for it; when it admits abcd's own
question where the badge shows, it also writes the waiting-on mode and the
marker saying a question is open (below).

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
differently. A script the command runs is resolved the same way: the check
reads it relative to its own working directory, and the hook relative to the
per-call working directory when the host names an existing one, else the
session directory.

## `hook` — the host adapter

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" guard hook
```

Reads a host pre-tool-use hook payload on stdin and applies the check's
decision before a shell command executes, adding the registry of a per-call
working directory's repository when the host names one (below). It is invoked by the plugin's hook manifest,
not by hand; a blocker returns the host's deny, exit 0 and one JSON object on
stdout whose reason carries the successor and the why, and a warn or an allow
lets the command run. The host shows that reason to you on one error line and
hands it to the agent. A notice the hook printed before it blocked, such as a
repository guard file that did not load, leads the reason, because the host
discards the hook's stderr when it exits 0.

Before it judges a shell command, the hook records the branches the command
creates, for the away mode's reading of which branches are the session's own
(`/abcd:mode`): `git worktree add -b|-B <b>`, `git switch -c|-C <b>`, `git
checkout -b|-B <b>` and `git branch <b>`, each only when no branch of that name
exists yet; a bare `git worktree add <path>`, recorded as the branch git names
after the folder, and `git worktree add <path> <branch>`; and `git branch
-m|-M <old> <new>`, which carries a record from the old name to the new. It
records them whether or not the person is away, under the working copy's key in
the away store (`~/.abcd.noindex/away/<root-sha>/`), matched by whole name, in a
list of the hundred most recent. The recorder **refuses nothing**: a store it
cannot write leaves one stderr line and the command goes on to the check as
before, and a record made for a command that then fails names a branch that
does not exist, which the reading ignores.

On the host's question tool the hook checks abcd's own questions instead of
consulting the registry. A question is abcd's when a header is in abcd's chip
grammar (a role word, then `Q` and the question's number, such as `Product Q2`)
or when `/abcd:mode` names somebody; any other question belongs to another tool
and runs unchecked. That includes a question with no abcd chip asked while
`/abcd:mode` reads `managed`, in a repository abcd manages or not: the hook
cannot tell an abcd question that forgot its chip from another tool's, so the
interview pages teaching the chip are what keep abcd's questions under the
check. abcd's question is held to the field limits of the asking
rules wherever the hook runs, in a repository abcd manages or not: the header,
the number of questions and options, each label's words and each meaning's
sentences, the decide-later option, no bold and no recommended mark, the `Now:`
and `Change later:` lines, the thing quoted before the question, the rows it
takes, and no side preview (the host hides every option's description while a
preview shows, so the meaning goes in the description). A question that breaks any of them is refused with the host's
deny, whose reason is one head line, "Blocked by the abcd guard (question tool): N part(s)
of this question break abcd's asking rules; fix each and ask again.", then one
line per part naming the tab, the part, the value, the limit and the remedy, so
one retry fixes them all. At most ten parts are named, and a closing line counts
the rest, so a refusal stays a few lines however large the question; the head
line keeps the full count. The rows limit is the exception: a question whose
only finding is that a tab runs past the rows limit is shown, and the hook tells
the agent afterwards which tab ran over, by how much, and to draft the next
question through the `abcd:question-drafter` agent, which counts rows the way
the check does. In a question refused for another part, a tab over the rows
limit is not one of the parts to fix and the head line does not count it: it
follows them, under a line saying the rows limit does not refuse on its own and
that the question would have been shown. A question with no abcd chip, held to the limits
because `/abcd:mode` names somebody, carries one more line under the head line
saying so: another tool's question asked while the mode names somebody is held
to abcd's rules. The hook refuses and never rewrites the question.

In a repository abcd manages, an admitted question carrying abcd's chip sets
`/abcd:mode` from the chip's role: `Product` to `product-thinker`, `Tech` and
`Setup` to `facilitator`. So a chipped question is never refused on the mode,
and the status line names whom the question on screen is for, a chip naming
the other person re-setting the mode. An admitted question is marked open in
`.abcd/.work.local/question_open`, and the answer resets the state to
`managed`: the plugin's hook on the question tool's result resets it when the
answer comes back, and the next human message resets it on a host that does
not run that hook. Where the state cannot be written or the question cannot be
marked open, such as a local tier that is not writable, the question runs with
a loud `NOT CHECKED` warning. A questions field the hook cannot read is the same case: the question runs, and the hook
warns loudly. So is a payload the hook cannot decode at all: where it names the
question tool the warning says the question runs, and where no tool name can be
read the warning names no tool.

While the away mode holds (`/abcd:mode autonomous`), the hook refuses every
question on the host's question tool whose payload it can decode, before any
of the checks above, abcd's or another tool's, whatever its chip, with one
reason, and sets no mode and marks nothing open; a payload it cannot decode
runs with the loud warning above, away or not. The reason routes the question by the act it asks about: an act
only the person may approve (an embargo `--approve` digest or triage verdict,
the drain's hand-back, the remote-configuration apply verb, a release sign-off,
pushing, opening or merging a pull request without a standing instruction,
publishing or posting outward, a new dependency, a secret, a credential or an
App, a website step, a destructive act, spending, or any other act a command
page or a brief invariant names as the person's) is queued with
`/abcd:mode queue "<step>"`, and every other question is decided and recorded
with `/abcd:decide`. Nothing detects the act: the agent classifies its own
question from the reason's list. An away record that cannot be read safely
reads as not away, and the hook says so on stderr, with the warning status
only where it would otherwise admit the question silently.

Anything the adapter cannot turn into a decision — an unreadable payload, a tool
call that is neither a shell command nor a question, a registry that does not
load — allows the command and warns loudly. A guard that cannot answer never
stops a session, and is never silently absent. A command line is never in that
set: one the guard cannot split is a **block** (`command-unparsable`), because a
line the guard misreads may be one bash runs, and letting it through would pass
every hazard in it; a trailing backslash and an unterminated here-document are
decided too. A here-document body is read as data, even when the line that
opened it ends in `&&`, and the command substitutions an unquoted delimiter lets
the shell run in it are read as commands. A substitution still open where that
line ends holds the body back until the line after it closes, and its own
lines are read as commands, as the shell runs them.

The hook judges only what the host hands it, and the plugin's hook manifest
hands it the shell tool, the question tool and the host's file-writing tools
(`Write`, `Edit`, `MultiEdit`, `NotebookEdit`) and nothing else. A file one of
those tools writes is not held to the registry: it is **denied** when it lands
in a main checkout two or more live sessions started in, with the refusal the
record verbs give, naming the worktree store (iss-2610280051498465). The file's
working tree is found by walking up to its `.git`, with no git subprocess, so a
linked worktree, a checkout one session or none works in, a path in no checkout,
and a file in a checkout's own local tier (`.abcd/.work.local/`, which git
ignores) are written silently. A file call naming no path, a relative path with
no absolute session directory, or a count that cannot be made, lets the write
run and warns loudly. A call through any other
tool never reaches the guard: a command a tool from another extension runs is
neither checked nor warned about. That is the guard's standing scope, not a
degradation, and the `guard:` line of `abcd ahoy` does not report it.

A host whose shell tool takes a per-call working directory passes it beside the
command as `tool_input.workdir`. The adapter resolves it against the session
directory. When it names an existing directory in another repository, the
command is checked against that repository's registry as well as the session's,
and the stricter verdict wins. So a workdir can add a hazard and can never take
one away. A workdir is not a `cd`: the one host that has the field fails the
call when the directory is missing, so no failed-cd hazard exists. A workdir that
no directory could be named by is refused with the host's deny and the
reason: a value that is not a string, holds a NUL byte, a control character or
invalid UTF-8, or is over 4096 bytes.

## `approve` — answering a headless agent's permission prompts

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" guard approve --dir=<the agent's working tree> --base=<the commit the run started on> [--run-dir=<the run's own directory>]...
```

You never run this yourself. When abcd starts an agent through its claude
runner without a terminal (a build lane's implementer or reviewer routed to a
runner, or an interview's role), and the role's contract grants the shell, the
launch carries the guard: the runner names `approve` as the harness's permission
prompt tool and starts this verb as the launch's one MCP server, on stdio, all
on that launch's command line. Nothing is written to the person's host settings,
interactive sessions are unchanged, and this adds no sandbox. On that launch the
shell is no longer granted without a prompt, so every shell command the harness
would ask a person about is put to the guard instead.

Each prompt is judged against the registry of the working tree `--dir` names,
read afresh each time, and that registry is in force only where it is at least
as strong as the one at `--base`: the lane's base for a build or a drain, the
working tree's `HEAD` when an interview launched. A new entry or a stricter tier
takes effect; any weakening since the base, of a warn as much as a blocker,
committed during the run or not, leaves no registry in force, and every command
is refused (`registry-not-in-force`). A shell launch that names no base is not
started.

A shell command is approved only when the guard reads it whole and clears it:

- no registry entry and no reading signal fired, a **warn included**, since
  nobody is there to see the warning;
- the check left no note, such as a script the command runs that does not
  exist (`check-note`);
- the repository's registry is loaded, switched on and in force as above;
- every program the line runs is named by the line, not by a variable, a
  substitution or a glob (`program-name-unread`); `[`, `[[` and `test` are the
  test commands they name, not globs, so `[ -f x ] && rm x` and
  `if [ -f a ]; then …; fi` are judged by their words;
- every operand bash evaluates as arithmetic is a written number
  (`arithmetic-operand-unfixed`). Bash evaluates a name it meets in arithmetic
  as arithmetic in turn, so a value the line does not fix can hold an index
  that runs a command. Refused are a name, a quote or an expansion inside
  `(( … ))`, `$(( … ))`, `$[ … ]` and a `for ((…))` header; the number
  comparisons of `[[ … ]]` (`-eq`, `-lt` and the rest) and its `-v`; `let`
  and an integer or name-reference declaration (`declare -i`, `local -n`); an
  array index that is not a number, in an assignment, an expansion or a
  compound assignment; a substring offset or length that is not a number;
  `${!name}`; and a variable name that is not written in the line, handed to
  `read`, `unset`, `printf -v`, `wait -p`, `test -v` or a declaration. The scan
  reads the whole text, quotes and here-documents included, so a script that
  only quotes such a construct is refused too. `[ "$n" -gt 0 ]` and `test`
  compare numbers without evaluating them, and stay approved;
- no text that bash reads as something other than what it shows
  (`arithmetic-operand-unfixed` too): an alias, or alias expansion turned on
  (`shopt -s expand_aliases`, `set -o posix`, `POSIXLY_CORRECT`); a `$'…'`
  string that writes a byte by its code (`\x28`, `\050`); and a builtin
  named above spelled with a quote or a backslash in its name (`"let"`,
  `\let`). A backslash-newline is joined before any of this is read, as bash
  joins it. These checks read the text as written, so they catch what an agent
  types by mistake, not every way a line can be built to hide a command: the
  approver is a mistake filter like the rest of the guard, and the boundary
  for a launched agent is the OS sandbox, not the approver;
- no code reaches an interpreter that the guard does not read
  (`inline-code-unread`): inline code for `python -c`, `node -e` or `-p`,
  `perl -e`, `ruby -e`, `php -r`, `osascript`, `lua` or `Rscript -e`,
  `deno eval` or `pwsh -Command`; code on the standard input (a pipe, a
  here-document, a here-string or a redirected file) of one of those, or of
  `ex`, when no script file is named; an `awk` program that calls `system(`,
  pipes to or from a command, or is a variable; a `sed` script with the `e`
  command or the `e` flag of `s`; `tar`'s options that run a program
  (`--checkpoint-action`, `--to-command`, `--use-compress-program`, `-I` and
  the like); a `vim` or `ex` command (`-c`, `--cmd`, `+`) holding a `!`.
  The words after a script's name, or after python's `-m` module, are the
  script's, so `node app.js -p 3000` and `python3 -m pytest -c pytest.ini` are
  approved; a word is taken for the script only where it cannot be the value
  of the flag before it (`node --title app.js -e …` is still refused);
- every write the guard's write model reads, and every removal, names its
  target in full (`path-not-fixed`). The write model is a list of known
  writers, so a program outside it writes without being read. The targets are
  read from the guard's own parse of the line: redirections, the writers its
  write model lists (`tee`, `cp`, `mv`, `install`, `ln`, `dd`,
  `curl -o`, `wget -O`, `sed -i` and a sed script's `w`, `patch`,
  `git checkout`/`restore`/`clone`, `tar -x`, `unzip`, `rsync`'s destination,
  `go build -o`, `sort -o`, `ditto`'s destination, `split`'s prefix, `uniq`'s
  output file, and what a command `find -exec` or `-execdir` runs writes, with
  each `{}` a name the line does not fix), and the operands of
  `touch`, `mkdir`, `chmod`, `chown`, `truncate`, `rm`, `rmdir`, `unlink`,
  `mv`'s sources and `ln`'s link targets. A target held in a variable, a
  substitution's output or a glob, or a relative name after a change of
  directory the line leaves unknown, is refused. A glob removal whose fixed
  directory is inside the working tree, followed through symlinks to where it
  leads, is the exception when no earlier part of the line writes or removes a
  file (which could make that directory a link first, as `ln`, `cp -s` or
  `cp -l` do): it is approved unless a protected name could match the glob, either one its literal text aims at
  (`build/.git*`, `.env*`), read with the shell's default that a name starting
  with a dot is matched only by a pattern that spells the dot, or one already
  standing in that directory, or a protected path below it (`tools/*` where
  `core.hooksPath` is `tools/hooks`). So `rm -rf build/*` and `rm -f build/*.o`
  are approved, while `rm -rf linkd/*`, where `linkd` is a symlink to a
  directory outside the tree, and `ln -s ~/Library l && rm -rf l/*` are
  refused (`path-not-fixed`). So is a glob with a wildcard in a directory part
  (`rm -rf build/*/*`, `rm -f build/*/x.o`) or one that ends in `/`
  (`rm -rf build/*/`): it lists directories the line never names, any of which
  may be a symlink out of the tree. So is any command where an unquoted `(`
  stands in a word, whatever the command and whatever follows the `(`
  (`rm -rf build/l(:A)`, `rm -f x(.)`, `sed x(.) -i`, `echo hi(:A)`,
  `ls x(/)`): zsh reads it as glob qualifiers or modifiers, which can change
  the path the word names (`build/l(:A)` is where the symlink `build/l`
  leads), and the words after it are not read as the command's own. bash
  refuses such a line, and every approved command runs under bash (below), so
  this refusal is a second layer. A quoted or escaped `(` is text, so
  `rm -f 'a(b)'` stays approved, and so do a function definition's `()`
  (`f() { … }`), the `((` of an arithmetic for-loop
  (`for ((;;)); do …; done`; a header holding a name is refused as arithmetic,
  above) and the `(` before a case pattern
  (`case a in (a) …;; esac`), which bash needs and which name no path. A
  relative name after a `cd` the line spells out and joins with `&&` is read
  in that directory;
- no write or removal reaches a protected path (`protected-path`), judged with
  its existing directories resolved through symlinks and its case folded where
  the filesystem folds it: the git common directory (its hooks and config
  included), each directory `core.hooksPath` names, each checkout's `.git`, the
  home directory's shell start-up files, `~/.gitconfig` and `~/.config/git`,
  the host's settings (`~/.claude`, `~/.claude.json`, a checkout's `.claude/`),
  `.mcp.json`, `.abcd/guard.json`, abcd's own folder in the home directory
  (`~/.abcd.noindex/`: its `trusted-roots`, `rules.json`, `config.json` and
  the rest of the layer, though the run's own worktree kept under
  `~/.abcd.noindex/worktrees/` stays writable), and the names the host
  protects in any working tree (`.husky/`, `.vscode/`, `.envrc`, `.npmrc` and
  the like). A removal of a directory holding one counts. A line that only
  names one of them, to read it, is not a write;
- no removal reaches outside the working tree (`removal-outside-tree`): a
  removal (`rm`, `rmdir`, `unlink`, `mv`'s sources, a `find` that deletes, an
  `rsync --delete`) whose target lies outside the launched working tree, the
  run's own directories (the brief's and the receipt's, which the runner
  passes as `--run-dir`) and the temporary directory (`$TMPDIR`, `/tmp`) is
  refused, so `rm -rf ~/Documents/x` and `rm ~/.ssh/id_ed25519` are. A write
  outside the working tree is judged by the protected paths alone; the OS
  sandbox for launched agents (spc-2610161751056096) is what bounds those;
- no `git config` with `--global`, `--system` or `--file` writes, sets a key
  that starts a program (`core.hooksPath`, `core.fsmonitor`, `core.pager`,
  `core.sshCommand`, an alias, a filter or diff driver, a credential helper,
  the signing keys and the like) or `core.worktree`, which moves the working
  tree git writes to another directory, or opens the editor, and no git
  command sets such a key for itself with `-c`, `--config-env` or `GIT_CONFIG_*`,
  except `-c core.pager=<command>` or `-c core.sshCommand=<command>` with a
  command the line spells out, which is judged as the command git runs and
  approved when it is clean, unless an earlier part of the line writes or
  removes a file (`git -c core.pager=cat log` is approved)
  (`git-config-write`). Reading configuration, and setting an ordinary key in
  the repository's own file, is approved;
- nothing removes, moves or recursively changes (`chmod -R`, `chown -R`) a
  path that is, or may be, the root, a top-level directory, the home directory
  or a directory directly in it, the working tree or a directory above it, or
  another checkout of the repository (the main one, or another registered
  worktree); no `find` deletes (`-delete`, or `-exec rm`) from home or above;
  and no `rsync --delete` writes into such a directory
  (`critical-path-removal`);
- the call does not ask to run outside the host's sandbox
  (`dangerouslyDisableSandbox`, refused as `sandbox-off`).

An approved call runs with the command judged and its description, and nothing
else it carried, from the working tree it was judged for and under bash: the
approval hands the command back as
`cd -- '<dir>' && /bin/bash --noprofile --norc -p -c '<command>'`, the
directory and the command each quoted as one literal word. The host keeps the
shell's directory from one call to the next, and the approver judges every
command as run from `--dir`; and the host runs the text in the person's login
shell, which may be zsh, while the guard reads every command in bash's grammar.
So the login shell only changes directory and, when that succeeds, starts
/bin/bash with its profile, rc and environment start-up files ignored (-p):
`--noprofile` and `--norc` skip the profile and rc files, and `-p` makes bash
ignore `BASH_ENV`, functions exported as `BASH_FUNC_*` and `SHELLOPTS` from
the environment. bash runs the command: what the guard judged is what runs.
The bash call is not an `exec`, so text the host joins after it (its own
`&& pwd -P >| <file>` record of the shell's working directory) still runs.
bash gives no zsh-only construct its zsh meaning:
glob qualifiers, `(:A)` modifiers and `{ … } always { … }` are syntax errors to
it, and it runs nothing of a line holding one. When the directory cannot be entered no
part of the command runs, a part after `||` or `;` included; a command that
ends in a here-document or a comment, or spans several lines, runs unchanged;
and its exit status is the call's. The text handed back is read again with the
guard's own parse, and the call is approved only when that text is exactly the
change of directory joined by `&&` to that one bash call, with the whole command
as its last word and nothing else (`tree-group-broken` otherwise). Nothing in
the command can end that word early, so a `}` or a `fi` with nothing open is
bash's syntax error inside the working tree, not a refusal. A change of
directory inside the command still holds for the rest of it, and a relative
name after a `cd` the line spells out and joins with `&&` is read in that
directory. Anything else is refused with the guard's reason and what to
run instead, which the harness hands the agent so it can rewrite the command. A
prompt for any tool but the shell is refused (`prompt-not-shell`), since nobody
is there to approve it, and so is a shell prompt whose input carries no
command string the guard can read (`prompt-input-unread`) or whose command
holds a NUL byte, which no shell passes on as written (`command-holds-nul`). A role whose contract grants no shell keeps its launch
as it was, with every prompt refused unasked. The opencode runner has no
per-launch prompt answerer, so its launches keep the permissions opencode's own
configuration sets. A script the line runs is read for the registry's hazards
as everywhere, but the checks above read the line itself, not the script's own
writes, and a command that writes into a directory (`tar -x`, `unzip -d`,
`cp -r`, `git apply`, `git checkout`) is judged by that directory, not by the
files that land inside it.

What the approver does not cover, plainly. The file tools the role is granted
on the launch (`--allowedTools`: reading, editing and writing files) write
without asking the approver, so the protected paths above bound only the shell.
The abcd binary that answers the prompts is not among the paths it protects.
Bounding those writes is the OS sandbox for launched agents
(spc-2610161751056096), not the approver.

## Branch deletes and worktree removals — proved, or refused

In a repository, a delete of a branch or a worktree runs only when the guard can
prove the target safe (iss-2610241239492159). That covers `git push --delete`
and a `:<branch>` refspec, `git branch -D` (and `-d` with `--force`) and
`git worktree remove`. Their registry entries are warnings, and a delete
proven safe still carries that warning; `check` and `hook` refuse every one
whose target they cannot prove (`git-delete-unproven`), which outside a git
checkout is every one.

A target is safe when every check holds, and each check refuses when it cannot
answer:

- **Merged** into origin's default branch as last fetched: its tip is an
  ancestor of it, or, for a rebased or squashed branch, its patches are there
  (the worktree store's own judgement), or a pull request merged at that tip.
- **No open pull request**, asked of the forge, for a branch on origin. A local
  `git branch -D` is judged offline against `origin/<default>` as last
  fetched; only a delete that touches origin asks the network, and a tip origin
  holds that this checkout has not fetched is refused until it is.
- **In no worktree**: a branch checked out anywhere is refused.
- **No live reservation**: an `abcd-reserved/` or `abcd-status/` branch is never
  deleted by hand (`abcd reserve --sweep --apply` clears the spent ones), and a
  worktree a live reservation, a running loop or a process holds is refused, as
  is every worktree removal in a checkout that has never fetched the
  reservations (`abcd peers` fetches them).
- **For a worktree, clean**: nothing staged, unstaged, untracked or ignored, not
  locked, not the main working tree, and on a branch (a detached HEAD is
  refused).

A line whose targets the guard cannot read (a variable, a glob, a name git
resolves from its history such as `@{-1}`, a `{}` placeholder or a delete
`xargs` hands its operands, a `git branch` name starting `refs/`, a `cd` or a
`GIT_` variable before the delete, an option it does not know) is refused too. A
pushed delete of a tag or another non-branch ref (`:refs/tags/v1`) is outside
the proof and keeps its warning. The refusal names the check that failed and
what settles it: merge the branch, close the pull request, fetch, or remove the
worktree first.

### `allow` — the person's one-shot way past

When the person decides a refused delete is safe anyway, they run, themselves,
in their own terminal, `abcd guard allow --once --command '<the exact command>'`.
That lets the one exact line (byte for byte) through the hook once, within ten
minutes, and lets past nothing else: a line any other hazard blocks stays
blocked, and the allow is left unused. The hook admits the line with a note
saying the person allowed it and when (exit 0, so no second judge re-reads it
with the allow already used). `guard check` never uses an allow, and
neither does the approver for a headless agent.

**Never run `allow` yourself.** The guard refuses it when an agent runs it, in
either spelling (`abcd …`, or `go run ./cmd/abcd …` in a source checkout), and refuses a shell write
into its store. Tell the person the exact command and why it is safe, and let
them decide.

The allow is recorded under `~/.abcd.noindex/guard-allow/<root-commit>/`, one
file per line, and read only when this account owns it and no one else can
write it or its folders, as the rules loader reads `~/.abcd.noindex/rules.json`.
It is removed as it is used. The folder's `log.jsonl` records each exact line,
when it was allowed and when it was used, and never who.

The trust model is a careless agent, not a hostile one, as for the guard's
other hazards. What it does not cover, plainly: a file tool writing into the
store is not a shell command, so the shell guard does not see it; `git push
--prune` and `--mirror`, which delete as a side effect, are not judged here; an
allow is keyed on the line, not the directory, so a relative worktree path
allowed in one directory of the repository is honoured from another; and a
local branch pushed to origin but not yet merged is refused like any other
unmerged branch.

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

The hazards are also taught before any command runs: the rules loader's `SHELL`
domain is generated from the registry this guard enforces, one rule per entry,
then one rule for the scripts the guard reads and one for the variable-built
`-c` script it refuses, and is injected when a prompt is about shell work (`abcd rules shell` renders
it). An entry a repo adds in `.abcd/guard.json` is taught there as well, its
rule marked `(repo)`; a guard file this guard refuses is named on stderr and
not taught. An entry's `why` and `successor` are each at most 1,024 bytes, and
a longer one refuses the file. With the guard switched off, each rule opens
`Hazard (guard off)` instead of `Refused by the guard`.

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
`git push --forc?`) is treated as produced and blocks. A bracket expression is
read exactly only where it is one plain set (`--for[c]e`); a class
(`[[:lower:]]`), a set that opens with `!`, `^` or `]` or holds `-`, `[` or a
backslash, and a pattern with a `]` after the set closes (a second set among
them) can produce any run of characters,
since the guard sees the line with its backslashes taken out, so
`r[[:lower:]] -rf /` blocks and `git clea[!n] -fd` warns. A glob anywhere else
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
`git push $(printf -- --force)` is not seen. A parameter expansion (`$VAR`,
`$1`, `$@`, `${VAR:-git}`, `${X:-$(…)}`) prints a value the line does not
hold, so it is read by the same rule: `--$VAR` is every long flag, `-$F` every
short one, and `$GIT` or `${GIT:-git}` as the program is any program its known
text allows; a single-quoted or escaped `$` is text. A word that is wholly a
variable is read as an operand, as a wholly-substituted one is, so `git push
origin "$branch"` stays allowed and `git push $X origin main` is not seen. A
string handed to a shell carries its variables for that shell to expand
(`sh -c "git push --$X …"` is read as that shell reads it), and the string's
own quotes apply to the value, so `'--$X'` inside it is a flag too. A variable
the line itself assigns before it uses it is read with that value in place, as
well as by the rule above: a pid list a search printed (`p=$(pgrep make); kill
$p`, `pgrep make | while read p; do kill $p; done`, `for p in $(pgrep make)`)
is a kill by name, a stream path (`f=<(curl …); bash "$f"`) is a shell reading
a stream, text a string runs (`X='…'; eval "$X"`, `sh -c "$X"`, `bash -c "… $X
…"`) is judged as the command it holds, and `P=pkill; $P make` is a `pkill`;
each is a **block** under the entry that names it. A string that runs a value
the line set from output the guard did not read — a command substitution
(`X=$(curl …); eval "$X"`) or a `read` — is a **block** as uninspectable. A
variable the line did not set is not read as what an earlier command carried
into it: as a shell's or `source`'s script it is not a stream (`bash
"$script"`), and as the program's name it is not `pkill` or `killall`, whose
entries name only the program and an operand, nor a bare interpreter inside a
string, so `"$GO" build ./...`, `$EDITOR notes.md`, `eval "$cmd"` and the
successor `kill "$pid"` stay allowed; so does a value the line set from a
source of none of these kinds (`pid=$(cat run.pid); kill "$pid"`, `GO=go;
"$GO" build`). Which assignment reaches which use is read by order alone, so a
name the line assigns more than once is read with each of its values. A variable standing as the program with
another variable as its first operand (`exec "$BIN" "$@"`) can be `git clean`
and warns, as a substitution there does. Inside double quotes a `${…}` ends at
its own `}`, and a `"` in it opens a nested string rather than closing the
outer one, so `echo "${MSG:-"don't"}"` is one word and runs. A
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
line where any other command names IFS (`IFS=x;`, `export IFS=x`, or an
assignment target that holds an expansion anywhere in the line's text, such as
`export ${I}FS=x`, `(( ${a}${b} = 1 ))` or `printf -v "$x" 1`, since bash sets the
variable the target's value names), an unquoted
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
<(curl …)`, `source <(curl …)`, zsh's `source =(…)`, `bash --init-file <(…) -i
-c true`), and a script the same line downloads to a file and then runs as a
shell's script or standard input, a `source` or `.` operand, or a path
(`curl -o i.sh … && bash i.sh`, `curl … > i.sh; source i.sh`, `wget …/i.sh &&
sh i.sh`, `curl -O …/i.sh && ./i.sh`): the file is one curl's `-o` or `-O`,
wget's `-O` or saved name, a `tee` a fetch feeds or a redirection of what a
fetch printed writes, matched by the name the line writes it under, and a
name the line does not fix can be any file. The shell's options before the
script are read as the script reading below reads them, so `bash -login
<(curl …)` and `zsh -oerrexit <(curl …)` block too. Download it in one
command, read it, and run it in the next. A command
line longer than 64 KiB is a **block** (`command-too-long`), because the guard
does not read it.

A script file a shell runs is **read and judged** before the command is: a
shell's script operand (`bash build.sh`; not under `-n` or `-o noexec`, a
syntax check that runs none of it, unless `-i` makes the shell interactive,
which ignores the option; each shell's options are read as that shell reads
them, so bash's `-login` is `--login`, zsh's and ksh's `-oerrexit` is one
option, and a lone `-` ends the options as `--` does, while a name that is
more than one shell, `sh` among them, is read every way it can be and any
script one of the ways runs is judged; the same reading finds a `-c` string
behind options that take a value, `bash -co errexit '…'`), a `source` or `.` operand (searched on
`PATH` and then in the working directory when it has no slash, as bash does,
while zsh's `source` looks in the working directory first and its `.` only on
`PATH`; a `PATH=` prefix
on the `source` itself is the `PATH` searched, a relative or empty entry resolving
against the working directory, and an entry of that prefix the line does not fix
is a **block**, `script-unread`; zsh ties its `path` array to `PATH`, so a
`path=(dir)` or `path=dir` the line sets is the `PATH` zsh searches and the one
a child of that zsh is handed, while in bash and the other shells `path` is an
ordinary variable; where the guard cannot tell which shell runs the line —
anything but a `zsh -c` string, another shell's string, or a file a named shell
runs — it reads the line both ways from the command that assigns `path` or
sources a bare name on, and judges each file either shell would run), the file a
redirection makes a shell's standard input when it has no `-c` string and no
script (`bash < s.sh`, `bash -s < s.sh`, `exec 3< s.sh; bash <&3`), and the
startup files the line selects — `BASH_ENV=f`, `ENV=f` on an interactive shell
(`-i`), `--rcfile f` and `--init-file f`, the zsh startup files under an
assigned `ZDOTDIR` or `HOME` (`.zshenv`, and `.zprofile`, `.zshrc`, `.zlogin`
for a login or interactive zsh), and a login or interactive bash's under an
assigned `HOME` (`.bash_profile`, else `.bash_login`, else `.profile`, and
`.bashrc`), the files a login bash or zsh reads as it exits (`.bash_logout`,
`.zlogout`), and the other shells' profile (`.profile`, yash's `.yash_profile`)
and rc file (`.kshrc`, `.mkshrc`, `.yashrc`) under an assigned `HOME`. An
assignment counts as a prefix, through `env`, or made earlier on the line,
exported or not, since a variable the environment exports stays exported when
the line assigns it (`FPATH=dir; zsh -c 'autoload f; f'` loads `dir/f`), unless
the line takes the export away (`export -n`, `typeset +x`). A function file `autoload` (or `functions -u`, `typeset -fu`)
loads is read too: an operand with a slash is that file, and a name is searched
in the `FPATH` or `fpath` the line sets, where the first directory holding it
(as `dir.zwc`, `dir/name.zwc` or `dir/name`) is the file read; a compiled
`.zwc` file is a **block**, and so is a name or a directory searched before the
file is found that the line does not fix (`script-unread`). A line that leaves
`FPATH` as the account set it loads the account's own functions, unseen. An
`fpath` element that is the array itself (`fpath=(~/.zfunc $fpath)`,
`"${fpath[@]}"`, `$FPATH`) is its value before the assignment, and where the
line set none, and for `fpath+=(dir)`, that is the account's own directories:
the search passes over them, the fixed directories around them are read, and
a name only the account's directories hold stays unseen. A path run directly (`./deploy.sh`) has its first 8 KiB read only to
classify it: a shell-family shebang, or no shebang and no NUL byte, makes it a
shell script, read the same way; anything else is a program, allowed unread. A
relative path resolves against the working directory, and against a `cd` or
`pushd` earlier on the line only when its target exists and the script is
chained after it with `&&`; after any other `cd` the path is not resolved, and
the script is allowed unread, as an operand held in a variable or a substitution
is. bash and zsh read some `cd` and `pushd` forms differently, and the reader
does not know which shell runs the line, so it reads the scripts after such a
`cd` in the directory each reading enters, and blocks on a hazard in any of
them. With bash's `-e` or `-@`, bash enters the operand; with `pushd -n`, bash
stays put; with a second operand or more, an older bash enters the first and
a newer one refuses the `cd`, so the first is read as entered. zsh reads a second
operand, or an option word it does not take (`-e`, `-@`, `-n`, or any word
with a letter other than `L`, `P`, `q` and `s`, which bash refuses), as its
substitution form: it replaces the first occurrence of the first word in the
current directory's path with the second and enters the result where that
directory exists; such an option word alone (`cd -x`) is a directory zsh
enters by that name where one exists, unless it is a directory-stack word
(`cd -2`). Only where neither reading names a directory is the directory
unknown. A `pushd` with no operand swaps the top two directories of the
stack, never going to `HOME`, so the scripts after it are read in the
directory the line's own `pushd` left, and the directory is unknown where
the line did not fix the stack. A `..` in a path resolves the way the kernel opens it, against the directory
a symlink points at, so after `cd link` the file `../s.sh` is the one beside the
link's target. A `cd` or `pushd` whose operand the shell searches `CDPATH` (or
zsh's `cdpath`) for — not absolute, not starting with `./` or `../` — while
`CDPATH` is set on the line or in the environment the check runs in enters the
one directory holding the operand when exactly one exists among the current
directory and each `CDPATH` entry, and the scripts after it are read there;
where none or two exist, `CDPATH` holds more entries than the guard looks
through, or an entry or the current directory is not fixed by the line, it is
a **block** (`script-unread`) whose reason says whether the line set
`CDPATH` or the environment exported it: name the directory as `./sub` or by
its absolute path. After `unset CDPATH` on the line (or zsh's `unset cdpath`,
which in bash is another variable) no `CDPATH` is searched, in that shell or in
a child it starts. The verdicts:

- a registry blocker at command position in the script is a **block**
  (`script-runs-hazard`), naming the script, the line and the entry, and so is
  an entry-less block inside it (a stream, a payload past the depth); a command
  the registry only warns on inside a script (`git clean` on its own scratch),
  and a speculative (Tier 2) hit, stay inside it, so running the script does
  not warn;
- a script written earlier on the same line — by a redirection, or as the
  target of `tee`, `cp`, `mv`, `install`, `ln`, `dd of=`, `curl -o`/`-O`, `wget -O`,
  `sed -i`, `patch`, `git checkout`/`restore`/`clone`, `tar -x`, `unzip`,
  `rsync`'s destination, `go build -o`, `sort -o`, `ditto`'s destination,
  `split`'s prefix, `uniq`'s output file, or a command a `find -exec` or
  `-execdir` runs — is a
  **block** (`script-written-then-run`): the file the guard reads now is not the
  file that runs. A file the line writes and then runs by path is a **block**
  the same way, whatever the file is (`curl -o x … && ./x`). Write it in one
  command and run it in the next. A write the guard cannot place (`> "$LOG"`)
  before a script a shell is pointed at is a **warn**;
- a script that does not exist is allowed with a note naming it (the shell
  refuses it); a startup file that does not exist is skipped, silently;
- a `BASH_ENV` or `ENV` value holding a `$` or a backquote (`BASH_ENV='$HOME/f'`,
  a substitution) is a **block** (`script-unread`): the shell expands it before
  it opens the file, so a substitution in it runs and the file it names is not
  one the guard can read;
- a file a shell is handed that holds a NUL byte in its first 8 KiB is a
  **block** (`script-unread`); one the guard cannot read, or a shell script over
  256 KiB, is a **warn** (`script-unread`);
- a script that runs a script is read in turn, sharing the two-layer depth with
  `sh -c`, and one past it is a **block**; one check reads at most 16 files and
  1 MiB, and past that it **warns** (`script-unread`).

What the reading does not see: another interpreter's file (`python3 f.py`,
`node f.js`, `ruby`, `perl`, a `Makefile`, `npm run`); a program run directly or
found through `PATH`, including a `PATH` assignment on the line that changes
which file a bare name finds; the files a shell loads from the account's real
`HOME` (`~/.bashrc`, `~/.zshenv`, `/etc/profile`), though a `source` naming one is
read like any other; file text substituted into a command string (`eval "$(cat
f)"`, `bash -c "$(<f)"`), which keeps its warn; a file changed between the
check and the run; a writer missing from the list above; a variable a sourced
file exports, or one `set -a` exports; a `ZDOTDIR` a `.zshenv` sets for the
files after it; a login shell made by its name (`exec -a -bash bash`) or for
another account (`su -l`, `sudo -i`); the `.profile` a login ksh can read from
the current directory; `INPUTRC` (key bindings, not commands); a
`PROMPT_COMMAND` set as an array; and any spelling not listed here.

Startup text a line carries in a variable, rather than in a file, is judged as
the command it becomes. A `PS0`, `PS1`, `PS2` or `PS4` value is decoded the way
bash decodes a prompt (an octal escape such as `\044` is read as the `$` it
spells), and each command substitution in it is judged, because bash expands a
traced command's `PS4` and an interactive prompt; a `PROMPT_COMMAND` value is
judged as the command line bash runs before each prompt. Either is read wherever
the line assigns it: as a prefix, as an `env` operand, or as the argument of a
declaration builtin (`export PS4=…`). `env` and `sudo` take every operand that
carries `=` before the command as an assignment, and so does the guard; the body
of a function they export (`BASH_FUNC_<name>%%=() { …; }`) is judged. A `trap`
action, and the callback of `mapfile -C` or `readarray -C`, is read the way an
`eval` string is: the reset and list forms (`trap - EXIT`, `trap -p`) carry no
command, and text the guard cannot read keeps the verdict `eval` gives it.

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

A kill is read with where its pids come from. A `kill` handed what a process
search prints — `kill $(pgrep -f make)`, `pgrep -f make | xargs kill`, `kill
$(pidof make)` — is a **block** (`kill-by-search`), because it signals every
matching process on the machine, as `pkill` does; a kill of a pid you name, one
you recorded (`kill $(cat pidfile)`), or a search of your own group (`kill
$(pgrep -g <pgid>)`) is not. Stopping whatever holds a port or a file is read
the same way: `fuser -k` (`fuser-kill`), a `kill` handed what `lsof` or
`fuser` prints (`kill $(lsof -t -i :8080)`, `lsof -ti:8080 | xargs kill`;
`kill-by-holder`) and `kill-port` (`npx kill-port 3000`; `kill-port`) are a
**block**, because they signal every process that holds it, another session's
server included; looking (`lsof -i :8080`, `fuser 8080/tcp`) is not. The
package runners — `npx`, `npm exec` (and `npm x`), `pnpm dlx`, `pnpm exec`,
`yarn dlx`, `bunx` and `bun x` — are read as wrappers whose first operand is
the program they run, a version after it (`kill-port@latest`) included, so a
blocker behind one is read as the blocker. The package manager's own options
before its subcommand are stepped with their values (`pnpm --filter web exec`,
`npm --prefix sub exec`, `yarn --cwd sub dlx`), by each manager's documented
global value flags; one it does not know leaves the runner unread, and a
blocker behind it takes the fail-safe warning. The shell string a runner hands to
a shell is read as a script, as a `bash -c` string is: the value of `-c` or
`--call` after `npx` or `npm exec`, and the command words after `pnpm exec -c`
or `--shell-mode`, so a blocker inside it is read as the blocker too; a `-c`
after the program a runner starts is that program's own. The search is followed through a group (`{ pgrep
…; } | xargs kill`) and into one, to every command in it (`pgrep … | { sleep 1;
xargs kill; }`), through a shell string that runs it (`kill $(sh -c 'pgrep
…')`), and into a shell string that `xargs` runs or that reads the pipe or a
redirect (`pgrep … | xargs sh -c 'kill "$@"' _`, `pgrep … | sh -c 'xargs
kill'`, `sh -c 'xargs kill' < <(pgrep …)`), or whose positional parameters or
own text hold the search's output (`sh -c 'kill "$1"' _ "$(pgrep …)"`); out
of an unquoted here-document whose body holds the search (`xargs kill <<EOF`
over `$(pgrep …)`), and on through a pipe from the command that reads it
(`cat <<EOF | xargs kill`); and into a substitution in a command that reads a pipe,
which runs with that pipe as its input (`pgrep … | echo "$(xargs kill)"`);
behind `xargs` and a launcher the guard does not know, the fail-safe warns. Every command of a string `xargs`
runs, and every command in a group a pipe feeds, is read as handed that input,
so `pgrep … | xargs sh -c 'kill 4242'` is a **block** too: the guard does not
read which of the commands uses it. The group ends only at a `}` the line
leaves wholly unquoted: a `}` with a quote in it (`}""`, `}''`) is a command's
name, so `echo … | { }""; bash; }` is a **block** (`interpreter-reads-stream`),
as `echo … | { true; bash; }` is. A `pkill` or `killall` that selects by user, group or terminal
(`-u`, `-t`, and `pkill`'s `-U` and `-G`, written apart from the value or
against it, as `pkill -tpts/3`) is a **block** too, under `pkill-by-owner`,
`killall-by-owner` or the entry for a kill by name, because every session under
the account is among what it selects. `pkill`'s signal is read as a signal
first, in any case and with or without its `SIG` prefix, as `pkill` reads it,
so `pkill -term -g <pgid>` stops a group and stays allowed. A pid list a search
printed into a variable the same line assigns is read as the search's output
(`p=$(pgrep make); kill $p`, `p=$(pgrep make); echo "$p" | xargs kill`), and
so is one written to a file the same line then reads (`pgrep make > p; kill
$(cat p)`, `xargs kill < p`, `while read q; do kill $q; done < p`): each is a
**block**. The file is matched by the name the line writes it under, and a
name the line does not fix can be any file, except a word whose only unfixed
parts are variables the line does not set: that is a value from before the
line, so `pgrep -l make > search.log; echo "$pid" | xargs kill` stays allowed,
though such a word still reads a file the line wrote under a name it does not
fix (`pgrep make > "$f"; kill $(cat "$f")`). One written under another
spelling of the path, or by a command that names it as an operand (`tee p`),
is not followed. A `ps` listing filtered by a name — through `grep`, or an
`awk` program that matches (`ps aux | grep make | awk '{print $2}' | xargs
kill`, `ps aux | awk '/make/ {print $2}'`) — is a search as `pgrep` is, and a
**block** under `kill-by-search`; a `ps` that reports the pid you name (`ps -o
pid= -p 4242 | xargs kill`) is not.
Every command of a string a shell is handed with such output in its words is
read as handed it, so `sh -c 'kill 4242' _ "$(pgrep …)"` is a **block** too.

A verb that is the person's own act is left to the person. `abcd library ledger
--flip N` records that a human chose to cite a source publicly, so an agent
running it is a **block** (`abcd-source-ledger-flip`); recording a line and
listing the ledger are not. The retired spelling, `abcd source ledger --flip N`,
is a **block** too (`abcd-source-ledger-flip-retired`), because a binary older
than the library verb still answers it. The entries name the program `abcd`,
so they match the binary on `PATH` and the plugin root's by basename, not a
`go run` of the source or a copy under another name.

Starting another agent harness with its own permission checks turned off is a
**block**, because that agent then runs every command with nobody asked and
none of its commands reach this guard: `claude --dangerously-skip-permissions`
(`agent-launch-claude-unchecked`), `claude --permission-mode bypassPermissions`
(`agent-launch-claude-unchecked-mode`), `codex --dangerously-bypass-approvals-and-sandbox`
or `--yolo` (`agent-launch-codex-unchecked`), `codex -a never`
(`agent-launch-codex-never-asks`), `codex -s danger-full-access`
(`agent-launch-codex-sandbox-off`), `codex -c approval_policy=never` or
`-c sandbox_mode=danger-full-access` (`agent-launch-codex-config-unchecked`),
and `opencode run --auto` (`agent-launch-opencode-auto`). The same harness
started with its checks on is allowed. The entries name each program by
basename, and match it however the line starts it: by name or path, behind `env`
or `command`, or through a package runner (`npx`, `bunx`, `bun x`, `pnpm dlx`,
`pnpm exec`, `yarn dlx`, `npm exec`, with or without `--`), by the program's
name or by the package that ships it (`npx @anthropic-ai/claude-code`, `npx
@openai/codex`, `npx opencode-ai`). opencode also reads permission rules from
its environment, so a line that sets `OPENCODE_PERMISSION` for the opencode it
starts — as an assignment before it, through `env`, or with an `export`,
`declare -x` or `typeset -x` earlier in the line — is a **block** too (`agent-launch-opencode-permission-env`), and
one that starts opencode without setting it is not. So is a line that sets
`OPENCODE_CONFIG`, a configuration file opencode loads, or
`OPENCODE_CONFIG_CONTENT`, configuration text it merges last, for the opencode
it starts (`agent-launch-opencode-config-env`), since either can carry
permission rules over opencode's own configuration. A harness started under
another name, a copied or linked binary run by that name, is not recognisable
as a harness at this layer; the OS sandbox for launched agents
(spc-2610161751056096) is what bounds what such a run can reach
(iss-2610162027075062). The list of agent launches is a mistake filter: it
steers an agent that starts another harness by an ordinary route, a harness
started by any route outside it (another name, a copied binary, a
configuration file it reads) is bounded by that sandbox and not by the list,
so a further form is added when it is common, not to close the class
(dcn-2610162040034117). <!-- record-lint: forward-looking -->

A recursive delete of the filesystem root or the home directory (`/`, `/*`,
`~`, `$HOME`, `${HOME}`, each also with a trailing `/`, `/*` or `/*/`, and the
home's dotfiles `~/.*`, `$HOME/.*`, `${HOME}/.*`, each also with a trailing `/`)
is a **block**
(`rm-rf-root-or-home`), with or without `-f`; one of the directory the shell is
in or the one above it (`*`, `*/`, `.`, `..`, `./*`, `./*/`, `../*`, `../*/`, `.*`,
`./.*`, and `$PWD` or `${PWD}`, each also with `/*` or `/*/`) is a **warn**
(`rm-rf-working-directory`), and so is a chain of parents, which holds the one
parent: a leading run of `..` segments reads as one `..`, so `../..`,
`../../..` and `../../*` are a **warn** as `..` and `../*` are. The target is compared as written, so `$HOME` and
`$PWD` are seen as those words, and read the way bash reads its text first: a
backslash-newline inside the name is dropped, a brace group's words keep their
variables (`{$HOME,x}`, `$HO{M..M}E`), an expansion that can leave the value
as it is reads as the variable (`${HOME%/}`, `${HOME:-x}`, `${HOME[0]}`), and
an alternative reads as its word (`${X:+$HOME}`, `${X:+/}`), split on
whitespace where it stands unquoted (`${X:+$HOME }`). A target is also read
with its redundant separators taken out (`//*`, `$HOME//`, `/./*`, `/../*`),
and one that begins at the root or the home with each `..` folded into the
directory before it, as the path reads: `/tmp/../*` is `/*`, and `~/../*`
globs the home's parent, which holds the home, so it is a **block** as `~` is.
A `..` past `$PWD` or past the start of a relative path is the directory above
the working directory, so `$PWD/../*` and `x/../../*` are a **warn** as `../*`
is, while `x/../*`, which stays inside it, is compared as written. A target is
also read the way its glob can expand: a run of `*` as one `*`, as every shell
without globstar expands `**`, so `/**` and `~/../**` are a **block**; and a
segment written with a leading `.` that can match `..` (`.?`, `.*`, `.[.]`) as
`..` where a further segment follows it, as the bash 3.2 and `/bin/sh` of
macOS expand `~/.?/*` to include `~/../*`. A bracket expression there that the
guard cannot decide reads as able to match `..`, so `~/.[[:punct:]]/*`,
`~/.[].]/*` and `~/.[--.]/*` are a **block**, and so is `~/.[a-z]/*`, which no
shell expands that far. That is an over-block on bash 5.3, which leaves the
segment unexpanded, and an escaped glob is read as unescaped (`~/*\*` blocks
as `~/*` does); a final dot-glob (`~/.?`) is not read so,
since rm refuses a last segment `..`. Quoted, `'/**'` and `"~/.?"/*` block as
`"/*"` does.

An `rm` whose path starts with a variable that can be empty, directly followed
by `/`, is a **block** whatever its flags (`rm-unguarded-variable-path`):
`"$VAR"/*`, `$VAR/x`, `"${VAR}"/x`, `${VAR}/x` and `"$VAR/x"` name a path from
the filesystem root when the variable is empty or unset, and nothing on the
line says which. So are `"$1"/x`, `"$A$B"/x`, an expansion that can print
nothing (`${VAR:-}/x`, `${VAR:+x}/x`), and a default whose word is such a
variable (`"${X:-$Y}"/x`), inside a string a shell runs as on the line. The
successor is the rewrite, `"${VAR:?}"/...`, which stops the shell on an empty
value, or a literal path; neither is refused, and nor is a default that cannot
be empty (`"${TMPDIR:-/tmp}"/x`), a variable later in the path
(`./build/$name`), a bare `"$VAR"`, which empties to no path at all, a trim, a
replacement, a substring or a case change (`"${DIR%/}"/x`), or `$HOME` and
`$PWD`, whose deletes the two entries above name. The guard never rewrites the
command itself. The same rule is restated in every agent definition that can
run a shell, in a block generated from this entry, because a sub-agent is
handed none of the `SHELL` rules the session is taught; the scribe's is the
one exception, its definition being held to ledger content alone.

What an allow still does not see is a hazard that never reaches command position
at all: a delete target printed whole by a substitution (`rm -rf $(echo /)`),
read by its known text the way `rm -rf $(find …)` names its targets every day,
or spelled any other way than the words above, as is a `..` after a symlink,
which the path is read past lexically, or after a segment holding a variable (`/tmp/$X/../*`) or a
`~user` home (`~root/../../*`); one launched through a known wrapper carrying a value-taking flag the
guard does not name (`sudo -u bob <hazard>` is seen; the bundled short form
`sudo -Hu bob <hazard>` reaches only the warn, not the entry that names it), one
whose API path an entry names by its ROOT
segment but the host serves under a prefix (a GitHub Enterprise Server install
mounts the same endpoints under `/api/v3/`; the `https://api.github.com/…` URL
form **is** read), what a variable the line did not assign carries in from an
earlier command — a pid list, a stream path, shell text run through `eval "$X"`
or a string a shell runs, or `pkill` or `killall` as the program a variable
names (`$P make`) — because reading each would refuse the ordinary commands a
variable carries a value for (a value the line assigns and then uses **is**
read; a value a `+=` adds to, or a word the value splits into when it stands
unquoted, is not), an IFS the shell already holds when
the line starts or gains during the line through a name the guard does not read
(a sourced file, a nameref set before the line, or an operator the line does
not write, such as a command's output an arithmetic context evaluates; every
line is read from the default IFS), a hazard inside a non-shell interpreter's payload (`python -c`,
`perl -e`) — one opaque token the tokenizer cannot read, today a silent allow, not
a warn (a warn for it is a recorded design target, not yet implemented), or a
dangerous form no entry describes. Nor does an allow see what a lone substitution
prints when it stands as the whole command (`$(cat msg.txt)`, `$(date)`): the
name can be any program, but with no operand after it no entry matches, so it
allows, because an allow means no entry matched. What a substitution prints is
read only for the exact shape `cat <<DELIM`, a newline, the body, the delimiter
line and blanks; `/bin/cat`, `command cat`, `cat -`, a redirection after the
delimiter word, a backslash-newline before the `<<`, a command after the
document, a backslash-newline after a backtick's document, and an output inside
a `${…}` all leave it unknown, so each of those alone as the command allows,
though bash runs what it prints (handed to `sh -c` as its string, each warns).
Coverage is what the registry names. Say exactly this if a user asks about
coverage — never that the guard cleared the command.

In a repository with more than one worktree, a `git stash` or `git stash pop`
that does not name its entry is a **warn** (`git-stash-shared-stack`): git keeps
one stash stack for the whole repository, so a bare pop can take another
worktree's work. A stash with a message and a pop by entry (`stash@{N}`) are not
warned about, and neither is anything in a single-worktree clone.

A `bash -c` or `sh -c` script built from variables that runs `rm`, `rmdir`,
`shred` or `unlink` is a **block** (`shell-c-variable-delete`): what it deletes
is named only when it runs, so no check made before it can read it, and a host
whose own check cannot read it stops to ask a person, where an agent working
alone waits. The way past is to write the script to a file in one command and
run it in the next, which the guard reads line by line, or to delete on the
line itself with literal paths (`/bin/rm -f -- /abs/path`) or a variable guarded
as `"${VAR:?}"`. It reads only a shell's own `-c` string: one with no variable
in it (`bash -c 'rm -f /x/a.png'`), or one that deletes nothing (`bash -c 'ls
"$D"'`), is not refused. Where `rm-unguarded-variable-path` fires inside the
same string, this refusal speaks for both.

In a repository that opted in with `ahoy install --outbound`
(`outbound.forge: true` in `.abcd/config.json`), the text a posting forge
command hands the forge is judged against the outbound policy before the
command runs, with the same check `abcd lint outbound` applies. A
`gh pr create`, `gh pr new`, `gh pr edit`, `gh pr comment`, `gh issue create`,
`gh issue new`, `gh issue edit` or `gh issue comment` whose title or body
carries a live agent-session URL or a tool attribution footer is a **block**
(`forge-text-outbound`). The refusal names which text it was (the title, the
body or the body file), the kind of finding and its line, never the text, and
teaches the safe form: remove the session URL, and
replace a tool footer with the repository's own disclosure trailer
(`Assisted-by: <Vendor>:<model-version>`). It reads `--title`/`-t`,
`--body`/`-b` and `--body-file`/`-F` in every spelling (`--body x`,
`--body=x`, `-b x`, `-bx`, `-b=x`, and each behind gh's boolean shorthands, `-db x`),
a value written out in full on the line, and
a `"$(cat <<'EOF' … EOF)"` body, read as the document it prints. A body file is
read through the guard's guarded read, from the directory the command runs in:
regular files only, no symlinked leaf (a symlink to a file is followed first),
at most 256 KiB, opened without waiting, so a named pipe can never hang the
check. A body file it cannot read is a **warn** (`forge-text-unread`), and the
command runs: one that does not exist, is a pipe, a directory or a file this
account may not read, is over 256 KiB, sits relative to a directory the line
leaves unknown, or is written earlier on the same line (the file the guard could
read is not the one gh will post). So is a `.abcd/config/pii.json` that does not
load, and a line handing the forge more than sixteen texts. A warn never stops a
command, and it never passes one in silence.

A pull request's own text is held to two more checks before it is posted,
each of which the hosted checks would otherwise make only after the push. A
`gh pr create`, `gh pr new` or `gh pr edit` whose body or body file carries no
`Assisted-by:` line in one of the three accepted forms
(`Assisted-by: <Vendor>:<model-version>`, `Assisted-by: None`, or the
`Assisted-by: abcd:<version>` label abcd writes) is a **block**
(`forge-text-disclosure`). So is one carrying a `Co-authored-by:` line, a
tool's "Generated with" footer, or an `abcd:` label in a form abcd never writes.
The body is judged as the attribution gate's body arm judges it: fenced blocks
are stripped first, since the forge renders them as examples, carriage returns
are dropped, and the line must stand alone, so a trailer in the middle of a
sentence does not count. A title, a comment and an issue are not held to it. In
a repository that also sets `outbound.declare_mentions: true`, a pull request
whose title and body name an issue id (`iss-N`) without a `Resolves: iss-N` or
`Refs: iss-N` line declaring it is a **block** too (`forge-text-mention`): the
rule CI's issue-resolution gate applies to the pull request (RS004), read from
both halves together, since a title has no room for a trailer. Neither check
judges a body the line does not fix or the guard could not read, and the mention
check judges nothing when any of the pull request's text is hidden from it (a
variable title, a body on standard input): the hidden part may hold the
declaration.

It does not see: a body or a path the line does not fix (`--body "$BODY"`,
`--body-file "$F"`, a substitution other than the `cat` document above), a body
on standard input (`--body-file -`), a flag spelled by a substitution
(`--$(echo body)`), text typed into an editor or a browser (`--editor`, `--web`), text gh takes from the
commits (`--fill`) or a template (`--template`), a posting command inside a
script the line runs, and every other posting path: `gh api`, `gh pr review`,
`gh release`, `gh pr merge --body`, `gh pr close --comment`,
`gh issue close --comment`, and any forge client but gh. The re-read
after posting stays the control for those.

Nothing changes in a repository that did not opt in. The opt-in is read from
the working tree on each check, and `outbound.declare_mentions` counts only
beside `outbound.forge`: RS004 is a rule a repository adopts, so `ahoy install`
never sets it. `--outbound` is one switch for this check and
the commit-message hook together, so to turn it off remove both
`outbound.forge` and `outbound.hook` from `.abcd/config.json` (or set both to
`false`): a later `ahoy install` leaves both off as they are, but completes a
repository that keeps only one of them on.

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
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

Whichever rung fires, the candidate still goes in on stdin via the
quoted-delimiter heredoc shown above.

**User input:** $ARGUMENTS
