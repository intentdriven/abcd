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
blocked_by: [itd-2610222215485265, itd-2611041656012738, itd-2611041815429986, itd-2611041549461012]
---

# A run the person started finishes after it dies

## Press Release

> A run a person started finishes by itself after it dies, inside the limit they gave it. Alice, a solo founder, starts ten fixes and goes out. An update restarts the computer after four. When the computer is back, abcd's check picks the run up, finishes the other six and tells Alice what it did. "I came home to ten fixes, not four," said Alice. "It never went past the limit I set, and the start of the run had told me it might do this." It is on by default wherever the check is on, a repository can switch it off, and a person's own no always wins.

## Why This Matters

A notice (itd-2611041656012738) tells the person a run died, and the work still waits for
them. The product thinker ruled on 2026-11-04 that a run they started may be
finished after it dies, inside its own limit, on by default, with each repository able to change it, and confirmed on 2026-11-07 that this is abcd's default in every repository. That changes a standing rule, so the trust rule is written down in
adr-2611041656056312 and this intent carries the capability.

The wider ask of iss-2610281328229800, a run starting by itself when paid credit
appears, is not decided here and stays open.

## Mechanism

We expect runs interrupted by a restart or a crash to finish without a person,
at no more spend than the run they started allowed, because the run's limit and
its progress are already recorded, so picking it up needs no new judgement and
no new permission beyond the one this intent's decision records.

## Scope Conditions

- Holds only once the plain-terminal run of itd-2610222215485265 has shipped; its terminal drive is already in the binary, and that intent is still planned.
- Holds only for a run a person started from a typed command, inside that run's own limit.
- Holds only where the check of itd-2611041656012738 is on; with no check, nothing picks a run up.
- Holds under every other condition of adr-2610222222326104: the walls proved live before any agent starts, the same bounds, and hand-back of what cannot be finished.
- Holds only for a death abcd can tell from a person's act. A process a person kills from outside abcd reads as a crash, so a run they want to stay stopped takes the stop of itd-2611041549471712.
- Holds for a stop the check can read. On a machine with two accounts, a stop set and lifted between two looks from the person's own account is not seen, because the record of it stays in their home, and the run may be picked up.

## Acceptance Criteria

- **Given** a person starts a run with a limit of ten and it dies after four, **when** the check sees it and nothing has switched this off, **then** abcd finishes at most the other six and tells the person what it did.
- **Given** a person is about to start a run, **when** the start shows what it will do, **then** it says the run may be finished alone if it dies.
- **Given** a repository's configuration switches this off, **when** a run there dies, **then** the run waits and the person is told.
- **Given** a person's own setting on their computer says no, **when** a repository's configuration says yes, **then** the run waits and the person is told.
- **Given** a person stopped a run on purpose, or it ended at its limit or an empty queue, **when** the check looks, **then** nothing is picked up.
- **Given** a stop stood when a run died, or has stood at any time since, **when** the check looks, **then** nothing is picked up, and the run waits for a person's take-over even after the stop is lifted.
- **Given** a person presses Ctrl-C a second time, which kills a terminal run at once, **when** the check looks, **then** that reads as the person's stop and nothing is picked up.
- **Given** a run is picked up, **when** it steps, **then** it goes through the restart path of itd-2611041549461012: it collects what arrived before it starts an agent, saves uncommitted work aside, reserves again first, and starts nothing twice.
- **Given** a wall fails its live check at pick-up, **when** the check would start agents, **then** none starts and the person is told which wall failed.
- **Given** a picked-up run keeps dying, **when** it reaches the limit on pick-ups, **then** abcd stops trying and tells the person.
- **Given** any configuration, **when** a run is picked up, **then** its limit is what the person started it with, never larger.
- **Given** a listed tool fails the test of itd-2611041815429986 at pick-up, **when** the check would start agents, **then** none starts, the person is told once for that run what is missing, the refusal does not count against the limit on pick-ups, and a later check picks the run up once the tool is registered or waived.

## Open Questions

- A picked-up run is unattended, so the rule that unattended lanes run only where the person's forge login is absent (dcn-2610230710469123) appears to apply to it. How that fits with the named profiles of itd-2611040344170206 decides whether this works on an ordinary personal machine (technical facilitator).
- The limit on pick-ups per run (technical facilitator).
- Whether a run started inside a host session can be picked up at all, since there is no host to continue it (technical facilitator).
- How this sits beside the autonomy grant of adr-2610240850193281, which is set only by an explicit command (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
