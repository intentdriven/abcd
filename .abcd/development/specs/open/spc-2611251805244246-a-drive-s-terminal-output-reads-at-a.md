---
id: spc-2611251805244246
slug: a-drive-s-terminal-output-reads-at-a
intent: itd-2611141610192096
origin: researcher-authored
production_mode: hand-written
---
# A drive's terminal output reads at a glance

## Summary

This spec delivers itd-2611141610192096. `abcd drain --drive` starts with at most seven short lines, and `?` at its question shows the full plan wrapped at word breaks. While the drive runs at a terminal, each event is one line cut to the window, and below the events three lines are redrawn in place: the state line of itd-2611141610197753, the progress of the lanes and the lane in hand. Where the output is not a terminal, or `TERM` is `dumb` or unset, the drive writes plain lines with whole ids and no control code. The log holds the full sentence of every event, its path is shown once after the yes, and no line the drive writes quotes a command meant for a host session.

The design is the footer the lab drew and measured (lab-261114145658-67e1f9b, findings F-13, F-20 and F-21, probe p6-output-renders-2), which the product thinker chose on 2026-11-15 (decision 3). The lab's prototype is the patch `state/patches/p6-drivescreen.patch`: 235 lines of renderers, 213 of a model, and 360 that read the drive's state back out of its sentences. The renderers and the model are the starting point for steps 1 and 2. The sentence patterns are not taken: they read the state wrong twice (F-11), and the intent rules them out. In their place the loop hands the surface a typed event (step 3).

The intent carries no open question. Decisions 1 and 2 (the option menu and the stop after the lane in hand) are built by itd-2611150636498012 and not here. No step adds a command, a flag, a dependency or a colour, and no step raises the run-state schema version.

Five steps, each one pull request. Steps 1 and 2 edit no file of the loop package and wait on nothing. Step 3 edits the loop and waits on step 4 of spc-2611141617199772, which makes the drive's state a value. Step 4 wires the drive and needs steps 2 and 3. Step 5 puts the full sentence one step away and waits on step 5 of spc-2611141617199772.

## Approach

### What is on the default branch today

- **The start.** `driveConsent` (`internal/surface/cli/drain_drive.go`) prints a heading and thirteen labelled lines (`cap`, `estimate`, `bound`, `review`, `runner`, `route`, `profile`, `gates`, `first`, `settings`, `updates`, `releases`, `others`), then `Start? [y/N] `, and reads one line. The program wraps nothing, so the terminal breaks the lines where the window ends: 34 rows at 50 columns, ten of fourteen lines wrapped (F-20). `--yes` prints the same lines and asks nothing. With no terminal on standard input and no `--yes` it refuses.
- **The lines.** `driveLog` (`internal/surface/cli/drain_drive.go`) returns `say`, the one writer of the drive's lines: each line goes to the output as it is and to the log with a time in front. The log is `.abcd/.work.local/logs/drain-drive-<stamp>.log`. `driveLog` is called twice in one drive when the live proof waits out a limit (`driveProve`, then `driveRun`), so such a drive has two logs. No line names the log (F-21).
- **The loop's sentences.** `terminal.say` (`internal/core/implement/loop/terminal.go`) formats a sentence, masks it with `termsafe.Sanitize` and hands it to `TerminalOptions.Say`. It has 59 call sites in `terminal.go` and seven more in `drive_profile.go`, `limit_evidence.go` (through the closure `terminal.go` hands it) and `preflightcheckout.go`. A few carry structure: `sayMove` (judged, handed back, finished, opened), `sayStep` (an agent ran, a stage done, a note), `sayWait` (a wait and its reason), `startCall` and `sayCallStill` (an agent or a preflight in flight). The rest are single sentences about a stop, a reservation or a take-over.
- **A host session's words in a wait.** Several waits pass a step's `Next` as their reason: the `sleep` and `idle` of a drain's move, the pace window's `sleep`, and `failed` with `"%s did not move: %s", runID, res.Next`. Where the lane awaits an agent's receipt, `Next` tells a host session to start an agent and run `abcd implement receipt <path>`. Nine wait lines of the lab's fixture run say so to a person at a terminal (F-21).
- **The second stream.** The drive's signal messages and its diagnostics go to standard error (`errw`, made by `sharedStderr` in `runDriveLocked`). A file is passed through as it is, so at a terminal those lines are written beside `say`'s with no shared lock.
- **The end.** `renderDriveEnd` prints how the drive ended and what it left, through `render`, and `driveEndExit` returns the exit whose message the root command prints on standard error.
- **What the repository already has for a terminal.** `internal/term` has the one terminal check (`IsTerminal`), the window's size (`Size`, through `golang.org/x/term`, with `COLUMNS` as its fallback), the colour-mode resolver (`ResolveColorMode`) and the locale check (`UTF8Locale`). `internal/termsafe` masks control codes and hidden runes. `internal/term/ptytest` opens a pseudo-terminal of a given size for a test. `boardWidth` (`internal/surface/cli/board_width.go`) is the pattern for a width a test can pin: a package variable that answers the terminal's columns, or 80 off a terminal.
- **What is not there yet.** Nothing of spc-2611141617199772 is on the default branch: no `DriveState`, no `state:` line, no set-aside. The footer's first line and its `aside` count are that spec's step 4 and step 3.

