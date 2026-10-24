---
id: spc-2610241007434505
slug: a-person-sets-abcd-up-in-a-repository-in
intent: itd-2610240823055889
origin: researcher-authored
production_mode: dictated-and-formatted
---
# a-person-sets-abcd-up-in-a-repository-in

## Summary

spc-2610241007434505 delivers the setup flow of itd-2610240823055889: `abcd
ahoy install` checks the basics before anything else, shows one plan screen
and asks once, asks each step only a person can approve as its own one-line
question, puts abcd on the `PATH` itself before the status line, and ends on a
three-part report, or on one line where nothing is left to do. Three parts of
the intent are their own records, built on sibling branches and merged with
this one: per-machine commit identity (itd-2610240918547728), verified tool
installs for the person alone (itd-2610240918594138), and the shell-profile
`PATH` line (itd-2610240919030303). This spec leaves each of them a seam
(below) rather than building it. Brief invariant 1 is amended for setup by
adr-2610241101463288.

## Scope

- **The basics** (`internal/core/ahoy/setup_basics.go`): no `git`, a folder
  that is not the top of a repository, or no agent harness ends the run before
  its first question with one message, status `refused`, exit 2.
- **The plan screen** (`setup_plan.go`): `SetupPlan` (the changes grouped by
  where they land, the defaults with where to change each, the person-only
  questions), `SetupPlanQuestion` (id `approve.plan`), `PlanPrompter`, and the
  `planDefaults` prompter that answers every approval yes and, at a terminal
  under proceed, takes the listed defaults, recording each through
  `DefaultRecorder` so a replay of the record answers the same.
- **The person-only questions** (`person_only.go`): the identity gaps leave the
  approvals (`personOnlyGap`), the plan lists the questions that follow
  (`personOnlyAsks`), and the status-line offer leaves the approvals where it
  cannot be wired (`approvalGaps`).
- **The abcd download** (`abcd_download.go`, `internal/core/update/fetch.go`):
  `stepDownload` before `stepSymlink`; `update.FetchVerified` checks the
  release's `checksums.txt` and its signed build provenance before returning
  bytes; the CLI wires it as `InstallOptions.DownloadAbcd`.
- **One line each** (`setup_brief.go`, `internal/surface/cli/ask`):
  `BriefQuestion` and `ExplainHint`; `ask.Terminal.Explain`, `ask.ErrExplain`
  and the picker's `Explain` outcome; the drawn door draws the brief form and,
  on `?`, the whole question.
- **The report** (`setup_report.go`): `SetupReport` (done, needs you, next,
  one line) on `InstallResult.Report`; the CLI prints it, and `--details`
  adds the full record.
- **The source checkout**: `vintage.GoRunModule` and `goRunOfCheckout` treat a
  `go run` build of the checkout being installed as current
  (iss-2610240824047504); the download's release there is the newest `v*` tag
  the checkout holds; the status line never runs the `--dev` shim.
- **Install mode without a plugin root** (`detect.go`): the copy
  `~/.abcd.noindex/path-entry` records reports `pinned`
  (iss-2610240916382802).
- **The docs**: commands/ahoy.md (the setup flow, the answers file's
  `approve.plan`, the download), the brief's 01-ahoy.md, invariants 1 and 7,
  the universal pattern for prompts, and adr-2610241101463288.

## Approach

