---
id: itd-2611041656012738
slug: a-person-who-turns-on-abcd-s-check-hears
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610161016391855]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A person who turns the check on hears when a run dies

## Press Release

> A person who turns on abcd's check hears when a run has died, without typing a command. A small timer on their computer looks every few minutes, uses no AI and starts nothing, and puts up one notice saying which run stopped and when. Alice, a solo founder, leaves a run of ten fixes going and steps into the next room. "The run stopped at two o'clock and I knew at five past," said Alice. "Before, I found out when I next typed something, sometimes hours later." The notice starts as a pop-up on the computer and reaches the phone once phone alerts exist.

## Why This Matters

A dead run stays unnoticed until someone next runs an abcd command, or until a
session that is still alive happens to look. Nothing runs between sessions, so
a run that dies when nobody is typing costs hours with no sign, and people have
been setting timers by hand to cover it. The hand-back of
itd-2610161016391855 only helps once something reads the records; this covers
the time when nothing does.

The product thinker settled the shape in an interview on 2026-11-04. The check
sits on the person's own computer, is off until they turn it on, and only tells
them. Finishing the work is a separate intent (itd-2611041656024409), filed apart so that
this one can ship on its own, while that one waits on the intent of the plain-terminal run.

## Mechanism

We expect a run that dies while its computer stays up to be noticed within
minutes rather than at the next typed command, because whether a run is still
moving can be read from what it leaves on disk, and a script reading that costs
almost nothing and needs no model.

## Scope Conditions

- Holds while the computer is on and the check is running. While the computer is off the check says nothing; it speaks once it is back. The product thinker chose this knowingly over a check hosted on GitHub, which could see a lapsed reservation with the computer off but would be up to about two hours late, could raise a false alarm while a run waits in the merge queue, and would be visible to everyone on a public project.
- Holds for runs on the computer the check sits on; a run on another computer is not seen.
- Until phone alerts exist (itd-2610222029228127) the notice is a pop-up, which helps only a person at the computer or returning to it.

## Acceptance Criteria

- **Given** the check is turned on and a run dies while the computer stays up, **when** the check next looks, **then** one notice names the run and when it stopped, and no model runs and no agent starts.
- **Given** the computer was off when the run died, **when** the computer is back and the check runs, **then** the notice says the run stopped and when, as far as that can be told.
- **Given** a run that is only waiting, in the merge queue for instance, **when** the check looks, **then** no notice is raised.
- **Given** a run a person stopped, by the stop of itd-2611041549471712 or by a second Ctrl-C, **when** the check looks, **then** no notice calls it dead.
- **Given** the same dead run, **when** the check looks again, **then** no second notice is raised for it.
- **Given** no run has died, **when** time passes, **then** nothing is shown and no model runs.
- **Given** the person never turned the check on, **when** abcd is used as normal, **then** nothing is scheduled on the computer.
- **Given** the check is on, **when** the person turns it off, **then** the timer is gone and nothing of it keeps running.

## Open Questions

- How a dead run is told from one that is only waiting, without a process that must stay alive (technical facilitator).
- Which timer on each operating system, how often it looks, and which verb turns it on and off (technical facilitator).
- After a restart, a computer with an encrypted disk can sit at the login screen with no account running. Whether the check can run before someone logs in needs a live test; if it cannot, the notice waits for the login (technical facilitator).
- This is the first thing abcd schedules on a person's computer. itd-2610180912189125 says abcd schedules nothing; this refines that for a timer the person turns on, and the wording there needs reconciling at planning (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
