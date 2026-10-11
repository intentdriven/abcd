---
id: spc-2610100613109045
slug: a-person-who-types-abcd-sees-only-the
intent: itd-2610090831227812
origin: researcher-authored
production_mode: hand-written
---
# a-person-who-types-abcd-sees-only-the-commands-people-use

## The verb audit

This table is the verb audit iss-2610090831317531 asks for, and the intent
requires it as the spec's first section. It has one row per page in
`commands/` at 3b8700ef7 (thirty-five), then one row per verb the binary
registers with no page of its own name. "Page block" is the page's `block:`
frontmatter today. "CLI block" is where `abcd --help --agent` lists the verb
today, read from `helpPlacements` in `internal/surface/cli/helpgroups.go` and
the committed snapshot `.abcd/development/release/surface.json`. "Ruled" is the
product thinker's placement and the dated entry under the intent's Decisions it
comes from. Nothing in this column is re-decided here.

The dated entries the column cites, all in the intent's Decisions:

- **P1**, 2026-10-09, "placed commands one by one": people get abcd (the
  board), ahoy, build, capture, decide, disembark, embark, intent, launch,
  update, ideate, dashboard, lab and reading. Agents only: lint, reflect, inbox,
  implement, history, site, docs, scribe, identity, guard, hook and statusline.
  source deferred ("Decide later").
- **P2**, 2026-10-09: drain stays with the agents (ruling BX1) until itd-82
  ships.
- **P3**, 2026-10-09: peers and mode are agents only.
- **P4**, 2026-10-09, "placed the remaining commands": report, spec and rules
  are agents only. The version page retires and the version shows on the
  board. prepare-this-repo folds into `ahoy install`. changelog merges into
  launch as a preview. consult, ingest, memory and source merge into a person's
  command named `library`. `/abcd:memory` later becomes an agent's command for
  memory notes (itd-2610091918433290).
- **P5**, 2026-10-09, settled without a question: ideate and dashboard are a
  person's, since the product thinker types both.
- **P6**, 2026-10-10: banlist, typed by no person, stays agents only.

Action words: **keep** (stays on the person's menu as it is); **hide** (gains
`user-invocable: false` and stays a command with its name); **move** (changes
block on the CLI and the page, in step 1); **merge** or **fold** (its job moves
into the named command and its own name goes, with no alias, under adr-40);
**retire** (the page goes and nothing replaces the name).

