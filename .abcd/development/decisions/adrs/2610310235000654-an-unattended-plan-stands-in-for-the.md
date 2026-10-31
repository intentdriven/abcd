---
id: adr-2610310235000654
slug: an-unattended-plan-stands-in-for-the
status: proposed
date: 2026-10-31
supersedes: null
superseded_by: null
related_intents: [itd-2609201916151817]
related_rfcs: []
related_adrs: [adr-2610222222326104]
---

# ADR-2610310235000654: An unattended plan stands in for the sign-off only on a recorded, reviewed, unchanged draft

## Context

`abcd intent plan` moves a draft to `planned/`, and running it is the product thinker's sign-off on the intent's acceptance criteria: the planning interview walks every criterion with them first, and the verb is never run unattended or on inferred consent (commands/intent.md, the planning interview's step 10; the glossary's `plan`).

itd-2609201916151817's decision 6 adds a second path, `abcd build <itd-N> --auto-plan`. It lets the planning path run with no human, the two adversarial reviews included, on a draft whose decisions are all recorded. The decision asks for a record of what stands in for the sign-off, minted before the path ships (spc-2609202134338445 step 11). The other reversal decision 5 gave the build, the loop driving itself with no host session, is recorded in adr-2610222222326104 and is not recorded again here.

Two rules were already in force. Adversarial review scales with blast radius: a draft crosses to `planned/` only after two independent reviewers with different lenses (the principle of that name; design and feasibility, and record discipline, are the proven pair). And only the loop writes a verdict: a verdict counts only when the loop parsed it from the reviewer's own return (itd-58, folded into the same intent as decision 9).

## Decision

We will let `abcd build <itd-N> --auto-plan` run `plan` without a human only when all four parts of this substitute hold, and refuse it, planning nothing, when any part does not.

1. **Recorded decisions.** The draft carries real acceptance criteria (at least one top-level bullet, by the parser `plan` itself applies), asks no open question, and answers its claim sections: the mechanism answered or absent, the scope conditions recorded. A condition with no identity marker yet passes, because `plan` is the verb that stamps one. It is not held, and nothing it names in `blocked_by` is unsettled.
2. **Grounds present.** The most recent ground a person wrote on the record is `pursued:`. None, or a deferred or declined one, refuses.
3. **Criteria provenance.** The record says where its text came from (`origin`) and how it was produced (`production_mode`), each a value of the closed vocabulary. The criteria the plan stamps as signed off have a stated source.
4. **The two review receipts.** Two fresh adversarial reviewers, one with the design and feasibility lens and one with the record-discipline lens, each judge the draft and write a return. The loop parses each verdict from that return and records it in the run's state. Both must pass. The reviews and the plan are bound to the bytes the start checked: the draft's digest is recorded when the run starts, and a draft whose bytes changed, or that left `drafts/`, before the plan is handed back unplanned.

The plan stage checks parts 1 to 3 again on the record as it then stands, and part 4 against the state, immediately before it runs `plan`. Anything less hands the run back to the person with what it found. A review that does not pass hands its findings back too: an unattended path has nobody to answer them, so the draft goes to the person's own planning interview.

The path plans and stops there. It writes the spec stub `plan` mints and moves the intent, in the checkout the run lives in, as a person's `abcd intent plan` does. Writing the spec's steps, committing the plan and building it stay separate acts.

## Alternatives Considered

- **The four-part substitute above (chosen).** Each part is a thing a person already wrote into the record or a verdict the loop recorded itself, so nothing is inferred. The digest closes the gap between what the reviewers read and what is planned.
- **Recorded decisions alone, with no reviews.** Rejected: the adversarial-review principle puts the drafts-to-planned crossing among the major ones, and the pilot that measured it found real defects in drafts whose authors thought them settled.
- **Reviews whose findings a fresh implementer applies, as the build's validators do.** Rejected for this path: applying a finding to a draft rewrites the person's record, and that rewrite is exactly the judgement the sign-off stands for. A finding goes back to the person.
- **No unattended path; keep `intent plan` the only door.** Rejected: decision 6 rules the flag in, and a run that stops at every draft has a person in the loop by construction.

## Consequences

- `abcd build` gains `--auto-plan`. On a draft, the run's one lane takes the two reviews, then the plan. On an intent already planned, the flag has nothing to do and the build runs as it would without it.
- `abcd intent plan` stays the human's act, and its page keeps saying so. The glossary's `plan` describes that act; this record names the one other door.
- The run's state gains the auto-plan's record on its lane: the draft, its digest, each review's verdict, and the planned record and spec. That is schema version 17.
- A planned intent from this path still needs its spec's steps before `abcd build` takes it, so an unattended path from a draft to delivered needs a spec author too. That is not built here.
