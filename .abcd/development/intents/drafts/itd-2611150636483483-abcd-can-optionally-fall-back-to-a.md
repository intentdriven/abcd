---
id: itd-2611150636483483
slug: abcd-can-optionally-fall-back-to-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611101419006280]
severity: minor
related_intents: [itd-2610240922461281]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd can fall back to a smaller model when a model's budget is exhausted

## Press Release

> **abcd can fall back to a smaller model when a model's budget is exhausted.** A person sets, on their own machine, that a role launched at a model class may run at the next smaller class when the class's model has no budget left. A long unattended run then carries on with the work that can be done at a smaller model, and goes back to the stronger one when its budget returns. Every receipt, review record and commit trailer names the model that did the work, so a fallback is never read as the stronger model's work. Some roles can be excluded: a role the person names as never falling back waits for its own model, and the run says so in one line.

> "The run had stopped and nothing said why: one model was used up. Now it tells me which lanes went on at the smaller model and which review waited for the strong one." — Iris, a product thinker (they/them)

> "A security review at a smaller model is a different review. I want the record to say which one I got, and I want to be able to say it never happens without my model." — Dave, a security engineer (they/them)

## Why This Matters

A long autonomous run leans on the strongest model for its hardest roles. The person reports that this model runs out of budget during such runs, and the work that depends on it then stalls until the budget returns (`.abcd/development/research/notes/autonomous-run-2026-11-14.md`, event 10). abcd has no setting today that lets a role run at another model when its own is exhausted.

The product thinker's request, restated: abcd needs a setting under which it lowers the model a role uses, automatically and only where the person has chosen it, and the design has to consider whether some tasks, such as security reviews, are excluded. Which tasks, and whether any, is to be decided.

### Where a fallback would act today

- **The class map.** `models.classes` maps four classes (`research`, `top`, `default`, `small`, strongest first) to one `<provider>/<model>` route each, with `models.effort` beside it. Both are read from the machine's configuration over the bundled map and never from a repository (itd-2611101419006280; `internal/core/implement/loop/reviewset.go`). The map already orders the classes, so "the next smaller class" has a meaning without new configuration.
- **The review set.** `review.reviewers` and `review.judge` name which classes review and judge. The set is resolved once when a run starts and written into the run's state, so that a later step honours the set the run started on. A fallback changes, part-way through a run, which model stands behind a class the state has already fixed.
- **The rate-limit checkpoint.** A runner's rate-limit response ends the window for the whole run: every lane is checkpointed, the time of the next attempt is written, and a short probe asks whether the limit has lifted (`internal/core/implement/loop/ratelimit.go`, `internal/core/runner/limit_probe.go`). The reason given in the code is that every lane spends the same budget.
- **The dispatcher.** A role whose runner is absent or fails already falls back to the host session or to a configured host, with one receipt for the event. A runner that reports a rate limit is the one failure the dispatcher does not fall back on (`internal/core/runner/dispatch.go`).
- **The budget check.** Before a run is created each runner is asked for its remaining quota, counted in agent runs. Neither shipped runner reports one, and the check says so and is skipped (`internal/core/implement/loop/budget.go`).
- **The records.** A runner's receipt carries the model the runner reported as served (`internal/core/runner/receipt.go`). Review records name the model of each reviewer, and the pull request body gathers the models that the commits' `Assisted-by:` trailers name (`internal/core/landing/prbody.go`).

### Who it is for

- The product thinker (Iris), who starts a long run and expects it to make what progress it can.
- The technical facilitator (Nia), who sets the machine's model classes and has to say afterwards which model did what.
- The security engineer (Dave), who relies on a review having been made at the class the record names.

### What a person does

- Sets, in their machine's configuration, whether roles may fall back and which may not.
- Reads, at the run's start, which roles may fall back and to what.
- Reads, in the run's record and on its pull requests, which work a fallback model did.

### What a person never has to know

- How the harness words a limit.
- Which lane met the limit first.

