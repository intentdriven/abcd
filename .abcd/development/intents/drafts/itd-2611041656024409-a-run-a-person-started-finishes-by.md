---
id: itd-2611041656024409
slug: a-run-a-person-started-finishes-by
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611041656012738]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2610222215485265, itd-2611041656012738, itd-2611041549461012, itd-2611141610197753]
---

# A run the person started finishes after it dies, from the one command that resumes it

## Press Release

> A drive tells the person at its start what ends it and how to go on. Alice, a solo founder, types `abcd drain --drive`, and before they say yes the start tells them that a reboot, a logout or a crash ends the drive, and names the one command that carries it on. They start ten fixes and go out. An update restarts the computer after four. When they are back they type that command, and the drive carries on from where the drain stood, finishes the other six inside the bound they agreed to, and tells them what it did. "Nothing ran while I was out that I had not started that day," said Alice, "and getting it going again was the one line it had shown me." abcd installs nothing that starts by itself.

## Why This Matters

A notice (itd-2611041656012738) tells the person a run died, and the work still waits for them. The product thinker ruled on 2026-11-04 that a run they started may be finished after it dies, inside its own limit, on by default, by the check picking it up, and confirmed on 2026-11-07 that this is abcd's default in every repository; adr-2611041656056312 was proposed to carry that trust rule.

In the planning of 2026-11-25 for drives that keep working across machines, the product thinker was asked whether abcd may set a drive to start again by itself after a reboot or a crash, and ruled that it may not (decision 1). A drive is not restarted automatically, and nothing self-starting is installed. What the person gets instead is a start that says plainly what ends a drive and names the one command that resumes it, which since itd-2611141610197753 carries on from the drain's state with nothing else typed. That ruling is this draft's answer, and it replaces the pick-up the draft first described.

The wider ask of iss-2610281328229800, a run starting by itself when paid credit appears, is not decided here and stays open. iss-2611050840353717 records that nothing restarts a long run after a reboot and that disk encryption can hold the machine at its login screen; its first half, a drive that says so at its start and names how to resume, is this intent.

## Mechanism

We expect an interrupted drive to be finished soon after the person returns, at no more spend than the bound they agreed to, because the start has already told them what ends a drive and the one command that resumes it, and that command carries on from the drain's state. Shown wrong if people who come back to a drive ended by a reboot or a crash still have to learn another command, or ask a session, to get its lanes going again.

## Scope Conditions

- Holds only once the plain-terminal run of itd-2610222215485265 has shipped, and once the start that carries on from the drain's state (itd-2611141610197753) is built.
- Holds only for a run a person started from a typed command; the resume is a new start, with its own yes, under every condition of adr-2610222222326104.
- A machine that waits at its login screen after an update holds the drive until a person signs in and types the command, by design (decision 1).

## What's Out of Scope

- A drive that starts again by itself after a reboot, a logout or a crash, on by default or as an opt-in: nothing that starts by itself is installed (decision 1).
- The check of itd-2611041656012738 picking a run up. That check tells a person that a run died, and starts nothing.

## Acceptance Criteria

- **Given** a person starts a drive, **when** the start shows what it will do, **then** before the yes it says that a reboot, a logout or a crash ends the drive, and names the one command that resumes it.
- **Given** a drive ended by a reboot, a logout or a killed process, **when** the person types the command the start named, **then** the drive carries on from the drain's state, inside the bound its new yes names, and says what it took up.
- **Given** any drive, **when** it starts or runs, **then** abcd registers nothing to start it again: no login item, no timer, no service, and no entry in a scheduler.
- **Given** the check of itd-2611041656012738 is on and a drive died, **when** the check looks, **then** it tells the person and starts nothing.

## Open Questions

None. The four questions the draft first carried were about the check picking a run up (the forge-login rule for a picked-up run, the limit on pick-ups, a run started inside a host session, and the autonomy grant of adr-2610240850193281). With no pick-up (decision 1), none of them arises.

## Decisions

1. **No automatic restart** (product thinker, 2026-11-25; the planning's fourth question for drives across machines).
   - Question: may abcd set a drive to start again by itself after a reboot or a crash?
   - Answer: no. The drive says at its start that a reboot, a logout or a crash ends it and prints the one command that resumes it, which carries on from the drain's state (itd-2611141610197753). Nothing self-starting is installed.
   - Alternatives and what each costs: an opt-in the person sets up once per account, which registers the drive to start when that account logs in and which the planning recommended; it survives reboots once someone signs in, at the cost of a program that starts itself and spends the subscription on a machine the person may have stopped thinking about. On by default for any drive with an end time, which installs a self-starting program as a side effect of another command.
   - What it costs, which the product thinker accepted: an overnight system update stops the drive until someone notices and types the command.
   - What it departs from: the ruling of 2026-11-04 that a run a person started is finished after it dies, on by default, by the check picking it up. For a drive, that pick-up is not built. adr-2611041656056312, which was proposed to carry it and never accepted, is left for its owner to withdraw or narrow; this draft no longer needs it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
