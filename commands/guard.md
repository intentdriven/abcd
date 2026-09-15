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

## `hook` — the host adapter

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" guard hook
```

Reads a host pre-tool-use hook payload on stdin and applies the same decision
before a shell command executes. It is invoked by the plugin's hook manifest,
not by hand; a blocker returns the host's blocking status with the successor and
the why as the message, and a warn or an allow lets the command run.

Anything the adapter cannot turn into a decision — an unreadable payload, a tool
call that is not a shell command, an unparsable command line, a registry that
does not load — allows the command and warns loudly. A guard that cannot answer
never stops a session, and is never silently absent. A command line a shell
would run is never in that set: a trailing backslash and an unterminated
here-document are decided, not failed open on, and a here-document body is read
as data however it is quoted, even when the line that opened it ends in `&&`.

## Registry and overrides

The bundled hazards ship inside the binary. A repo overrides them in one file in
the repository, `.abcd/guard.json`: add an entry key to change one field (for
example `{"tier": "warn"}`) or to declare a new hazard, and set
`{"disabled": true}` to switch the guard off entirely.

There is no flag, environment variable, or prompt that turns the guard off for a
session — the file is the only route, so the change lands in a diff someone
reviews. Two things follow, and both must be said plainly if a user asks. The
file is read from the working tree, so an edit takes effect on the very next
command, before anyone has reviewed it. And a repo whose guard is switched off is
an unguarded session: every command it lets through carries an UNGUARDED warning
naming the file, so the state cannot pass unnoticed.

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

A git alias declared in the command line is expanded before the match, because
git resolves it before it runs: `git -c alias.p='push --force' p origin main` is
a force push, and so are its `--config-env`, `GIT_CONFIG_KEY_n`/`VALUE_n` and
`GIT_CONFIG_PARAMETERS` spellings. A `!` body is read as a shell command. Two
consequences to report accurately: an alias that shadows a git builtin
(`-c alias.push='push --force' push`) is refused even though git would ignore
it — an accepted over-block — and configuration delivered from a FILE
(`GIT_CONFIG_GLOBAL`, `-c include.path=`, an `--config-env` variable the line
does not set) is a **warn** under `git-config-rewrite-unread`, because the
directive is visible and its body is not.

What an allow still does not see is a hazard that never reaches command position
at all: one launched through a known wrapper carrying a value-taking flag the
guard does not name (`sudo -u bob <hazard>` is seen; the bundled short form
`sudo -Hu bob <hazard>` reaches only the warn, not the entry that names it), one
whose API path an entry names by its ROOT
segment but the host serves under a prefix (a GitHub Enterprise Server install
mounts the same endpoints under `/api/v3/`; the `https://api.github.com/…` URL
form **is** read), a hazard inside a non-shell interpreter's payload (`python -c`,
`perl -e`) — one opaque token the tokenizer cannot read, today a silent allow, not
a warn (a warn for it is a recorded design target, not yet implemented), or a
dangerous form no entry describes.
Coverage is what the registry names. Say exactly this if a user asks about
coverage — never that the guard cleared the command.

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
