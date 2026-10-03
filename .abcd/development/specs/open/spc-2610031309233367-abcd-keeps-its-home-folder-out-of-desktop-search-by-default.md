---
id: spc-2610031309233367
slug: abcd-keeps-its-home-folder-out-of-desktop-search-by-default
intent: itd-2610030720038073
origin: researcher-authored
production_mode: hand-written
---
# abcd-keeps-its-home-folder-out-of-desktop-search-by-default

## Summary

This spec delivers
[itd-2610030720038073](../../intents/planned/itd-2610030720038073-abcd-keeps-its-home-folder-out-of-desktop-search-by-default.md)
(abcd's own folder gives the desktop search indexer no work to do) under the
rule its decision record states,
[adr-2610030720195401](../../decisions/adrs/2610030720195401-abcd-keeps-its-own-folders-out-of-desktop-indexing-only-by.md):
abcd changes only its own folder and never the computer's search settings. The
file name and slug are the draft's, kept by contract; the intent's title is the
one that describes the work.

Four things land. One home resolver, `internal/abcdhome`, becomes the only Go
code that spells the home folder's name, and an AST literal test holds every
other package to it. The name then changes in that one place, `~/.abcd` to
`~/.abcd.noindex`, and the same resolver decides the stop: with an old
`~/.abcd` standing, every command, every hook and the status line write
nothing and name the folder and the one rename command. The plugin hooks'
shell wrapper and the bootstrap script, which run ahead of the binary, make the
same stop before they provision or read anything in the home. Two superseding
decision records carry the location change decision 4 lists (the worktree
store's location and the transcript store's spelling under brief invariant
15), and every text naming the old folder follows, the managed block included,
which a project receives when setup runs there again.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 6 (decision 6 replaces decision 2's automatic move: abcd moves
nothing and stops until the person renames the folder), its confirmed criteria
D1 to D6, its four scope conditions, and the decision record's rule.

## Scope

In:

- A new leaf package, `internal/abcdhome`: the home's name, its old name, the
  paths and display forms built from them, and the stop check. It imports only
  the standard library.
- Every Go spelling of the home moves onto the resolver. Today the name is
  spelled in more than thirty non-test files, for example `WorktreeStoreRel`
  (`internal/core/implement/loop/lane.go`, line 34), `userStoreRelPath` and
  `LocalRootsRelPath` (`internal/core/history/location.go`, lines 62 and 72),
  `UserRelPath` (`internal/core/rules/rules.go`, line 49),
  `TrustedRootsRelPath` (`internal/core/rules/root.go`, line 17),
  `pathEntryRel` (`internal/core/ahoy/owned_copy.go`, line 110), the
  `EnsureHomeScope(home, ".abcd", …)` writers in `internal/core/credential`
  (`store.go`, line 231; `credential.go`, line 199) and
  `internal/core/ahoy/oracle_routing.go` (line 140), `File.MachineOrigin` in
  `internal/core/layered/layered.go` (line 120), and the `~/.abcd/…` text in
  user-facing messages across `internal/core` and `internal/surface/cli`.
- The stop in the binary's one front door, `cli.Run`
  (`internal/surface/cli/cli.go`, line 6247), with a form for each hook verb
  `applyHookPlaneFailOpen` lists (line 144), for `guard hook`, and for
  `statusline`.
- The plugin hooks' shell wrapper (each command in `hooks/hooks.json`, whose
  PATH fallback reads `~/.abcd/path-entry` and refuses a symlinked `~/.abcd`)
  and `hooks/bootstrap.sh` (the symlink refusal at line 126, the path-entry
  record at line 477, the cache attestation at line 966).
- The texts the binary embeds: the managed block
  (`internal/core/ahoy/defaults/claude-md-marker-block.md`, its
  `~/.abcd/rules.json` and `~/.abcd/trusted-roots` lines) and the scratch rule
  in `internal/core/rules/defaults/rules.json`.
