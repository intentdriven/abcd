---
id: itd-2611150636498012
slug: the-autonomous-run-gets-a-refreshed
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611141610192096]
severity: minor
related_intents: [itd-2611141610197753, itd-2611141507299828, itd-2610030810370060]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The autonomous run gets a refreshed terminal interface and a headless mode

## Press Release

> **The autonomous run gets a refreshed command line and terminal interface, and a headless mode a native app can connect to.** While a drive runs at a terminal, a person presses Space to move through a short list of options and Enter to accept the one shown. The option shown first does nothing, so an Enter pressed by accident changes nothing. One of the options stops the drive after the lane in hand. Below the events, three lines are redrawn in place and say the state, the progress of the lanes and the lane in hand. The same run can be started with no terminal interface at all, in a headless mode that reports the same state and takes the same options, so that a native macOS app can later connect to the command line.

> "I wanted to stop it after the lane it was on, and I did not want to learn a command for that. Space until it says so, then Enter." — Iris, a product thinker (they/them)

> "The app and the terminal show the same three lines because they ask the same process the same thing." — Nia, a technical facilitator (they/them)

## What It Looks Like

The words and the options below are illustrations. The footer is the one drawn in itd-2611141610192096; this intent adds the line of options under it.

At rest, the option shown does nothing:

```text
09:16 wait 30s: …0101 did not move
working · lane 2 of 3 · since 09:11
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
…0101 The renderer panics on an…  implement 5m
Space: next option   Enter: carry on (no change)
```

After one press of Space:

```text
Space: next option   Enter: stop after this lane
```

## Why This Matters

The drive's terminal today takes one answer at its start and an interrupt after that. An interrupt stops the drive at once and leaves a lane to take up, and the incident note records what a stop by signal cost a person (`.abcd/development/research/notes/drive-operations-2026-11.md`, events 7 to 9). The terminal-output draft (itd-2611141610192096) drew the screen and left three questions open: whether the drive reads keys, whether "stop after the current lane" is wanted, and whether the running display is a footer or a table.

The product thinker has since ruled on all three, restated here:

- The drive offers options that a person cycles through with Space and accepts with Enter. The default option does nothing, so an accidental Enter changes nothing.
- Stopping after the current lane is one of those options.
- The terminal uses the live footer: three live lines, with the events above them.
- This is one part of a larger refresh of the command line and the terminal for the autonomous run. The refresh also needs a headless mode, so that a native macOS app can later connect to the command line.

The first three points answer the terminal-output draft's questions DQ-10, DQ-11 and DQ-12. That draft still records them as open; bringing it into step is part of planning this intent.

### Who it is for

- The product thinker (Iris), who starts a drive, looks in on it and sometimes wants it to finish what it is doing and stop.
- The technical facilitator (Nia), who watches the drive in a narrow pane, and who would build or use an app that shows the same state.

### What a person does

- Presses Space to see the next option and Enter to accept the option shown.
- Starts the run headless when an app, and not a terminal, is to show it.

### What a person sees

- Three live lines under the events, and one line that names the option Enter would accept.
- In an app connected to a headless run: the same state, the same lanes and the same options.

### What a person never has to know

- A command or a signal for stopping after the lane in hand.
- Which process the app talks to, or how.

## Mechanism

We expect a person to stop a drive after its lane without a lane left to take up, because the option is on the screen and one key away, and because the option at rest does nothing, a stray Enter in the drive's terminal costs nothing. Shown wrong if people still interrupt the drive when they mean to stop it after the lane, which the run's record of how each stop was set can count. We expect one headless interface to serve both the terminal and an app, because both then read state the loop hands out as typed values and neither reads it back out of sentences; the lab found that reading state out of sentences was wrong twice (lab-261114145658-67e1f9b, F-11). Shown wrong if the terminal interface needs something the headless interface cannot carry.

## Scope Conditions

- A terminal that can hand the drive single key presses. The repository has no code for that terminal mode today, and the terminal-output draft lists what it costs to add (its DQ-10).
- Where input is not a terminal, no key is read and no option line is drawn. The drive runs as it does today, and the headless mode or a flag carries the choice.
- The footer's three lines and their words are those of itd-2611141610192096 and itd-2611141610197753. This intent adds the option line and the keys.
- A stop after the lane in hand is recorded state a person sets, as every stop is (adr-2611050740453233). Its place among the existing stops is part of planning.
- The native app is not part of this intent. The headless mode is built so that one can connect later, on the same machine and for the same account.

## What's Out of Scope

- The native macOS app itself.
- A connection from another machine or another account.
- The state line's words and stall rules (itd-2611141610197753).
- The host status line's row (itd-2611141507299828).
- A full-screen interface that takes over the terminal. The events stay as scrollback above the footer.

## Acceptance Criteria

- **Given** a drive running at a terminal, **when** nothing has been pressed, **then** the option line names an option that changes nothing, and Enter leaves the drive exactly as it was.
- **Given** a drive running at a terminal, **when** a person presses Space, **then** the option line names the next option, and after the last option it returns to the one that changes nothing.
- **Given** the option to stop after the current lane is shown, **when** a person presses Enter, **then** the lane in hand runs to its end, no further lane opens, the drive stops, and the state line and the run's record say that a person set the stop from the terminal.
- **Given** an option has been accepted, **when** the option line is drawn again, **then** it shows the option that changes nothing, so a second Enter does not repeat or undo the first.
- **Given** a drive running at a terminal of 50 columns or more, **when** it runs, **then** the three live lines of itd-2611141610192096 are redrawn in place with every event a line above them, and the option line does not wrap.
- **Given** input or output that is not a terminal, or `TERM=dumb`, **when** the drive runs, **then** no key is read, no option line and no control code is written, and the output is the plain lines of itd-2611141610192096.
  - Evidence: itd-2610030810370060, criteria B3 and B4.
