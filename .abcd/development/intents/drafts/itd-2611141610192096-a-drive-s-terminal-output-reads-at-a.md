---
id: itd-2611141610192096
slug: a-drive-s-terminal-output-reads-at-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611141610197753]
severity: minor
related_intents: [itd-2610222215485265, itd-2610030810370060, itd-2611141610201875]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A drive's terminal output reads at a glance

## Press Release

> **A drive's terminal output reads at a glance.** A person types `abcd drain --drive` and reads at most seven short lines before they answer: how many lanes and agent runs, when it ends, what it runs on, the review, the first check, and the question. None of the lines wraps in a narrow window, and `?` at the question shows the full plan. While the drive runs at a terminal, each event is one short line, and below the events a small region is redrawn in place: the state line, the progress of the lanes, and the lane in hand. Where the output is not a terminal, the drive writes plain lines with whole ids and no control codes. The full sentence of anything a person may have to act on is in the log, the log's path is shown once at the start, and no line shown to a person quotes a command meant for a host session.

> "The start used to be a page I skimmed and answered yes to. Now it is six lines and a question, and I read them." — Iris, a product thinker (they/them)

> "I look at the bottom three lines. They tell me the state, how many lanes are done and which lane is in hand, in a window fifty columns wide." — Nia, a technical facilitator (they/them)

## What It Looks Like

The mock-ups are the lab's prototype drawing one recorded fixture run (lab-261114145658-67e1f9b, probe p6-output-renders-2). The words and the short forms are illustrations. Which of the two running designs is built is an open question below.

The start screen, the same at 50 and 80 columns:

```text
abcd drain --drive: works through the issues
  lanes    up to 8, about 40 agent runs
  ends     8h on any one wait (--until: a time)
  runs on  claude, login walled
  review   ruthless and security reviewers
  first    one short live check of the login
Start? [y/N, ? shows the full plan]
```

The footer design, nearly three hours into an outage, at 50 columns. Every event is a line, and the last three lines are redrawn in place:

```text
09:00 …0103 judged: its lane may open
09:00 …0103 lane opened
09:00 …0103 worktree done
09:00 …0103 brief done
09:03 wait 30s: the implementer of …0103 did not…
09:07 …0103 implement done
09:07 …0103 validate done
09:08 …0103 preflight running
09:10 …0103 land done
09:11 …0101 judged: its lane may open
09:11 …0003 handed back: needs a decision record
09:11 …0101 lane opened
09:11 …0101 worktree done
09:11 …0101 brief done
09:15 wait 1m0s: the subscription has met its ra…
09:16 wait 30s: …0101 did not move
09:17 wait 1m0s: outage: …0101 did not move
stalled 2h44m · on outage: …0101… · needs a look
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
…0101 The renderer panics on an…  implement 2h48m
```

The footer design at the same moment, at 80 columns:

```text
09:00 …0103 judged: its lane may open
09:00 …0103 lane opened
09:00 …0103 worktree done
09:00 …0103 brief done
09:03 wait 30s: the implementer of …0103 did not run
09:07 …0103 implement done
09:07 …0103 validate done
09:08 …0103 preflight running
09:10 …0103 land done
09:11 …0101 judged: its lane may open
09:11 …0003 handed back: needs a decision record
09:11 …0101 lane opened
09:11 …0101 worktree done
09:11 …0101 brief done
09:15 wait 1m0s: the subscription has met its rate limit
09:16 wait 30s: …0101 did not move
09:17 wait 1m0s: outage: …0101 did not move
stalled 2h44m · on outage: …0101 did not move · needs a look
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
…0101 The renderer panics on an empty list  implement 2h48m
```

The table design at the same moment, at 50 columns. Only notable events are lines, and the region below them has a row per lane:

```text
09:00 …0103 judged: its lane may open
09:03 wait 30s: the implementer of …0103 did not…
09:11 …0101 judged: its lane may open
09:11 …0003 handed back: needs a decision record
09:15 wait 1m0s: the subscription has met its ra…
09:16 wait 30s: …0101 did not move
09:17 wait 1m0s: outage: …0101 did not move
issue  stage      time    what
…0103  landed     10m03s  The renderer panics on…
…0101  implement  2h48m   The renderer panics on…
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
stalled 2h44m · on outage: …0101… · needs a look
```