The redesign is a layer over the existing apply pass rather than a rewrite of
it: the plan is asked through an optional prompter interface, so every door
that does not put a plan (a piped stream, a test's fake prompter, `--yes`)
runs the pass exactly as before, and the plan's answer becomes a prompter
(`planDefaults`) that answers the approvals and the defaults the plan listed
and passes every other question through. That keeps the steps' own questions,
and their order, as they were, and keeps the edits to `apply.go` to the plan's
two places (before the adoption of an unmanaged repository, after the
idempotency check of a managed one), the download step, the shell seam and the
approvals' gap list.

## Criteria

1. **No git, no repository or no harness stops before any question.** Met:
   `setupBasicsStop` and `missingGitStop`, held by
   `TestSetupStopsBeforeAnyQuestionWithoutTheBasics`.
2. **One plan screen, asked once.** Met at a terminal and through an answers
   file: `askSetupPlan`, held by `TestPlanProceedAsksOnceAndTakesTheDefaults`,
   `TestPlanCancelWritesNothing`, `TestPlanCustomiseAsksEachKindOfChange` and
   the drawn door's `TestInterviewOffATerminalIsPlainAndMatchesTheTerminalRecord`.
3. **A person-only step is its own question naming value and where.** Met for
   the identity (`TestIdentityQuestionStandsWithoutTheSettingsApproval`,
   `TestSettingsApprovalDoesNotListTheIdentity`), the download and the status
   line (`TestDownloadComesBeforeTheStatusLine`), the commit checks
   (`TestArmingTheHooksIsItsOwnQuestion`), and each tool (its question
   unchanged, now with the explanation as its material, and at a terminal
   never behind the dependency approval:
   `TestCustomiseStillAsksTheToolQuestion`). The plan lists exactly the
   questions the run puts, in the order it puts them (download, shell
   settings, commit checks, status line, identity, tools):
   `TestThePlanListsAToolOnlyWhenItWillAsk`,
   `TestPersonOnlyQuestionsComeInThePlansOrder`. The shell settings question
   is `stepShellPath`/`shellSettingsAsks`, which itd-2610240919030303 fills.
4. **The download before the status line.** Met:
   `TestDownloadComesBeforeTheStatusLine` (a yes wires the status line in the
   same run; a no never asks about it), `TestFetchVerifiedChecksBothRecords`.
5. **At most four questions on a first run.** Met for the plan, the download
   and the identity; the shell question is the seam above. The status-line
   offer is a person-only question too (criterion 3), so a fresh account whose
   assistant settings exist is asked it as a fifth where it can be wired; the
   product thinker is asked whether criterion 5 counts it.
6. **One line, `?` for the whole question.** Met at the drawn door (arrows and
   numbered) and the stdin door's plan: `TestQuestionMarkAsksForTheExplanation`,
   `TestQuestionMarkDrawsTheWholeQuestion`. The answers door writes each
   question whole, since it is a record, not a person.
7. **A three-part report.** Met: `TestReportHasThreeParts`; the CLI prints
   it. A change the person approved that then failed is a needs-you step with
   its reason, not only a note (`TestAFailedAbcdDownloadIsANeedsYouStep`,
   `TestAFailedShellEditIsANeedsYouStep`), the step after a yes to the shell
   settings is to open a new terminal
   (`TestAfterAYesTheReportSaysOpenANewTerminal`), and committed hooks git
   does not run keep the install from being reported clean
   (`TestUnarmedHooksAreNeverClean`, iss-2610241134315056).
8. **A fully set-up repository prints one line.** Met:
   `TestFullySetUpRerunIsOneLine`. At the stdin door's plan screen a bare
   Enter is no answer and asks again (`TestPlanScreenBareEnterAsksAgain`).

## Footprint

- packages: internal/core/ahoy, internal/core/update, internal/core/vintage, internal/surface/cli, internal/surface/cli/ask, commands/ahoy.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/05-internals/04-universal-patterns.md, .abcd/development/decisions/adrs, docs/reference/cli/commands.md
- tests: TestSetupStopsBeforeAnyQuestionWithoutTheBasics, TestGitInstalledReadsPath, TestPlanProceedAsksOnceAndTakesTheDefaults, TestPlanCancelWritesNothing, TestPlanCustomiseAsksEachKindOfChange, TestDownloadComesBeforeTheStatusLine, TestFetchVerifiedChecksBothRecords, TestIdentityQuestionStandsWithoutTheSettingsApproval, TestSettingsApprovalDoesNotListTheIdentity, TestQuestionMarkAsksForTheExplanation, TestQuestionMarkDrawsTheWholeQuestion, TestOldApprovalIdIsRefusedNamingThePlan, TestReportHasThreeParts, TestFullySetUpRerunIsOneLine, TestGoRunOfThisCheckoutIsCurrent, TestGoRunModuleOf, TestInstallModeReadsTheRecordWithoutAPluginRoot

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