- Two superseding decision records (decision 4): one for the worktree store's
  location, which
  [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
  binds as `~/.abcd/worktrees/<root-sha>/<name>/`, and one for the transcript
  store's spelling, which
  [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)
  sets and brief invariant 15 locks through
  `TestOnlyTheHistoryPackageNamesTheStorePath`
  (`internal/core/history/store_boundary_test.go`).
- abcd's own tree: `AGENTS.md` (the rule-loader section, the working-tree
  layout and § Concurrent sessions, fourteen lines naming the home), the
  `.githooks/pre-commit` corpus path (line 508), `README.md`, `commands/`,
  `docs/how-to`, the generated `docs/reference/cli/commands.md`, the brief
  chapters that name the home, `internal/README.md`, and the principle
  [`the-users-directory-is-theirs`](../../principles/the-users-directory-is-theirs.md)
  (its Enforcement paragraph names the store's path).
- The store records this intent refines:
  [itd-2609091014076309](../../intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)
  and its open spec
  [spc-2609301811532881](spc-2609301811532881-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)
  spell the store's path; the spelling follows the superseding record, and
  nothing else in them changes.
- The D6 receipt, a dated research note.

Out:

- The repository tier. A project's own `.abcd/` folder (`development/`,
  `work/`, `.work.local/`, `rules.json`, `config.json`, `guard.json`) keeps its
  name; only the home in the person's home folder is renamed.
- Any move, copy or read of an old `~/.abcd` (decision 6). No reader falls
  back to the old name, so there is no window in which both names are read:
  the stop replaces it.
- A project abcd manages whose setup has not run again: its block keeps the old
  paths until it does (the press release, D4). abcd never reaches into it.
- Scripts a person wrote: theirs to update.
- The computer's search settings (`mdutil`, the Spotlight privacy list): the
  decision record's rule, held by D5.
- Windows, a later change with `FILE_ATTRIBUTE_NOT_CONTENT_INDEXED` named for
  it, and Linux, settled by its indexers (decision 3; scope conditions 3 and
  4). The folder has one name on every platform; on Linux and Windows the
  suffix is inert.
- The worktree store's verbs, which are spc-2609301811532881's. This spec moves
  only where its root is spelled.

## Approach

### One home resolver

`internal/abcdhome` holds the only spellings:

```go
// name is the home folder abcd keeps under the person's home directory. The
// ".noindex" suffix is what the macOS indexer honours at scan time
// (itd-2610030720038073, adr-2610030720195401).
const name = ".abcd.noindex"

// oldName is the folder the home was called before; it is read by Check and
// by nothing else.
const oldName = ".abcd"

// RenameCommand is the one command the stop names.
const RenameCommand = "mv ~/.abcd ~/.abcd.noindex"

func Rel(leaf ...string) string            // slash path below the home directory: ".abcd.noindex/trusted-roots"
func Path(home string, leaf ...string) string // absolute path under home
func Display(leaf ...string) string        // "~/.abcd.noindex/trusted-roots", for messages
func Check(home string) *Stop              // nil, or the stop and its line
```

`Rel` is what the home-scope primitives in `internal/fsutil/home.go`
(`EnsureHomeScope`, `OpenHomeScope`, `HomeScopeLink`, `ReadHomeDeclaration`
and its two variants, `HomeDeclarationNames`) already take as their `rel`, so
the readers' guards are unchanged: each level of the path, the home folder
itself included, is judged and a symlinked level is refused as it is today. A
package that resolved `os.UserHomeDir()` keeps doing so; it joins the result
with `Path` instead of spelling the name.

The resolver is a leaf at `internal/`, beside `internal/fsutil`, because the
spellings sit in `internal/core`, `internal/surface/cli` and `cmd`, and a leaf
every one of them can import holds no import edge back. It imports nothing
from `internal/core`, so the transport-agnostic boundary is untouched.

