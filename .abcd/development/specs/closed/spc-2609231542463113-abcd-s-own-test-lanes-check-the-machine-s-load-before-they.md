---
id: spc-2609231542463113
slug: abcd-s-own-test-lanes-check-the-machine-s-load-before-they
intent: itd-2609231434459890
origin: researcher-authored
production_mode: hand-written
---
# The load check: preflight and the eval harness warn about strays and extreme load, and never refuse

## Summary

spc-2609231542463113 delivers itd-2609231434459890: before `make preflight` and
the eval harness run abcd's own tests, a check reads the machine's load
average and process table once. It warns when a program outside the running
lanes has used a near-full core for longer than the stray limit (30 minutes by
default), or when the one-minute load average is above the extreme limit (four
times the online core count by default). The caller's own strays are named in
full with the owned-group remedy the `LOAD` rule domain teaches; other
accounts' strays appear only as a count and a total CPU share. Inside an
autonomous run the same warning is also written to the run log as a `load`
event. The check always exits 0: it never refuses, never waits and never
kills.

The check is one core function, `implement.CheckLoad`, behind one thin
sub-verb, `abcd implement load`, in the implement family that already owns the
run log's single writer and the report-never-enforce verdict. Beneath it is
the one new primitive, a stdlib-only machine read in a new leaf package,
`internal/core/machineload`. `make preflight` runs it as its first
prerequisite (the pre-push hook inherits it); the eval harness's `TestMain`
runs it once after building the binary under test; CI runs a single step that
prints why the check is skipped there.

Two facts measured for this spec shape the harness wiring. `go test` in
package-list mode prints nothing from a passing package, `TestMain` output
included, and a cached eval lane starts no harness at all (both probed on this
machine, see Risks). The design accounts for both rather than assuming the
harness's output reaches a person.

## Scope

### In

- **The machine read**: One-minute, five-minute and fifteen-minute load
  averages, the online core count, and a process table (pid, parent pid,
  process group, effective uid, age, cumulative CPU time, executable name) on
  macOS and Linux, with no new dependency.
- **The classifier**: A pure function from a snapshot, the limits and the
  caller's identity to a verdict with two triggers (stray, extreme), own
  strays in full and other accounts' strays as a count and a CPU total.
- **The machine settings file**: `~/.abcd/load-limits`, read-only, with
  defaults derived from the online core count and a loud report when it is
  malformed.
- **The core function and the verb**: `implement.CheckLoad` and
  `abcd implement load --site preflight|eval-harness [--json]`, with four
  statuses (`ok`, `warning`, `skipped`, `unchecked`) and exit 0 on every one.
- **The private-names scrub**: Own strays' executable names pass through the
  private banned-names layer, on the same engine the pre-commit guard uses,
  before they are printed or logged.
- **The run-log event**: A verb-owned `load` event, written only when the
  check warns inside an autonomous run.
- **Wiring**: The first prerequisite of `make preflight`, one call in
  `evals/harness_test.go`'s `TestMain`, and one CI step per harness job that
  prints the skip reason.
- **Loud staging**: A platform the reader does not cover, or a read that
  fails, says it could not check and carries on.
- **Doc surfaces**: The plugin command page, the CLI reference, the brief's
  implement and configuration pages, the compatibility snapshot, and the six
  surfaces that restate the preflight prerequisite list.

### Out

Carried from the intent and not reopened here:

- Refusing to start, waiting for the load to fall, or any override flag: the
  product thinker ruled warn, never refuse (Decision 2), so nothing exists to
  override.
- Killing, pausing or renicing anything. The warning names the remedy; the
  caller acts.
- Running the check on CI, beyond the step that says it is skipped.
- Windows and every platform other than macOS and Linux, beyond the loud
  "could not check" line.
- Load experiments themselves, which the `LOAD` rule domain governs.
- A check that keeps watching after the tests start: the check runs once at
  each start, so a burner launched mid-run is seen by the next start, not this
  one.
- An allowlist of the caller's deliberate long-running programs. Raising
  `stray-minutes` is the only knob (see Risks).

## Approach

### 1. Where the check lives

**The seam is the implement family; the new primitive is only the machine
read.** Nothing in `internal/`, `cmd/` or `evals/` reads the load average, the
core count or the process table today (a `git grep` for `loadavg`,
`runtime.NumCPU`, `syscall.Sysctl` and `_NPROCESSORS` finds only comments in
the banned-names code about `/proc/<pid>/cmdline`). So the read is new, and it
goes in a leaf package, `internal/core/machineload`, that imports only the
standard library.

The decision the read feeds is not new. `internal/core/implement` already owns
the two things this check needs from the record's side: the run log's one
writer (`Run.append`, reached through the run's lock, one `O_APPEND` line per
event) and the verdict shape that reports a limit without enforcing it
(`Verdict.Ceiling` in `bounds.go`, "recorded and reported, never enforced" in
`session.go`). The check is a sibling of `implement check`, so it lives there:

