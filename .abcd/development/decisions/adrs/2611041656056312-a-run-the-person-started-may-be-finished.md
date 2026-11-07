---
id: adr-2611041656056312
slug: a-run-the-person-started-may-be-finished
status: proposed
date: 2026-11-04
supersedes: null
superseded_by: null
related_intents: [itd-2611041656024409, itd-2611041656012738, itd-2610222215485265, itd-2611041549461012]
related_rfcs: []
related_adrs: [adr-2610222222326104, adr-2611050740453233]
---

# ADR-2611041656056312: A run the person started may be finished after it dies, by default, inside its own limit

## Context

[adr-2610222222326104](2610222222326104-abcd-drives-agents-with-no-host-session.md)
lets abcd start agents with no host session only under four conditions. Its
first says a drive starts only from a command the person typed, after a yes that
covers that one drive and is never remembered, and that a repository's
configuration can never start one or widen one.

A run that dies part-way therefore waits for the person, however little of it is
left. On 2026-11-04 the product thinker ruled that a run they started may be
finished after it dies, inside its own limit, on by default, with each repository able to change it (itd-2611041656024409). That narrows condition 1, so it needs its own record. The
timer that notices a dead run is itd-2611041656012738.

## Decision

We will let a run be finished after it dies, on these terms.

1. **Only a run the person started.** The run began from their typed command
   under the other conditions of adr-2610222222326104. Nothing here starts a
   new run; iss-2610281328229800's wider ask stays undecided.
2. **Inside the limit they gave it.** Finishing never raises the limit, and what
   was already done counts against it.
3. **On by default, and said at the start.** The default is yes wherever the
   person has turned on the check, and the start of each run says the run may be
   finished alone. This is the one part of condition 1 that changes: this one
   permission is standing rather than asked per run. The product thinker
   confirmed on 2026-11-07 that this is abcd's default in every repository.
4. **A repository may switch it off and never on against the person.** A
   repository's configuration may say no. A person's own no on their computer
   always wins, so a repository's yes changes nothing where they have said no.
   The product thinker asked that a repository be able to change this and did not
   rule on this direction: it was decided without a question because condition 1's
   ban on a repository widening a drive, and the standing rule that abcd adds no
   vulnerability, leave one defensible answer.
5. **The other conditions hold afresh.** Each wall is proved live before any
   agent starts, and a failure starts nothing and tells the person. The bounds
   and the hand-back of what cannot be finished stand as written.
6. **Repeated death stops it.** A run that keeps dying reaches a limit on
   pick-ups, and then waits for the person.
7. **A person's stop outranks this.** adr-2611050740453233 says nothing resumes
   by itself when a stop lifts, so nothing is picked up under a stop, and a run
   that died under one waits for a person's take-over. A second Ctrl-C, which
   kills a run at once, is a stop and not a death. A pick-up steps through the
   restart path of itd-2611041549461012 and starts nothing twice.

## Alternatives Considered

- **Finish by default, a repository may only switch it off (chosen).** The work finishes while the person is away, at no more spend than they already allowed, and no repository can make their account spend against their no. It spends with nobody watching, which the product thinker accepted.
- **Only tell the person.** Keeps condition 1 whole and spends nothing unwatched. Rejected by the product thinker: the work would wait until they are back.
- **Ask at each start.** Keeps every yes per run. Offered and not chosen: one more question at every start, and a forgotten yes means the run waits.
- **A repository's setting decides, over the person's.** One behaviour for a whole team. Rejected without a question: anyone who can change a repository could make a person's account spend, unwatched, against their stated no.

## Consequences

- Agents can now run and spend with nobody watching on a run the person started earlier. The start of the run and the notice afterwards are what keep that visible.
- A picked-up run is unattended, so dcn-2610230710469123's allowance for a drive the person starts and watches does not obviously reach it. itd-2611041656024409 carries this as an open question, and it may limit where this works.
- Nothing here takes effect until itd-2610222215485265 ships and itd-2611041656024409 is built.
- adr-2610222222326104 is not edited while this is proposed. On acceptance it needs a pointer naming this narrowing of condition 1.