### Where the code goes, and why it is three layers

1. **`internal/surface/cli/drivescreen/`, a new package that only draws.** It imports `internal/termsafe` and the standard library, and nothing of the loop. Its inputs are its own small types (`Plan`, `Event`, `State`, `View`), a `Frame` holding the width and whether the locale takes more than ASCII, and a time. Its outputs are lines. It reads no environment, no clock and no file descriptor, so every test of it is a table of values and widths. This is the lab's package name and shape, without its `events.go`.
2. **The typed event, in the loop.** `internal/core/implement/loop/driveevent.go` declares `DriveEvent`, and `TerminalOptions.Event` receives it. The loop decides what an event is and which of its words are a host session's. It formats nothing for a window.
3. **The wiring, in the surface.** `internal/surface/cli/drain_drive_screen.go` holds `driveScreen`, the one writer: it maps a `loop.DriveEvent` and a `loop.DriveState` onto the drawing package's types, writes the log, and chooses between the live region and plain lines.

The reason for the split is itd-2611150636498012. Its option line is one more line appended to the region the drawing package returns, and its headless mode is a second consumer of the same `loop.DriveEvent` and `loop.DriveState` that never touches the drawing package. Neither is built here. What is built here is the seam: `Frame.Region` returns a slice of lines and `Live.Print` draws any number of them, so a fourth line needs no change to the redraw; and nothing the terminal shows is computed in the drawing package from anything but the two typed values.

### Measuring and cutting (every criterion that says "wraps")

A line never wraps because the program cuts it to one column less than the window before it is written. One column less, because a line that exactly fills the window leaves some terminals' cursor on the next row, and then the count of rows to go back up over is wrong.