The sweep lands first with the name unchanged (step 2), so the change of name
is one constant (step 3). Test fixtures that build a home under a temporary
`HOME` move onto `Path` in the same sweep, so the change of name moves them
too.

### The stop

`Check(home)` reads two names with `os.Lstat` and nothing else:

| `~/.abcd` | `~/.abcd.noindex` | Result |
| --- | --- | --- |
| absent | either | no stop |
| present (a folder, a file or a link) | absent | stop: the old folder stands |
| present | present | stop: both stand |

An empty or relative `HOME` is no stop: the existing refusals for that shape
stand. The two lines, written once in the resolver:

> abcd's folder is now ~/.abcd.noindex, a name the Mac's search indexer passes
> over, and ~/.abcd still stands, so abcd has written nothing. Rename it with
> `mv ~/.abcd ~/.abcd.noindex`, then run abcd again.

> Both ~/.abcd and ~/.abcd.noindex exist, so abcd has written nothing and
> moves neither. Keep the one you want, named ~/.abcd.noindex, and take the
> other out of your home folder, then run abcd again.

Where it is checked, before any write:

- **`cli.Run`**, before `root.Execute()`. Every verb, every hook verb and the
  status line reach the binary through it, and nothing before it writes. It
  finds the command `args` name (`root.Find`) and renders the stop in that
  command's form:
  - an ordinary verb: `abcd: <line>` on stderr and exit 1, or the existing
    `--json` error envelope (`newErrorEnvelope`) on stdout;
  - `hook prompt-router` and `hook session-start`: the line on stdout, the
    channel the host adds to the agent's context, so the agent tells the
    person; exit 0, the hook plane's contract that a hook never wedges a
    session;
  - `hook prompt-router-reset`, `hook session-end` and `hook subagent-stop`:
    the hook result `emitHookResult` already writes for an uncaptured
    transcript (`hookOutcomeNotCaptured`), with the line as its reason;
  - `guard hook`: refused with the blocking status and the line on stderr,
    every shell command and question alike, except the exact rename command,
    which it admits (open question 3);
  - `statusline`: the short form, `abcd stopped: rename ~/.abcd to
    ~/.abcd.noindex`, on stdout, exit 0, in every checkout. The previous
    status command is not run, because its record is in the home abcd does
    not read while stopped.
- **The hooks' shell wrapper**, at the top of each command in
  `hooks/hooks.json`, before the bootstrap attempt marker, the bootstrap call
  and the PATH fallback. With `~/.abcd` present (`[ -e ]` or `[ -L ]`) and an
  executable plugin-root binary, the wrapper skips the bootstrap and runs the
  binary, which renders the stop in its own form. With no plugin-root binary, the wrapper
  provisions nothing, reads no path-entry record, prints the line on stderr
  and exits as its missing-binary branch does today. The PATH fallback reads
  `~/.abcd.noindex/path-entry`, and its symlink refusal names
  `~/.abcd.noindex`.
- **`hooks/bootstrap.sh`**, which a person can also run by hand: right after
  its `HOME` checks, the old folder ends the script with the line before any
  download or record is written. Its home records (`path-entry`,
  `cache-attestation`) move to `~/.abcd.noindex`, and its symlink refusal
  (line 126) names the new folder.

Nothing else is a door into the home: `cmd/record-lint` and the redaction in
`internal/fsutil/paths.go` resolve the home directory only to hide it in
output, and the generators under `cmd/` write into the checkout.

### The trust paths

`trusted-roots` (`rules/root.go`, line 374), the user `rules.json`
(`rules/rules.go`, line 451) and `path-entry` (`ahoy/owned_copy.go`, line 177)
are read through `fsutil.HomeDeclarationNames` and `fsutil.ReadHomeDeclaration`
with `abcdhome.Rel(…)`. The guards apply to `.abcd.noindex` exactly as they
applied to `.abcd`: a symlinked home is refused, a record another account owns
or can write is refused. No reader ever reads `~/.abcd`, so a link there
admits nothing (open question 2); while it stands, the stop holds anyway. The
shell wrapper's path-entry check keeps its three-part guard (a regular file,
owned by the caller, writable by nobody else) on the new path.

