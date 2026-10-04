---
id: spc-2610031156364295
slug: evaluate-whether-claude-md-can-be-removed-safely-now-that
intent: itd-2610030814013772
origin: researcher-authored
production_mode: hand-written
---
# evaluate-whether-claude-md-can-be-removed-safely-now-that

## Summary

This spec delivers
[itd-2610030814013772](../../intents/shipped/itd-2610030814013772-evaluate-whether-claude-md-can-be-removed-safely-now-that.md)
(abcd's projects keep one conventions file, AGENTS.md, and Claude Code reads
it directly) under its standing rule,
[adr-2610030814023326](../../decisions/adrs/2610030814023326-agents-md-is-the-one-conventions-file-abcd-writes-it-never.md).
The file name and slug are the origin issue's, reused by contract; the
intent's title is the one that describes the work.

Five things land. abcd's own repository drops its two committed links,
CLAUDE.md and GEMINI.md, and proves with a canary word that a fresh session
loads AGENTS.md; this lands first, as the fix of iss-2609291925136841. The
write side then narrows: setup and embark plant abcd's block into AGENTS.md
alone, and a saved setup choice of CLAUDE.md or both files stops setup with
the one setting to change, while every reader of the old choices stays. Setup
learns what a tool's own conventions file in an adopted project is: one
holding the owner's words is left untouched and named in a loud warning; one
that only links to or repeats AGENTS.md is offered for retirement and removed
only on a yes. Setup also warns, by presence alone, where a file in a folder
above the project, a personal CLAUDE.local.md, or an old Claude Code would
hide AGENTS.md. prepare-this-repo, the brief and the docs stop describing a
CLAUDE.md abcd writes.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 9, its confirmed criteria A1 to A8, its four scope conditions,
and the ADR's Decision.

## Scope

In:

- abcd's own tree: the CLAUDE.md and GEMINI.md links (both git mode 120000,
  both pointing at AGENTS.md) removed; a canary line in AGENTS.md's first
  section; the brief's one link to the root CLAUDE.md
  (`.abcd/development/brief/02-constraints/01-platform.md`, line 7) re-pointed
  at AGENTS.md; the two repository tests that list the links
  (`internal/core/lint/lintscope_test.go`, `linkScopeExempt`, and
  `internal/core/lint/preflightgates_test.go`, the two surface lists naming
  "AGENTS.md's committed mirror"); the inert-path comment in
  `.github/workflows/ci.yml` (line 140) that names the three routers.
- The write side in `internal/core/ahoy`: the conventions targets setup may
  write (`docsTargetChoices` in `detect.go` stays the readable set; a writable
  set beside it), the setup question's choices (`prompt_help.go`, lines 98 to
  100), the stop on a saved retired choice, and the `--docs-target` flag in
  `internal/surface/cli/cli.go` (line 4060).
- The write side in `internal/core/lifeboat`: `embarkMarker` (`embark.go`, line
  576) follows the target's conventions target instead of the hard-coded
  CLAUDE.md, and `MarkerResult.Target` (`embark_types.go`) names the file it
  chose.
- New in `internal/core/ahoy`: the registry of tools' own conventions files,
  their classification, the owner's-file warning, the retirement offer, the
  presence-only warnings above the project root, and the host-version warning.