## Mechanism

We expect a run with fallback switched on to deliver more lanes in one unattended period than the same run without it, because the roles that may fall back keep working while the stronger model's budget is out, and the run today stops every lane at the first rate-limit response. Shown wrong if the provider counts every class against one shared budget on the person's plan, so that the smaller model is refused at the same moment as the stronger one: the scope conditions of itd-2611101419006280 say that on a subscription the classes share one usage window. Shown wrong also if work done at the smaller class is sent back in review more often than it saves waiting, which the per-reviewer record of itd-2610240922461281 can count once a record names the model that ran.

## Scope Conditions

- A fallback acts only where abcd launches a role at a class through a runner. A role the host session runs is only asked for a class, and the model of the host session itself is not abcd's to change.
- A fallback helps only where the smaller class has budget left when the stronger one does not: a limit counted per model, a second provider mapped to a smaller class, or a budget the person sets per class. Where one limit covers every class, the run waits as it does today.
- The class order is the one `models.classes` already has. A fallback goes to a class, never to a model the map does not name.
- The implementer is launched with no class today, so it has nothing to fall back from until it is given one.
- The setting is read as the class map and the load settings are read: from the person's own machine. Whether a repository may narrow it is an open question below.

## What's Out of Scope

- Choosing a model by the difficulty of a task. This intent is about a model that cannot be used, not about which model suits the work.
- Raising the model above the class a role was given.
- Changing which classes review, or how many reviewers a role has. That stays with `review.reviewers`.
- Metering what each model has spent. The budget check's unit and the runners' missing quota reports are as they are.
- The fallback the dispatcher already has, from a runner to the host. Its receipts and its count are kept.

## Acceptance Criteria

The criteria below hold whichever way the open questions are answered. A criterion that depends on an answer says which.

- **Given** no fallback setting in any layer, **when** a run starts, **then** its behaviour at a limit is whatever open question 1 decides the default to be, and the start says in one line whether roles may fall back.
- **Given** fallback switched on and a role that may fall back, **when** the role's class is found exhausted by the detection open question 3 decides, **then** the role is launched at the next smaller class the map names, and the run's record says in one line which role, from which class and model, to which class and model, and why.
- **Given** a role the person has named as never falling back, **when** its class is exhausted, **then** the role is not launched at any other class, and the run does what open question 2 decides (waits for the class, or stops the lane), saying which in one line.
- **Given** a role that ran at a fallback class, **when** its receipt, its review record and any commit it leads to are written, **then** each names the model that did the work and the class that was asked for, and no surface shows the asked-for class's model as the one that ran.
- **Given** a review made at a fallback class, **when** the run decides whether the review round is complete, **then** it counts the review as open question 5 decides, and the record shows which rule was applied.
- **Given** a class that was exhausted and is available again, **when** the next role is launched at that class, **then** it runs at the class's own model, and the record says the fallback has ended.
- **Given** fallback switched on and the smallest class exhausted, **when** a role at that class is due, **then** the run waits as it does today, since no smaller class exists.
- **Given** a fallback setting in a repository's configuration, **when** a run resolves its settings, **then** it is treated as open question 6 decides, and a value that is not honoured is skipped with one line naming the machine's file.
- **Given** a run state written before this change, **when** a newer abcd reads it, **then** the run goes on with no fallback unless the person switches it on, and nothing in the stored review set changes.

## Open Questions

Each question below is the product thinker's to decide, and none is answered here. Each is given with the answers that can be defended and what each costs.

1. **Is fallback off by default, or on?**
   - Off: a run behaves as it does today until the person switches it on. Cost: the stall the person reported stays for everyone who has not found the setting.
   - On for roles that may fall back: an unattended run makes progress with nothing configured. Cost: work is done at a smaller model without the person having asked, and the first sign is in the record.
   - On, with a question at the run's start: the start names what may fall back and asks. Cost: one more question at a start the terminal-output draft is trying to shorten (itd-2611141610192096), and no answer where nobody is at the terminal.