### The managed block

The embedded block names `~/.abcd.noindex/rules.json` and
`~/.abcd.noindex/trusted-roots`, and abcd's own `AGENTS.md` section that
mirrors it matches. Nothing new is needed for D4: `classifyMarker`
(`internal/core/ahoy/marker.go`, line 82) already reports a block that differs
from the embedded one as outdated, only install (`apply.go`, line 1040) and
embark write the block, and the replacement rewrites the fenced block alone.
Before setup runs in a project, nothing touches its file.

### The two superseding records

Each is minted with `abcd decide` and lands `proposed` in step 1, its
`related_adrs` naming the record it will supersede. Step 3, the change that
makes each true, accepts it, writes `supersedes` and the old record's
`superseded_by` and `status: superseded` (record-lint holds both directions),
and changes the texts it binds:

- **The worktree store's location**, `~/.abcd.noindex/worktrees/<root-sha>/<name>/`,
  carrying adr-2609091248200336's trust rule forward unchanged. Its texts:
  `AGENTS.md` § Concurrent sessions (line 236, and the `trusted-roots` command
  at line 64); the principle's Enforcement paragraph (lines 43 to 48); the
  brief's § The worktree store (`05-internals/03-configuration.md`, line 374);
  `commands/build.md`, `commands/implement.md` and
  `04-surfaces/34-build.md`; the scratch rule in the bundled rules; and the
  path spelling in itd-2609091014076309 and spc-2609301811532881, which gains
  a dated decision line naming the new record.
- **The transcript store's spelling**, `~/.abcd.noindex/transcripts/<root-sha>/`
  and the declaration `~/.abcd.noindex/local-transcript-roots`, carrying
  adr-2609091248201071 forward unchanged otherwise. Its texts: brief invariant
  15 (`02-constraints/03-invariants.md`, line 43), the working-tree layout in
  `AGENTS.md` (lines 197 and 198), `04-surfaces/11-history.md`,
  `05-internals/03-configuration.md`, `commands/history.md` and
  `docs/how-to/install.md`.

### The boundary tests

`TestOnlyTheHomeResolverNamesTheHome` lives in `internal/abcdhome`, in the
shape of `TestOnlyTheHistoryPackageNamesTheStorePath`: it parses every non-test
Go file under `internal/` and `cmd/` (testdata skipped, the walk required to
read at least a hundred files) and refuses, outside the resolver's package:

1. a string literal containing `.abcd.noindex`;
2. a string literal containing `~/.abcd`;
3. a call that takes a home value and a `.abcd`-led literal (`.abcd`, or
   `.abcd/` and more), or a constant of the same package holding one: a home
   value is an identifier or selector whose name ends in `home` in any case
   (`home`, `s.home`, `homeDir`), or a call to `os.UserHomeDir` or
   `userHome`. This catches `filepath.Join(home, ".abcd")`,
   `fsutil.ReadHomeDeclaration(home, UserRelPath, …)` with
   `UserRelPath = ".abcd/rules.json"`, and
   `workingTreeAbove(home, ".abcd")` (`internal/core/credential/store.go`,
   line 223), and leaves `filepath.Join(repoRoot, ".abcd", "config.json")`
   alone.

Each finding names its file, line and literal. `TestHomeNameScannerIsArmed`
feeds each shape as a hostile source and a comment naming the home as a benign
one, as the history test's armed twin does.

`TestNoCodeNamesTheSearchSettings` walks the same tree and refuses any literal
containing `mdutil`, `.Spotlight-V100` or `VolumeConfiguration.plist`, with
its armed twin. Literals, not comments: the resolver's own comment explains
why abcd never touches those.

