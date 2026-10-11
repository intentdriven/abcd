---
id: spc-2610031342374947
slug: when-a-person-runs-abcd-s-check-on
intent: itd-2610031026190632
origin: researcher-authored
production_mode: hand-written
---
# when-a-person-runs-abcd-s-check-on-purpose-abcd-also-says

## Summary

This spec delivers
[itd-2610031026190632](../../intents/planned/itd-2610031026190632-when-a-person-runs-abcd-s-check-on.md)
(abcd's update check says when a harness is older than abcd needs). The file
name and slug are the ones minted on filing; the intent's title is the one
that describes the work.

When a person runs `abcd update --check`, the report gains one line for each
agent harness abcd knows of that is installed on this computer, and a
`harnesses` field in the JSON beside the unchanged `check` object. A harness
below the version abcd needs gets a warning naming the version installed, the
version needed, and the harness's own update command. A harness at or above
it gets no warning. A harness abcd cannot read a version from is named as not
checked, with the reason. The line is read from this computer alone: it never
reaches the network, even within the one command that does.

Nothing about the reading is new. The installed-version reading and the floor
are owned by
[spc-2610031156364295](../closed/spc-2610031156364295-evaluate-whether-claude-md-can-be.md)
step 5, which ships first; this spec widens that one reading from one harness
to a table of them, gives each outcome a reason, and puts the result in the
update check.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 9, its confirmed criteria H1 to H7, and its four scope
conditions.

## Scope

In:

- `internal/core/ahoy`: the harness table (each harness abcd knows of, its
  command, whether abcd has an adapter for it, its floor with the floor's
  source cited beside it, and its own update command), the reading step 5
  writes widened to take any table entry's command and to say why a version
  was not read, and one function that checks every entry against its floor.
- `internal/core/runner`: the runner's command admission (`launcher.admit`,
  `proc.go`, line 79) and its shipped route names (`runnerNames`,
  `runner.go`, line 49) exported for the table to use and to be checked
  against (open question 2).
- `internal/surface/cli/version.go`: a `Harnesses` field on `versionOutput`
  (lines 36 to 48), a sibling of `Check` (line 47) and filled only when
  `runVersion` runs with `check` set (lines 94 to 96), and one text line per
  harness after the `next:` line (lines 109 to 117).
- `internal/surface/cli/update.go`: the verb's long help (lines 47 to 49) and
  the `--check` flag's help (line 109) widened from abcd alone to abcd and the
  agent harnesses on this computer; the regenerated
  `docs/reference/cli/commands.md` that carries both.
- Pages and record: the plugin page `commands/update.md` ("Only asking",
  lines 27 to 37, which tells the agent which fields to relay);
  `docs/how-to/install.md` (lines 296 to 299, the paragraph on the check);
  the brief's `04-surfaces/21-update.md` (lines 23 to 28, the check) and
  `04-surfaces/12-version.md` (line 45, the report the check adds to).
- H7's dated receipt, in the local tier.

Builds on and reuses:

- spc-2610031156364295 step 5: the installed-version reading (the command on
  the search path, one subprocess, a short timeout, the first
  `major.minor.patch` it prints) and the one floor it defines. Reused, never
  restated: this spec changes that code in place, and step 5's own test
  (`TestHostVersionWarning`) stays as it is. <!-- record-lint: forward-looking -->
- itd-111 and adr-38: the update check is the one explicit check; the harness
  line adds no fetch to it, so adr-38's tiers and brief invariant 7 are
  untouched (decision 7).
- The update check's existing release-fetcher seam (`newReleaseFetcher`,
  `version.go`, line 22) and its recording stand-in (`recordingFetcher`,
  `version_check_test.go`, line 16), for H1.

Out:

- Any lookup of a harness's newest release, and any mention of a release above
  the floor (decisions 4 and 7).
- The running session's own copy of a harness, and a newer version already
  downloaded but not yet running: itd-2610031325050110 (decisions 6 and 8).
- `abcd --version`, `abcd ahoy`, the session start and the status line: none
  of them reads a harness's version for this line (decision 3).
  `TestOnlyUpdateCheckTouchesTheNetwork` (`version_check_test.go`, line 28) is
  unchanged.
- Setup's own warning for a harness below the floor: step 5's, kept as it is;
  the cost of a second place stating it was accepted in decision 7.
- Whether a harness's own settings hold its updates (open question 3).
- Hosts that run only inside an editor, with no command on the search path
  (scope condition 4).

