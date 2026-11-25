# `/abcd:start` — Start a Session on abcd Alone

`/abcd:start` is the one command a person types to work in a session on abcd alone
(itd-2611251725203320, spc-2611251757236507): the few tools the work needs, abcd's
hooks and the repository's conventions file, and nothing else of their set-up. It is
the drive's wall turned to a session, not a second wall. It is a person's verb, listed
in the set-up group, and theirs to type: it refuses where a host session's marker is set
or a stream is not a terminal, which catches a start made by mistake and is not a wall.
It takes one optional prompt, the profile's name and a permission mode; bare, it is the start
itself.

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

## The two outcomes

**An account with no profile.** It makes one directory under
`~/.abcd.noindex/profiles/`, named `drive` unless the person gives another plain name,
and writes abcd's settings there, the drive's own rendering, with an empty `skills`
folder closed to writes, so the host syncs none of the account's skills in. It prints
the host's own login command and starts nothing. The checkout is not looked at, since
the profile is the account's. The login is the person's, in their own shell: abcd never
performs it, and never reads, stores or forwards its token. It makes an account's first
profile and no other.

**A profile that is logged in.** It runs one short launch with the session's arguments,
refuses naming anything that launch reports which the printed list does not name, and
otherwise replaces the abcd process with the host. It changes no file in the profile,
and says what it found. After its last line abcd says nothing more.

A profile is a separate login and settings on one operating-system account, never a
sandbox: a command the session runs can read whatever the account can, the profile
directory included.

## The order of the refusals

Each is exit 2 and opens nothing, and each carries its own code but the third, which is
a usage refusal that repeats no value:

1. A host session's marker is set in the environment (`start/outside-a-terminal`).
2. Standard input, output or error is not a terminal (the same code). Those two are the
   only things read about where the command runs. They prove nothing about who typed it.
3. A permission mode outside the set, or a profile name that is not plain.
4. The profile (`start/profile-unusable`): several and none named, a name the account
   does not hold, a directory abcd does not trust, or one holding something a launch
   would load.
5. The checkout (`start/repository-unready`): none here, one abcd does not manage, or a
   conventions file that is absent, a link, not regular, holds a NUL byte or is over
   100 KiB.
6. The host or the copy of abcd cannot be run, or the machine's runner configuration
   cannot be read, so no ledger opens (`start/host-not-runnable`).
7. The login (`start/not-logged-in`, with the login command; `start/no-verdict` where the
   host gives no answer).
8. The short launch: `start/no-verdict` where it met a limit, did not finish, was
   interrupted or could not be recorded, and nothing waits or is tried again;
   `start/wall-not-held` where it reported what the list does not name, or no list.
   That code is also the refusal where the checkout's own guard is switched off or does
   not refuse the command it is meant to refuse, read as the guard hook reads it, before
   the launch and again before the session opens.

## What the session holds

The host is handed a fixed list of tools, and no settings source, server, browser tool
or plugin but its own. Its settings are one launch argument: abcd's hooks, the host's
attribution off, the built-in plugins abcd knows of off, and a rule denying every
server's tool without naming one. The conventions file, `AGENTS.md` at the checkout's
root, goes whole as a launch argument, as it stands at that moment. The environment is
the person's own, home included, less what the host signs in or bills through, every
other variable of the host's and a walled stage's; the names left out are printed and
no value is. A permission mode is passed only when the person names one. No repository
file starts a session, adds a tool or chooses a mode.

The hooks cover every event the plugin's manifest wires, to the same verbs, and a test
holds the two together. An event whose verb the binary lacks is left out and said,
never wired to a verb that would fail.

They run a copy of the abcd that was typed, kept in `~/.abcd.noindex/session-bin/` under
its content's digest, so an update, a rebuild or a removed worktree changes nothing an
open session runs. In a checkout listed in `dogfood-roots` the hooks are that build, not
the reviewed branch's. Nothing reclaims a copy, since a live session cannot be told from
an ended one; one may be cleared when no session started from it is open and no
profile's `settings.json` names it.

## What the short launch reads

It runs on the profile in an empty repository of its own, in print mode, with abcd's own
prompt and no permission mode of the person's, and it is written to the launch ledger as
`session-proof`; the session is not. It reads the host's first event, for the tools, the
servers, the plugins, the skills, agents and commands, and the key source, and the
guard's answer to one command it is meant to refuse. The repository holds settings, a
server, an agent, a skill and a command page the launch must not load, and a launch that
names any of them, or a plugin the settings switch off, is refused. A listed tool it does not report is said and refuses
nothing, because a print launch reports no question tool.

It does not read a checkout's own `.claude` folder, what an interactive session alone
loads, or what the host adds after the start. abcd says the last of these as it opens
the session.

## Where it is named

The drive's refusal on an account with no profile, and the warning that a session's
setting reads as off a profile ([`26-mode.md`](26-mode.md)), name this verb. The drive
([`35-drain.md`](35-drain.md)) runs on the same profile and stays the unattended start.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd start`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--permission-mode` | string |
| `--profile` | string |

<!-- surface-appendix:end -->