- **Order.** Text the drive did not write (an issue's title, a reason, a path) is masked with `termsafe.Sanitize` first, then measured, then cut. The mask can change a text's length, so measuring first would be measuring something that is never drawn.
- **Columns.** `Columns` counts one column for a rune below U+1100 and two for a rune at or above it, with the drive's own three glyphs (`…`, `█`, `░`) listed as one. This over-counts some narrow runes, and over-counting only makes a line shorter; it never lets one wrap. The alternative is a width table from a library, which is a new dependency for a case the over-count already makes safe, and the intent rules a terminal library out.
- **The cut.** `Cut` keeps the head of a line and ends it with `…`. Where the locale is not UTF-8 (`term.UTF8Locale`), `Frame.ASCII` is set and the glyphs are `...`, `#`, `.` and ` - ` for `·`.
- **Ids.** At a terminal an id reads as `…` and its last four digits. Two ids of one drive that share those four digits are told apart by lengthening both to the shortest ending that differs, so a short id never names two records. A run's id reads as its issue's short id, as the mock-ups show ("the implementer of …0103"): the view keeps the run-to-issue pairs it was told at each lane's opening. Ids inside the state line's `on` part are shortened the same way, by their record-id shape (`iss-`, `itd-`, `run-` and digits) and not by the sentence around them. Plain output and the log keep every id whole.
- **Times.** A short time with no zone (`09:17`) is the machine's local time, because it is read against the clock in the room. Plain output and the log write the time in UTC with its zone, as the log does today. An elapsed time reads in seconds below a minute and in hours and minutes above it (`42s`, `5m`, `2h44m`), so after its first minute a region line changes at most once a minute.
- **Below the scope.** The intent's scope is 50 columns or more. Down to 20 columns the same cut applies and the lines are short and poor. Below 20 the drive writes plain lines.

### The start (criterion 1)

`Plan` holds the values `driveConsent` already has. `StartLines` returns six lines and the question:

```text
abcd drain --drive: works through the issues
  lanes    up to 8, about 40 agent runs
  ends     8h on any one wait (--until: a time)
  runs on  claude, login walled
  review   ruthless and security reviewers
  first    one short live check of the login
Start? [y/N, ? shows the full plan]
```

- The six lines are written to fit 49 columns with the longest values the drive can give them (a cap of three digits, a count of runs of five, an end time under `--until`). The profile's name and the reviewers' names are the only free text, and each is cut. So the start is the same at 50, 80 and 120 columns.
- Under `--until` the `ends` line reads `at <date and time> (--until)`, in local time, and the full plan gives it in UTC with its zone.
- The `review` line names the reviewers and no class. The models and efforts are in the full plan. The product thinker ruled so on 2026-11-25: the reviewers' names only, with the models and efforts on `?` and in the log. The shipped criterion of itd-2611101419006280 keeps its letter, since the pair is named before the question and no class name is shown alone.
- `PlanLines` is the full plan: every line of today's start, its label in a column and its text wrapped at word breaks to the window, a word longer than the room standing on a line of its own. The answer `?` prints it and asks again; any other answer but `y` or `yes` is a no, as today. One `bufio.Reader` serves every read of the question, so a second answer typed ahead is not lost.
- `--yes` prints the six lines and no question. The full plan is not printed then, and it is written to the log at every start, whichever way the drive was started, so what was agreed to can be read later.
- Where the output is not a terminal the six lines are written uncut.

### One log, and its path (criterion 4, the path)

The log is opened once, straight after the yes, and handed to the live proof's wait and to the run; `driveProve` no longer opens one of its own. Its path is then said once, as two lines, because the path is 55 characters and one line of it would wrap or be cut at 50 columns:

```text
09:00 log in .abcd/.work.local/logs/
      drain-drive-20261112T090000Z.log
```

A drive that re-executes itself onto a new release is a new process with a new log, and says that log's path once. Where the log cannot be opened the drive says so in one cut line, as it does today in a long one.

### The typed event (criterion 4, the host command; the ground for criteria 2 and 3)

`DriveEvent` (`driveevent.go`) is what the loop hands the surface for each line it would have said:

| Field | What it holds |
| --- | --- |
| `At` | the time |
| `Kind` | one of the kinds below |
| `Issue`, `Run`, `Lane`, `Stage`, `Role` | the records and the stage the event is about, each empty where it has none |
| `Title` | the issue's title, on an opening and a take-over |
| `Opened` | the drain's count of lanes opened so far, on an opening and a take-over |
| `Wait` | how long a wait will last |
| `Cause` | the clause the drive itself wrote, with whole ids: "the implementer of run-… did not run" |
| `Detail` | what follows the cause in the sentence |
| `Host` | true where `Detail` is a step's `Next` or a refusal's `Remedy` written for a host session |
| `Acts` | true where a person may have to act on the event |
| `Quiet` | true for an event that changes the region and is no line at a terminal |
| `Sentence` | the full sentence for the log |

The kinds are `judged`, `handed-back`, `held-back`, `passed-over`, `set-aside`, `opened`, `took`, `stage-done`, `stage-running`, `agent-started`, `agent-ran`, `still-running`, `wait`, `finished`, `stopping` and `note`.

- **One choke point.** `terminal.say` stays and becomes the `note` kind: its sentence is the event's `Sentence` and its `Cause`. So all 66 call sites emit an event with no edit, and only the sites whose short form needs a field are rewritten to build the event themselves: `sayMove`, `sayStep`, `sayWait`, `startCall`, `sayCallStill`, the set-aside, the take-over and take-up, the stop's lines, and the preflight's start in `preflightcheckout.go`. This keeps the diff in `terminal.go`, the most contended file of the package, to those functions.
- **Say or Event, never both.** Where `TerminalOptions.Event` is set the loop hands each line to it and not to `Say`. Where it is nil, `Say` gets the event's `Sentence`, so `WaitOutProof` and every existing caller and test of `Say` work as they do today.
- **A host session's words.** `Host` is set from the step's typed result, a lane that awaits an agent's receipt, and never from the words of `Next`. Step 4 of spc-2611141617199772 makes the same judgement when it decides whether a wait's remedy may be the state line's act, and this step calls that one function. Where that step left the judgement inline, this step lifts it into one function that both call, so it is still written once. `Sentence` is composed without a `Detail` marked `Host`. The log therefore no longer tells its reader to run `abcd implement receipt` on a run the drive is driving. Nothing is lost by it: the step's `Next` is still in the run's record, where `abcd implement status` shows it to the host session it was written for.
- **Who may have to act.** `Acts` is set at the call site, for a hand-back, an issue held back, an issue passed over for a person, a set-aside, a lane found stuck, a stop that could not be kept or applied, a reservation that was not released, and the drive's end. A test lists them, so a new line of that kind cannot be added without deciding.
- **Quiet events.** An agent starting, an agent having run, a call still running and evidence kept are `Quiet`: at a terminal they change the lane-in-hand line and are no line of their own, as the mock-ups show. In plain output and in the log each is a line.

### Event lines and plain lines (criteria 2 and 3)

`Frame.EventLine` is an event's line at a terminal: the short time, the short form, cut. `PlainLine` is its line elsewhere: the UTC time, the short form with whole ids, uncut and with no control code.

| Kind | Short form |
| --- | --- |
| `judged` | `<id> judged: <its outcome's first clause>` |
| `opened` | `<id> lane opened` |
| `took` | `<id> taken over` or `<id> taken up` |
| `stage-done` | `<id> <stage> done` |
| `stage-running` | `<id> <stage> running` |
| `wait` | `wait <duration>: <cause>` |
| `handed-back` | `<id> handed back: <cause>` |
| `held-back`, `passed-over`, `set-aside` | `<id> held back: <cause>`, and so on |
| `finished` | `<id> finished` |
| `stopping` | `stopping: <cause>` |
| `note` | the sentence |

The short forms are composed from the event's fields. No short form is made by cutting or matching a sentence, except the `note` kind, which is its sentence cut to the window. An event a person may have to act on is told apart by its words (`handed back`, `held back`, `set aside`, `stopping`), never by a colour or a glyph alone.

### The region (criterion 2)

`View` is what the region is drawn from: the latest `State`, the cap, the lanes opened, and the lane in hand (its issue, title, stage, and when the stage began). `View.Apply` takes an event and `View.Set` takes a state. `Frame.Region` returns three lines:

```text
stalled 2h44m · on outage: …0101… · needs a look
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
…0101 The renderer panics on an…  implement 2h48m
```

- **The state line.** Its words are those of spc-2611141617199772: the state word, the elapsed time, `on <what>` and the act. This spec draws them and defines none. `State` carries the four parts and not the composed line, because fitting the line means cutting its middle: the act is kept whole, the `on` part takes the room left, and only when the act alone is too wide is the line's end cut. Where the window is wide enough, the line is character for character the one that spec's composing function returns, and a test in step 4 holds the two together. At 80 columns the mock-up's line is whole.
- **The progress line.** A bar of at most sixteen cells, the lanes opened over the cap, and the four counts from the state value. Where the words do not fit it falls back to `1 done 1 fly 0 aside 1 back`, and where the drain has no cap it reads `lanes:` and the counts with no bar.
- **The lane in hand.** The short id, the title cut to the room left, the stage and how long it has run; `(agent)` after the stage while an agent is in its call. Between lanes it says what the drive is doing (`judging`, or `no lane in hand`).

### Redrawing in place (criterion 2)

`Live` writes three control sequences and no other: cursor up *n* rows (`ESC [ n A`), carriage return, and erase from the cursor to the end of the screen (`ESC [ J`). `Live.Print(lines, region)` goes up over the region that stands, erases it, writes the new event lines, and writes the region again. One write holds all of it, so the screen is never seen half drawn.

- **When.** On every event, on every change of the state value, and on a tick of one second. A tick writes only when the three lines differ from the ones standing, which after a stage's first minute is at most once a minute. A second is short enough that a resized window is right before a person has finished looking at it.
- **Resize.** The width is read from the terminal at each draw (`term.Size`), which is one system call. The drive installs no handler for the window-change signal: its signals are a contract of their own (today the first asks for a stop, the second kills, and a hang-up is outlived; `drain_drive_signal.go`), and a tick needs none. What the first interrupt means is changed by itd-2611150636498012 and not here. When the window has become narrower, the terminal may already have re-wrapped the three lines standing, so `Live` remembers each standing line's columns and goes up by the rows they now take.
- **The second stream.** While a region stands, the drive's standard error goes through the same writer when it is the same terminal as its output (`os.SameFile` on the two descriptors): a line written there erases the region, is written above it, and the region is drawn again. A standard error that is a file or another terminal is written as it is.
- **A writer the drive does not own.** A child process that writes to the terminal directly lands inside the region. The next draw goes up three rows from where the cursor then is, so the cost is one stale copy of the region left in the scrollback, and the screen's last three lines are right again. Agents and the preflight write to files and not to the terminal, so this is rare; the step that wires the region checks which children inherit the terminal and says so in its pull request.
- **The end.** `Live.Close` leaves the region as the screen's last lines, with the final state, and what `renderDriveEnd` prints goes below it. At a terminal its lines are wrapped at word breaks by `PlanLines`' wrap, and a command a person is to type (`to resume: …`) stands whole on a line of its own and is never cut. The exit's own message is the root command's and is not changed.

### Terminal or plain (criterion 3)

One function decides, behind a package variable that tests pin, in the pattern of `boardWidth`:

```go
// driveWindow reports the window the drive's lines are drawn for, and whether
// a region may be redrawn in it.
var driveWindow = func(w io.Writer, getenv func(string) string) (cols int, live bool)
```

`live` is true only when the writer is a file that is a terminal, `TERM` is set and is not `dumb`, and the window is 20 columns or more. `NO_COLOR` is not read by it: the region is redrawn under `NO_COLOR`, as the intent's third criterion says.

- **Plain.** One line per event, `Quiet` ones included, each the UTC time and the short form with whole ids. A `state: <line>` line, with whole ids, each time the state word, its `on` part or its act changes, which is the line step 4 of spc-2611141617199772 prints. No control code of any kind.
- **Live.** The event lines and the region. No `state:` line is printed, since the region's first line is the state. The log still gets one.
- **Colour.** None is written in either mode. The intent puts colour out of scope ("added last, if at all"), every criterion is met without it, and a build with no colour code cannot convey anything by colour alone. A test holds that no output of the drive carries a colour code, with `NO_COLOR` set and unset, so a later change that adds colour meets a failing test and has to take the repository's resolver.
- **`--json`.** The lines go to standard error then, as today, and the same decision is made on that writer.

### One step away (criterion 4)

The intent's scope says that until the option menu is built, one step away means `abcd implement status` in a second terminal and the log's path. So `abcd implement status` shows both.

The drive's writer keeps a small file, `.abcd/.work.local/logs/drive-attention.json`, written whole by rename each time an event with `Acts` set arrives: a version of its own, the time, the event's kind and ids, its full sentence, and the log's path. `abcd implement status` (`internal/surface/cli/build.go`) prints, under the state line that step 5 of spc-2611141617199772 puts there, the log's path and the last such sentence, wrapped at word breaks; `--json` carries the same fields. A missing, unreadable or newer-version file prints nothing and names why on standard error, as the state file's reader does.

It is a file of its own and not a field of the state value, because the state value is the loop's, is versioned by the other spec and is read by the status line and the board, none of which wants a paragraph. It is the surface's file because the sentence and the log's path are the surface's to know. The headless mode of itd-2611150636498012 can serve the same event from memory and need not read it.

### Coordination

- **The loop package.** Step 3 is the only step that edits `internal/core/implement/loop`. It does not start until that package's open lanes are on the default branch: the remaining steps of spc-2611201455051897 (the drain's bundles, which today's surge lands first, in `drain.go`, `terminal.go` and others) and steps 1 to 4 of spc-2611141617199772, each of which edits `terminal.go`. It is cut from the default branch as it then stands.
- **`drain_drive.go`.** Steps 1, 4 and 5 here edit it, and so do steps 3 and 4 of spc-2611141617199772. Each lane reserves it, so they land one at a time. Step 1 touches only `driveConsent`, `driveProve`'s log and the opening of the log, and may land before or after that spec's step 3.
- **The documentation pages.** `commands/drain.md` and `.abcd/development/brief/04-surfaces/35-drain.md` are edited by steps 1 and 4 here and by step 5 of spc-2611141617199772. Each step edits only the paragraphs it names.
- **The start's seven lines are full.** Three rulings of 2026-11-25 on other work touch the start: the drive ends at an end time that defaults to seven days and no longer at a drain's cap, which changes what the `lanes` and `ends` lines say; the start says that a reboot, a logout or a crash ends the drive and gives the command that resumes it (itd-2611041656024409); and the start says what the first and the second interrupt do (itd-2611150636498012). None is built here, and the mock-up's two lines state what the drive does today. `Plan` holds values and `StartLines` alone holds the words, so each of those changes is to one function. Where its words go (one of the seven lines, the full plan, or a line after the yes) is for the spec that builds it to say, and one that needs an eighth line amends the first criterion.

## How the Acceptance Criteria are met

The criteria are numbered in the order the intent lists them. The third and fourth each hold several clauses, lettered here.

1. **The start is at most seven lines, none wraps, and `?` shows the full plan wrapped at word breaks.** `StartLines` and `PlanLines`, and `driveConsent`'s loop on `?` (step 1). Proved by `start_test.go` (seven lines; every line under 50 columns with the longest values; the plan wrapped at 50, 80 and 120 with no line over the window and no word split) and by `drain_drive_start_test.go`, which runs the command with the window pinned at 50 and at 80.
2. **Each event is one line no wider than the window, the region is redrawn in place, its first line is the state line, and nothing wraps at 50, 80 or 120.** `Frame.EventLine`, `Frame.Region` and `Live` (step 2), fed by the typed event (step 3) and wired by `driveScreen` (step 4). Proved by `footer_test.go` and `live_test.go` on the recorded fixture run at the three widths, through a test terminal that interprets the three sequences and counts wrapped rows; by `drain_drive_screen_test.go`, which drives the command's writer at 50 and 80 columns and compares the region's first line with the other spec's composed state line; and by one pseudo-terminal test at 50 columns.
3. a. **Not a terminal, or `TERM=dumb`: plain lines, no control code, whole ids, a `state:` line on a change.** `driveWindow` and `PlainLine` (steps 2 and 4). Proved by `drain_drive_plain_test.go`: no byte below 0x20 but the newline and no 0x1b in the whole output of a fixture run, every id whole, one `state:` line per change, for a pipe, for `TERM=dumb` at a terminal and for `TERM` unset.
   b. **`NO_COLOR` at a terminal: no colour code, and the region is still redrawn.** `driveWindow` does not read `NO_COLOR` (step 4). Proved by `drain_drive_screen_test.go`: with `NO_COLOR=1` the output carries the cursor-up sequence and no `ESC [ … m`.
   c. **Nothing is conveyed by colour alone.** No colour is written (steps 2 and 4). Proved by the same test with `NO_COLOR` unset, and by `event_test.go`, which holds that each kind a person may act on differs from the others in its words.
   d. **A title carrying control codes is masked before it is measured or drawn.** `Cut` and `Columns` take masked text only, and the mapping in `driveScreen` masks every field it did not write (steps 1, 2 and 4). Proved by `text_test.go` and `footer_test.go` with the lab's hostile titles (an erase-screen code, a cursor move, a zero-width run, a title of wide runes): no control byte in the drawing and no line over the window.
4. a. **The full sentence is in the log.** `DriveEvent.Sentence`, written by `driveScreen` for every event, quiet ones included (steps 3 and 4). Proved by `drain_drive_screen_test.go`: the log of a fixture run holds each event's sentence whole, at a window of 50 columns.
   b. **The log's path is shown once at the start.** One log per drive and its two lines (step 1). Proved by `drain_drive_start_test.go`: the path appears once, whole, with no line over 50 columns, and a drive whose proof waited has one log file.
   c. **The full sentence is one step away at the terminal.** The attention file and `abcd implement status` (step 5). Proved by `drain_drive_attention_test.go`.
   d. **No line shown to a person quotes a command meant for a host session.** `DriveEvent.Host` and the composed `Sentence` (step 3), and short forms that take only `Cause` (step 2). Proved by `driveevent_test.go` (a wait at a lane that awaits a receipt carries `Host` and a `Sentence` without `abcd implement receipt`) and by `drain_drive_plain_test.go` and `drain_drive_screen_test.go` (no line of the terminal, the plain output or the log of the fixture run names `abcd implement receipt` or `abcd implement step`).

## Open Questions

_All resolved: no question is open, and the checks below are kept for the reader, each with the rule its step follows._

The one product question the planning raised, what the start's `review` line holds, was answered by the product thinker on 2026-11-25: the reviewers' names only, with the models and efforts on `?` and in the log. The spec is written on that answer.

Three things a step checks on the default branch when it is cut, each with the rule it follows whichever way the check comes out:

- How step 4 of spc-2611141617199772 hands the state value to the surface for its `state:` line. Step 4 here reads the value through that same hand-off and adds no second one.
- Whether the loop holds an issue's title where a lane opens. If it does not, step 3 reads it from the issue's record through the reader the judgement already uses, and an opening whose title cannot be read carries none; the lane-in-hand line then shows the id, the stage and the time.
- Which children of the drive inherit its terminal (see "A writer the drive does not own").

## Footprint

- packages: internal/surface/cli/drivescreen/text.go, internal/surface/cli/drivescreen/start.go, internal/surface/cli/drivescreen/event.go, internal/surface/cli/drivescreen/footer.go, internal/surface/cli/drivescreen/live.go, internal/core/implement/loop/driveevent.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/preflightcheckout.go, internal/surface/cli/drain_drive.go, internal/surface/cli/drain.go, internal/surface/cli/drain_drive_screen.go, internal/surface/cli/build.go, commands/drain.md, docs/reference/cli/commands.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/27-implement.md
- tests: internal/surface/cli/drivescreen/text_test.go, internal/surface/cli/drivescreen/start_test.go, internal/surface/cli/drivescreen/event_test.go, internal/surface/cli/drivescreen/footer_test.go, internal/surface/cli/drivescreen/live_test.go, internal/surface/cli/drivescreen/vt_test.go, internal/surface/cli/drivescreen/testdata/run-three-lanes.json, internal/surface/cli/drivescreen/testdata/footer-50.txt, internal/surface/cli/drivescreen/testdata/footer-80.txt, internal/surface/cli/drivescreen/testdata/footer-120.txt, internal/surface/cli/drivescreen/testdata/plain.txt, internal/core/implement/loop/driveevent_test.go, internal/surface/cli/drain_drive_start_test.go, internal/surface/cli/drain_drive_screen_test.go, internal/surface/cli/drain_drive_plain_test.go, internal/surface/cli/drain_drive_attention_test.go, internal/surface/cli/drain_drive_test.go, internal/surface/cli/drain_drive_reviewset_test.go, internal/surface/cli/drain_drive_until_test.go, internal/surface/cli/drain_drive_update_test.go, internal/surface/cli/drain_drive_reprove_test.go, internal/surface/cli/drain_drive_stage_test.go

## Steps

1. The start in seven lines, the full plan on `?`, and one log with its path
   - criteria: 1, 3d, 4b
   - needs: none
   - packages: internal/surface/cli/drivescreen/text.go, internal/surface/cli/drivescreen/start.go, internal/surface/cli/drain_drive.go, internal/surface/cli/drain.go, commands/drain.md, docs/reference/cli/commands.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/surface/cli/drivescreen/text_test.go: `Columns` counts a rune at or above U+1100 as two and the drive's own glyphs as one; `Cut` never returns a line wider than asked, ends a cut line with `…`, or `...` under ASCII, and leaves a line that fits alone; a title carrying an erase-screen code, a cursor move and a zero-width run is masked before it is measured, with no control byte in the result; the wrap breaks only at spaces and puts a word longer than the room on a line of its own; two ids sharing their last four digits get short forms that differ. internal/surface/cli/drivescreen/start_test.go: `StartLines` is seven lines, each under 50 columns with a cap of 999, 99,999 runs, a bound under `--until` and a profile name of 64 characters; `PlanLines` at 50, 80 and 120 holds every label of today's start and no line over the window. internal/surface/cli/drain_drive_start_test.go, with the window pinned at 50 and at 80 through `driveWindow`: the start is at most seven lines; `?` prints the full plan and asks again, and `y` after it starts; `--yes` prints six lines and no question; the full plan is in the log whichever way the drive started; the log's path is said once, in two lines under 50 columns, and a drive whose proof waited out a limit has one log file; a pipe gets the six lines uncut. The tests that pin today's labelled lines in the start, most of them through `--yes`, are rewritten to pin those lines in the full plan (on `?`) and in the log: `TestTheConsentNamesTheReviewPair` in internal/surface/cli/drain_drive_reviewset_test.go, which also pins that the short start's `review` line names both reviewers before the question and no model; the `cap:`, `runner:` and `profile:` assertions in internal/surface/cli/drain_drive_test.go; the `bound:` and estimate assertions in internal/surface/cli/drain_drive_until_test.go, which also pin the short `ends` line with and without `--until`; the `releases:` and `settings:` assertions in internal/surface/cli/drain_drive_update_test.go; `TestTheConsentSaysAnUpdateIsProvedInTheRun` in internal/surface/cli/drain_drive_reprove_test.go; and `TestThePublicTextSaysWhatAProfileIs` in internal/surface/cli/drain_drive_stage_test.go, which answers `?` before its no. The two checks in internal/surface/cli/drain_drive_update_test.go that a resumed start shows no start look for today's heading, `works through this repository`, and would pass on nothing once the heading changes; they are rewritten to the new heading. Watched failing first: on the default branch the start is fourteen lines and `?` is a no. This step declares `driveWindow` in `drain_drive.go` and uses only its width; step 4 uses its second answer. The command page and the brief page say what the start shows and that `?` shows the rest; the verb's help in `drain.go` says the same in place of its sentence on what the start shows, and `docs/reference/cli/commands.md` is generated again from it, since its drift test fails on a help text changed alone. Waits on no other step and on no file of the loop package.
2. The drawing package: event lines, plain lines, the region and its redraw
   - criteria: 2, 3a, 3c, 3d
   - needs: 1
   - packages: internal/surface/cli/drivescreen/event.go, internal/surface/cli/drivescreen/footer.go, internal/surface/cli/drivescreen/live.go
   - tests: internal/surface/cli/drivescreen/testdata/run-three-lanes.json is the lab's recorded fixture run written as typed events and states (39 events over 8h35m, three lanes, an outage, a hand-back at the judgement), with whole ids. internal/surface/cli/drivescreen/event_test.go: each kind's short form from its fields; a `wait` whose `Detail` is a host session's shows its cause and nothing of the detail; the kinds a person may act on differ in their words; `PlainLine` keeps ids whole, is not cut and carries no byte below 0x20; internal/surface/cli/drivescreen/testdata/plain.txt is the whole run in plain lines. internal/surface/cli/drivescreen/footer_test.go: `Region` is three lines at every width from 20 to 200, none as wide as the window; the state line keeps its act whole and cuts its `on` part, and at 80 columns is the uncut line; the progress line falls back to its short words and to no bar where the drain has no cap; the lane in hand shows `(agent)` during a call and `no lane in hand` between lanes; the lab's hostile titles and a title of wide runes draw with no control byte and no line over the window. internal/surface/cli/drivescreen/vt_test.go is a test terminal of a given width that interprets cursor up, carriage return and erase to the end of the screen, wraps a line that reaches its width, and reports the rows and how many were made by wrapping; any other escape byte fails the test. internal/surface/cli/drivescreen/live_test.go: the fixture run drawn through `Live` at 50, 80 and 120 leaves the event lines and three region lines, with no wrapped row, equal to internal/surface/cli/drivescreen/testdata/footer-50.txt, footer-80.txt and footer-120.txt; a draw with unchanged lines writes nothing; a region of four lines is redrawn as cleanly as one of three, which is the seam the option line of itd-2611150636498012 will use; after the window narrows from 80 to 50 the next draw leaves no fragment of the old region on the screen; `Close` leaves the region standing and the next write goes below it. Watched failing first: the package does not exist. Waits on step 1 for `text.go` and on no file of the loop package.
3. The loop hands the surface a typed event
   - criteria: 4a, 4d
   - needs: none
   - waits on: step 4 of spc-2611141617199772 reaching the default branch; `needs:` holds only this spec's own step numbers, so whoever starts this step holds it until then
   - packages: internal/core/implement/loop/driveevent.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/preflightcheckout.go
   - tests: internal/core/implement/loop/driveevent_test.go, on the package's fixture worlds: a drive with `Event` set gets one event per line and `Say` gets none, and a drive with `Event` nil gets on `Say` each event's `Sentence`; a judgement, a hand-back, an opening, a take-over, each stage done, the preflight's start, an agent's start and end, a call still running, a wait, a set-aside and the stop each carry their kind and their ids, and an opening carries the title and the count of lanes opened; a wait at a lane that awaits an agent's receipt carries `Host`, a `Cause` in the drive's own words, and a `Sentence` that names neither `abcd implement receipt` nor `abcd implement step`, while the run's record still holds the step's `Next`; a wait on the pace window and one on a rate limit carry no `Host`; the list of sites that set `Acts` is held by a table, and an agent's start, an agent's end, a call still running and kept evidence are `Quiet`; every field the loop did not write is masked. Watched failing first: on the default branch the wait's line quotes the receipt command (the lab's fixture run, lines 24 to 37) and `TerminalOptions` has no `Event`. Waits on step 4 of spc-2611141617199772, and so on its steps 1 to 3, because all four edit `terminal.go`, because the judgement of which words are a host session's is that step's and is used here and not written twice, and because the set-aside's line does not exist before its step 3. Waits also on the loop package's other open lanes, the remaining steps of spc-2611201455051897 among them, being on the default branch.