- **Given** `NO_COLOR` at a terminal, **when** the drive runs, **then** no colour code is written, the keys still work, and the option shown is told apart by its words and never by colour alone.
  - Evidence: itd-2610030810370060, criterion B4.
- **Given** an issue title or any other text the drive did not write, **when** it is drawn in the footer or sent over the headless interface, **then** it is made safe first.
  - Evidence: itd-2610030810370060, criterion B7.
- **Given** a run started in headless mode, **when** a client connects by the interface open question 1 decides, **then** it can read the state line, the progress of the lanes and the lane in hand as typed values, and can set each option the terminal offers, with the same effect and the same record.
- **Given** a headless run and a terminal run of the same recorded fixture, **when** both are read, **then** the state, the lanes and the events agree in every value.
- **Given** a headless run with no client connected, **when** it runs, **then** it works as a drive at a plain terminal does and waits on no client.

## Open Questions

Each question below is the product thinker's to decide, and none is answered here. Each is given with the answers that can be defended and what each costs.

1. **What is the headless interface?**
   - A local socket with a versioned message protocol. An app can connect to a run that is already going, and more than one client can read. Cost: a protocol to design, version and keep; a socket's place and lifetime to manage across a stop and a restart.
   - JSON lines on the standard input and output of one long-running process that the app starts. No address to find, and the process ends with the app. Cost: the app cannot join a drive a person started at a terminal, and only the starter can read it.
   - The existing `--json` outputs, polled. Nothing new to keep running, and every verb's output is already there. Cost: no event arrives until the next poll, each poll starts a process, and setting an option is a separate command with its own race against the loop.

2. **How does the command line know that the app is the person's own, on the same machine?**
   - The interface is reachable only by the account that started the run, by where it is kept and who may open it. Cost: nothing tells two programs of the same account apart.
   - The run writes a one-time value where only the account can read it, and a client presents it. Cost: one more thing to create, store and clear, and an app has to find it.
   - The app starts the process itself and talks to its own child. Cost: this fits only the second answer to question 1.

3. **What does the menu offer beyond stopping after the lane?**
   - Only the two options: carry on, and stop after this lane. Cost: every other wish still needs a second terminal.
   - Also showing detail: the full sentence of the last event that may need a person, and the log's path. Cost: the menu both shows and changes things, and a person has to read which.
   - Also acting on the run: stop now, set the lane in hand aside, open no new lane until told. Cost: each is a new recorded state with its own rules, and a longer list makes the option that does nothing harder to return to.

4. **Does the refresh cover only the drive, or every interactive verb?**
   - Only the drive. Cost: the plain-terminal interviews and the build keep their own way of asking, and a person learns two.
   - The drive first, with the keys and the option line written so the other interactive verbs can take them. Cost: a shared piece designed before its second user exists.
   - Every interactive verb at once. Cost: a large change to verbs whose criteria on plain output and on answers from a file were confirmed separately (itd-2610030810370060).

5. **How do screen readers and piped output fare?** The plain-terminal interview intent's criteria on plain output apply.
   - The plain output is the accessible form: a person using a screen reader runs the drive with output that is not redrawn, and sets an option by a flag or a second command. Cost: that person has no option line and no single key.
   - A setting that keeps the keys and writes each change of option as a new plain line, with no redrawing. Cost: a third way of drawing to build and test.
   - The headless interface is the accessible form, and an app speaks for it. Cost: nothing until an app exists.

6. **Does the first interrupt change its meaning?** The terminal-output draft's DQ-11 offered "stop after the current lane" as the first interrupt's meaning. The ruling puts it in the menu.
   - An interrupt keeps today's meaning and stops at once. Cost: the quickest key still leaves a lane to take up.
   - The first interrupt stops after the lane and a second stops at once. Cost: it changes what an interrupt does today, and there are then two ways to ask for the same stop.

## Typed Links

- builds_on: itd-2611141610192096 (a drive's terminal output reads at a glance). This intent takes that draft's footer as ruled, adds the option line and the keys, and restates the ruling that answers its questions DQ-10, DQ-11 and DQ-12.
- related: itd-2611141610197753 (a drive carries on from one command, and says in one line when it cannot). Its state line is the first live line, and a stop set from the menu is one more way a stop is set in the state it stores.
- related: itd-2611141507299828 (the status line shows the progress of the lanes). It shows the same stored state in the host's row, and a headless interface would hand out the values it reads.
- related: itd-2610030810370060 (a person can run abcd's interviews in a plain terminal). Its criteria B3, B4 and B7 on plain output, colour and text the program did not write are followed here.
- evidence: `.abcd/development/research/notes/drive-operations-2026-11.md`, events 5 and 7 to 9; `.abcd/development/research/notes/autonomous-run-2026-11-14.md`; lab-261114145658-67e1f9b, F-11 and F-13.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