- **`machineload`** (new leaf): `Snapshot`, `Proc`, `Read() (Snapshot,
  error)` with per-platform files, the pure parsers, `Limits`,
  `DefaultLimits(cores)`, `ReadLimits(home)`, and `Classify(snap, limits,
  self) Verdict`.
- **`implement.CheckLoad(LoadRequest) LoadResult`** (new, `load.go`): The one
  core function. It decides CI, reads the machine through an injectable
  reader, reads the limits, classifies, scrubs own names, and writes the run
  log event when a run is live. It never writes to stdout.
- **`abcd implement load`** (new sub-verb in `internal/surface/cli`): Parses
  `--site` and `--json`, calls `CheckLoad`, renders the result. No logic.

Rejected, with the reason each loses:

- **A new `implement check load` step.** `Run.Check` requires a joined session
  and returns an `ErrRefused`-classed error on a bound; preflight runs on no
  session's word, and a step whose verdict can never refuse would break the
  contract every other step keeps.
- **A guard entry.** The guard is a stateless command-position token matcher
  held to a fixture corpus; "tests while the machine is loaded" is a machine
  state, not a command shape.
- **`peers`.** It reads sibling worktrees for records and knows nothing of
  processes.
- **A new top-level verb.** It would open a second front door for
  run-adjacent machinery, a second plugin page and a second surface family, for
  a check whose only record-side act is a line in the run log the implement
  family already writes.

### 2. Reading the machine

`machineload.Read` dispatches by build tag. Every parser is a pure function
over bytes or an `fs.FS`, so both platforms' parsers are unit-tested on both CI
legs.

**macOS** (`read_darwin.go`):

- **Load**: `syscall.Sysctl("vm.loadavg")` returns the kernel's `loadavg`
  struct as a string with one trailing NUL stripped (23 bytes here): three
  little-endian `uint32` fixed-point values, four bytes of padding, then the
  `int64` scale. The parser restores the stripped byte, reads the scale from
  offset 16, and divides. Probed here in Go: 13.79, 18.40, 18.80, equal to
  `sysctl -n vm.loadavg`.
- **Online cores**: `syscall.SysctlUint32("hw.activecpu")`, the value
  `sysconf(_SC_NPROCESSORS_ONLN)` reports. Probed here: 16, equal to
  `getconf _NPROCESSORS_ONLN`.
- **Processes**: `/bin/ps -axo pid=,ppid=,pgid=,uid=,etime=,time=,comm=` run
  through `os/exec` by absolute path with `LC_ALL=C`, so a planted `ps` on
  `PATH` is never run. The first six fields split on whitespace; `comm` is the
  rest of the line (names with spaces survive) and is reduced to its base
  name, because macOS prints the executable's full path, which can carry a
  home directory. `etime` is `[[dd-]hh:]mm:ss`; `time` is cumulative user plus
  system CPU as `[dd-][hh:]mm:ss.cc` with minutes allowed past 59 (`114:03.49`
  seen here). Probed: all seven keywords accepted, other accounts' processes
  listed (45 distinct uids here), 63 ms for about 1,870 processes. `etimes` is
  not available on macOS, which is why `etime` is parsed.

**Linux** (`read_linux.go`), from `/proc` alone, so a minimal `ps` does not
matter:

- **Load**: The first three fields of `/proc/loadavg`.
- **Online cores**: `/sys/devices/system/cpu/online`, a range list such as
  `0-3,8-11`, which is what glibc's `sysconf(_SC_NPROCESSORS_ONLN)` reads. Not
  `runtime.NumCPU`, which counts the affinity mask and can be smaller.
- **Processes**: Each numeric `/proc/<pid>/stat`: `comm` is the text between
  the first `(` and the last `)` (it may contain spaces and parentheses), then
  ppid, pgrp, utime, stime and starttime by field position. Age is
  `/proc/uptime` minus starttime; CPU time is utime plus stime. Both are in
  clock ticks, converted with `AT_CLKTCK` read from `/proc/self/auxv` (100
  when the entry is absent). The effective uid comes from the `Uid:` line of
  `/proc/<pid>/status`, read only for processes that already pass the age and
  CPU tests, since the owner of the `/proc/<pid>` directory is root for a
  non-dumpable process. A process that exits between the listing and the read
  is skipped, not an error. Zombies (state `Z`) are skipped.

**Both**: A core count that cannot be read falls back to `runtime.NumCPU()`
and the report says so. No `go.mod` change: `golang.org/x/sys/unix.SysctlRaw`
would read `vm.loadavg` more cleanly, and `x/sys` is already an indirect
requirement, but promoting it edits `go.mod`, which needs sign-off this change
does not need.