Invariant 15's test changes shape in step 2, since the history package no
longer spells the home: its needles keep the old literal spellings and gain
`.abcd.noindex/transcripts`, and it also refuses a call into `abcdhome` whose
literal leaf begins `transcripts` or `local-transcript-roots` outside
`internal/core/history`. Its declared exception for the cold-reading
assembler's exclusion row is unchanged, because that row names the repository
tier's `.abcd/.work.local/transcripts`, which this change does not rename.

## How each acceptance criterion is met

**D1.** `TestFirstRunCreatesOnlyTheNoindexHome`, in `internal/surface/cli`,
sets `HOME` to an empty temporary folder, runs `abcd ahoy install --yes` in a
fresh repository and then a verb that writes a run log, and asserts that
`<home>/.abcd.noindex` exists, holds the run log, and that no
`<home>/.abcd` exists. Every home writer reaches the folder through `Path` or
`Rel`, which D3 holds, so no other writer can create the old name.

**D2.** `TestHomeStopCheck`, in `internal/abcdhome`, covers each row of the
table, an old folder that is a symlink, and an empty `HOME`.
`TestOldHomeStopsEveryVerb` and `TestOldHomeStopsEveryHook`, in
`internal/surface/cli`, set `HOME` to a folder holding `.abcd/` with a file in
it and run, through `cli.Run`, an ordinary verb that would write (plain and
`--json`), each hook verb, `guard hook` with an ordinary command, a question
and the rename command, and `statusline`. Each asserts the line naming
`~/.abcd` and the rename command in that entry's form and exit status, the
rename command admitted, and the home folder's tree (names, sizes and
modification times) identical before and after, with no `.abcd.noindex`
created. `TestBothHomesStopAndNameBoth` repeats it with both folders present
and asserts the second line. `TestHookWrapperStopsBeforeProvisioning`, beside
the existing `hooks_selfprovision_test.go` tests, runs each `hooks/hooks.json`
command under `sh` with no plugin-root binary and the old folder present, and
asserts the line, no bootstrap attempt marker, and no change under `HOME`;
`TestBootstrapWritesNothingBesideTheOldHome` does the same for
`hooks/bootstrap.sh`.

**D3.** `TestOnlyTheHomeResolverNamesTheHome` and
`TestHomeNameScannerIsArmed`, above. The criterion's example, a new reader
that joins the home with `.abcd` itself, is the third shape, and the failure
names its file and line.

**D4.** `TestSetupRefreshesTheBlocksHomePaths`, in `internal/core/ahoy`, writes
an `AGENTS.md` holding text above and below the block as it was planted before
the change (kept as a testdata fixture), and asserts two states: after
`Detect`, the file is byte-identical and the block is reported outdated; after
install, the block names `~/.abcd.noindex/rules.json` and
`~/.abcd.noindex/trusted-roots`, no `~/.abcd/` remains in it, and every line
outside the block is byte-identical. `TestEmbeddedDefaultsNameTheNewHome`
asserts that neither embedded default (the block, the bundled rules) names
`~/.abcd/`.

**D5.** `TestNoCodeNamesTheSearchSettings` and its armed twin, above. No
literal names any of the three today, so the armed twin is the test watched
fail first.

**D6.** A dated research note,
`.abcd/development/research/notes/<yyyy-mm-dd>-noindex-scan-receipt.md`
(open question 4). On a Mac with indexing on, macOS 27.0 first, a person
creates eight working copies of one repository within four minutes with plain
`git worktree add`, first under a dot-folder without the suffix in the home
folder (the old shape) and then under `~/.abcd.noindex/worktrees/<root-sha>/`,
and samples corespotlightd, mds_stores and mdworker CPU every five seconds for
a minute after each (`top -l 13 -s 5`). The note records the date, the macOS
version, both sample sets and their combined figures, and the run log of one
autonomous run that opened its lanes in the new store, searched for a `stop`
naming indexing. The pass is the after set under 20% combined and no such
`stop`. It is a load experiment on a live development machine, so the person
consents before it runs, and every process it starts is its own.

