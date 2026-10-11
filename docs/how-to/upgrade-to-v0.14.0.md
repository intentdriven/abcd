# Upgrade to v0.14.0

v0.14.0 is a breaking release. The `/abcd:` menu in your agent session lists
only the commands people run, and seven command names go with no alias: three
fold into a command that already does their job, and four become one command,
`library`. A script or a hook that runs an old command name gets a refusal
that names what replaced it. This guide lists every retired name with its
successor, updates the pre-commit hook in each project abcd manages, and says
where the agents' commands went.

## Replace the retired commands

Each command-line spelling below is an unknown command in v0.14.0: the
refusal exits 2 and its second line names the successor. Each `/abcd:` page
below is gone from the menu, and the page named beside it does its job:

| Retired | Use instead |
|---------|-------------|
| `/abcd:version` | `/abcd:abcd` (or bare `abcd`): the board's last line is the installed version. `abcd --version` still answers. |
| `abcd changelog` | `abcd launch --dry-run`, which renders the release cut after the bundle report. |
| `/abcd:prepare-this-repo` | `/abcd:ahoy install` |
| `abcd memory` | `abcd library` |
| `abcd source` | `abcd library` |
| `/abcd:memory`, `/abcd:source` | `/abcd:library` |
| `/abcd:consult` | `/abcd:library`, which searches the sources corpus and records what influenced a decision with `abcd library ledger`. |
| `/abcd:ingest` | `/abcd:library`, where `abcd library add` registers a document in the sources corpus. |

Search your scripts, hooks and CI workflows for each spelling in the left-hand
column, change it to the right-hand one, and run the changed command once to
see it answer.

## Move memory and source commands under library

`abcd library` holds two stores under one command: your sources corpus and the
project's memory store. They stay separate, and each keeps its own sub-verbs
under their old names:

| Spelling in v0.13 | Spelling in v0.14.0 |
|-------------------|---------------------|
| `abcd memory ingest`, `ask`, `lint` | `abcd library ingest`, `ask`, `lint` |
| `abcd source init`, `add`, `declassify`, `ledger`, `sync-banlist`, `cite-check` | `abcd library init`, `add`, `declassify`, `ledger`, `sync-banlist`, `cite-check` |
| `abcd source add --confidential` | `abcd library add --private` |
| `abcd memory`, `abcd source` (bare) | `abcd library` (bare) |

Two of these sub-verbs do different things, so keep them apart when you change
a script:

- `abcd library add`: Registers a document in the sources corpus, filed under
  its class.
- `abcd library ingest`: Distils a local file or an https source into cited
  pages in the memory store, and registers nothing in the corpus.

`abcd library add` takes `--private` or `--public` for the class. Given
neither, it asks at a terminal and refuses anywhere else, writing nothing, so
a script passes one of the two. `--confidential` is an unknown flag; the
folder a private source sits in keeps its name, `confidential/<key>/`.

Bare `abcd library` renders both stores. A store that cannot be read is one
line naming what would create it, and the command refuses only when neither
store can be read. Every corpus sub-verb but `init` still exits 3 when there is
no corpus.

## Read the release cut from the launch preview

`abcd launch --dry-run` renders the cut the next `abcd launch ship` would make:
the derived version, the deciding impact, the records, and the guard verdict.
Under `--json` the cut is the `cut` field of the preview's report, with
`cut_error` in its place when the cut cannot be read.

The preview's exit code is the bundle's. A refused cut is reported and changes
nothing about that exit, but a failing bundle gate does, and the preview
refuses a repository with no `.abcd/config/artefact.json`. A script that read
the cut from a command that always exited 0 checks the `cut` field rather than
the exit code.

## Refresh each managed project's pre-commit hook

The pre-commit hook abcd commits in a project, `.githooks/pre-commit`, runs
the sources refresh. A hook written by an earlier release runs it as `source
sync-banlist`, which v0.14.0 does not know, so the hook skips the refresh with
one line on stderr and the block of confidential names stops being kept up to
date. In each project abcd manages:

1. Run `abcd ahoy`. It reports the gap as `the committed guard runs the
   sources refresh under its retired name`.
2. Run `abcd ahoy install`. It rewrites `source sync-banlist` to `library
   sync-banlist` in the hook and leaves every other line as it is.
3. Commit the changed hook.

A hook abcd did not write is yours: `abcd ahoy` does not report it and `abcd
ahoy install` does not change it, so change any `source sync-banlist` in it by
hand. The git setting that opts a clone into the refresh keeps its name,
`abcd.sourcesBinary`.

## Find the agents' commands

The `/abcd:` menu lists the commands people run: `abcd`, `ahoy`, `build`,
`capture`, `dashboard`, `decide`, `disembark`, `embark`, `ideate`, `intent`,
`lab`, `launch`, `library`, `reading`, and `update`. The commands only agents
run keep their names and their pages, and the host leaves them off the menu:
`banlist`, `docs`, `drain`, `guard`, `history`, `identity`, `implement`,
`inbox`, `lint`, `mode`, `peers`, `reflect`, `report`, `scribe`, and `site`.

- In the agent session: Typing one of the agents' commands in full gets the
  host's refusal. Ask your agent in plain words instead, and it runs the
  command as before.
- In a terminal: Every command runs as before. `abcd --help` lists the
  people's commands; `abcd --help --agent` lists the agents' too, each naming
  the page an agent reads next.