**Elsewhere** (`read_other.go`, `//go:build !darwin && !linux`): `Read`
returns `ErrUnsupported`, and its doc comment names itd-2609231434459890 and
the scope condition that excludes other platforms. See section 8.

### 3. Classifying what is running

**Near full, and for how long.** A process's CPU share is its cumulative CPU
time divided by its age, in cores (1.0 is one core flat out). It is a stray
candidate when its age exceeds the stray limit and its share is at least 0.9.
The same definition holds on both platforms and needs no second sample, so the
classifier is a pure function of one snapshot, and a fixture can express a
two-day burner without spawning one. A multi-threaded process can exceed 1.0
and counts once, with its whole share.

**abcd's own lanes, by ancestry and by time, never by name.** Two things
exempt a process:

- **The invocation's own ancestry**: The check's own pid and every ancestor up
  the parent chain (the `ps` or `/proc` read, `make`, the pre-push hook, the
  shell, the agent host). They are the run that asked; they cannot be strays
  from outside it.
- **Age**: Everything else abcd's lanes start is short-lived. `go`,
  `compile`, `link`, `vet` and the 57 package test binaries of `go test ./...`
  live for seconds to minutes, so a concurrent preflight in another worktree
  never reaches the stray limit. This is the intent's third scope condition
  made operational.

A name list (`go`, `*.test`, `make`) is rejected deliberately. It would exempt
exactly the hung test binary that the third scope condition says must read as
a stray, and a burner named `go` would pass it.

**Same uid or other uid.** A candidate whose effective uid equals the check's
`os.Geteuid()` is the caller's own; any other uid, root included, is another
account's. There is no third class and no allowlist of operating-system
daemons: a system process at a near-full core for over half an hour is real
load, and it appears only in the anonymous count.

**The two triggers.**

- **Stray**: At least one candidate outside the ancestry. Own and other
  candidates both trigger it.
- **Extreme**: The one-minute load average is strictly above the extreme
  limit. Only the one-minute value decides, because the five-minute and
  fifteen-minute averages remember preflights that finished a quarter of an
  hour ago; all three are reported.

**The verdict type carries the privacy rule.** `Verdict.Own` is a list of
`OwnStray{Name, PID, PGID, Age, Share}`; `Verdict.Others` is
`OtherStrays{Count, Cores}`, a type with no field that could hold a name, a
command line, a uid or an account. A renderer cannot leak what the verdict
cannot carry. The classifier also returns the remedy plan for own strays (see
section 5), computed from the snapshot's process groups.

Measured on this machine while the run's own lanes were busy (load 18 to 26
on 16 cores): no process older than 30 minutes had a share above 0.19, so the
nearest normal program sits far below the 0.9 line.

### 4. The machine settings file

**Name and home.** `~/.abcd/load-limits`, in the caller's own machine tier,
beside `trusted-roots` and `local-transcript-roots`. Load is a property of the
machine, so neither the repository's `.abcd/rules.json` nor an environment
variable is its home. The check never creates the file or `~/.abcd/`.

**Format.** Line-oriented, following the sibling files: `#` starts a comment,
blank lines are ignored, every other line is `<key><space or tab><value>`.

```
# abcd's load check on this machine (itd-2609231434459890)
stray-minutes 30
extreme-load 64
```

- **`stray-minutes`**: An integer from 1 to 10080 (one week).
- **`extreme-load`**: A decimal number above 0 and at most 100000, an
  absolute load in the `_NPROCESSORS_ONLN` unit the `LOAD` rule's cap uses.
- Either key may be omitted; the omitted one takes its default.

**Defaults.** `stray-minutes` 30; `extreme-load` four times the online core
count (64 on this 16-core machine).

**Read.** Through `fsutil.ReadDeclaration`, the guarded read the other
home-scope files use: a regular file, not a symlink, owned by the caller, not
writable by group or other, at most 4 KiB.

**Malformed.** An unknown key, a repeated key, a value that does not parse or
is out of range, or any `ReadDeclaration` refusal other than "absent" makes
the whole file unusable. The check then uses the defaults for **both** limits,
never a mix, and prints a loud line naming the file, the line number and the
fault class, even when there is otherwise nothing to warn about:

```
LOAD CHECK SETTINGS UNUSABLE: ~/.abcd/load-limits line 3: unknown key "stray-mins" (known: stray-minutes, extreme-load); using the defaults for both limits: 30 min, load 64 (4 x 16 online cores)
```

An offending key is echoed through `termsafe.Sanitize`, capped at 32 bytes; a
value is never echoed. An absent file is silent.

### 5. Output