## Open design questions

For the technical facilitator; the records do not settle these.

1. **How D3's test tells the home's `.abcd` from a repository's `.abcd/`.**
   The two share a name, and the repository tier keeps it. (a) Literal needles
   for `.abcd.noindex` and `~/.abcd`, plus the call shape: a `.abcd`-led
   literal, or a same-package constant holding one, passed beside a home value
   (designed to). It catches every home spelling in the tree today, including
   `rules.json`, whose relative spelling is the same in both tiers; a home
   value under a name the rule does not know passes, and the armed test pins
   the shapes it does know. (b) Move the repository tier's spelling into one
   constant in a second declared package too, and refuse every bare `.abcd`
   literal outside the two: exact, but a sweep of about fifty repository-tier
   literals this intent does not otherwise touch. (c) Literal needles only,
   with a list of leaves only the home holds: simplest, but
   `filepath.Join(home, ".abcd", "rules.json")` passes unseen.
2. **What a `~/.abcd` that is a symlink counts as.** A person may link the old
   name to the new folder to keep their own scripts working. (a) Any entry at
   `~/.abcd` stops, a link included (designed to): D2's "an existing
   `~/.abcd`" read as it is written, one `Lstat`, and the press release
   already leaves scripts to their owner. (b) A link whose target resolves to
   `~/.abcd.noindex` is admitted as the person's alias, never read through:
   kinder to scripts, at the cost of resolving a link in the check every
   entry point runs, and of a second name that leads into the folder.
3. **What the guard hook does while the old folder stands.** (a) Refuse every
   shell command and question with the blocking status and the line, and admit
   only the exact rename command (designed to): nothing runs unguarded, D2's
   "stops" holds, and the person can have their own session run the rename,
   which is their act and not abcd's. (b) Refuse everything, the rename
   included: the strictest, but the person must leave the session to rename.
   (c) Keep judging as today and add the line: the guard's own contract that
   it never stops a session holds, but a hook carries on working, against D2.
4. **Where the D6 receipt lives.** (a) A committed, dated research note
   (designed to): scope condition 1 holds the claim only for versions with a
   dated receipt, so the record has to carry them, and the audit reads them.
   (b) The local tier, as the conventions-file canary's receipt is kept: no
   record churn, but the scope condition and the audit could not see it from
   another checkout.
5. **When the rename step lands.** In abcd's own checkout the installed
   plugin's hooks keep writing `~/.abcd` (the live transcript drain runs on
   every prompt) until the plugin is updated, while a source build of step 3
   stops on that folder, `go run ./cmd/abcd lint docs` included. (a) Step 3
   is the last change merged before a release cut, which follows at once, and
   the person updates the plugin and renames the folder together (designed
   to): the window is one update long. (b) Land it whenever it is ready: no
   coupling to the cut, but every source run on a machine with the old plugin
   stops until the next release.

## Footprint

- packages: internal/abcdhome, internal/core/ahoy, internal/core/credential, internal/core/history, internal/core/implement, internal/core/implement/loop, internal/core/lab, internal/core/layered, internal/core/lifeboat, internal/core/report, internal/core/rules, internal/core/source, internal/core/statusline, internal/surface/cli, internal/README.md, hooks/, .githooks/pre-commit, AGENTS.md, README.md, commands/, docs/how-to, docs/reference, .abcd/development/brief, .abcd/development/decisions/adrs, .abcd/development/principles, .abcd/development/intents/planned, .abcd/development/specs/open, .abcd/development/research/notes
- tests: TestOnlyTheHomeResolverNamesTheHome, TestHomeNameScannerIsArmed, TestNoCodeNamesTheSearchSettings, TestSearchSettingsScannerIsArmed, TestOnlyTheHistoryPackageNamesTheStorePath and TestStorePathBoundaryScannerIsArmed (reshaped), TestHomeStopCheck, TestFirstRunCreatesOnlyTheNoindexHome, TestOldHomeStopsEveryVerb, TestOldHomeStopsEveryHook, TestBothHomesStopAndNameBoth, TestHookWrapperStopsBeforeProvisioning, TestBootstrapWritesNothingBesideTheOldHome, TestSetupRefreshesTheBlocksHomePaths, TestEmbeddedDefaultsNameTheNewHome; the dated D6 receipt; the command reference regenerated; docs-lint and record-lint clean