- Pages and record: `commands/prepare-this-repo.md` (line 123 scaffolds the
  link today; line 96 names the bridges), `commands/embark.md` (lines 14, 39
  and 54), `docs/how-to/install.md` (lines 305 to 307), the generated
  `docs/reference/cli/commands.md` (line 118, the flag's help), and the brief:
  `02-constraints/01-platform.md` (the rule's platform-constraint line, owed by
  the ADR's Consequences), `04-surfaces/01-ahoy.md` (lines 270 and 641),
  `04-surfaces/03-embark.md` (lines 126 and 194), `04-surfaces/15-prepare-this-repo.md`
  (line 72), `05-internals/02-adapters.md` (line 214) and
  `05-internals/03-configuration.md` (line 38).

Out:

- The read side, which stays so that a project set up through `claude_md` or
  `both` still reads as managed and still uninstalls cleanly: `classify`
  (`internal/core/ahoy/detect.go`, line 151), `Managed`
  (`internal/core/ahoy/managed.go`, line 33), `Uninstall`
  (`internal/core/ahoy/apply.go`, line 1621, which strips both files whatever
  the target), `markerTargets` (`internal/core/ahoy/marker.go`, line 382, still
  mapping both retired values, because `markerFilesDropped` needs them to
  retract a block), the lifeboat packer (`internal/core/lifeboat/sources_native.go`,
  lines 370 and 383), the conventions-router rule
  (`internal/core/repolint/rule_router.go`), the `stray_root_docs` link
  exemption (`internal/core/lint/lint.go`, lines 722 to 760) and the citations
  bridge exemption (`internal/core/lint/citations.go`, line 145). Their
  comments, and those in `rewritelock.go`, `store.go`, `runner/claude.go`,
  `runner/opencode.go` and `surface/cli/build.go`, describe files abcd still
  reads and stay as they are.
- The size of AGENTS.md (37,109 bytes, past one reader's 32,768-byte default):
  iss-2610031012543135. The one coupling is kept here: the canary sits in
  AGENTS.md's first section, so the canary measures the rule and not the cap.
- The managed block's content and where it lands in an adopted file:
  iss-2610020700529281, which stays open; this spec only gives the block one
  home.
- Writing any tool's settings: the host's instructions setting is honoured
  only in the person's own settings, and Gemini CLI's settings naming
  AGENTS.md are the person's to write. abcd writes neither (decision 7).
- The user-level CLAUDE.md in the person's home settings folder, which the
  research finds does not switch native reading off.
- The intents this rule amends, already amended at planning and needing no
  work here: itd-22 (harness portability, decision 7), itd-21 (no lifeboat
  scaffolding, decisions 1 and 7) and itd-10 (purge uninstall, decisions 4
  and 8). The CLAUDE.md clause of the shipped itd-3's criterion is reversed by
  the ADR; no code enforces that clause, so none changes.
- A canary run in CI: CI holds no model credentials and runs no host session,
  so A8 is a host receipt, not a gate.
- The internal file name `internal/core/ahoy/defaults/claude-md-marker-block.md`,
  which names the block's history, not where it is written.

## Approach

### abcd's own switch

The two links go with `git rm`. The brief's "where things live" link in
`02-constraints/01-platform.md` points at `../../../../AGENTS.md`, since the
CLAUDE.md it names stops existing and `links_resolve` would refuse it.
`linkScopeExempt` loses its two link entries (the files are no longer
committed), and the two surface lists in `preflightgates_test.go` lose their
"committed mirror" rows, which would otherwise fail to read a missing file.
The CI comment names AGENTS.md alone; the classifier's behaviour is unchanged,
since none of the three was ever inert.

The canary is one line between AGENTS.md's H1 and the managed block's BEGIN
fence, outside the fence so that setup's drift refresh never rewrites it:

```markdown
Conventions-file check word: <word>. A session that can name it without
reading any file loaded this file as its instructions (itd-2610030814013772).
```

The word is chosen in step 1, one invented word that appears nowhere else in
the tree. A go test holds the line before the BEGIN fence and inside the first
32,768 bytes.

The receipt is taken by a person in a fresh session at the step's branch tip,
in a worktree with no CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md in it or
above it. The first prompt is "Without reading any file or running any tool,
what is the conventions-file check word in your project instructions?". The
receipt, `.abcd/.work.local/logs/agents-md-canary-<yyyy-mm-dd>.md`, records the
date, the commit, the Claude Code version the `--version` flag prints, the
question and answer quoted, the canary word, the line `/memory` shows naming
AGENTS.md, and that the transcript holds no tool call before the answer.

### The write side narrows; the read side stays

`detect.go` keeps `docsTargetChoices` (`claude_md`, `agents_md`, `both`,
`skip`) as the set a saved value is read against, so `detectConfigValues` never
reports a saved retired value as missing. Beside it, `docsTargetWritable`
(`agents_md`, `skip`) is the set setup writes. One core function,
`RetiredDocsTarget(v string) (explanation string, retired bool)`, holds the
explanation every front door shows, so no surface restates it:

> This project's saved setup choice `docs.target` in `.abcd/config.json` is
> `claude_md`. abcd now writes its block into AGENTS.md alone, which Claude
> Code and most other agent tools read directly. Change that one setting with
> `abcd ahoy install --docs-target agents_md` (or `skip` for no block); the
> block already in CLAUDE.md is then taken out of it, and nothing else of
> yours changes.

Where it acts:

- Detection raises one required, non-resolvable gap,
  `config.docs_target_retired`, whose detail is the explanation, so the dry
  run and the doctor name it.
- Install stops before its first write when the saved value is retired and no
  override names a writable one: status `refused`, nothing written, the
  explanation in `Notes`. An override to `agents_md` or `skip` is the one
  setting changed; it runs as today, and the existing `markerFilesDropped`
  retraction takes the block out of CLAUDE.md.
- `--docs-target claude_md` or `both` is refused at the flag with the same
  explanation, and the setup question offers `agents_md` and `skip` alone.
- `stepMarker` writes only through `docsTargetWritable`.

`classify`, `Managed`, `Uninstall` and `markerTargets` do not change.

### Embark follows the conventions target

`embarkMarker` reads the target's `.abcd/config.json` `docs.target` with the
reader setup uses, and chooses its file:

- `agents_md`: AGENTS.md;
- `skip`: no file, action `skip`, note "this project's setup chose no
  conventions file for abcd's block";
- a retired value: no file, action `skip`, the note `RetiredDocsTarget` gives;
  the record families still land, as a skipped marker never stops them today;
- none chosen (no settings file, or no `docs.target`): AGENTS.md (open
  question 2).

`MarkerResult.Target` carries the chosen file name, or is empty with the
action `skip`. Probe and from share the one path, as they do now, so the probe
cannot mispredict. `EnsureMarker` keeps refusing to write through a link.

### Tools' own conventions files in an adopted project

One registry, `toolConventionsFiles` in `internal/core/ahoy/conventions.go`,
names each file a tool reads in place of AGENTS.md, with the tool it serves,
from the [2026-10-03 note](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md)
(open question 4):

| File | Tool that then does not read AGENTS.md |
| --- | --- |
| `CLAUDE.md`, `.claude/CLAUDE.md` | Claude Code |
| `GEMINI.md` | Gemini CLI, at its default settings |
| `.rules`, `.cursorrules`, `.github/copilot-instructions.md` | Zed, which reads the first match in a fixed order |

Each file found at the project root is classified with one guarded read
(`fsutil.ReadGuardedInRoot`, `maxAhoyFileBytes`), never followed through a link:

- **Repeats AGENTS.md** (open question 3): a link whose target resolves to the
  root AGENTS.md, or, while there is no root AGENTS.md, a link whose target
  text names it in the plain spelling; a regular file holding only the text `AGENTS.md`, which is
  such a link checked out where links are not supported; a byte-for-byte copy
  of AGENTS.md; a file whose only non-blank line is `@AGENTS.md`; or a file
  that is blank once abcd's own block is stripped (`StripMarkerBlock`), which
  is what retracting a retired target leaves behind.
- **The owner's words**: anything else, including a file that cannot be read,
  is too large, or is not a regular file. Nothing more is read from it.

The owner's file is never edited, moved, merged or removed (decision 6). It
raises a non-resolvable warning gap, `conventions.owner_file`, which install
reports in a new `Warnings` list on `InstallResult`, printed first by the CLI
before the headline and carried as `warnings` in the JSON:

> CLAUDE.md holds your own words, so Claude Code reads it and not AGENTS.md:
> abcd's rules stay hidden from Claude Code in this project until you move
> those words into AGENTS.md and remove CLAUDE.md. abcd never edits or
> removes this file.

A file that repeats AGENTS.md raises an optional, resolvable gap,
`conventions.retire_offered`, in a new category `conventions-file`, and the
offer follows the drain rule's precedent (`drain_rule.go`): asked only of a
person at a terminal (`atTerminal`), never under `--yes` and never off a
terminal, and listed in `optional_skipped` when not asked; the gap id joins
`optionalGapIDs` (`apply.go`, line 1789). `stepConventionsFiles` runs after
`stepDrainRule`, the last consent question, and so after `stepMarker`, whose
retraction may have just blanked a CLAUDE.md. Each file is one question, on
the `Prompter.Prompt` seam with three answers, `retire`, `keep` and `later`,
`later` the default. On `retire` the file is classified again at that moment
and removed only if it still repeats AGENTS.md; a file that changed while the
question was open is left and named. `keep` and `later` write nothing and
record nothing, so the next setup asks again, as the drain offer does. The
removal is from the working tree only; the person commits it.

The question is a fixed setup question, so it is held to the asking rules the
two question specs set
([spc-2610030944505997](spc-2610030944505997-asking-and-layout.md) and
[spc-2610030911534855](spc-2610030911534855-a-person-can-run-abcd-s-interviews-in-a-plain-terminal.md))
once they land: the file and what it repeats quoted first, the question last,
and the decide-later answer last.

### The host-reach warnings: presence only

`detectHostReach` raises warning gaps, never a refusal (decision 3):

- **A personal file at the root.** CLAUDE.local.md at the project root is
  named. It is the person's own, usually kept out of git, so it is never
  classified, read or offered for retirement.
- **A file above the project.** From the root's parent up to the file-system
  root, each folder is checked with `os.Lstat` alone for `CLAUDE.md`,
  `.claude/CLAUDE.md` and `CLAUDE.local.md`. In the person's home folder the
  user-level `.claude/CLAUDE.md` is skipped, since it does not switch native
  reading off; a `CLAUDE.md` directly in the home folder is still named. A
  folder that cannot be searched ends the walk quietly. Each path is shown
  through `fsutil.RedactHome`.

Each warning names the path, says Claude Code then reads that file and not
AGENTS.md, and carries one fixed sentence explaining why this check differs
from the rule that abcd reads no `.abcd/` above the working tree:

> abcd reads no settings from folders above this project. This check only asks
> whether a file of this name exists there; it reads nothing in it and changes
> nothing abcd does, because the agent tool itself reads that folder.

The difference, for the record: the `.abcd/` rule bounds where abcd takes
configuration that changes its own behaviour, so a hostile file above the
tree could steer it; a presence check takes nothing, and the most a hostile
file can cause is one warning.

The version warning (open question 5) asks the `claude` command on `PATH` for
its version with a short timeout, parses the first `major.minor.patch` it
prints, and compares it with one floor constant, v2.1.281, the release from
which every session type the research names reads AGENTS.md. Below the floor
it warns that this Claude Code is older than the release that reads AGENTS.md
on its own and loads none of abcd's rules there until it is updated. The text
names no version number, the host's or the floor's. No command, no answer, or
no parsable version raises nothing. The floor is the one place a vendor's
version lives, with its source cited beside it.

The two warnings that read the folder tree and the host are install checks
only; `Managed`, which the status line calls on every refresh, does neither.

### prepare-this-repo, the brief and the docs

`commands/prepare-this-repo.md` step 3 creates AGENTS.md when absent and
scaffolds no other conventions file of any tool, link or copy; its Phase 2
audit names a found tool's file as setup does (owner's words, or repeats
AGENTS.md) and points at setup's offer rather than acting on it. The embark
page and brief say the block goes into the target's chosen conventions file,
AGENTS.md, and the brief's "Never `AGENTS.md`" sentence goes. The ahoy brief's
inventory tree shows AGENTS.md as the block's home, and its target list and
the configuration chapter show `claude_md` and `both` as read for detection
and uninstall and refused at setup with the explanation. The platform chapter
gains the rule's constraint line: AGENTS.md is the one conventions file abcd
writes, in its own project and every project it sets up. The adapters table's
`claude_md` reader row names AGENTS.md beside CLAUDE.md, since the lifeboat
packer reads both. The install guide
says the managed block goes into AGENTS.md only, names the warnings in
host-agnostic words with no version string, and names no tool's file beyond
CLAUDE.md, so the docs-lint harness rules pass without an allow escape.

## How each acceptance criterion is met

**A1.** Two halves. `TestInstallWritesNoToolConventionsFile` runs install in an
empty repository at `agents_md` and at the `skip` default, and asserts AGENTS.md
is written at `agents_md` and that no file in `toolConventionsFiles`, nor
CLAUDE.local.md, exists after either. `TestPrepareThisRepoScaffoldsNoToolConventionsFile`
(beside the onboarding test in `internal/core/ahoy/onboarding_test.go`) reads
the page and fails on any instruction creating, linking or copying a registry
file. Open question 1 records how the `skip` default reads against the
criterion's example.

**A2.** `TestEmbarkPlantsInAgentsMD` embarks a lifeboat into a target holding
only AGENTS.md, and asserts the block lands in AGENTS.md, `MarkerResult.Target`
reads `AGENTS.md`, no CLAUDE.md exists, and the target's file list is
otherwise unchanged; the probe predicts the same.
`TestEmbarkFollowsTheChosenTarget` covers `skip` and a retired value.

**A3.** `TestOwnersToolFileIsUntouchedAndNamed` writes a CLAUDE.md reading
"Always run make check first", runs install with `--yes` and interactively,
and asserts the file is byte-identical with the same mode, and that
`Warnings` holds exactly one line naming CLAUDE.md, saying AGENTS.md stays
hidden, and saying how to end it.

**A4.** `TestRepeatingToolFileIsOfferedForRetirement` runs a GEMINI.md link to
AGENTS.md and a CLAUDE.md copy through each answer: `retire` removes the file;
`keep` and `later` leave it, the link still a link; `--yes` and a piped run do
not ask and list the gap in `optional_skipped`. `TestRetireRechecksTheFile`
changes the file while the question is open and asserts it is left and named.
`TestToolFileClassification` holds each class.

**A5.** `TestHostReachWarningsArePresenceOnly` builds a parent folder holding
CLAUDE.md, a project root holding CLAUDE.local.md, and a fake home folder whose
`.claude/CLAUDE.md` exists, and asserts: one warning per file naming its path;
the user-level file not named; install not refused; an unreadable file (mode
000) named all the same, so no read is needed; each warning carrying the fixed
sentence on the `.abcd/` rule; no warning text matching a version pattern.

**A6.** `TestSavedRetiredTargetStopsSetup` saves `claude_md`, then `both`, and
asserts install is refused with nothing written and the note naming
`docs.target` and the command. `TestRetiredTargetStillReadsAsManaged` asserts
`classify` and `Managed` report a block in CLAUDE.md as managed.
`TestUninstallStripsARetiredTargetsBlocks` saves `both` and asserts uninstall
strips the block from both files. `TestChangingTheOneSettingMovesTheBlock`
runs `--docs-target agents_md` over a saved `claude_md` and asserts the block
leaves CLAUDE.md and lands in AGENTS.md. The flag's refusal is
`TestDocsTargetFlagRefusesRetiredValues` in `internal/surface/cli`.

**A7.** `TestRepositoryKeepsOneConventionsFile`, in `internal/core/lint` beside
the other tests over the repository's own tree, asserts that neither
`git ls-files` nor `os.Lstat` finds CLAUDE.md or GEMINI.md at the root.

**A8.** The dated receipt above, from a fresh session in abcd's own checkout.
`TestAgentsMDCanarySitsInTheFirstSection` holds the canary line before the
BEGIN fence and inside the first 32,768 bytes, so the receipt can be taken
again at any later host version.

## Open design questions

These were for the technical facilitator. Each was decided on 2026-10-03
without a question, under the person's ruling that an obvious answer is
decided rather than asked; the reason is given beneath each.

1. **A1 and the `skip` default.** A default setup writes no conventions file
   (iss-2609110944498549), so "setup writes AGENTS.md" holds where setup writes
   one. (a) The default stays; A1 is met by install at `agents_md` and by
   prepare-this-repo, which always writes AGENTS.md, and its test asserts no
   tool's file at either target (designed to). (b) The default becomes
   `agents_md`, so a bare install writes AGENTS.md: it meets the example
   literally, but reverses the ruling that a default install names abcd in no
   committed conventions file, which is the product thinker's to reopen.
   - Decided: (a). It keeps the product thinker's standing ruling on the
     default; nothing in this intent's decisions asks to reopen it.
2. **Embark with no chosen target.** (a) Plant into AGENTS.md (designed to):
   embark plants a block today whatever the target holds, and A2's target,
   holding only AGENTS.md, has no settings file. (b) Follow the `skip` default
   as setup does: consistent with setup, but an unconfigured target receives
   no block and A2 holds only for a target that chose `agents_md`.
   - Decided: (a). It is today's behaviour pointed at the one file, and A2
     is written for exactly that target.
3. **What counts as repeating AGENTS.md.** A4 names a link and a byte-for-byte
   copy. (a) Also the link's checked-out text, a lone `@AGENTS.md` line, and a
   file blank once abcd's block is stripped (designed to): none holds a word
   of the owner's, and the last is what retracting a retired target leaves, a
   file that still hides AGENTS.md. A lone `@AGENTS.md` would otherwise draw a
   "hidden" warning that is untrue, since that file loads AGENTS.md. (b) Only
   the two A4 names: narrower, and the warning misreports those three cases.
   (a) widens the class A4 names, so the product thinker confirms it.
   - Decided: (a), without a question. The press release the product
     thinker confirmed (decision 9) says a file that "only repeats or links
     to AGENTS.md" is offered for removal: a lone `@AGENTS.md` line links, and
     a file left blank once abcd's block is stripped only repeats. A4's
     confirmed wording is unchanged; its two named cases are read as examples
     of that class, and its test covers all five.
4. **Which tools' files the registry names.** (a) The files the research names
   as read in place of AGENTS.md, each with its tool, as in the table
   (designed to): decision 7 reaches every tool's own file, and the warning
   says which tool loses AGENTS.md. (b) CLAUDE.md, `.claude/CLAUDE.md` and
   GEMINI.md alone: the criteria's files, but decision 7 then holds for two
   tools. (c) Every known tool file, folders included: a rules folder is
   never a copy of one file and Cursor reads AGENTS.md beside it, so it would
   only ever draw a warning that is untrue.
   - Decided: (a). It is what decision 7 reaches, and it names the tool
     that loses AGENTS.md.
5. **How the version warning knows the version.** (a) Ask the `claude` command
   for its version and compare with one floor constant (designed to): it
   names the old host where one is installed, at the cost of one subprocess
   during setup and one vendor version in the source. (b) No probe, a
   standing line in every setup that plants into AGENTS.md: no coupling, but
   noise on every run and nothing named. Neither sees an IDE-only host or the
   first session after an upgrade.
   - Decided: (a). It names the old host where one is installed; the
     reading is shared with itd-2610031026190632 (the harness-version draft),
     whose spec takes it over when that draft is planned.
6. **Whether the canary stays.** (a) Committed for good in the first section
   (designed to): the receipt can be taken again whenever the host changes,
   and the test holds its place against the size work in
   iss-2610031012543135. (b) Planted only for the run: no extra line, but the
   run then mutates a live AGENTS.md or needs a scratch copy, which sits
   inside the checkout and is not "abcd's own checkout".
   - Decided: (a). A8's receipt can then be retaken whenever the host
     changes, and the canary's place guards against the size work.

## Footprint

- packages: internal/core/ahoy, internal/core/lifeboat, internal/surface/cli, internal/core/lint, commands/, docs/how-to, docs/reference, .abcd/development/brief, .github/workflows, AGENTS.md, CLAUDE.md, GEMINI.md
- tests: TestRepositoryKeepsOneConventionsFile, TestAgentsMDCanarySitsInTheFirstSection, TestInstallWritesNoToolConventionsFile, TestSavedRetiredTargetStopsSetup, TestRetiredTargetStillReadsAsManaged, TestUninstallStripsARetiredTargetsBlocks, TestChangingTheOneSettingMovesTheBlock, TestDocsTargetFlagRefusesRetiredValues, TestEmbarkPlantsInAgentsMD, TestEmbarkFollowsTheChosenTarget, TestOwnersToolFileIsUntouchedAndNamed, TestRepeatingToolFileIsOfferedForRetirement, TestRetireRechecksTheFile, TestToolFileClassification, TestHostReachWarningsArePresenceOnly, TestHostVersionWarning, TestPrepareThisRepoScaffoldsNoToolConventionsFile; the dated canary receipt in the local tier; the command reference's drift test after regeneration; docs-lint and record-lint clean

## Steps

1. abcd's own switch: the two links go, and a fresh session loads AGENTS.md
   - criteria: A7, A8; lands first, as iss-2609291925136841's fix, with a `Resolves: iss-2609291925136841` trailer and the issue moved to resolved/ in the same change (decision 5)
   - packages: CLAUDE.md, GEMINI.md, AGENTS.md, .abcd/development/brief/02-constraints, internal/core/lint (tests only), .github/workflows, .abcd/work/issues
   - tests: TestRepositoryKeepsOneConventionsFile and TestAgentsMDCanarySitsInTheFirstSection, each watched fail first; TestEveryCommittedMarkdownFileHasItsLinksChecked and the two preflight surface tests pass with the link rows gone; record-lint clean with the brief's link re-pointed; the dated receipt taken at the branch tip before the pull request
   - landed: #786
2. Setup writes AGENTS.md alone, and a saved retired choice stops setup
   - criteria: A6, and the install half of A1
   - packages: internal/core/ahoy, internal/surface/cli, docs/how-to, docs/reference, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals
   - tests: TestInstallWritesNoToolConventionsFile, TestSavedRetiredTargetStopsSetup, TestRetiredTargetStillReadsAsManaged, TestUninstallStripsARetiredTargetsBlocks, TestChangingTheOneSettingMovesTheBlock, TestDocsTargetFlagRefusesRetiredValues; the 29 test files that build CLAUDE.md fixtures pass, those that install at `claude_md` or `both` moved to `agents_md` or kept as read-side cases; the command reference regenerated with `go generate ./internal/surface/cli`
   - landed: #792
3. Embark follows the conventions target
   - criteria: A2
   - packages: internal/core/lifeboat, internal/surface/cli, commands/embark.md, .abcd/development/brief/04-surfaces
   - tests: TestEmbarkPlantsInAgentsMD, TestEmbarkFollowsTheChosenTarget; the existing embark tests over a symlinked or unwritable target moved to AGENTS.md and still skipping; the surface's embark tests read the chosen target
   - landed: #794
4. Tools' own files: the owner's-file warning and the retirement offer
   - criteria: A3, A4
   - packages: internal/core/ahoy, internal/surface/cli, docs/how-to, .abcd/development/brief/04-surfaces
   - tests: TestOwnersToolFileIsUntouchedAndNamed, TestRepeatingToolFileIsOfferedForRetirement, TestRetireRechecksTheFile, TestToolFileClassification; the existing piped-install tests unchanged in their answer order, since the offer is asked only at a terminal
   - lands after step 2; it adds `Warnings` to `InstallResult`, and its question meets the asking rules of the two question specs where those have landed
   - landed: #800
5. The host-reach warnings: presence above the root, the personal file, and the version
   - criteria: A5
   - packages: internal/core/ahoy, docs/how-to, .abcd/development/brief/04-surfaces
   - tests: TestHostReachWarningsArePresenceOnly, TestHostVersionWarning (a fake `claude` on `PATH` below the floor, at it, absent, and printing nothing parsable); `Managed` unchanged in cost, asserted by its existing tests
   - lands after step 4, whose `Warnings` list it reuses
   - landed: #803
6. prepare-this-repo, the remaining brief, and the close
   - criteria: the page half of A1; the intent's close
   - packages: commands/prepare-this-repo.md, internal/core/ahoy (test only), .abcd/development/brief/02-constraints, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals
   - tests: TestPrepareThisRepoScaffoldsNoToolConventionsFile; docs-lint and record-lint clean; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031156364295` (the intent already declares `impact: breaking`) with a `Delivers: itd-2610030814013772` trailer
   - landed: feat/agentsmd-close