The verb writes its report to stdout. Every name passes through
`termsafe.Sanitize`, because process names are free text, before the
private-names scrub below.

**`ok`**: One quiet line, so a log shows the check ran:

```
load check (preflight): nothing to warn about; load 18.1 on 16 online cores
```

**`warning`**: A loud block. Own strays are listed in full, up to 20 with an
"and N more" line; then the remedy; then the others' line; then the load line:

```
LOAD WARNING (preflight): programs that are not abcd's tests are keeping this machine busy.
  Your programs at a near-full core for over 30 min:
    yes    pid 41233  group 41230  running 2d 07h  CPU 99% of a core
    yes    pid 41234  group 41230  running 2d 07h  CPU 99% of a core
  To stop them, if you did not mean to leave them running, check each target first:
    pgrep -g 41230        lists only 41233 41234, then:  kill -- -41230
    ps -o pid=,comm= -p 41250   still shows it, then:  kill 41250   (its group holds other programs)
  Never by pattern (pkill -f, killall): a pattern also matches other sessions' programs.
  Other accounts: 38 programs at a near-full core for over 30 min, using about 14.1 cores.
  Load 440.2 over 1 min (431.0 over 5, 425.7 over 15) is above the extreme limit of 64 (4 x 16 online cores).
abcd carries on; stopping them is your call.
```

The load line is always present; when only the stray trigger fires it reads
"Load 12.3 on 16 online cores" instead. Limits taken from the settings file
are marked "(set in ~/.abcd/load-limits)".

**The remedy follows the `LOAD` rule domain.** The domain teaches one owned
process group, killed as a group only after a re-check that the handle still
names what was started, and never a kill by pattern (the no-pattern rule and
the re-check are the loadtrust fix round's wording, not yet on `main` at this
tip). The warning never signals anything itself; it prints commands:

- **Group form** (`pgrep -g <pgid>`, then `kill -- -<pgid>`): Offered only
  when every live member of that group in the snapshot is a stray this warning
  names, and the group is neither the check's own group nor the group of any
  ancestor. So a group that holds the caller's shell, agent host or anything
  unnamed is never offered.
- **Process form** (`ps -o pid=,comm= -p <pid>`, then `kill <pid>`): Every
  other own stray.
- **Never** a pattern, and never anything about another account's programs.

**Others.** Only the count and the summed share, rounded to one decimal, as
"about N cores". No name, command line, uid, account name or pid, by type
(section 3).

**The private-names scrub.** The one free-text field left is the own strays'
executable names. `CheckLoad` asks a new `banlist.MatchPrivate(repoRoot,
names) ([]bool, error)` which names match the private layer
(`.abcd/.work.local/private-names.txt`). It asks the enforcing engine itself,
exactly as `grepAccepts` and the pre-commit guard do: `grep -iE` under
`LC_ALL=C`, the patterns on stdin through `-f -` (never in argv, where
`/proc/<pid>/cmdline` would expose them), and the names in a 0600 temporary
file that is removed afterwards. So the scrub and the guard cannot disagree
about a match. A matching name becomes `[private name]`. An absent store
matches nothing. An unreadable or malformed store, or no usable `grep`, makes
every own name `[name withheld: private-names layer unreadable]`, and the
block says so. The scrub runs before the verdict is rendered or logged, so
the printed text and the run-log event carry the same masked names. The
package comment that says banlist does "not the matching" gains one sentence
naming this one read-only matcher.

**`--json`**: The same result as data: `status`, `site`, `load` (three
values), `cores`, `cores_source`, `limits` (`stray_minutes`, `extreme_load`,
`source` of `default`, `file` or `default-after-malformed`, and `malformed`
when it is), `triggers`, `own_strays`, `own_strays_more`, `other_strays`
(`count`, `cores`), `remedy`, `run_log` (`logged`, `session`, `error`) and
`reason` for `skipped` and `unchecked`.

### 6. The run-log event and the autonomous run

**Knowing it is in a run.** `CheckLoad` resolves the repository's root
commit and calls `implement.Peek` on it, which creates nothing. The check is
in an autonomous run when the run state holds at least one joined session
record (`Run.Sessions()` non-empty). A session's record is removed when it
leaves, so a closed run is not a live one. Outside a repository, or with no
joined session, the check writes nothing anywhere.

**When it writes.** Only when the status is `warning` and a run is live
(criterion 5). `ok`, `skipped` and `unchecked` write no event.

**The event.** `load` joins `verbOwnedEvents` beside `claim` and
`session_open`, so `implement log load` is refused as a hand-written
imitation. It is written through the run's lock and `Run.append`, as every
other verb-owned event is:

```json
{"ts":"2026-09-24T09:12:03Z","session":"804bdd81","event":"load",
 "cores":16,"extreme_limit":64,"limits_source":"default",
 "load1":440.2,"load5":431,"load15":425.7,
 "other_strays":{"count":38,"cores":14.1},
 "own_strays":[{"name":"yes","pid":41233,"pgid":41230,"age_s":198000,"cpu_pct":99}],
 "own_strays_more":0,"site":"preflight","stray_limit_min":30,
 "triggers":["stray","extreme"],"within_preflight":false}
```

- **`session`**: The run's first-role session, else the earliest joined. The
  check runs inside a lane on no session's word, and `Run.Check`'s role
  bounds deliberately never read a session from the environment; the line
  records the machine's state for the run, and `site` records where it was
  seen.
- **"The same warning"**: The event holds the verdict's facts, not its prose;
  the CLI's renderer is a pure function of them, so rendering a logged event
  reproduces the printed block byte for byte (a test pins it). Own strays are
  capped at 20 per line, as printed, which keeps the line far under the 16 KiB
  `maxLineBytes`.
- **The hand-run samples**: The run log already holds 67 `load` lines from the
  run's hand-written sampler, which carry `ncpu` and `cpu_pct_by_user` and no
  `triggers`. A reader tells the two apart by `triggers`. The sampler's
  per-account CPU map is exactly what criterion 2 forbids in a warning, so the
  check does not copy its shape.
- **A failed write**: Contention on the run lock (3 s) or an I/O error prints
  one loud line, "could not write the load warning to the run log: …; the
  warning above stands", and the exit stays 0.

### 7. Wiring

**`make preflight`.** A new phony target, first in the prerequisite list, and
a target-specific exported marker placed after the prerequisite line (the
gate-list parser in `preflightgates_test.go` reads the first line that starts
with `preflight:`):

```make
preflight: load-check lint-reviews lint-issues lint-decisions record-lint docs-lint site-render smoke evals-cold-reading
preflight: export ABCD_LOAD_CHECKED := preflight

load-check:
	-go run ./cmd/abcd implement load --site preflight
```

- **Order**: The pre-push hook runs plain `make preflight`, and serial GNU make
  builds prerequisites left to right, so the check runs first. Under `make
  -j` it may run beside the first gates; it reads nothing they change.
- **The pre-push hook** inherits it with no edit to its logic.
- **The leading `-`**: If the check cannot even be compiled, make reports the
  error as ignored and carries on; a broken build is the `go build` gate's to
  fail, never the load check's.