## Steps

1. The two superseding records, proposed
   - criteria: none directly; decision 4's two records, written before the change that makes them true
   - packages: .abcd/development/decisions/adrs
   - tests: record-lint clean, the decisions index current; each record minted with `abcd decide`, `proposed`, naming in `related_adrs` the record it will supersede, and stating the new spelling, the texts it binds (above) and that it changes nothing else of the record it replaces
2. One home resolver, the name unchanged
   - criteria: D3, D5
   - packages: internal/abcdhome, internal/core/ahoy, internal/core/credential, internal/core/history, internal/core/implement, internal/core/implement/loop, internal/core/lab, internal/core/layered, internal/core/lifeboat, internal/core/report, internal/core/rules, internal/core/source, internal/core/statusline, internal/surface/cli, internal/README.md
   - tests: TestOnlyTheHomeResolverNamesTheHome watched fail on the current tree, naming each spelling, and pass after the sweep; TestHomeNameScannerIsArmed, TestNoCodeNamesTheSearchSettings, TestSearchSettingsScannerIsArmed; invariant 15's test reshaped and still armed; every existing test passing with no path changed, test fixtures moved onto `Path`
   - spc-2609301811532881's step 1 creates `internal/core/ahoy/worktree` from `WorktreeStoreRel`; whichever lands second takes the store root from the resolver, and the D3 test refuses the other order's literal
3. The rename and the stop
   - criteria: D1, D2, D4
   - waits on steps 1 and 2; lands as the last change before a release cut (open question 5)
   - packages: internal/abcdhome, internal/surface/cli, internal/core/ahoy, internal/core/rules, hooks/, .githooks/pre-commit, AGENTS.md, .abcd/development/decisions/adrs, .abcd/development/brief/02-constraints
   - tests: TestHomeStopCheck, TestFirstRunCreatesOnlyTheNoindexHome, TestOldHomeStopsEveryVerb, TestOldHomeStopsEveryHook, TestBothHomesStopAndNameBoth, TestHookWrapperStopsBeforeProvisioning, TestBootstrapWritesNothingBesideTheOldHome, TestSetupRefreshesTheBlocksHomePaths, TestEmbeddedDefaultsNameTheNewHome, each watched fail first; the existing hook-plane, wrapper and bootstrap tests passing on the new path; the two records accepted with both supersession directions, invariant 15 and `AGENTS.md` changed in the same diff; record-lint clean
4. Every other text names the new folder
   - criteria: the docs half of D4 (the press release's "a project … keeps pointing at the old folder until abcd's setup runs in that project again", said in the install guide)
   - packages: commands/, docs/how-to, docs/reference, README.md, .abcd/development/brief, .abcd/development/principles, .abcd/development/intents/planned, .abcd/development/specs/open
   - tests: docs-lint and record-lint clean; the command reference regenerated; a search of the tree for `~/.abcd/` and `~/.abcd ` finding only the stop's own text, superseded records and dated history (the decision log, resolved issues, research notes), listed in the pull request
   - lands after step 3, so no text names a folder the shipped binary does not use
5. The receipt and the close
   - criteria: D6; the intent's close
   - packages: .abcd/development/research/notes, .abcd/development/decisions/adrs
   - tests: the dated receipt on macOS 27.0, taken with the person's consent; adr-2610030720195401 accepted, D5's test now holding its rule; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031309233367` (the intent already declares `impact: breaking`) with a `Delivers: itd-2610030720038073` trailer
