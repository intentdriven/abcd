---
id: itd-2609231434459890
slug: abcd-s-own-test-lanes-check-the-machine-s-load-before-they
spec_id: spc-2609231542463113
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-2609210828122412]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# abcd's own tests warn before running on an overloaded machine

## Press Release

> Before abcd runs its own tests on your machine, it looks at what is already running. If a program that is not part of abcd's work has been running flat out for over half an hour, or the machine is loaded far beyond its cores, abcd says so loudly before it starts: your own stray programs by name, with how to stop them; other people's only as a count and how much of the machine they use. Then it carries on, because the choice to stop is yours. In an autonomous run the warning also lands in the run's log, so nobody finds out two days later from a crash.
>
> "Two days of leftover busy loops took my machine down and nothing said a word," said Maya, an autonomous-development practitioner. "Now the first test run after I leave something burning tells me what it is and how to stop it."

## Why This Matters

On 2026-09-21 the development machine panicked: eight leftover CPU burners from
a load experiment had run for more than two days, abcd's own test binaries were
found running beneath them, and nothing had said a word
(iss-2609210828122412, promoted into this intent; `promoted_from` (historical) was
written by hand because the issue route of `capture promote --intent` stamped only
`promoted_to` (historical)). On 2026-09-22/23 the autonomous run then worked for a whole
session under a load of about 440 from another account's orphaned loops that it
could not stop, and the product thinker chose to carry on.