4. The drive draws through one writer: the region at a terminal, plain lines elsewhere
   - criteria: 2, 3a, 3b, 3c, 3d, 4a, 4d
   - needs: 2, 3
   - packages: internal/surface/cli/drain_drive_screen.go, internal/surface/cli/drain_drive.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/surface/cli/drain_drive_screen_test.go, with `driveTerminal` standing in a loop that replays the fixture run's events and states, the window pinned at 50 and at 80 and the output read through the test terminal: every event is one line under the window, the last three rows are the region, and its first row is the state line the other spec's composing function returns for the same value, cut as `Region` cuts it; a tick with nothing changed writes nothing, and a tick past a minute's turn redraws; a line written to standard error while the region stands lands above it and the region is whole after it; with `NO_COLOR=1` the output holds the cursor-up sequence and no colour code, and with it unset no colour code either; the log holds every event's full sentence, the quiet ones included, and a `state:` line per change; no line of the screen or the log names `abcd implement receipt`; after the end the region stands with the final state and the end's lines are below it, wrapped, with the resume command whole on its own line. One test opens a pseudo-terminal of 50 columns (`internal/term/ptytest`) and asserts the same of the real writer. internal/surface/cli/drain_drive_plain_test.go: for a pipe, for `TERM=dumb` at a terminal and for `TERM` unset, the output holds no 0x1b and no byte below 0x20 but the newline, every id is whole, each event is one line with a UTC time, quiet events included, and a `state:` line follows each change of state; under `--json` the same holds of standard error and standard output is the one JSON value. Watched failing first: on the default branch the screen at 50 columns wraps (198 rows for the fixture run, F-13) and nothing is redrawn. The command page and the brief page say what a line holds, what the three lines are, and when the drive writes plain lines. Waits on steps 2 and 3, and through step 3 on step 4 of spc-2611141617199772, whose state value is the region's first line.