## Approach

### The harness table, beside the reading and the floor

Step 5 writes one reading for one harness and one floor constant beside it.
This spec turns the floor constant into the first row of one table in the
same file, `harnessFloors`, each row holding:

- `Name`: the harness's route name as the runner spells it (`claude`,
  `opencode`), or a plain name for a harness abcd has no adapter for;
- `Command`: the command looked for on the search path;
- `Adapter`: whether abcd has an adapter for it;
- `Floor` and `FloorSource`: the version abcd needs and where that version
  was established, cited beside it, or empty when abcd needs no particular
  version (scope condition 3);
- `UpdateStep`: the harness's own update command (open question 3).

Today the table has three rows. The two adapters: `claude`, whose floor is
step 5's, and `opencode`, which has none. One harness with no adapter,
`gemini`, the terminal harness spc-2610031156364295's conventions-file
registry already names (open question 1). A harness the table does not name
is never looked for.

`TestHarnessTableCoversEveryAdapter` asserts the table's adapter rows are
exactly the runner's shipped route names, so a third adapter added to the
runner without a row fails a test rather than going unchecked.

### One reading, with a reason for every outcome

Step 5's reading takes the command name from the table row instead of a
literal, and returns a result rather than a silent nothing:

- the version read, the first `major.minor.patch` the command printed; or
- the reason none was read: not on the search path; refused before it was run
  (open question 2); no answer within the timeout, or a non-zero exit; no
  version printed.

Step 5's setup warning keeps its behaviour by treating every reason as
"raise nothing", as it does today. A row with no adapter is never run: it is
looked up on the search path and nothing more, since abcd does not know what
its version command prints.

`CheckHarnessFloors()` walks the table in order and returns one
`HarnessFloor` per row that applies:

| Row | Verdict | Fields set |
| --- | --- | --- |
| adapter, version read, below the floor | `below_floor` | installed, needed, update step |
| adapter, version read, at or above the floor | `at_or_above_floor` | installed, needed |
| adapter, version read, no floor in the table | `no_floor` | installed |
| adapter, no version read | `not_checked` | reason |
| no adapter, command on the search path | `not_checked` | reason: abcd has no adapter for it |
| no adapter, command absent | not listed | |

Versions are compared part by part as numbers, never as strings, so 1.10.0
sits above 1.9.2. No verdict reads as current: `at_or_above_floor` and
`no_floor` say what was compared and nothing more (brief invariant 16). A
row with no adapter is listed only when its command is present, since
naming every absent harness abcd has no adapter for would only add noise.

`CheckHarnessFloors` takes no fetcher and no network client, and nothing it
calls does: the reading is a subprocess on this computer, and the floor is in
the binary.

### The update check carries the lines

`versionOutput` gains

```go
Harnesses []harnessLine `json:"harnesses,omitempty"`
```

beside `Check`, never inside it (criterion H5), each `harnessLine` carrying
`name`, `verdict`, and, where they apply, `installed`, `needed`, `reason` and
`update_step`. `runVersion` fills it only when `check` is set, after
`runReleaseCheck`, and independently of it: a failed or refused release
fetch leaves the harness lines exactly as they would otherwise be.

The text render prints one line per harness after the `next:` line, in the
report's existing column, every value passed through `termsafe.Sanitize`
since a harness's own output is untrusted:

```text
  harness:   claude 1.4.0 is older than the 1.5.0 abcd needs; update it with `claude update`
  harness:   opencode 1.18.31 (abcd needs no particular version of it)
  harness:   gemini not checked: abcd has no adapter for it
```

A harness at or above its floor prints one line saying so, with no warning
word:

```text
  harness:   claude 1.9.2 meets the 1.5.0 abcd needs
```

The numbers in these samples are made up, as the intent's criteria are.

### The help, the page, the docs and the brief

The verb's long help and the `--check` flag's help say the check also reads
the version of each agent harness installed on this computer, offline, and
warns only below the version abcd needs. The plugin page tells the agent to
relay each `harnesses` entry's `verdict` and, where present, `update_step`
verbatim. The how-to paragraph and the two brief chapters say the same in
host-agnostic words: "the agent harness" and "the harness's own update
command", with no harness's name and no version string outside fenced sample
output, so the `harness/*` docs-lint rules pass without a new allow escape.

## How each acceptance criterion is met