- **The marker**: GNU make passes a target-specific exported variable to the
  target's prerequisites and recipe, and not to a prerequisite run on its own
  (probed with the system's GNU Make 3.81). The eval harness reads it below.

**The eval harness.** `TestMain` in `evals/harness_test.go` runs the check
exactly once, right after building the binary under test and before
`m.Run()`: `abcdBin implement load --site eval-harness`. That is the start of
the one choke point both tagged lanes share, and it exercises the verb through
the built binary the harness exists to test. Its exit code is ignored and it
never panics. Where its output goes is decided by two facts probed for this
spec:

- **Package-list mode hides it.** `go test -tags smoke ./evals/...` prints
  nothing from a passing package, `TestMain` included.
- **So the output goes to the person directly.** With `ABCD_LOAD_CHECKED`
  unset (the harness started on its own), `TestMain` writes the report to
  `/dev/tty` when that opens, else to stderr. With `ABCD_LOAD_CHECKED=preflight`,
  it writes to stderr only: the preflight's own check already put a warning on
  the terminal, so one preflight shows one warning, not three. The verb stamps
  `within_preflight` on any event it writes in that case, so a reader can fold
  the three events of one preflight into one moment.

The check therefore runs once at each start and never per package: `go test
./...` builds 57 package test binaries, and no `TestMain` outside the eval
harness calls the check (a test enforces this).

**CI.** CI never runs `make preflight`: its `check` job runs `go test ./...`
and the race lane directly, and the eval lanes run as `make smoke` and `make
evals-cold-reading` in their own jobs. Each of those two jobs gains one step
before the harness:

```yaml
      - name: Load check (skipped on CI, says why)
        run: go run ./cmd/abcd implement load --site eval-harness
```

`CheckLoad` sees `GITHUB_ACTIONS=true` (or a `CI` value other than empty,
`false` or `0`), reads nothing, and returns `skipped` with the reason, which
the step prints into the job log:

```
load check (eval-harness): skipped on a CI runner (GITHUB_ACTIONS=true): a fresh runner carries no programs left from earlier work, so there is nothing to warn about
```

The harness's own call on CI takes the same branch; its line is hidden by
package-list mode, which is why the visible line is the step's. The step
shares the Go build cache with the harness build that follows. The exemption
is earned only while the runner is fresh, so a test pins that every job that
starts the harness runs on a GitHub-hosted label; moving one to a self-hosted
runner fails it.

### 8. Loud staging for the cannot-check case

Two paths lead to `unchecked`, and both print a loud line and exit 0:

- **A platform the reader does not cover**: `read_other.go` returns
  `ErrUnsupported`. The three disclosure sites the loud-staging principle asks
  for are the doc comment naming itd-2609231434459890 and its scope condition,
  the surface line below, and the intent's own out-of-scope entry for other
  platforms.

  ```
  LOAD CHECK UNAVAILABLE (preflight): abcd reads load and processes on macOS and Linux only, and this is freebsd; carrying on without checking the machine's load
  ```

- **A read that fails on a covered platform**: `/bin/ps` missing or exiting
  non-zero, `/proc` unreadable, or a parse that fails. The line names what
  could not be read and the error, for example "could not read the process
  table (/bin/ps: exit status 1)". A load average that reads while the process
  table does not is still reported, and the extreme trigger still applies.

### 9. Exit status

`abcd implement load` exits 0 for all four statuses, and for a failed run-log
write and an unusable settings file. The one non-zero exit is a malformed
invocation (an unknown flag, or a `--site` outside the closed vocabulary),
which is cobra's usage error, not a load verdict; the placement tests pin the
only two invocations that exist, so it cannot arise from the wiring.

### 10. Doc surfaces

- **`commands/implement.md`**: A "Check the machine's load" section and the
  sub-verb in its argument hint and table.
- **`docs/reference/cli/commands.md`**: The `implement load` entry, the four
  statuses, `~/.abcd/load-limits` and its format.
- **`.abcd/development/brief/04-surfaces/27-implement.md`**: The sub-verb and
  the `load` event in the run-log vocabulary.
- **`.abcd/development/brief/05-internals/03-configuration.md`**: The user
  scope's inventory gains the limits file. Its sentence "no home-scope one is
  read at all" names this one exception; the tree in
  `04-surfaces/01-ahoy.md`, which must agree, gains the same line.
- **`.abcd/development/release/surface.json`**: Regenerated for the new
  sub-verb and its flags.
- **The preflight prerequisite list**: `TestPreflightGateListIsNotRestatedWrongly`
  requires every prerequisite to be named where the list is restated, so
  `load-check` is named in the Makefile comment, `docs/how-to/install.md`,
  `CONTRIBUTING.md`, `AGENTS.md`, `CLAUDE.md` and `.githooks/pre-push`'s
  comment, each as "the load check first (a warning, never a failure)". It is
  not counted among the gates: the gate count in `AGENTS.md` stays six.

## Footprint and estimate

About a day:

- **`internal/core/machineload`** (new): `machineload.go`, `parse.go`,
  `read_darwin.go`, `read_linux.go`, `read_other.go`, `limits.go`,
  `classify.go`, tests and `testdata/`. About three and a half hours.
- **`internal/core/implement`**: `load.go` (`CheckLoad`, `LoadResult`,
  `LogLoad`), `log.go` (`EventLoad` among the verb-owned events). About an
  hour.
- **`internal/core/banlist`**: `private_match.go` (`MatchPrivate`). About half
  an hour.
- **`internal/surface/cli`**: `implement.go` registers the sub-verb;
  `implement_load.go` renders. About an hour.
- **Wiring**: `Makefile`, `evals/harness_test.go`, `.github/workflows/ci.yml`.
  About half an hour.
- **Doc surfaces** (section 10). About an hour.

## How the acceptance criteria are satisfied

Test names are the ones to be written; files are where they go. Fixture
snapshots are Go values in `internal/core/machineload/classify_test.go` built
from the recorded incidents and the run's own telemetry, so no test spawns a
long-lived burner.

1. **Own strays named with the remedy, and the check carries on.** Sections 3
   and 5. In `classify_test.go`: `TestMechanismFlagsIncidentOne` (eight `yes`
   processes of the caller's uid, one owned group, 2 days 7 hours old at 99%:
   stray trigger, all eight named with pid, group, age and share, group form
   offered), `TestStrayBoundaries` (29 and 31 minutes; shares 0.89 and 0.90),
   `TestAncestryIsNeverAStray` (a two-day agent host at 100% in the check's
   parent chain is exempt), `TestRemedyNeverGroupKillsAMixedGroup` and
   `TestRemedyNeverGroupKillsTheCallersGroup` (process form only). In
   `internal/surface/cli/implement_load_test.go`:
   `TestImplementLoadNamesOwnStraysWithTheRemedy` (the block names each field,
   the `pgrep -g` re-check precedes every `kill`, the text holds no `pkill`
   or `killall` except in the "Never by pattern" line, exit 0).
2. **Other accounts' strays only as a count and total CPU.** Sections 3 and 5.
   `TestMechanismFlagsIncidentTwo` in `classify_test.go` (38 other-uid `zsh`
   loops since the Saturday: count 38, share summed). In
   `implement_load_test.go`: `TestImplementLoadKeepsOtherAccountsAnonymous`
   (the fixture gives the others a distinctive name, command path, uid and
   account; neither the text nor the `--json` output contains any of them,
   and `other_strays` has exactly the keys `count` and `cores`; exit 0).
   `TestOtherStraysTypeCarriesNoIdentity` reflects over `OtherStrays` and
   fails if a field is added. `TestMatchPrivateMasksAMatchingName` and
   `TestMatchPrivateFailsClosed` in `internal/core/banlist/private_match_test.go`
   cover the scrub, including an unreadable store and a missing `grep`.
3. **Only abcd's own short-lived processes, however many: no warning, proven
   by a test.** Section 3. Two tests, a fixture and a live one:
   - `TestEightConcurrentPreflightsAreQuiet` in `classify_test.go` replays the
     run's busiest recorded sample (14:59:13Z on 2026-09-23: load 41.91 on 16
     cores, eight preflights, twelve `.test` binaries) as a process table of
     eight `make`, eight `go`, twelve test binaries and their compilers, all
     younger than 15 minutes at full share: status `ok`, no trigger, and
     `TestImplementLoadIsQuietUnderOwnParallelWork` asserts the rendered
     output holds no `LOAD WARNING`.
   - `TestEightConcurrentShortLivedProcessesDoNotWarn` in
     `internal/core/machineload/live_test.go` starts eight re-executions of
     the test binary that each busy-loop for three seconds, as one owned
     process group (the first child leads it and the rest join by
     `SysProcAttr.Pgid`), reads the real machine with `Read`, and classifies
     with the default stray limit: none of the eight pids appears among the
     strays. The extreme limit is set above the live load, because the test
     machine's real load is not the subject. It then classifies the same
     snapshot with the eight children's age and CPU time advanced past the
     limit and asserts all eight are named, so the test can fail. Cleanup
     kills the group and proves it gone with a fresh `Read`, as the resolved
     iss-2609210828122412 requires of any test that spawns load.