5. The full sentence is one step away
   - criteria: 4c
   - needs: 4
   - waits on: step 5 of spc-2611141617199772 reaching the default branch
   - packages: internal/surface/cli/drain_drive_screen.go, internal/surface/cli/build.go, commands/drain.md, .abcd/development/brief/04-surfaces/27-implement.md
   - tests: internal/surface/cli/drain_drive_attention_test.go: an event with `Acts` set writes the attention file whole, with its own version, the event's ids, its full sentence and the log's path, and an event without it leaves the file as it was; `abcd implement status` prints the log's path and that sentence under the state line, wrapped at word breaks at 50 and at 80 columns, and `--json` carries the same fields; a title carrying control codes is masked in both; a missing, unreadable or newer-version file prints nothing and names why on standard error; the next drive's first such event replaces the file. Watched failing first: on the default branch `abcd implement status` names no log and no sentence. The command page and the brief page say where the full sentence is. Waits on step 4 here and on step 5 of spc-2611141617199772, which puts the state line in `abcd implement status` and edits the same function of `build.go`.

## Size

Five pull requests. Steps 1 and 2 start from the lab's renderers and model, 448 lines before tests, and drop its 360 lines of sentence patterns; the measuring, the id shortening and the redraw after a narrower window are new. Step 3 is the smallest in new lines and the one most exposed to other lanes: one new file, and edits to about a dozen functions of `terminal.go`. Step 4 is new code with no prototype, since the lab did not wire its renderers into the drive, redraw on a timer or a resize, or put the two streams through one writer. Step 5 is a small file and a few lines of `abcd implement status`.

Steps 1 and 2 can be built today. Steps 3 to 5 wait on spc-2611141617199772, of which nothing is on the default branch yet (checked at b8250514c): step 3 on its step 4, and step 5 on its step 5.