2. **Which roles may never fall back, and what does such a role do when its model is exhausted?**
   - Which roles: the candidates raised are the security reviewer, the reviewer of a change that touches a trust boundary, and a design review. A fixed bundled list is predictable and cannot be loosened by mistake. Cost: a person who wants a cheaper security review on a small repository cannot have one. A list the person sets, with a bundled default, fits each machine. Cost: one more setting, and a person can remove the exclusion without noticing what it protected. No exclusions at all is the simplest rule. Cost: the review that matters most can be made by the smallest model, with only the record to say so.
   - What the role does: it waits for its class while other lanes go on. Cost: the lane holds its reservation and its worktree for as long as the wait lasts. Or the lane stops and is handed back. Cost: finished and passed work in the lane is set aside, which the drive-operations note records as a loss (event 2).
   - How a change is known to touch a trust boundary is part of this question: by the paths it changes, by a mark on the issue or intent, or by the security reviewer being in the review set at all. Each can be wrong in one direction, and the cost of a miss is a review at the wrong class.

3. **How is exhaustion detected, and how does the run return to the stronger model?**
   - A rate-limit response from the harness, which the runner already recognises, with the reset time it names. Cost: the response does not say whether the limit is the model's or the whole plan's, so a fallback can be tried and refused at once.
   - Any error from the harness that names the model. Cost: an outage or a refused route reads as exhaustion, and work drops to a smaller model for the wrong reason.
   - A budget the person sets per class, in agent runs, which the run counts itself. Cost: the count is abcd's and not the provider's, other sessions of the same account spend the same budget unseen, and the person has to pick a number.
   - The return: the existing short probe asks whether the limit has lifted. Cost: each probe is a launch at the stronger model. Or the reset time the response named is trusted. Cost: the response does not always name one. Or the run stays at the smaller class until it ends. Cost: a run of many hours finishes below the class it was given long after the budget is back.

4. **How is a fallback recorded?** Reviews and `Assisted-by:` trailers name models, so a record that names the asked-for model when another ran is wrong.
   - A receipt for the event, as the dispatcher's existing fallback writes one, and the model that ran in every place a model is named. Cost: every place has to be found and changed together, and a missed one is a false record.
   - The same, and a marker in the commit trailer and the pull request body that the work ran below its class. Cost: a new trailer form that the attribution check and the pull request body composer both have to read.
   - A line in the run's record alone. Cost: a commit and a pull request read later, away from the run's record, do not show it.

5. **Does a review made by a fallback model count as the same review class?**
   - Yes: the round is complete, and the record names the model. Cost: the review set the person chose is not the one that was applied, and the merge goes ahead on it.
   - No: the review is kept as advice, and the round stays open until the class's own model has reviewed. Cost: the fallback saves no waiting for reviews, only for building.
   - It counts for the correctness review and not for the security review. Cost: two rules for two roles, which every surface that shows a round has to explain.

6. **Is the setting per machine, or per repository?** The load settings and the budget-related settings are per machine and are never read from a repository.
   - Per machine only, as `models.classes` and `models.effort` are. Cost: a repository cannot say that its reviews never fall back, whoever runs them.
   - Per machine, and a repository may narrow it (name more roles that never fall back) and never widen it. Cost: a second layer to read and to show, and the merged result has to be explained at the start.
   - Per repository. Cost: a repository would decide how the person's budget is spent on their machine, which the class map's rule exists to prevent.

## Typed Links

- builds_on: itd-2611101419006280 (the model set is the person's, in classes named for their use). The fallback acts on that intent's class map and its order, and follows its rule that the map is the machine's.
- related: itd-2610240922461281 (a person sets how many models review). Its review set is what a fallback changes part-way through a run, and its per-reviewer record is where the mechanism's second test is counted.
- evidence: `.abcd/development/research/notes/autonomous-run-2026-11-14.md`, event 10.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
