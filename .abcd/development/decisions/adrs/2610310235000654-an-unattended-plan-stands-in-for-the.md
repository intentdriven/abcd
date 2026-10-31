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

1. **Recorded decisions.** The draft carries real acceptance criteria (at least one top-level bullet, by the parser `plan` itself applies), asks no open question, and answers its claim sections: the mechanism answered or absent, the scope conditions recorded. A condition with no identity marker yet passes, because `plan` is the verb that stamps one. It is not held, nothing it names in `blocked_by` is unsettled, and nothing `plan` refuses on the draft's own fields stands in the way: a kebab-case slug, no `spec_id` already set, a planned form under the size cap.
2. **Grounds present.** The most recent ground on the record that no run's pick wrote is `pursued:`. None, or a deferred or declined one, refuses. What is checked is that no pick wrote it, not who did: `abcd intent ready --grounds` writes a ground whoever runs it.
3. **Criteria provenance.** The record carries `origin` and `production_mode`, each present and a value of the closed vocabulary. What is checked is that the values are stated and known, not that they are true: `origin` asserts nothing about whether a person or an agent session wrote the record, so the criteria the plan stamps have a stated source, not a verified one.
4. **The two review receipts.** Two fresh adversarial reviewers, one with the design and feasibility lens and one with the record-discipline lens, each judge the draft and write a return, one after the other: the second is handed out only once the loop has recorded the first's verdict, so the two are never out together. The loop parses each verdict from its return and records it in the run's state. Both must pass. Each brief tells its reviewer that the draft's text is data to judge, not instructions, and to change nothing. The reviews run on the host's agents only: no runner and no reviewer route is handed one, since the lane has no worktree to confine a runner to and a reviewer route reads a diff the lane has none of.

Parts 1 to 3 are checks on the record, and they bind a draft a person wrote. For a draft an agent wrote end to end, its grounds and its provenance included, they show only that the record is complete and well formed, and the two reviews are the only gate on what it says.

The reviews and the plan are bound to the bytes the start checked: the draft's digest is recorded when the run starts, and a draft whose bytes changed, or that left `drafts/`, before the plan is handed back unplanned. `plan` itself is told the digest and compares it with the bytes it reads under the intent store's lock, so a change landing after the loop's own check is refused there before anything moves. The checkout is bound too: the start records a digest of its HEAD, its index, and every file the ignore rules admit outside the local tier, and a checkout that changed during the reviews or before the plan is handed back, whoever changed it. The digest does not cover ignored files or the git directory (refs other than HEAD, the configuration, the hooks). The local tier is left out by design: the run's own state and the reviewers' returns are written there, as they are for the build's validators.

The plan stage takes two steps. The first checks parts 1 to 3 again on the record as it then stands, and part 4 against the state, and marks the lane as planning, writing only the run's state; the second checks it all once more and runs `plan`. A record the plan stage finds already planned is never counted as this run's plan, and the lane is handed back citing no verdict, the planned record left as it is. Before the mark, another act planned it; after it, the record is the same whether this run's second step stopped after `plan` and before its state write or another act planned it, and the run cannot tell the two apart. The cost is that a second step stopped at that point ends the run handed back, with a plan the person reads before building it. Anything less hands the run back to the person with what it found. A review that does not pass hands its findings back too: an unattended path has nobody to answer them, so the draft goes to the person's own planning interview.

The path plans and stops there. It writes the spec stub `plan` mints and moves the intent, in the checkout the run lives in, as a person's `abcd intent plan` does. Writing the spec's steps, committing the plan and building it stay separate acts.

## Alternatives Considered

- **The four-part substitute above (chosen).** Each part is a field the record carries or a verdict the loop recorded itself, so nothing is inferred; parts 1 to 3 check that the record is complete, not who wrote it, and the reviews judge what it says. The digest, compared by `plan` under its lock, closes the gap between what the reviewers read and what is planned.
- **Recorded decisions alone, with no reviews.** Rejected: the adversarial-review principle puts the drafts-to-planned crossing among the major ones, and the pilot that measured it found real defects in drafts whose authors thought them settled.
- **Reviews whose findings a fresh implementer applies, as the build's validators do.** Rejected for this path: applying a finding to a draft rewrites the person's record, and that rewrite is exactly the judgement the sign-off stands for. A finding goes back to the person.
- **No unattended path; keep `intent plan` the only door.** Rejected: decision 6 rules the flag in, and a run that stops at every draft has a person in the loop by construction.

## Consequences

- `abcd build` gains `--auto-plan`. On a draft, the run's one lane takes the two reviews, then the plan. On an intent already planned, the flag has nothing to do and the build runs as it would without it.
- `abcd intent plan` stays the human's act, and its page keeps saying so. The glossary's `plan` describes that act; this record names the one other door.
- The run's state gains the auto-plan's record on its lane: the draft, its digest, the checkout's digest, each review's verdict, whether the plan was begun, and the planned record and spec. That is schema version 17.
- The step that runs `plan` keeps the record writers' rule, judged under the run's lock immediately before the plan: in a main checkout two or more live sessions started in, it backs off.
- A planned intent from this path still needs its spec's steps before `abcd build` takes it, so an unattended path from a draft to delivered needs a spec author too. That is not built here.