The table design at the same moment, at 80 columns:

```text
09:00 …0103 judged: its lane may open
09:03 wait 30s: the implementer of …0103 did not run
09:11 …0101 judged: its lane may open
09:11 …0003 handed back: needs a decision record
09:15 wait 1m0s: the subscription has met its rate limit
09:16 wait 30s: …0101 did not move
09:17 wait 1m0s: outage: …0101 did not move
issue  stage           time    what
…0103  landed          10m03s  The renderer panics on an empty list
…0101  implement       2h48m   The renderer panics on an empty list
[██░] 2/3  1 done, 1 in flight, 0 aside, 1 back
stalled 2h44m · on outage: …0101 did not move · needs a look
```

Both designs after the run has ended, at 80 columns, show the same last state line:

```text
stopped 2s · on eight hours of runner failures · you: abcd drain --drive
```

Output that is not a terminal, one plain line per event with whole ids:

```text
2026-11-12T09:00:25Z iss-2609300000000103 judged: its lane may open
2026-11-12T09:00:26Z iss-2609300000000103 lane opened
2026-11-12T09:03:47Z wait 30s: the implementer of run-2611141543093746 did not run
2026-11-12T09:08:24Z iss-2609300000000103 preflight running
```

## Why This Matters

The drive's output today is written as full sentences, one per step, and a terminal breaks them wherever the window ends. On the pinned build the lab counted, at 50 columns (F-13, F-20):

| | Pinned | Footer | Table |
| --- | --- | --- | --- |
| Start screen | 14 lines, 34 rows | 7 rows | the same start |
| First 10 minutes (12 events) | 23 rows | 11 rows | 6 rows |
| Whole run, 8h35m (39 events) | 198 rows | 23 rows | 15 rows |

Ten of the start screen's fourteen lines wrap at 50 columns, and the program does not wrap them, so the terminal breaks inside words (F-20). Nine wait lines of the fixture run tell the reader to start an agent by hand and run `abcd implement receipt`, which is a host session's command: a person who followed it would hand a receipt to a run the drive is driving. The drive never says where its log is (F-21). During the three days the incident note records, learning the drive's state took reading roughly forty thousand log lines.

### Who it is for

- The product thinker (Iris), who starts a drive and leaves. They read the start before they answer, and on return they read the bottom of the screen.
- The technical facilitator (Nia), who is asked "is it working?" and looks at the drive's terminal, often in a narrow pane beside other work.

### What a person does

- Types `abcd drain --drive`, reads at most seven short lines, and answers y, or `?` for the full plan.
- Looks at the terminal and reads the region at the bottom.

### What a person sees

- At the start: lanes and agent runs, when it ends, what it runs on, the review, the first check, and the question. The full plan on `?`.
- While it runs, at a terminal: a short line for each event, and below them a region redrawn in place (the state line, the progress of lanes, the lane in hand).
- Where output is not a terminal: one plain line per event, and a `state:` line when the state changes.
- Once, at the start: where the log is.

### What a person never has to know

- The difference between the loop's kinds of wait.
- A command meant for a host session.
- The leading digits of an id, while watching. The whole id is in the log and in plain output.

## Mechanism

We expect a person to learn the drive's state from the screen without scrolling, because the same recorded run that takes 198 rows at 50 columns today takes 23 or 15, and its state is always the same line in the same place. Shown wrong if the short forms lose something a person needs to act: the lab found they drop the question of a hand-back, every command, the cause after its first clause, the ids' leading digits and the log's place (F-13). The claim stands only while the full sentence of the last event that may need a person, and the log's path, are one step away.

## Scope Conditions

- A terminal of 50 columns or more. The lab drew 50, 80 and 120 and nothing narrower.
- A terminal that honours the three control sequences the prototype uses to redraw a region in place. Anywhere else (output that is not a terminal, `NO_COLOR`, `TERM=dumb`) the drive writes plain lines.
- The state line is the one itd-2611141610197753 defines and stores. This intent draws it and defines none of its words or rules.
- The lab drew one recorded fixture run of three lanes and a set of hostile titles. It did not wire the renderers into `abcd drain --drive`, redraw on a resize or on a timer, or put the drive's two output streams through one writer; each is part of building this.
- Without keys, "one step away" means `abcd implement status` in a second terminal and the log's path. Whether the drive reads keys is an open question below.

