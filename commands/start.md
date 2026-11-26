---
name: start
description: "Start a session on abcd alone, making its separate login where none exists: Writes that profile once, a copy of abcd, a ledger entry; refuses off a terminal."
argument-hint: "[--profile <name>] [--permission-mode <mode>] [prompt]"
block: people
---

# `/abcd:start`

<!-- generated: ask-the-binary -->
Before you fill in an input for this command, run `"${CLAUDE_PLUGIN_ROOT}/abcd" explain start`, and where its answer and this page disagree, follow the binary, since this page may be older than it.
<!-- /generated -->

A session on abcd alone: the few tools the work needs, abcd's hooks and the
repository's conventions file, and nothing else of the person's set-up.

**This command is the person's to type; never run it for them.** It refuses, with exit
2 and nothing written, when a variable a host sets inside its own session is set (an
agent's shell) or when standard input, output or error is not a terminal. Those two
checks catch a start made by mistake. They are not a wall, and prove nothing about who
typed the line. Give the person the line to type, in a plain terminal, in a checkout abcd
manages:

- `abcd start` opens waiting for them;
- `abcd start "read the handover first"` opens with that prompt.

`--profile <name>` names the profile where the account holds several, and
`--permission-mode <mode>` the mode the session starts in. With no mode named the
host's default holds, and no file is read for one.

## The first run

On an account with no profile it makes one, a directory under
`~/.abcd.noindex/profiles/`, named `drive` unless `--profile` gives another plain
name. It writes abcd's own settings there (`settings.json`, and an empty `skills`
folder closed to writes) and nothing of the person's. Then it prints the host's
own login command for that directory, and starts nothing.

**The login is the person's, in their own shell.** abcd never performs it, and never
reads, stores or forwards its token. After it, they type `abcd start` again.
`abcd drain --drive` runs on the same profile. A second profile is made by hand, as
`/abcd:drain` shows: with one or more there, a name the account does not hold is
refused and nothing is made, so a mistyped name makes no second login.

A profile is a separate login and settings on this one account, never a sandbox: a
command the session runs can read whatever the account can, the profile directory
included. The host's Read tool is kept out of that directory; its other tools and the
shell are not.

## A later run

On a profile that is there it changes no file in it and says what it found. It refuses,
opening nothing, when:

- the account holds several profiles and none is named, or the one named is not held
  (it lists those that are), or the directory is one abcd does not trust;
- the profile holds something a launch would load, a synced `skills` folder among
  them: a start of `abcd drain --drive` on that profile sets that one aside, and any
  other is moved out by hand;
- this is not a checkout abcd manages, or its `AGENTS.md` is absent, a link, not a
  regular file, holds a NUL byte or is over 100 KiB, since it is handed over whole;
- the host cannot be run, or a copy of abcd cannot be kept;
- the profile is not logged in, which prints the login command again;
- the short launch came to no verdict: a limit, a launch that did not finish, an
  interrupt, or a launch ledger that could not record it. Nothing waits and nothing
  is tried again;
- the short launch did not show that the wall holds: it reported something the printed
  list does not name, or reported no list;
- the guard of this checkout is switched off in `.abcd/guard.json`, or that file leaves
  the guard not refusing the command it is meant to refuse. That is read before the
  launch, and again before the session opens.

## The short launch

Before the session opens, abcd runs one short launch on that profile with the
session's own arguments, in print mode, with abcd's own prompt and without the person's
permission mode. It spends a little of the subscription and is written to the launch
ledger (`/abcd:agents`); the session itself is not. A tool, a server, a plugin or a key
that launch reports and the list does not name refuses the start, naming it. So does a
plugin the settings switch off, an agent, a skill or a command page planted in the
launch's own directory, and a guard that was not seen to refuse the command it is meant
to refuse. A listed tool it does not report is said and refuses nothing.

The short launch runs in an empty repository and does not read a checkout's own
`.claude` folder. It is one launch, not the open session: what the host adds after the
start abcd cannot see.

## What the session holds

- **Tools:** `Bash`, `Read`, `Edit`, `Write`, `Agent`, `TaskStop`, `AskUserQuestion`,
  `Monitor`, `ScheduleWakeup`, `CronCreate`, `CronList` and `CronDelete`.
- **abcd's hooks**, handed over as a launch argument: every event the plugin's manifest
  wires. An event whose hook this binary lacks is left out, and said.
- **The conventions file**, `AGENTS.md` at the checkout's root, whole, as it stands at
  that moment, handed over as a private file the session reads: a copy only this
  account can read, kept in `~/.abcd.noindex/session-conventions/` under its content's
  digest. A later edit does not reach an open session.
- **The person's home and shell environment**, less what the host signs in or bills
  through, every other variable of the host's and a walled stage's. The names left out
  are printed, never a value.

It holds no settings file, no server and no server's tool, no browser tool, no plugin
but the host's own, and none of the person's instructions, memory, skills or connected
services. No repository file starts a session, and none adds a tool to one.

## The copy of abcd its hooks run

The hooks run a copy of the abcd that was typed, kept in
`~/.abcd.noindex/session-bin/` under its content's digest, so an update, a rebuild or a
removed worktree changes nothing an open session runs. In a checkout listed in
`dogfood-roots` the session's hooks are the build that typed `abcd start`, not the
reviewed branch's. abcd reclaims no copy. One may be cleared when no session started
from it is open and no profile's `settings.json` names it: a session started by hand on
such a profile would otherwise run with every hook missing. The same holds for the copies
of the conventions text: abcd reclaims none, and one may be cleared when no session started
from it is open.

After its last line abcd has nothing more to say: the host takes its place in the
terminal. `/abcd:session` is another thing, the live brief sessions.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

<!-- generated: command-facts -->
**Command facts, format version 2.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain start` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd start [prompt] [flags]`
  - An example call is `abcd start`.
<!-- /generated -->