**H1.** `TestHarnessLinesNeverGoOnline`, in `internal/surface/cli` beside
`TestOnlyUpdateCheckTouchesTheNetwork`, puts a fake harness below its floor
on the search path and runs `update --check --json` twice: once with the
recording stand-in behind `newReleaseFetcher`, asserting exactly one
`LatestTag` call, and once with a stand-in that fails every call, asserting
the `harnesses` field is byte-identical between the two runs and still carries
the `below_floor` warning.

**H2.** `TestUpdateCheckWarnsBelowTheFloor`: a fake `claude` printing
`1.4.0` against a test floor of `1.5.0` gives exactly one `harness:` text line
naming both numbers and `claude update`, and a JSON entry with `verdict`
`below_floor`, `installed` `1.4.0`, `needed` `1.5.0` and `update_step`
`claude update`. The table's floor is substituted through a package seam, so
the test never depends on the real floor.

**H3.** `TestUpdateCheckQuietAtOrAboveTheFloor`: fakes printing `1.5.0` and
`1.9.2` against a floor of `1.5.0` give no warning line and two JSON entries
with `verdict` `at_or_above_floor`. In the core, `TestHarnessFloorVerdicts`
holds every row of the verdict table above, and
`TestHarnessVersionsCompareAsNumbers` holds `1.10.0` above `1.9.2`.

**H4.** `TestUpdateCheckNamesUncheckedHarnesses`: a fake `gemini` present (no
adapter), no `opencode` on the search path (adapter, command absent), and a
fake `claude` printing `unknown` (no version printed). Each reads
`not_checked` with its own reason, in the text and the JSON, and none reads
`at_or_above_floor`. `TestHarnessNotCheckedReasons`, in the core, holds every
reason the reading can give, the timeout and the refused command included.

**H5.** `TestUpdateCheckObjectIsUnchanged` compares the `check` object, taken
from the JSON as raw bytes, with
`internal/surface/cli/testdata/update-check-object.json`, captured from
today's code with the recording stand-in in the step's first commit, before
any other change. The comparison runs with harnesses present and with none,
and asserts `harnesses` is a sibling key of `check` at the top level.