4. **Extreme load gives the load and the core count, and carries on.**
   Sections 3 and 5. `TestExtremeTriggerIsStrictlyAbove` in
   `classify_test.go` (64.0 on 16 cores is quiet, 64.1 warns; a five-minute
   or fifteen-minute value above the limit alone does not warn);
   `TestImplementLoadReportsLoadAndCores` in `implement_load_test.go` (the
   line carries all three averages, the online core count and the limit's
   derivation; exit 0).
5. **In an autonomous run, the same warning is written to the run log.**
   Section 6. In `internal/core/implement/load_test.go`, under a temporary
   `HOME`: `TestLoadWarningIsLoggedInALiveRun` (a joined first session; one
   `load` line with `triggers`, attributed to that session),
   `TestNoEventOutsideARun` (no joined session: no directory and no file
   created), `TestNoEventWhenQuiet`, `TestLoadIsVerbOwned` (`Run.Log` refuses
   `load`). In `implement_load_test.go`:
   `TestLoggedEventRendersToThePrintedWarning` (the event read back through
   `ReadLog` renders byte-identical to the printed block, masked names
   included) and `TestRunLogFailureIsLoudAndExitsZero`.
6. **Settings file limits; defaults from the online core count; malformed is
   loud and falls back.** Section 4. In
   `internal/core/machineload/limits_test.go`: `TestDefaultLimitsFromCores`
   (16 cores gives 64, 10 gives 40, stray 30),
   `TestLimitsFileSetsEitherOrBoth`, and `TestMalformedLimitsFallBackWhole` (a
   table: unknown key, repeated key, non-numeric, zero, out of range,
   oversize, symlink, group-writable, another owner; each yields both
   defaults and a report naming the line and fault class, never the value).
   `TestImplementLoadMalformedLimitsIsLoud` in `implement_load_test.go`
   asserts the `LOAD CHECK SETTINGS UNUSABLE` line appears even when the
   status is `ok`.