| # | Page or verb | Page block | CLI block | Ruled | Action | Notes |
|---|---|---|---|---|---|---|
| 1 | `abcd` (the board) | none | the bare root | people (P1) | keep; add `block: people`; show the version | Step 2 adds the version. No verb, so `TestCommandPagesDeclareTheirBlock` skips it today; the menu test in step 1 reads it. |
| 2 | `ahoy` | people | people, set-up | people (P1) | keep; prepare-this-repo folds in | Step 4. |
| 3 | `banlist` | agents | agents | agents (P6) | hide | |
| 4 | `build` | people | people, records | people (P1) | keep | |
| 5 | `capture` | people | people, records | people (P1) | keep | |
| 6 | `consult` | none | no verb (host-delegated) | merge into library (P4); people until then (decision 14) | step 1 writes `block: people`; merge; page goes | Step 1 writes the interim block, step 6 retires the page. Runs `abcd source` today. |
| 7 | `dashboard` | agents | agents | people (P1, P5) | move to people | Step 1. |
| 8 | `decide` | people | people, records | people (P1) | keep | |
| 9 | `disembark` | people | people, portability | people (P1) | keep | |
| 10 | `docs` | agents | agents | agents (P1) | hide | |
| 11 | `drain` | agents | agents | agents until itd-82 ships (P2) | hide | Frontmatter only. The body belongs to the itd-82 lane in flight. When itd-82 ships, drain moves to people and drops the key; the menu test makes that one edit. |
| 12 | `embark` | people | people, portability | people (P1) | keep | |
| 13 | `guard` | agents | agents | agents (P1) | hide | `guard hook` keeps its agents-block line. |
| 14 | `history` | agents | agents | agents (P1) | hide | |
| 15 | `ideate` | agents | agents | people (P1, P5) | move to people | Step 1. `ideate record` keeps its agents-block line. |
| 16 | `identity` | agents | agents | agents (P1) | hide | |
| 17 | `implement` | agents | agents | agents (P1) | hide | |
| 18 | `inbox` | agents | agents | agents (P1) | hide | The first 2026-10-09 question named the inbox as typed by people; the placement after the inventory put it with the agents. |
| 19 | `ingest` | none | no verb (host-delegated) | merge into library (P4); people until then (decision 14) | step 1 writes `block: people`; merge; page goes | Step 1 writes the interim block, step 6 retires the page. Runs `abcd source add` today. |
| 20 | `intent` | people | people, records | people (P1) | keep | `intent prepass`, `intent audit ingest` and `intent consistency ingest` keep their agents-block lines (the 2026-10-09 ruling on agent-only steps). |
| 21 | `lab` | agents | agents | people (P1) | move to people | Step 1. |
| 22 | `launch` | people | people, release | people (P1) | keep; changelog merges in as a preview | Step 3. |
| 23 | `lint` | people | people, checks | agents (P1) | move to agents; hide | Step 1. reading and lab join Checks, so the group keeps a purpose (decision 1). |
| 24 | `memory` | people | people, records | merge into library (P4) | merge; page and verb go | Step 5. The name is free afterwards; itd-2610091918433290, a draft, gives it to memory notes. |
| 25 | `mode` | agents | agents | agents (P3) | hide | |
| 26 | `peers` | agents | agents | agents (P3) | hide | |
| 27 | `prepare-this-repo` | none | no verb (host-delegated) | fold into `ahoy install` (P4); people until then (decision 14) | step 1 writes `block: people`; fold; page goes | Step 1 writes the interim block, step 4 retires the page. |
| 28 | `reading` | agents | agents | people (P1) | move to people | Step 1. |
| 29 | `reflect` | people | people, release | agents (P1) | move to agents; hide | Step 1. |
| 30 | `report` | agents | agents | agents (P4) | hide | Asking permission before it files (iss-2610091903077581) is separate work. |
| 31 | `scribe` | agents | agents | agents (P1) | hide | The trial page of 2026-10-10. |
| 32 | `site` | agents | agents | agents (P1) | hide | |
| 33 | `source` | people | people, records | merge into library (P4, after P1 deferred it) | merge; page and verb go | Step 5. |
| 34 | `update` | people | people, set-up | people (P1) | keep | |
| 35 | `version` | none | no verb (the root's `--version`) | retire; the version shows on the board (P4); people until then (decision 14) | step 1 writes `block: people`; retire; page goes | Step 1 writes the interim block, which keeps the page visible as it is today; step 2 retires the page. `abcd --version` still answers. |
| — | `changelog` | no page | agents (names `commands/launch.md`) | merge into launch as a preview (P4) | merge; verb goes | Step 3. |
| — | `rules` | no page | people, set-up | agents (P4) | move to agents | Step 1. Its agents-block line names `commands/ahoy.md`, which installs the rule loader (decision 2). P4 supersedes itd-146's criterion that kept it on the person's list (decision 3). |
| — | `spec` | no page | people, records | agents (P4) | move to agents | Step 1. Its agents-block line names `commands/intent.md` (decision 2). P4 supersedes itd-146 here too (decision 3). |
| — | `statusline` | no page | agents (names `commands/ahoy.md`) | agents (P1) | keep | |
| — | `hook` | no page | hidden | agents (P1) | keep | Already hidden from every help. |
| — | `help`, `completion` | no page | people, set-up | not ruled | keep | The CLI framework's own; filed under set-up by `applyHelpPlacement`. Counted apart from the verbs, as today. |
| — | `library` (new) | people (new page) | people | people (P4) | new | Step 5 builds the verb and page; step 6 folds consult and ingest in. |

At the end, `commands/` holds thirty pages: the thirty-five, less version,
prepare-this-repo, consult, ingest, memory and source, plus library. Fifteen
are a person's: abcd, ahoy, build, capture, dashboard, decide, disembark,
embark, ideate, intent, lab, launch, library, reading and update. Fifteen are
hidden: banlist, docs, drain, guard, history, identity, implement, inbox, lint,
mode, peers, reflect, report, scribe and site. `abcd --help` lists the same
fourteen verbs a person runs (every person's page but the board, which is bare
`abcd`), plus the framework's `help` and `completion`. Fourteen is the
aspiration of the 2026-10-09 menu ruling, met but not gated.

## Summary

This spec delivers
[itd-2610090831227812](../../intents/planned/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md):
a person who types `/abcd:` sees the fifteen commands people use, and the
fifteen only agents run are hidden but keep their names. `abcd --help` lists
the same commands. Four commands about sources become `library`, and three
fold into the commands that already do their job.

The approach, in plain words:

- **Hide with the host's own switch.** Every agent page gets
  `user-invocable: false` in its frontmatter. The 2026-10-10 trial showed the
  host then leaves it off the menu, refuses it when a person types it, and
  still runs it when an agent is asked. No page moves, no name changes, and the
  flat `commands/` directory stays as it is.
- **One source, two surfaces, one test.** A page's `block:` stays the one
  statement of whose command it is. A new test holds `user-invocable: false` to
  `block: agents` page by page, and a second holds the plugin's person list
  equal to the command line's. The fifteen-verb cap test retires, as the
  2026-10-09 menu ruling says.
- **The merges each land as their own step.** Version onto the board,
  changelog into launch, prepare-this-repo into `ahoy install`, then memory and
  source into `library`, then consult and ingest into the library page. Under
  adr-40 each old name goes with no alias, and each step changes the brief,
  the docs and the generated pages that name it in the same change
  (spec-moves-with-the-surface).
- **The record that cannot be tested is captured live.** The host's menu
  rendering is not visible to a test, so a live session captures the menu
  as it stood before step 1 and after the last step, and the captures are
  attached here. Both are taken in step 8's session with the person: the
  before capture from a checkout of step 1's base commit, 314e9c3ce, since
  step 1 was built by an agent with no live host session to capture in.

How each acceptance criterion is met, numbered A1 to A13 in the order the
intent lists them:

| Criterion | Met by |
|---|---|
| A1, the menu lists only people's pages | Step 1 hides every agent page; the after capture in step 8 shows it. |
| A2, an agent still runs `/abcd:guard` as before | Step 1 changes no invocation; `TestPlacementChangesNoInvocation` keeps running `guard check`; step 8 captures an agent running it. |
| A3, `--help --agent` is the one place both blocks render, each agent line naming a page | Unchanged renderer; step 1 has rules name `commands/ahoy.md` and spec name `commands/intent.md` (decision 2); a test holds every agents-block page to an existing file. |
| A4, a typed hidden name gets the host's refusal | Step 8's live capture. |
| A5, a test fails on a page whose block and key disagree | Step 1, `TestCommandPagesMatchTheMenu`. |
| A6, the table names every page, and only merged pages leave | This section; the steps delete only the six pages above. |
| A7, before and after captures attached | Step 8's live session takes both: the before capture from step 1's base commit, 314e9c3ce, and the after capture at the merged tip. |
| A8, the docs set the library beside memory notes | Waits on open question 12, for the product thinker. Step 7 builds everything else, and the close waits on the answer. |
| A9, the board shows the version and the retired `/abcd:version` is gone | Step 2. |
| A10, launch previews the changelog and `changelog` is gone | Step 3. |
| A11, `ahoy install` does what prepare-this-repo did | Step 4. |
| A12, `library` does what the four did, and the four names are gone | Steps 5 and 6. |
| A13, the library asks about confidentiality unless told | Step 6. |

## Design

### Hiding: the key, the menu test and the list parity test

Every page whose `block:` is `agents` gains one frontmatter line,
`user-invocable: false`, placed after `block:`. Nothing else in the page
changes. `go generate ./internal/surface/cli` rewrites only the
`description:` line (`withDescription` in `internal/surface/cli/sentences.go`),
so the new line survives regeneration.

The key cannot be read with `frontmatter.Fields`. Its key pattern is
`^([A-Za-z0-9_]+)` (`internal/core/frontmatter/frontmatter.go`), so a
hyphenated key is skipped, the way `argument-hint` already is. Widening that
pattern would change what every record reader accepts, capture's strict
parser included, for one page key. The command pages already have a reader
that keeps hyphens: the deep smoke's `pageFields` in
`internal/core/launch/deepsmoke.go`, which reads `argument-hint`. Step 1
exports one small page-frontmatter reader there (or beside it, in a leaf the
CLI tests can import) and reads `block` and `user-invocable` through it, so
the smoke and the gate read a page's keys the same way.

Two new tests in `internal/surface/cli`:

- **`TestCommandPagesMatchTheMenu`** walks `commandFileBodies`. Every page
  declares `block:`, either `people` or `agents`. The four pages with no
  `block:` today (consult, ingest, prepare-this-repo and version) get
  `block: people` in step 1 and keep it until their own step deletes them
  (decision 14). A `people` page carries no
  `user-invocable` key. An `agents` page carries `user-invocable: false` and
  nothing else in that key. Each failure names the page and the defect. The
  board page `abcd.md` is in the walk with `block: people`. A synthetic page
  set proves each defect is named (a negative control in the style of
  `TestPersonVerbsCountsEveryListedVerb`).
- **`TestPluginPersonListEqualsTheCLIs`** compares two sets. One is the pages
  whose `block:` is `people` and that back a CLI verb. The other is the
  verbs `abcd --help` lists under the person's groups, less `help` and
  `completion`. The board page is the bare root on one side and nothing on the
  other, so it is left out of both. Pages in `pagesWithNoVerb`
  (`internal/surface/cli/staleusage.go`) back no verb, so they are left out
  while they exist. Steps 2, 4 and 6 empty that set down to the board, and
  step 6 adds an assertion that the board is all it holds. Each failure names
  the verb that is on one list and not the other.

`TestCommandPagesDeclareTheirBlock` stays as it is: it holds each page to the
CLI tree, and the new pair holds the menu to the page and the plugin's list to
the CLI's.

Two existing tests retire in step 1, each with its reason in the commit:

- `TestPersonsListHoldsAtMostFifteenVerbs` (`consolidate_test.go`) and its
  `maxPersonVerbs`. The 2026-10-09 menu ruling removes the cap for both
  surfaces. `TestPersonVerbsCountsEveryListedVerb` goes with it, since its only
  job is to make the cap able to fail. `personVerbs` stays if the parity test
  reuses it.
- `TestRulesAndSpecKeepTheirPlaces` (`helpgroups_test.go`), which holds
  itd-146's criterion that rules and spec stay on the person's help. P4 puts
  both with the agents. The intent flags the cap reversal but not this one;
  decision 3 records that P4 supersedes the criterion.

### The command line follows the same ruling

The CLI half of step 1 edits `helpPlacements`. lint and reflect move to
`groupAgents`, naming `commands/lint.md` and `commands/reflect.md`. rules and
spec move to `groupAgents`. rules names `commands/ahoy.md`, which installs
the rule loader, and spec names `commands/intent.md`, the user-facing half of
the spec lifecycle (decision 2). ideate joins Records, reading and lab join
Checks, and dashboard joins Set-up (decision 1); library joins Records when
step 5 creates it. The sub-verb entries `ideate record`, `guard hook` and the
three `intent` ones stay agents-block lines. With lint gone, Checks holds
reading and lab, so the group keeps a purpose.

The marker block `ahoy` installs in every managed repository tells agents to
run `abcd rules`. That still works: the verb is listed in the agents' block,
not hidden or removed (decision 3).

The comment above `helpPlacements`, which still describes the fifteen-verb
ceiling and the 2026-09-25 technical ruling, is rewritten to cite P1 to P6.
The snapshot records each verb's group and block, so the moves show in the
regenerated `surface.json` diff and in `docs/reference/cli/commands.md`, and
the surface diff never reads a placement as a break (04-surfaces/README.md,
"How the help lists the verbs").

### The version on the board; `/abcd:version` retires

`/abcd:version`, the page step 2 retires, runs `abcd --version --json` and
relays it. Step 2 deletes the page and puts the version on the board, in every
place a person reads the board: bare `abcd` in a terminal, the markdown form
`commands/abcd.md` pastes, and `--json`. `abcd --version` is untouched and
still answers alone (`TestRootVersionFlagPrintsWhatVersionPrinted` passes
unchanged). The board
reads the version from the same `core.VersionInfo` the flag reports. It
reads no network and adds no subprocess, under adr-38. The version is the
board's last line, in both views and both forms (for example `abcd v0.13.4`),
and `--json` gains a `version` field (decision 4). The first line stays the
view label alone, as the board's spec (spc-2610031844142274, decision 2)
requires.

`commands/abcd.md` gains the version line and the parts of the version page an
agent still needs: the superseded-root note (say it first), binary
resolution, and what to do when no binary resolves. The update check stays on
`commands/update.md`. `version` leaves `pagesWithNoVerb`. Its index entry,
registry row 12 and chapter `12-version.md` leave the brief, and the root's
generated appendix in `08-abcd.md` keeps listing `--version`.
`help_truth_test.go` does not read `commands/version.md`, but its failure
message names version.md; repoint the message at the page that now carries
the network sentence. `README.md` stops naming the retired `/abcd:version`.

### `changelog` merges into launch as a preview

`abcd changelog` (`newChangelogCommand`, `internal/surface/cli/ship.go`) is
the read-only, deterministic emit of the next cut: derived version, deciding
impact, records, guard verdict and findings. It runs the same `emitCut` and
`renderCut` that `launch ship` runs, and it exits 0 on a refused cut. Step 3
makes `launch --dry-run` render that cut beside its bundle report, and removes
the `changelog` verb (decision 5). The preview is the deterministic cut,
exactly what `abcd changelog` shows today: no composed prose, so no agent runs
in a preview. A refused cut is information in the dry run, as it was in the
changelog verb, and changes no exit code the dry run gives today. `emitCut`,
`renderCut` and `internal/core/release` do not change. Only the front door
moves, and the render's verb label becomes `abcd launch --dry-run`.

What moves with the verb: its placement and its `cliOnlyVerbs` note in
`surfaceparity_test.go`; its sentence in `internal/core/surface/sentences.go`;
`changelog` in the `.abcd/rules.json` DOGFOODING recall list (held by
`rules_dogfooding_recall_test.go`); the preview lines in `commands/launch.md`;
`commands/intent.md`'s mention; `agents/release-changelog-composer.md`, which
names `abcd changelog --json` as its read-only preview; and the brief:
`04-launch.md` and the operator-internal row in `04-surfaces/README.md`. The
existing changelog tests move onto the preview rather than being deleted, so
what they pin (the root resolution, the uncommitted-records refusal, exit 0 on
a refused cut) still holds.

### prepare-this-repo folds into `ahoy install`

`commands/prepare-this-repo.md` is a host-run workflow: refuse a repository
the person does not own, orient, audit with `abcd lint`, then adopt (the three
tiers, the migration of a historical root-level work folder, the AGENTS.md merge with verified repo
facts, the identity block, `ahoy install` and its attribution opt-in), and a
definition of done. `abcd ahoy install` is a binary sub-verb that already
writes `.abcd/`, the name-guard hooks and the PATH entry. Step 4 moves the
workflow into the `install` section of `commands/ahoy.md`, around the binary
call it already makes, and deletes the page. The fold is page-only: the
binary does not change, and `ahoy install` gains no new refusal (decision 6).
The ownership check stays a step the page tells the host to take before it
runs install, as prepare-this-repo's first phase is today; a refusal in the
binary would break installs that work now.

Code that names the page or its verb changes in the same step. The loop's
conventions-file hint in `internal/core/implement/loop/brief.go` and
`issuebrief.go` tells the reader to run `abcd prepare-this-repo`, which was
never a verb, so it becomes `abcd ahoy install`. Also:
`internal/core/repolint/rule_docs.go`'s message; `commands/lint.md`'s pointer;
`pagesWithNoVerb`; `host_delegated` in `.abcd/record-lint.json` and the comment
in `internal/core/lint/config.go`. `onboarding_test.go` and
`persona_conventions_test.go` (in `internal/core/ahoy`) read the
prepare-this-repo page, so they now read `commands/ahoy.md`. The brief changes
too: `15-prepare-this-repo.md` folds into `01-ahoy.md`, and the registry row,
the index and the host-delegated paragraphs of `04-surfaces/README.md`
change, as does `README.md`'s first-run line.

### `library`: memory and source become one command

Today the two are separate stores with separate verbs.

- `abcd memory` (`newMemoryCommand`, `internal/surface/cli/cli.go`) addresses
  the checkout's `.abcd/memory/` and refuses outside a checkout. Its
  sub-verbs are `ingest`, `ask` and `lint`.
- `abcd source` (`internal/surface/cli/source.go`) addresses the user-level
  corpus under `~/.abcd.noindex/sources/` and exits 3 when there is none. Its
  sub-verbs are `init`, `add`, `declassify`, `ledger`, `sync-banlist` and
  `cite-check`, and it has a `--corpus` flag.

Step 5 builds one top-level `library` verb carrying both sub-trees and removes
`memory` and `source`. `internal/core/memory` and `internal/core/source` stay
two packages: the store, the redactor and the guard checks are not touched,
only the front door and the user-facing strings that name a verb. Every
sub-verb keeps its name (decision 7): any rename beyond the merges is out of
the intent's scope. The memory store's `ingest` (distil into pages) then sits
beside the corpus's `add` (register a document), while
the retired `/abcd:ingest` meant the second. Step 5's agent files a capture
recording that double meaning, and renames nothing.

Bare `library` renders both halves: the corpus and the checkout's memory
store. A half that cannot be read is one line saying so and naming what would
create it (`library init` for the corpus, a checkout for the memory store). The
verb refuses only when neither half can be read (decision 8). So bare
`library` with no corpus, inside a checkout, renders the memory half and a
line about the corpus, where bare `source` exited 3.

Places where the old name is load-bearing, and so change in step 5:

- **The guard.** The registry entry `abcd-source-ledger-flip`
  (`internal/core/guard/defaults/guard.json`) blocks an agent from running
  `abcd source ledger --flip`, the person's own act under adr-41. Its
  `subcommand` becomes `library`, and its fixtures and `sourceflip_test.go`
  move with it. Left alone, the guard would stop matching the moment the verb
  is renamed, and an agent could flip a citation line. The SHELL rule domain
  is generated from the same registry, so it follows.
- **The pre-commit hooks.** Both `.githooks/pre-commit` and the template
  `ahoy install` seeds (`internal/core/ahoy/defaults/pre-commit`) run
  `"$bin" source sync-banlist --refresh`. When the binary lacks that verb,
  they say so on stderr and skip. A managed repository whose committed hook
  predates the rename would then stop refreshing its confidential-name block,
  with one stderr line as the only sign. The templates and
  `internal/core/banlist/hook_sources_test.go` move to `library sync-banlist`
  in the same step. A hook already committed in a managed repository still
  names `source`. `ahoy` detects a committed hook that names the old verb and
  reports it as a gap, and `ahoy install` rewrites it; the upgrade guide says
  so too (decision 13). The `abcd.sourcesBinary` git setting keeps its name.
- **Strings a reader acts on.** The memory status drift lines (`run abcd
  memory ingest`, `run abcd memory lint` in `internal/core/memory/bare.go`),
  the report headings and the corpus refusal naming `abcd source init`
  (`internal/core/source/source.go`) all name the new verb. The provenance
  strings written into memory pages (`ingestedBy`, `filedBy`) also name the
  new verb for new pages. Pages already written keep the verb that wrote
  them, since provenance records a historical fact, and nothing reads the
  value back.
- **The generated and gated surfaces.** These are the sentences and examples
  manifests (`internal/core/surface`), the placement, the DOGFOODING recall
  list (drop `memory` and `source`, add `library`), the snapshot, and the CLI
  reference. The `## Sub-verbs` tables come from a new brief chapter for the
  library, folded from `07-memory.md` and `33-source.md`, and registry rows 7
  and 33 become one library row. `commands/guard.md` documents the flip
  entry.

The page side of step 5 writes `commands/library.md` from the binary-backed
halves of `memory.md` and `source.md` (bare status, ingest, ask, lint, init,
add, declassify, ledger, sync-banlist, cite-check, and the hard rule), and
deletes those two pages. consult and ingest still exist after step 5, as
host-delegated pages repointed at `abcd library`. Step 6 folds them in. The
split keeps each step inside one agent's reach while main stays green. The
verb rename forces its pages, registry rows and chapter into the same change,
because `surface_coverage` and the parity test refuse a page with no verb and a
verb with no page.

### The library asks whether an item is confidential

`abcd source add` refuses today without `--confidential` or `--public`, and
says the class is never guessed. The ingest page asks the person when in
doubt. Step 6 makes the ask the library's own behaviour, A13. The design
follows the pattern `ahoy install` uses for a tool install: the binary asks
at a terminal and refuses off one, and the page asks through the host's
question tool.

- `abcd library add` given no class, at a terminal, asks one question naming
  the two answers and what each means, then proceeds with the answer. Off a
  terminal (a pipe, CI, a host's shell), it refuses with exit 2, naming the
  flags. Nothing is written in either refusal.
- `commands/library.md` tells the agent to ask the person through the host's
  question tool before it runs `library add` without a class, unless the
  person already said, and to pass the answer as the flag. The question
  follows the GRILL asking rules, and `abcd guard hook`'s question check
  applies to it as to any question. Nothing in `cmd/asking-sync`'s generated
  blocks changes, so `make asking-sync` has nothing to write.
- The class flags are `--private` and `--public` (decision 9). `--private`
  replaces `--confidential`, which goes with no alias under adr-40, and the
  stored folder `confidential/` and the field `custom.confidential` keep their
  names. Either flag means the person has said, so neither is asked about.
- The two stores stay unconnected under the one verb (decision 10): nothing
  distils a corpus item into `.abcd/memory/`.

Step 6 also folds the judgement halves of consult and ingest into
`commands/library.md`: searching the corpus, recording an influence, reading a
document first, choosing class and key. It deletes both pages, folds chapters
`13-consult.md` and `14-ingest.md` into the library chapter, removes the last
two entries from `pagesWithNoVerb` and `host_delegated`, and rewrites the
host-delegated and "No skills" paragraphs of `04-surfaces/README.md` and the
page-to-verb paragraph of `05-internals/08-skills.md`. After it, every page but
the board backs a verb.

### Old names on the command line and the plugin

Under adr-40 there is no alias. `abcd changelog`, `abcd memory` and `abcd
source` become unknown commands. Each step adds a test that runs the old
spelling and expects cobra's unknown-command refusal, in the style of
`TestRemovedSpellingsAreUnknownCommands`. The stale-usage note in
`staleusage.go` names what replaced each old verb, rather than calling the
binary stale for not knowing it. That is the existing mechanism for a token
the binary does not register, and it is not an alias: the command still
refuses. Five plugin pages retire with their files: version,
prepare-this-repo, consult, ingest and source. The memory page goes too, and
itd-2610091918433290 later gives its name to memory notes.

Step 7 adds one entry to record-lint's `banned_tokens` (decision 11). It bans
only the `/abcd:` spellings of the five retired page names, which nothing
reuses. Written in a fence, which the ban does not scan:

```json
{
  "id": "retired-plugin-pages",
  "pattern": "/abcd:(consult|ingest|prepare-this-repo|version|source)\\b",
  "message": "a retired plugin page (itd-2610090831227812): the version shows on the board, prepare-this-repo is ahoy install, and consult, ingest and source are library",
  "severity": "blocker",
  "successor": "/abcd:abcd (version), /abcd:ahoy install (prepare-this-repo), /abcd:library (consult, ingest, source)",
  "allow_context": ["historical", "retire", "^ +evidence: [^ ]+:[0-9]+ — \""]
}
```

There is no per-record escape. `allow_context` is a list of regular
expressions (`BannedToken.AllowContext` in `internal/core/lint/config.go`),
and `checkBannedTokens` in `internal/core/lint/lint.go` tries them against the
one line that holds the banned spelling: a match suppresses that line's finding
and no other. So a record that keeps a retired spelling as history says so on
the same line, with `historical` or a word on the stem `retire` (retired,
retires, retirement). That is the inline-word escape the existing bans use:
`historical` on most entries, and `retired` beside it on
`retired-promote-back-links`. A quoted audit-evidence line cannot take a word
without misquoting its source, so the entry carries the evidence-line pattern
`retired-promote-back-links` already carries. The research tree and superseded
intents are exempt paths and need nothing. This spec's own lines that name a
retired slash spelling each carry one of the words, so it passes the ban it
orders.

The bare command-line spellings `abcd source` and `abcd changelog` are not
banned. They run through the brief and the records as prose about the command
line, and 69 lines under `.abcd/development` carry one today with neither
word on the line; banning them would spend step 7 on records no reader is
misled by. The unknown-command tests and the stale-usage notes guard those
spellings instead. `/abcd:memory` and `abcd memory` are left out, because
itd-2610091918433290 reuses them (decision 11).

### Pages, docs and the brief that change

| Where | Step |
|---|---|
| Frontmatter of all thirty-five pages (block and key) | 1 |
| `commands/abcd.md`, `commands/version.md` (deleted) | 2 |
| `commands/launch.md`, `commands/intent.md`, `agents/release-changelog-composer.md` | 3 |
| `commands/ahoy.md`, `commands/lint.md`, `commands/prepare-this-repo.md` (deleted) | 4 |
| `commands/library.md` (new), `commands/memory.md` and `commands/source.md` (deleted), `commands/consult.md` and `commands/ingest.md` (repointed), `commands/guard.md` | 5 |
| `commands/library.md`, `commands/consult.md` and `commands/ingest.md` (deleted) | 6 |
| `README.md` | 2, 4, 7 |
| `docs/reference/cli/commands.md` (generated) | 1, 3, 5, 6 |
| `docs/reference/terminology.md`, `docs/explanation/`, `docs/how-to/` | 7 |
| `04-surfaces/README.md`: the register, the help table, the index, the operator-internal and host-delegated paragraphs | each step, for its own rows |
| `04-surfaces/` chapters: 12 (deleted), 08, 04, 15 (folded into 01), 01, 07 and 33 (folded into a library chapter), 13 and 14 (folded in) | 2 to 6 |
| `05-internals/08-skills.md`: the page-to-verb enumeration | each of steps 2 to 6 keeps it true; step 6 rewrites it |
| `02-constraints/04-naming.md`: the exemption list | each step keeps it true; step 7 rewrites it |
| `01-product/04-scope.md`: the command list and the operator-internal verbs | steps 3 and 5 keep it true; step 7 checks it again |
| `05-internals/07-memory.md`: the verb it names | 5 |

### The generated blocks

- `go generate ./internal/surface/cli` writes the snapshot
  (`.abcd/development/release/surface.json`), the CLI reference, each surface
  chapter's appendix and every page's `description:`. It runs in each step
  that changes a verb, a placement or a sentence (1, 3, 5, 6). The drift
  tests (`TestSurfaceAppendicesMatchCommandTree`, the reference drift test,
  the sentence test) fail on a step that forgets it.
- `make asking-sync` renders the asking rules into `commands/intent.md` and
  `agents/question-drafter.md`. No step changes an asking rule, so it has
  nothing to write. The library's question in step 6 is asked under those
  rules, not added to them.
- The SHELL rule domain is generated from the guard registry at load time, so
  step 5's registry edit reaches it with no regeneration step.

### Gates that find a page by its verb name

Hiding renames nothing, so every gate that maps a verb to `commands/<verb>.md`
keeps working. The steps that remove a name each update every gate that lists
it:

- **`surface_coverage`** (`.abcd/record-lint.json`): a `shipped` row in the
  register needs `commands/<name>.md`, and every page needs a row. Its
  `host_delegated` list (consult, ingest, prepare-this-repo) empties in steps
  4 and 6.
- **`index_drift`** (id `commands`): the marked list in
  `04-surfaces/README.md` must equal `commands/`.
- **The surface-parity test** (`surfaceparity_test.go`): `cliOnlyVerbs` loses
  `changelog` in step 3 and gains a note for nothing. `pagesWithNoVerb` loses
  version, prepare-this-repo, consult and ingest.
- **`TestCommandPagesDeclareTheirBlock`** and the new menu and parity tests.
- **The sentence pages** (`SentencePages`), which find `commands/<verb>.md`
  for each visible top-level verb.
- **The appendix generator**, which maps a chapter to its commands through the
  register's Command and File columns. A chapter folded away must also leave
  the register in the same step.
- **The DOGFOODING recall test**, which holds the recall list to the verb
  list.
- **The deep smoke** (`launch --deep-smoke`), which renders every page's help
  from the payload. A deleted page simply stops being rendered.

## Steps

1. The menu follows the ruling: agents' pages are hidden, and the eight verbs that change sides move
   - packages: commands (the frontmatter of all thirty-five pages), internal/core/launch (the exported page-frontmatter reader), internal/surface/cli (helpgroups.go, helpgroups_test.go, consolidate_test.go, a new menu test file), .abcd/development/release/surface.json, docs/reference/cli/commands.md, .abcd/development/brief/04-surfaces/README.md (the help table and its paragraph), .abcd/development/brief/02-constraints/04-naming.md where it states the cap
   - tests: TestCommandPagesMatchTheMenu with its synthetic negative control, and TestPluginPersonListEqualsTheCLIs, each watched fail before the frontmatter and placement edits; TestPersonsListHoldsAtMostFifteenVerbs, TestPersonVerbsCountsEveryListedVerb and TestRulesAndSpecKeepTheirPlaces retired, each with its ruling in the commit; TestRootHelpListsThePersonsGroups (internal/surface/cli/helpgroups_test.go) rewritten to the new table, since it hardcodes the person's groups (Records with spec, Checks with lint, Release with reflect) and names reading among the agent verbs, so this step breaks it; TestCommandPagesDeclareTheirBlock, TestPlacementChangesNoInvocation and TestPeopleBlockIsTheSameUnderBothHelps pass; the snapshot and reference regenerated with go generate ./internal/surface/cli
   - the BEFORE capture (what `/abcd:` lists, and what typing `/abcd:scribe` answers) is deferred to step 8's live session, taken from a checkout of this step's base commit, 314e9c3ce, and added to this spec under a `## Live captures` heading after the Steps section in step 8's pull request (A7): this step was built by an agent with no live host session
   - placements per decisions 1 to 3: ideate under Records, reading and lab under Checks, dashboard under Set-up; rules names commands/ahoy.md and spec names commands/intent.md; TestRulesAndSpecKeepTheirPlaces retires because P4 supersedes itd-146's criterion
   - drain.md's frontmatter gains the key only; its body belongs to the itd-82 lane, so coordinate the order of the two edits
   - per decision 14: consult.md, ingest.md, prepare-this-repo.md and version.md, which carry no `block:` today, each gain `block: people` and no `user-invocable` key; steps 2, 4 and 6 delete them
2. The board shows the installed version, and the version page retires
   - packages: internal/core/board, internal/surface/cli (cli.go board output, staleusage.go, surfaceparity_test.go, help_truth_test.go), commands/abcd.md, commands/version.md (deleted), README.md, .abcd/development/brief/04-surfaces (12-version.md deleted, README.md row 12 and index, 08-abcd.md), .abcd/development/brief/05-internals/08-skills.md (the enumeration), .abcd/development/brief/02-constraints/04-naming.md (the exemption list)
   - tests: TestBoardShowsTheVersion over the text form, the markdown form and --json, watched fail first; TestRootVersionFlagPrintsWhatVersionPrinted passes unchanged; the menu and parity tests pass with version gone from pagesWithNoVerb; the board's golden tests updated, with the first line still the label alone
   - peer finding dissolved: 12-version.md's claim that the retired verb answers for one release goes with the chapter
   - per decision 4: the version is the last line of both views and both forms, and `--json` gains `version`
3. Launch previews the cut, and the changelog verb goes
   - packages: internal/surface/cli (ship.go, the launch command in cli.go, launch_preflight.go, helpgroups.go, surfaceparity_test.go, the changelog tests moved), internal/core/surface (sentences.go), internal/core/release (the comment naming the verb), .abcd/rules.json (DOGFOODING), .abcd/development/brief/01-product/04-scope.md (the operator-internal list), commands/launch.md, commands/intent.md, agents/release-changelog-composer.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md, .abcd/development/brief/04-surfaces (04-launch.md, README.md operator-internal row), .abcd/development/brief/05-internals/08-skills.md, .abcd/development/brief/02-constraints/04-naming.md, .abcd/development/principles/pre-existing-is-not-a-defence.md (names the verb)
   - tests: TestLaunchDryRunRendersTheCut (version, records, guard; a refused cut changes no exit code) and TestChangelogIsAnUnknownCommand, each watched fail first; the changelog tests rehomed onto the preview; rules_dogfooding_recall_test passes with changelog dropped; TestRootHelpListsThePersonsGroups, which hardcodes the person's group table and names changelog among the agent verbs it checks, has changelog dropped from that list (it would still pass, checking the absence of a verb that no longer exists)
   - peer findings folded into 04-launch.md: "The preview always exits 0" (line 15, repeated near line 229) is false, since `--baseline` naming a non-tag exits 2 and an undeclared artefact kind is refused; the README's changelog row says itd-73 and itd-67 are "both in intents/planned/" when both are shipped, which dissolves with the row
   - per decision 5: the preview is `launch --dry-run` rendering the deterministic cut
4. ahoy install carries what prepare-this-repo did, and the page goes
   - packages: commands/ahoy.md, commands/prepare-this-repo.md (deleted), commands/lint.md, internal/core/ahoy (onboarding_test.go, persona_conventions_test.go), internal/core/implement/loop (brief.go, issuebrief.go), internal/core/repolint (rule_docs.go), internal/surface/cli (staleusage.go), internal/core/lint (config.go comment), .abcd/record-lint.json (host_delegated), README.md, .abcd/development/brief/04-surfaces (15-prepare-this-repo.md folded into 01-ahoy.md, README.md row, index and host-delegated paragraphs), .abcd/development/brief/05-internals/08-skills.md, .abcd/development/brief/02-constraints/04-naming.md
   - tests: the onboarding and persona tests repointed at commands/ahoy.md and watched fail before the workflow moves; a loop-brief test asserting the conventions hint names `abcd ahoy install`, watched fail first; the parity test passes with prepare-this-repo gone from pagesWithNoVerb
   - peer findings folded into 01-ahoy.md: it calls `status` a plugin-page alias for the bare form, but commands/ahoy.md has the binary refuse `status` like any other word, and `abcd ahoy status` is an unknown command; its user-scope tree, declared the one inventory, leaves out oracle-routing.json, interviews/, cache-attestation and statusline.json, all of which the shipped code reads or writes and the 03-configuration.md symlink rule lists
   - per decision 6: a page-only fold; the binary and its refusals do not change
5. One library verb carries the memory and source sub-trees, and the memory and source pages become the library page
   - packages: internal/surface/cli (source.go and the memory builder in cli.go become one library front door, helpgroups.go, staleusage.go, banlist.go:71 and barerender.go:48, which name `abcd source sync-banlist` and `abcd source init`, the source and memory CLI tests renamed), internal/core/surface (sentences.go, examples.go), internal/core/memory and internal/core/source (user-facing strings only), internal/core/guard (defaults/guard.json, sourceflip_test.go), internal/core/ahoy (defaults/pre-commit, and detection of a committed hook naming the old verb as a gap that install rewrites), .githooks/pre-commit, internal/core/banlist (hook_sources_test.go), .abcd/rules.json (DOGFOODING), commands/library.md (new), commands/memory.md and commands/source.md (deleted), commands/consult.md and commands/ingest.md (repointed), commands/guard.md, commands/banlist.md (line 127 names `abcd source sync-banlist`), .abcd/development/release/surface.json, docs/reference/cli/commands.md, .abcd/development/brief/04-surfaces (07-memory.md and 33-source.md folded into a new library chapter, README.md rows, index and the library-and-memory section), .abcd/development/brief/05-internals (07-memory.md, 08-skills.md), .abcd/development/brief/02-constraints/04-naming.md, .abcd/development/brief/01-product/04-scope.md (the memory line)
   - tests: TestLibraryCarriesEveryMemoryAndSourceSubVerb and TestMemoryAndSourceAreUnknownCommands, watched fail first; the guard fixtures moved to `abcd library ledger --flip` and watched fail against the old registry entry; hook_sources_test driven through `library sync-banlist`; TestBareLibraryRendersBothHalves (no corpus inside a checkout renders the memory half and a corpus line; refuses only when neither half reads) and TestAhoyReportsAHookNamingTheOldVerb, each watched fail first; TestRootHelpListsThePersonsGroups rewritten, since its Records row names memory and source, which become library, so this step breaks it; the menu, parity, sentence, appendix and dogfooding tests pass after go generate ./internal/surface/cli
   - most of the diff is renames and moves; if the non-move lines pass about 800, split the strings in internal/core, or the ahoy hook gap, out as a step of its own that changes no verb
   - per decisions 7, 8 and 13: every sub-verb keeps its name; bare library renders both halves; ahoy reports and install rewrites a hook naming the old verb
   - the step's agent files a capture recording that `ingest` now means distil, while the retired `/abcd:ingest` meant register
6. The library asks whether an item is confidential, and consult and ingest fold into its page
   - packages: internal/core/source (add.go, the class resolution), internal/surface/cli (library add's terminal question and its seam, staleusage.go), commands/library.md, commands/consult.md and commands/ingest.md (deleted), .abcd/record-lint.json (host_delegated emptied), internal/core/lint (config.go comment), .abcd/development/brief/04-surfaces (13-consult.md and 14-ingest.md folded into the library chapter, README.md rows, index, host-delegated and No skills paragraphs), .abcd/development/brief/05-internals/08-skills.md
   - tests: TestLibraryAddAsksTheClassAtATerminal, TestLibraryAddRefusesWithoutAClassOffATerminal, TestLibraryAddPrivateIsNotAsked, TestLibraryAddPublicIsNotAsked and TestConfidentialFlagIsUnknown, each watched fail first; a page test that commands/library.md tells the agent to ask through the host's question tool unless the person said; the parity test's new assertion that pagesWithNoVerb holds only the board, watched fail while consult and ingest remain
   - peer findings folded into 08-skills.md, which this step rewrites: "The release payload declares all four kinds" contradicts .abcd/config/launch-payload.json and the chapter's own later paragraph (no skills directory); "Five verbs have a Go verb and no command page" leaves out statusline; "Three command pages carry no Go verb" leaves out version (moot once step 2 lands); "Two of those three call no part of the binary" is false, since consult and ingest both run `abcd source` (moot once this step lands); the enumeration is rewritten from the tree as it stands after this step
   - per decisions 9 and 10: `--private` and `--public`, `--confidential` gone with no alias, stored names unchanged; the two stores stay unconnected
7. The user-facing and brief sweep, and the library beside memory notes once open question 12 is answered
   - packages: docs/explanation (the library and memory comparison, A8, only once open question 12 is answered), docs/reference/terminology.md, docs/how-to (an upgrade guide for the breaking release, on the precedent of upgrade-to-v0.13.0.md, naming each retired command and its successor, and the `ahoy install` run that rewrites a pre-commit hook naming the old verb), docs/how-to/README.md, mkdocs.yml, README.md, .abcd/development/brief/01-product/04-scope.md, .abcd/development/brief/02-constraints/04-naming.md, .abcd/development/brief/04-surfaces/README.md, .abcd/development/brief/glossary (library and memory-note entries, if they name a retired command), .abcd/record-lint.json (the one `retired-plugin-pages` entry in banned_tokens, per decision 11), and the lines that ban finds: .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/principles/script-first-mvp.md, the planned intent itd-2610090831227812 and the draft itd-2609292108089653
   - tests: record-lint with the new entry watched fail on the lines below before they are edited, then clean; docs-lint and site-render clean; a docs-currency review of the changed pages; the docs page read against the brief's library-and-memory table so both say the same thing
   - peer findings folded into 02-constraints/04-naming.md: the ahoy row lists install, uninstall and remote apply as the write paths, but connect and credential write too; the exemption list leaves out build, dashboard, drain, implement, inbox, lab, mode, peers, report, scribe, source and statusline, and is rewritten from the post-merge set; the task_classes enum omits intent_consistency, which agents/intent-auditor.md declares; "no cross-check test reads the field" is false, since agentcontract.go checks task_classes is present (not its values); "two verbs refuse instead of rendering" is stale, as launch, decide and build all refuse bare
   - peer findings folded into 04-surfaces/README.md: the four agents said to "serve no verb, and no command page calls on them" are partly called (build.md and the loop's validator roles dispatch ruthless-reviewer and security-reviewer; guard.md and intent.md call question-drafter; only sota-researcher is uncalled); the claim that a registered sub-command must have a row leaves out the exemption of hidden subtrees (launch smoke-pages, dashboard serve; subverbs.go); the agents paragraph cites iss-110, a resolved issue about a different defect, as the tracker of an open gap
   - peer finding folded into 01-product/04-scope.md: it calls `/abcd:reflect` "a design target (itd-24, planned)", but itd-24 is in shipped/, so it reads "(itd-24, shipped)"; the same page's `/abcd:memory` line and its list of operator-internal verbs (which names `changelog`, `rules` and `spec`) are kept true by steps 3 and 5 and checked again here
   - the ban's line count, measured at 9d990d87f with the entry applied to a scratch copy of the config: 53 lines under .abcd/development, outside the exempt paths, name a retired slash spelling with neither allow word on the line. Eight were this spec's, which now carry the word. Thirty-eight sit in the chapters and paragraphs steps 2 to 6 delete, fold or rewrite (12-version.md 2, 13-consult.md 6, 14-ingest.md 9, 15-prepare-this-repo.md 6, 33-source.md 3, 04-surfaces/README.md 9, 05-internals/08-skills.md 3), and those steps must leave none behind in what they write. That leaves seven lines in six files for step 7, and it edits six of them: 16-lint.md's opening paragraph (one line) and two lines of 03-configuration.md's section "The two `.abcd/` scopes" are forward-looking and name the successor; script-first-mvp.md's live-instance paragraph (one line) names the successor or says the pages retired; the intent's A9 line (77) and the draft's line 84 gain the word. The seventh, itd-147's audit-evidence line 271, is a verbatim quote the evidence-line pattern covers with no edit. A line steps 2 to 6 add or leave is theirs to fix; record-lint names it
   - builds everything except A8, which waits on open question 12; the ban follows decision 11
8. The live captures, the doc-fidelity review and the close
   - packages: this spec (the BEFORE and AFTER captures under Live captures), .abcd/development/intents (the intent moves to shipped by the close), .abcd/work/issues (iss-2610090831317531 resolved, if it is still open)
   - tests: the BEFORE capture in a live host session at a checkout of 314e9c3ce, step 1's base (what `/abcd:` lists, and what typing `/abcd:scribe` answers); the AFTER capture at the merged tip: what `/abcd:` lists (the fifteen, A1); typing `/abcd:guard` in full gets the host's refusal (A4); an agent asked in plain words runs `/abcd:guard` and it behaves as before (A2); a docs review recorded for HEAD with go run ./cmd/abcd docs fidelity record; then go run ./cmd/abcd spec close spc-2610100613109045 with a `Delivers: itd-2610090831227812` trailer (the intent already declares `impact: breaking`, so no --impact is passed)
   - the live session is the only evidence for A1, A2 and A4 at the host; script output does not stand in for it
   - waits on the product thinker's answer to open question 12 and the A8 work it decides

## Footprint

- packages: commands, agents/release-changelog-composer.md, internal/core/launch, internal/core/board, internal/core/surface, internal/core/release, internal/core/memory, internal/core/source, internal/core/guard, internal/core/ahoy, internal/core/banlist, internal/core/implement/loop, internal/core/repolint, internal/core/lint, internal/surface/cli, .githooks/pre-commit, .abcd/rules.json, .abcd/record-lint.json, .abcd/development/release/surface.json, docs/reference, docs/explanation, docs/how-to, mkdocs.yml, README.md, .abcd/development/brief/01-product, .abcd/development/brief/02-constraints, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/brief/glossary, .abcd/development/principles/pre-existing-is-not-a-defence.md, .abcd/development/principles/script-first-mvp.md, .abcd/development/intents (the planned intent's A9 line and draft itd-2609292108089653)
- tests: TestCommandPagesMatchTheMenu, TestPluginPersonListEqualsTheCLIs, TestBoardShowsTheVersion, TestLaunchDryRunRendersTheCut, TestChangelogIsAnUnknownCommand, TestLibraryCarriesEveryMemoryAndSourceSubVerb, TestMemoryAndSourceAreUnknownCommands, TestBareLibraryRendersBothHalves, TestAhoyReportsAHookNamingTheOldVerb, TestLibraryAddAsksTheClassAtATerminal, TestLibraryAddRefusesWithoutAClassOffATerminal, TestLibraryAddPrivateIsNotAsked, TestLibraryAddPublicIsNotAsked, TestConfidentialFlagIsUnknown, the library page's ask test, the loop-brief hint test, the guard flip fixtures under library, hook_sources_test under library sync-banlist, the onboarding and persona tests on commands/ahoy.md; TestPersonsListHoldsAtMostFifteenVerbs, TestPersonVerbsCountsEveryListedVerb and TestRulesAndSpecKeepTheirPlaces retired; the snapshot, CLI reference, appendix and sentence drift tests after go generate ./internal/surface/cli; record-lint, docs-lint and site-render clean; the before and after live captures

## Decisions (technical facilitator, 2026-10-10)

The technical facilitator ruled on twelve of the thirteen questions this spec
first left open (numbered as first written, so 12 is missing), in an autonomous run the person authorised on 2026-10-10.
Decision 14 closes a gap an independent review of this spec found the same
day. Each ruling is folded into the design and the step it affects.

- **1. The CLI groups.** ideate and library go under Records, reading and lab
  under Checks, and dashboard under Set-up. Grounds: least change, and Checks
  keeps a purpose once lint leaves it.
- **2. The pages the rules and spec lines name.** Both name an existing page:
  spec names `commands/intent.md`, and rules names `commands/ahoy.md`.
  Grounds: no ruling covers adding a page.
- **3. rules and spec leave the person's help.** Type: supersession.
  Supersedes: itd-146, the criterion "No verb hidden, renamed, moved, or
  nested", in its clause that `rules` and `spec` stay listed on the person's
  help. By: the product thinker's placement of 2026-10-09 (P4), which makes
  both agents only. `TestRulesAndSpecKeepTheirPlaces` retires in step 1.
  Grounds: the later ruling is the product thinker's, and the marker block's
  `abcd rules` still runs, because the verb is listed elsewhere, not removed.
- **4. The version on the board.** A last line in both views and both forms
  (for example `abcd v0.13.4`), plus a `version` field in `--json`; the first
  line stays the label alone, per spc-2610031844142274's decision 2. Grounds:
  the product thinker ruled the version shows on `/abcd:abcd` and on bare
  `abcd`, which is the person's view.
- **5. The launch preview.** Spelling: `launch --dry-run` renders the cut.
  Content: the deterministic cut, which is what `abcd changelog` shows today.
  Grounds: no new name, and no agent in a preview.
- **6. The prepare-this-repo fold.** Page-only, into `commands/ahoy.md`'s install
  section, with no new refusal of `ahoy install`. Grounds: least change; a
  new refusal would break installs that work today.
- **7. The library's sub-verb names.** Every sub-verb keeps its name; step 5's
  agent files a capture recording the double meaning of `ingest`. Grounds:
  the intent puts any rename beyond the merges out of scope.
- **8. Bare `library`.** It renders both halves, each absent half as a line, and
  refuses only when neither can be read. Grounds: loud staging.
- **9. The class flags.** `--private`, as accepted criterion A13 names it, and
  `--public`; neither is asked about, because the person has said.
  `--confidential` goes with no alias (adr-40), and the stored folder and
  field keep their names. Grounds: the criterion's letter, and the intent's
  "when the person did not say".
- **10. The two stores.** Two stores under one verb, unconnected. Grounds: least
  change.
- **11. Banning retired names.** Ban only the `/abcd:` spellings of the five
  retired page names nothing reuses (consult, ingest, prepare-this-repo,
  version and source), as one `banned_tokens` entry whose allow context is
  the words `historical` and `retire` and the audit-evidence line pattern. The
  allow context is per line, so each historical line that keeps a retired
  spelling carries one of the words on that line; there is no per-record
  escape. The bare `abcd changelog` and `abcd source` CLI spellings are not
  banned, to keep step 7's line budget sane (69 lines carry them today), and
  `/abcd:memory` and `abcd memory` are left out. Grounds:
  itd-2610091918433290 reuses the memory names, and the unknown-command tests
  and stale-usage notes guard the CLI spellings.
- **13. Hooks already committed in managed repositories.** `ahoy` reports a hook
  that names the old verb as a gap, and `ahoy install` rewrites it; the
  upgrade guide says so too. The `abcd.sourcesBinary` git setting keeps its
  name. Grounds: renames are out of scope.
- **14. The four pages that leave, until they leave.** consult, ingest,
  prepare-this-repo and version carry no `block:` today, and the menu test
  requires one on every page. Step 1 writes `block: people` on each, and the
  step that deletes the page retires it (step 2 for version, step 4 for
  prepare-this-repo, step 6 for consult and ingest). Grounds: the intent's
  2026-10-09 entry classes consult and ingest by the audit on the same rule
  as every page, that a command a person needs is a person's; prepare-this-repo
  folds into `ahoy install`, a person's command; and version keeps its current
  visibility on the menu until step 2 retires it.

## Open Questions

12. **Which docs page carries the library-and-memory comparison, and how it
    speaks of memory notes** (step 7's A8 part, and the close in step 8). This
    one stays open for the product thinker. Accepted criterion A8 asks the
    user-facing docs to set the library beside memory notes, but memory notes
    are a draft intent with nothing built, and the docs describe only what
    ships, in the present tense. Step 7 builds everything except A8, and the
    close waits on this answer.
    - (a) A new `docs/explanation/` page, which says memory notes are coming
      and names no command for them until they ship.
    - (b) A section of `docs/reference/terminology.md`, worded the same way.
    - (c) Wait: ship the library half now, and add the comparison when
      itd-2610091918433290 ships memory notes (A8 would then move to that
      intent).