**H6.** `go run ./cmd/abcd lint docs` (preflight's docs-lint gate) is clean
over the regenerated command reference and the how-to page, and
`TestUpdateCheckDocsAddNoAllowEscape` asserts the count of
`docs-lint: allow` markers in `docs/how-to/install.md` and
`docs/reference/cli/commands.md` equals the count at the step's base,
written into the test.

**H7.** A dated receipt,
`.abcd/.work.local/logs/harness-floor-check-<yyyy-mm-dd>.md`, taken on the
product thinker's machine with `go run ./cmd/abcd update --check` from the
source checkout. It records the date, the commit, each harness line quoted,
and, for each adapter's command, what that command's own version flag prints
the same day, and states that each line agrees with that output and with the
table's floor.

## Open design questions

These are for the technical facilitator; the records do not settle them.
Each is designed to the marked option, and the reason is given beneath it.

1. **Where a harness with no adapter comes from (H4's first case).**
   (a) A short list of harnesses abcd knows by name but has no adapter for,
   each named only when its command is on the search path and never run,
   seeded with `gemini`, the one terminal harness the records already name
   (designed to): the case H4 names occurs on a real machine, at the cost of
   a list to keep. (b) No such list: abcd names nothing it has no adapter
   for, and H4's first case is only a test fixture, so decision 5's "named
   as not checked" never happens for a harness without an adapter. (c) The
   harness this session runs in, read from its environment: names the
   harness actually in use, but each host's environment marker is its own
   undocumented contract, and a terminal verb may run in none.
   - (a): it is the only option under which the criterion describes
     something a person can see, and it reads nothing abcd does not own.
   - Decided: (a), on 2026-10-03 without a question: naming a known harness only when its command is present answers H4's first case without running anything.
2. **Which command is run.** The reading runs a program found on the search
   path. (a) Through the runner's admission, exported as `runner.Admit`: a
   command that resolves to a relative path, sits inside the checkout the
   check runs from, or is writable by group or others is not run and reads
   as `not_checked` with that reason (designed to): the same rule that
   already decides which harness binary abcd runs a role through, applied to
   the one reading, step 5's setup warning included. (b) A plain search-path
   lookup, as step 5 is written: fewer moving parts, but `abcd update
   --check` run inside a cloned repository could start a program that
   repository placed on the search path.
   - (a): one rule for running a harness's binary, and the check never runs
     repository content. `internal/core/runner` imports no `ahoy` code, so
     the import adds no cycle.
   - Decided: (a): one admission rule for the one shared reading.
3. **The harness's own update step.** (a) One fixed command per adapter, the
   one the harness documents for updating itself (`claude update`,
   `opencode upgrade`) (designed to): no settings read and no install layout
   read, at the cost that an install another tool manages is pointed at a
   command that may hand off to that tool. (b) One per install shape, read
   from where the command resolves (the design review's finding 4): the
   right command per shape, but each shape is the vendor's undocumented
   layout and moves between releases. (c) (a), plus "held" when the
   harness's own settings stop its updates (the intent's proposal): it
   avoids naming a command the person turned off, but reads settings keys
   abcd has not confirmed against the vendor's documentation, and for the
   second adapter a settings file abcd reads nowhere today.
   - (a): it names the harness's own way to update, which is what the
     press release promises, and adds no new read of another tool's
     configuration. The "held" half of the intent's proposal is not built;
     a person who has turned updates off sees the command and chooses.
   - Decided: (a): one fixed update command per adapter, each confirmed in its tool's help.
4. **A harness abcd needs no particular version of.** The intent proposes
   naming it as having no floor to check against, never as current.
   (a) A `no_floor` verdict, printed as "abcd needs no particular version of
   it" (designed to): the reading is shown and nothing is claimed beyond it.
   (b) Leave it out of the report: quieter, but a harness abcd ran a version
   command on would then be invisible.
   - (a): it is the intent's proposal, and brief invariant 16 requires the
     report to say what it compared.
   - Decided: (a): no_floor never reads as current (invariant 16).

## Footprint

- packages: internal/core/ahoy, internal/core/runner, internal/surface/cli, commands/update.md, docs/how-to, docs/reference, .abcd/development/brief/04-surfaces
- tests: TestHarnessTableCoversEveryAdapter, TestHarnessFloorVerdicts, TestHarnessVersionsCompareAsNumbers, TestHarnessNotCheckedReasons, TestHarnessCommandAdmission, TestHostVersionWarning (unchanged), TestHarnessLinesNeverGoOnline, TestUpdateCheckWarnsBelowTheFloor, TestUpdateCheckQuietAtOrAboveTheFloor, TestUpdateCheckNamesUncheckedHarnesses, TestUpdateCheckObjectIsUnchanged, TestOnlyUpdateCheckTouchesTheNetwork (unchanged), TestUpdateCheckDocsAddNoAllowEscape; the command reference's drift test after regeneration; docs-lint and record-lint clean; the dated receipt in the local tier

## Steps

1. The harness table and one reading with a reason for every outcome
   - criteria: the core halves of H2, H3 and H4
   - packages: internal/core/ahoy, internal/core/runner
   - tests: TestHarnessTableCoversEveryAdapter, TestHarnessFloorVerdicts, TestHarnessVersionsCompareAsNumbers, TestHarnessNotCheckedReasons, TestHarnessCommandAdmission, each watched fail first; TestHostVersionWarning and the runner's adapter tests pass unchanged
   - waits on spc-2610031156364295 step 5, which writes the reading and the floor this step widens <!-- record-lint: forward-looking -->
2. The update check carries the harness lines
   - criteria: H1, H2, H3, H4, H5
   - packages: internal/surface/cli, docs/reference, commands/update.md
   - tests: TestUpdateCheckObjectIsUnchanged, its testdata captured from the base in the step's first commit; TestHarnessLinesNeverGoOnline, TestUpdateCheckWarnsBelowTheFloor, TestUpdateCheckQuietAtOrAboveTheFloor, TestUpdateCheckNamesUncheckedHarnesses, each watched fail first; TestOnlyUpdateCheckTouchesTheNetwork unchanged; the command reference regenerated with `go generate ./internal/surface/cli`
   - lands after step 1
3. The how-to, the brief, the receipt and the close
   - criteria: H6, H7; the intent's close
   - packages: docs/how-to, .abcd/development/brief/04-surfaces
   - tests: TestUpdateCheckDocsAddNoAllowEscape; docs-lint and record-lint clean; the dated receipt taken at the step's branch tip before the pull request; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031342374947` (the intent already declares `impact: additive`) with a `Delivers: itd-2610031026190632` trailer
   - lands after step 2