7. **Once at the start of preflight and the eval harness, never per package;
   skipped on CI with a logged reason.** Section 7. In
   `internal/core/lint/loadcheck_placement_test.go` (beside the preflight
   gate pins): `TestLoadCheckIsPreflightsFirstPrerequisite` (and the marker
   line follows the prerequisite line), `TestEvalHarnessRunsTheLoadCheckOnce`
   (exactly one invocation in `TestMain`, before `m.Run()`),
   `TestNoOtherTestMainRunsTheLoadCheck` (no `TestMain` in any other
   `_test.go` file calls the verb or `CheckLoad`; the verb's own unit tests
   call it from ordinary test functions, which run on demand, not at a lane's
   start), `TestCIRunsTheSkipStepBeforeEachHarness`
   (both harness jobs carry the step before `make smoke` or `make
   evals-cold-reading`, and no workflow runs `make preflight`), and
   `TestHarnessJobsRunOnHostedRunners` (the exemption's failable half). In
   `internal/core/implement/load_test.go`: `TestCheckLoadSkipsOnCIWithReason`
   (with `GITHUB_ACTIONS=true`, and separately `CI=true`, the injected reader
   fails the test if called; the result is `skipped` with the variable named),
   and `TestCIValueFalseIsNotCI`.
8. **macOS and Linux with no new dependency; elsewhere, "could not check".**
   Section 2 and section 8. In `internal/core/machineload`:
   `TestParseLoadavgSysctlBytes` (the 23-byte form, including the restored
   NUL), `TestParseDarwinPS` (a fixture listing with names containing spaces,
   full paths reduced to base names, `etime` with days, `time` with minutes
   past 59), `TestParseLinuxProc` (an `fstest.MapFS` of `/proc` with a `comm`
   holding `) (`, a zombie, a process that vanished between listing and
   read, a `status` `Uid:` line and an `auxv` `AT_CLKTCK`),
   `TestParseOnlineRange`, `TestReadSeesThisProcess` (the real reader on
   whichever platform runs it: CI's macOS and Linux legs cover both; the
   snapshot holds `os.Getpid()` with `os.Geteuid()`, at least one core and
   non-negative loads), `TestMachineLoadImportsOnlyTheStandardLibrary` (no
   import path with a dot in its first element), and
   `TestMachineLoadCompilesElsewhere` (`go vet` of the package with
   `GOOS=freebsd`). `TestImplementLoadCannotCheck` in
   `implement_load_test.go` injects `ErrUnsupported` and a failing process
   read: both print the `LOAD CHECK UNAVAILABLE` line and exit 0, and the
   second still reports the load average. `go.mod` is unchanged in the diff.

## Risks and not verified

- **Lifetime share lags a late spinner.** A program idle for days that starts
  spinning today reaches a 0.9 lifetime share late, or never. Both recorded
  incidents spun from birth, which is the shape the rule catches; a
  recent-share reading would need a second sample on Linux and a different
  definition per platform. Accepted; the extreme trigger still sees the load.
- **A deliberate long-running program of the caller's warns on every run.** A
  virtual machine or an indexer at a full core is named each time. The only
  knob is `stray-minutes`; an allowlist is out of scope.
- **Cached and swallowed harness runs.** A cached eval lane (`ok (cached)`,
  seen here on the second of two identical runs) starts no harness, so no
  harness check runs; the preflight's own check is unaffected. An agent with
  no terminal that runs the eval lanes on their own, outside a run, sees the
  harness's report only when the lane fails or runs with `-v`. Inside a run
  the event still lands.
- **Up to three events per preflight in a run.** The preflight's check and
  each harness start log separately; `within_preflight` lets a reader fold
  them. The criterion asks for the warning to be logged whenever it is
  printed, so none is suppressed.
- **Attribution to the first session.** The event names the run's first
  session even when a second session's lane ran the preflight. Session records
  of a run that crashed without leaving make later checks log into that run;
  harmless, and visible in `abcd implement`.
- **Linux `hidepid`.** With `/proc` mounted `hidepid=2`, other accounts'
  processes are invisible and their count reads 0. Not detected.
- **The `LOAD` rule's wording.** The remedy follows the loadtrust fix round's
  text (the one-job group, the re-checked handle, no kill by pattern), which
  is on its branch and not yet on `main` at this tip. If that text changes
  before it merges, the remedy lines follow it in the same change.
- **Not verified**: The Linux readers against a real `/proc` (no Linux host in
  this session; the field positions are from proc(5), and the CI Linux leg
  runs `TestReadSeesThisProcess`); that `/sys/devices/system/cpu/online` is
  what glibc's `_NPROCESSORS_ONLN` reads on every distribution; the `AT_CLKTCK`
  auxv entry on every architecture (100 is the fallback); that
  `hw.activecpu` equals `_NPROCESSORS_ONLN` on a Mac with cores taken offline
  (equal here, all 16 online); the day form of macOS `ps` `time`; that
  `/dev/tty` opens from a test binary started by `go test` in a terminal
  session; and GitHub-hosted runner core counts. Probed on this machine and
  so verified: the `vm.loadavg` parse in Go, `hw.activecpu`, the macOS `ps`
  keywords and cost, the lifetime-share margin (highest 0.19 among programs
  over 30 minutes old at load 18 to 26), package-list mode hiding a passing
  `TestMain`'s output, eval-lane caching, and GNU Make 3.81 passing a
  target-specific exported variable to prerequisites.