The product thinker ruled on 2026-09-23 that a load experiment is trusted only
under four conditions. The first three (one owned process group killed
together; "clean" proven by what is running, never by a list; consent and a cap
below the core count on a live development machine) ship as the bundled `LOAD`
rule domain. The fourth, abcd's own test lanes checking the machine before they
start, is this intent. At the planning interview the product thinker changed
that fourth condition from "refuse when the load is too high" to "warn, never
refuse": the load average alone cannot tell the incident from the run's normal
state (one preflight peaks near 7, eight concurrent preflights near 42, the
incident's burners near 8 to 12 on 16 cores), and a refusal would have stopped
every lane of a run the product thinker chose to continue.

## Decomposition (itd-84 hand-run, 2026-09-23, confirmed by the product thinker at the planning interview)

Verdict: **FILE-AS-IS**. One capability; the rest is already shipped, carried by
this intent's criteria, or plumbing.

| Part | Type | Home |
|------|------|------|
| A warning before abcd's own tests start, naming the caller's stray programs and counting other accounts', written to the run log in an autonomous run | capability | this intent |
| One owned process group killed together; clean proven by what is running; consent and a cap for load experiments; never kill by pattern | rules | **already shipped** as the bundled `LOAD` rule domain (the last added in the loadtrust fix round) |
| abcd never starts its tests under foreign load | trust rule | **not recorded**: the product thinker ruled warn-only, so no refusal invariant exists to record |
| What a warning may say about other accounts' programs | stance | carried by acceptance criteria 2 and 4 and the scope conditions; no new principle |
| The settings file under `~/.abcd/`, the process reading on each platform, and the preflight and eval-harness wiring | plumbing | the spec `intent plan` mints |

Relations, in prose because the schema carries no typed field for them
([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)):
this intent refines the fourth condition of the 2026-09-23 ruling on
iss-2609210828122412, and changes that condition from a refusal to a warning.

## What's In Scope

- A check that runs once at the start of `make preflight` (so the pre-push hook
  inherits it) and once at the start of the eval harness, never once per test
  package.
- Two warning triggers: a process outside abcd's own lanes at near full CPU for
  longer than the stray limit (default 30 minutes), and a load average above the
  extreme limit (default four times the online core count).
- The caller's own strays named in full (executable name, pid, age, CPU share)
  with how to stop them; other accounts' strays reported only as a count and a
  total CPU share. The output passes the private-names scrub.
- A `load` warning event in the run log when the check runs inside an autonomous
  run.
- Per-machine limits in a settings file under `~/.abcd/`, defaulting from the
  online core count in the same unit as the `LOAD` rule's cap.
- macOS and Linux, with no new dependency.

## What's Out of Scope

- Refusing to start. The check never refuses and never waits.
- Killing or pausing anything. The warning names the remedy for the caller's own
  strays; the caller acts.
- CI runners. A fresh runner carries no strays; the check does not run there and
  says so.
- Windows.
- Load experiments themselves, which the `LOAD` rule domain governs.

## Acceptance Criteria

- **Given** one of the caller's own programs outside abcd's work has run at near full CPU for longer than the stray limit, **when** `make preflight` or the eval harness starts, **then** it prints a warning naming the program's executable name, pid, age and CPU share with how to stop it, and carries on.
- **Given** stray programs that belong to other accounts, **when** the check runs, **then** the warning reports only how many there are and their total CPU share, with no program name, command line or account name, and carries on.
- **Given** only abcd's own short-lived test processes are running, however many at once (for example eight concurrent preflights at a load near 42 on 16 cores), **when** the check runs, **then** no warning is printed, and a test proves it.
- **Given** a load average above the extreme limit (four times the online core count by default), **when** the check runs, **then** the warning gives the load and the core count, and carries on.
- **Given** an autonomous run, **when** the check prints a warning, **then** the same warning is written to that run's log as an event.
- **Given** a settings file under `~/.abcd/` that sets either limit, **when** the check runs, **then** it uses those limits; without the file it derives the defaults from the online core count; a malformed file is reported loudly and the defaults are used.
- **Given** the check's placement, **when** the local check run or the eval harness starts, **then** the check runs once at its start and never once per test package; **and given** a CI runner, **then** the check does not run and the log says it was skipped and why.
- **Given** macOS or Linux, **when** the check runs, **then** it reads load and processes with no new dependency; **and given** a platform where it cannot read them, **then** it says it could not check and carries on.

## Decisions

Ruled by the product thinker on 2026-09-23 at the planning interview, after two
independent adversarial reviews of the draft (design/feasibility and record
discipline):

1. **Signal.** Both triggers: foreign sustained CPU (the incident's shape) and an
   extreme load ceiling. The load average is not the trigger for the incident.
2. **Warn, never refuse.** For the caller's own strays and for other accounts'
   alike. This changes the 2026-09-23 ruling's fourth condition from "refuse
   when it is too high".
3. **Where.** Once at the start of the local check run and of the eval harness;
   in an autonomous run, also the run log; not on CI.
4. **Privacy.** Other accounts' programs appear only as a count and a total CPU
   share.
5. **Limits.** Defaults of 30 minutes near full CPU for a stray and four times
   the online core count for extreme load, adjustable per machine in a settings
   file under `~/.abcd/`.

## Open Questions

_None open; decisions 1 to 5 settle the four the draft carried (the ceiling,
which lanes, what a refusal names, an override) and the signal the reviews
raised._

## Mechanism

We expect the foreign-sustained-CPU signal to flag both recorded incidents (the 2026-09-21 leftover burners and the 2026-09-22/23 other-account loops) and none of the run's 58 recorded normal-work load samples, because normal work is entirely short-lived test processes; shown wrong if it flags normal parallel work or misses a real stray.

## Scope Conditions

- macOS and Linux developer machines; Windows is not covered. <!-- cond: cond-2609231542464428 -->
- abcd's tests running on a person's own development machine, not on CI. <!-- cond: cond-2609231542466193 -->
- abcd's own test processes finish within minutes; a test running flat out for longer than the stray limit would read as a stray. <!-- cond: cond-2609231542465241 -->
- The machine may be shared by several accounts, which is why other accounts' programs are only counted. <!-- cond: cond-2609231542469551 -->

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-46f69cff6b50 -->
Fidelity review — receipt rcp-46f69cff6b50 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:98e58bb42916e7f0648e4befdeefa2dea3cf5f9503c15495bec70b925497f3a1
Input attestations: diff:tree at 4c09b5c749de3dc3d1a1e0ab85d9dcd58ffcb4d5 (main lineage, itd-2609231434459890 shipped)@-;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Classify names the caller's own processes past the stray limit at near-full share, excluding the invocation's ancestry; the render prints each with name, pid, group, age and CPU share and the pgrep/kill remedy, and ends with abcd carries on; TestImplementLoadNamesOwnStraysWithTheRemedy covers it and the Makefile target ignores the exit
  evidence: internal/core/machineload/classify.go:146 — "if ancestry[p.PID] || p.Age <= limit {"
  evidence: internal/surface/cli/implement_load.go:140 — "pid %d group %d running %s CPU %d%% of a core"
  evidence: internal/surface/cli/implement_load.go:168 — "still shows it, then: kill %d%s"
  evidence: internal/surface/cli/implement_load.go:201 — "abcd carries on; stopping them is your call."
  evidence: internal/surface/cli/implement_load_test.go:69 — "func TestImplementLoadNamesOwnStraysWithTheRemedy(t *testing.T) {"
  evidence: Makefile:343 — "-go run ./cmd/abcd implement load --site preflight"
- ac-2 — MET: a stray with another uid only increments a count and a cores total, the OtherStrays type carries no identity field, and the render prints the count and share alone; TestImplementLoadKeepsOtherAccountsAnonymous and TestOtherStraysTypeCarriesNoIdentity cover it
  evidence: internal/core/machineload/classify.go:157 — "v.Others.Count++"
  evidence: internal/surface/cli/implement_load.go:173 — "Other accounts: %d programs at nearly all the CPU they can get, for over %s, using about %.1f cores."
  evidence: internal/core/machineload/classify_test.go:355 — "func TestOtherStraysTypeCarriesNoIdentity(t *testing.T) {"
  evidence: internal/surface/cli/implement_load_test.go:112 — "func TestImplementLoadKeepsOtherAccountsAnonymous(t *testing.T) {"
- ac-3 — MET: TestEightConcurrentPreflightsAreQuiet models eight make/go/core.test/compile trees under the caller's uid and asserts no trigger, TestEightConcurrentShortLivedProcessesDoNotWarn does it with live busy children, and TestImplementLoadIsQuietUnderOwnParallelWork covers the front door; the machineload package is green at BASE
  evidence: internal/core/machineload/classify_test.go:298 — "func TestEightConcurrentPreflightsAreQuiet(t *testing.T) {"
  evidence: internal/core/machineload/live_test.go:47 — "func TestEightConcurrentShortLivedProcessesDoNotWarn(t *testing.T) {"
  evidence: internal/surface/cli/implement_load_test.go:196 — "func TestImplementLoadIsQuietUnderOwnParallelWork(t *testing.T) {"
- ac-4 — MET: the extreme trigger fires strictly above lim.ExtremeLoad, which defaults to four times the online cores, and the render states the load averages, the limit and the core count; TestExtremeTriggerIsStrictlyAbove and TestImplementLoadReportsLoadAndCores cover it
  evidence: internal/core/machineload/classify.go:171 — "if snap.HasLoad && snap.Load1 > lim.ExtremeLoad {"
  evidence: internal/core/machineload/limits.go:30 — "DefaultExtremeFactor = 4"
  evidence: internal/surface/cli/implement_load.go:189 — "Load %.1f over 1 min (%.1f over 5, %.1f over 15) is above the extreme limit of %s (%s)."
  evidence: internal/surface/cli/implement_load_test.go:213 — "func TestImplementLoadReportsLoadAndCores(t *testing.T) {"
- ac-5 — MET: logLoadWarning appends an EventLoad event to the live run's log with the result's fields, LoadResultFromEvent reads it back, and a failed write is reported with the warning standing; TestLoadWarningIsLoggedInALiveRun, TestNoEventOutsideARun and TestLoggedEventRendersToThePrintedWarning cover it
  evidence: internal/core/implement/log.go:62 — "EventLoad = "load""
  evidence: internal/core/implement/load.go:402 — "_, err := run.append(session, EventLoad, fields)"
  evidence: internal/core/implement/load_test.go:114 — "func TestLoadWarningIsLoggedInALiveRun(t *testing.T) {"
  evidence: internal/surface/cli/implement_load_test.go:288 — "func TestLoggedEventRendersToThePrintedWarning(t *testing.T) {"
- ac-6 — MET: readLimits reads ~/.abcd/load-limits with stray-minutes and extreme-load keys over DefaultLimits(cores), a fault discards the whole file and the render prints LOAD CHECK SETTINGS UNUSABLE with the defaults; TestLimitsFromTheSettingsFile, TestMalformedLimitsFileFallsBackWhole and TestImplementLoadMalformedLimitsIsLoud cover it
  evidence: internal/core/implement/load.go:306 — "path := filepath.Join(home, ".abcd", machineload.LimitsFileName)"
  evidence: internal/core/machineload/limits.go:53 — "func DefaultLimits(cores int) Limits {"
  evidence: internal/surface/cli/implement_load.go:87 — "LOAD CHECK SETTINGS UNUSABLE: %s%s%s; using the defaults for both limits"
  evidence: internal/core/implement/load_test.go:277 — "func TestMalformedLimitsFileFallsBackWhole(t *testing.T) {"
- ac-7 — MET: load-check is the first prerequisite of make preflight and runs one `implement load` invocation, the eval harness runs its own once from TestMain with ABCD_LOAD_CHECKED keeping one preflight to one terminal report, and a CI runner is skipped with a reason the render prints; TestCheckLoadSkipsOnCIWithReason and TestImplementLoadSkipsOnCI cover it
  evidence: Makefile:307 — "preflight: load-check lint-reviews lint-issues"
  evidence: evals/harness_test.go:85 — "if os.Getenv("ABCD_LOAD_CHECKED") != "preflight" {"
  evidence: internal/core/implement/load.go:196 — "skipped on a CI runner (" + reason + "): a fresh runner carries no programs left from earlier work"
  evidence: internal/surface/cli/implement_load.go:93 — "load check (%s): %s"
  evidence: internal/core/implement/load_test.go:77 — "func TestCheckLoadSkipsOnCIWithReason(t *testing.T) {"
- ac-8 — MET: read_darwin.go shells /bin/ps and sysctl, read_linux.go reads /proc, read_other.go returns ErrUnsupported which CheckLoad turns into LOAD CHECK UNAVAILABLE naming the platform and carrying on; TestMachineLoadImportsOnlyTheStandardLibrary proves no dependency and TestImplementLoadCannotCheck covers the unavailable path
  evidence: internal/core/machineload/read_other.go:10 — "func Read() (Snapshot, error) { return Snapshot{}, ErrUnsupported }"
  evidence: internal/core/implement/load.go:211 — "abcd reads load and processes on macOS and Linux only, and this is"
  evidence: internal/surface/cli/implement_load.go:98 — "LOAD CHECK UNAVAILABLE (%s): %s; carrying on without checking the machine's load"
  evidence: internal/core/machineload/read_test.go:50 — "func TestMachineLoadImportsOnlyTheStandardLibrary(t *testing.T) {"
  evidence: internal/surface/cli/implement_load_test.go:250 — "func TestImplementLoadCannotCheck(t *testing.T) {"

Gap audit:
- honoured:
  - warn, never refuse: the check exits 0 on every status and the Makefile ignores even a build failure
    evidence: Makefile:343 — "-go run ./cmd/abcd implement load --site preflight"
  - own strays' names pass the private banned-names scrub
    evidence: internal/core/implement/load.go:340 — "func scrubOwnNames(repoRoot string, own []LoadOwnStray) bool {"
  - the remedy never group-kills a mixed or the caller's own group and never names a pattern kill
    evidence: internal/surface/cli/implement_load.go:170 — "Never by pattern (pkill -f, killall)"
  - the verb is reachable from the plugin surface
    evidence: commands/implement.md:147 — "implement load --site preflight|eval-harness [--json]"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609231542464428 — survived: the readers are darwin and linux build files and every other platform takes read_other.go's ErrUnsupported, reported as unavailable
  evidence: internal/core/machineload/read_other.go:10 — "return Snapshot{}, ErrUnsupported"
- cond-2609231542466193 — survived: a CI runner is detected from GITHUB_ACTIONS or CI and the check is skipped there with a reason, as the condition assumed
  evidence: internal/core/implement/load.go:273 — "if getenv("GITHUB_ACTIONS") == "true" {"
- cond-2609231542465241 — survived: the classifier keys a stray on age past the limit and near-full share alone, excluding only the invocation's ancestry, so a test running flat out past the limit would read as a stray exactly as assumed; the eight-preflight fixtures stay under the limit
  evidence: internal/core/machineload/classify.go:146 — "if ancestry[p.PID] || p.Age <= limit {"
- cond-2609231542469551 — survived: strays under another uid are counted and summed with no identity carried, as the condition's reason requires
  evidence: internal/core/machineload/classify_test.go:355 — "func TestOtherStraysTypeCarriesNoIdentity(t *testing.T) {"

## Grounds

- pursued: we expect that no leftover program will again run for days unnoticed on a machine abcd tests on; shown wrong if a stray runs for a day or more while abcd's tests keep running without a warning.