## What's Out of Scope

- The state line's words, its stall rules and where it is stored. That is itd-2611141610197753.
- Keys, until the first open question is answered. The design works without them.
- A terminal library or a full-screen interface. Three control sequences were enough in the prototype.
- Colour. It is added last, if at all, from the repository's existing colour-mode resolver, and nothing is conveyed by colour alone.
- The prototype's sentence patterns. It read the drive's state back out of its sentences, and twice read it wrong (F-11); the product hands the surface a typed event from the loop.
- The host status line's row (itd-2611141507299828).
- Re-arming pull requests and fast-forwarding the checkout (itd-2611141610201875).

## Acceptance Criteria

- **Given** a terminal of 50 columns or more, **when** the drive starts, **then** the start is at most seven lines and none wraps, and `?` at the question shows the full plan, wrapped at word breaks.
  - Evidence: lab-261114145658-67e1f9b, F-13 and F-20.
- **Given** output that is a terminal, **when** the drive runs, **then** each event is one line no wider than the window, the region below is redrawn in place and its first live line is the state line of itd-2611141610197753, and no line wraps at 50, 80 or 120 columns.
  - Evidence: lab-261114145658-67e1f9b, F-13.
- **Given** output that is not a terminal, or `NO_COLOR`, or `TERM=dumb`, **when** the drive runs, **then** it writes plain lines with no control code, whole ids, a `state:` line when the state changes, and nothing conveyed by colour alone. **Given** an issue title carrying control codes, **then** it is masked before it is measured or drawn.
  - Evidence: lab-261114145658-67e1f9b, F-13; itd-2610030810370060, criteria B3, B4 and B7.
- **Given** an event a person may have to act on, **when** the drive reports it, **then** its full sentence is in the log, the log's path is shown once at the start, the full sentence is one step away at the terminal, and no line shown to a person quotes a command meant for a host session.
  - Evidence: lab-261114145658-67e1f9b, F-13 (its condition) and F-21.

## Open Questions

Each question below is the product thinker's to decide, and none is answered here. Each is quoted as the lab met it, with the answers it can defend and what each costs.

1. **DQ-10. "Should the drive read keys?"** (F-13, the lab's note on keys)
   - No keys; detail is `abcd implement status` and the log's path. Cost: no way to ask for detail without a second terminal.
   - A typed letter and Enter (`d`, `s`): no terminal mode, safe once the read is guarded against a drive in the background. Cost: not a single key; the letter echoes into the live region until the next redraw.
   - Single keys by a new terminal mode written for the drive. Cost: termios code the repository does not have, a direct use of `golang.org/x/sys`, and four failure cases to test (hang-up, Ctrl-C as a byte, a killed drive leaving the terminal raw, a background read).

2. **DQ-11. "Is 'stop after the current lane' wanted, and what reaches it?"** (F-13)
   - Yes, as the first Ctrl-C's meaning, with a second to stop at once as now. Cost: changes what Ctrl-C does today.
   - Yes, by a typed letter. Cost: depends on DQ-10.
   - No. Cost: a stop always leaves a lane to take up.

3. **DQ-12. "Footer or table at a terminal?"** (F-13, the probe p6-output-renders-2 and its side-by-side drawing)
   - Footer: three live lines, every event a line above. Cost: more scrollback (23 rows for the whole fixture run against 15).
   - Table: a row per lane, only notable events above. Cost: a lane's stages leave no trace on the screen; the region grows with the lanes (the prototype shows the last six).

## Typed Links

- builds_on: itd-2611141610197753 (a drive carries on from one command, and says in one line when it cannot). The first live line of either design is that intent's state line, so this intent is built after its state is a value the loop hands the surface.
- related: itd-2610222215485265 (the drive whose output this is; its criterion that the drive prints one line per step stands, and this intent says what a line holds). itd-2610030810370060 (the plain-terminal interviews, whose criteria B3, B4 and B7 on plain output, colour and untrusted text this intent follows). itd-2611141610201875 (the merge queue and the checkout, whose one-line reports appear among these events).
- evidence: lab-261114145658-67e1f9b (findings F-13, F-20 and F-21) and `.abcd/development/research/notes/drive-operations-2026-11.md`, event 5.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
