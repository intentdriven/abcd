---
id: itd-2611251834408574
slug: ideate-needs-a-web-reader-a-walled
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_issues: [iss-2611251828573245]
origin: extracted-from-record
production_mode: hand-written
---

# An ideate run says what each leg needs before it starts

## Press Release

> Before its first leg, an ideate run now says what each leg needs and whether this session has it. Maya, an autonomous-development practitioner, asks for an idea to be judged in a session started on abcd alone, which holds no tool that reads the web. In place of a research leg that fails part-way, the agent's first command answers at once: the research leg cannot run here, because it opens primary sources on the web; the record grill and the adversarial review can; and no verdict is recorded from fewer than all three. Maya reads which parts are on offer and what a full run needs, and chooses: the grill now, or the whole gauntlet in a session that can read the sources.

## Why This Matters

Graduated from iss-2611251828573245, which holds the observation. An ideate run's first leg opens primary sources, and a source that is neither a record nor a filed source is a web address. The tool list of a session on abcd alone holds no tool that reads one: the hand-written start script and the list of itd-2611251725203320 were both checked on 2026-11-25. Neither the page (`commands/ideate.md`) nor the verb says so before the first leg starts, so the gap is met once the work has begun, and the recorder then has nothing it can accept, since `abcd ideate record` refuses a verdict whose legs are missing. The page says what each leg does and not what it needs. The gauntlet (itd-104) is optional and stays so; what changes is when a run that cannot finish is told.

## Mechanism

We expect the stop to come before the first leg because the page's first step becomes a command whose answer names each leg as able or unable to run here, and an agent that has just read that the research leg cannot run holds that in its context at the point where it would begin. The answer rests on what the session says it holds, the one thing an agent knows and a command it runs is not shown. Shown wrong if a session with no reader of web sources, handed the new page, still starts the research leg, read from its transcript as a search or a fetch attempted, or as a claim marked against a source it never opened.

## Scope Conditions

- Holds for a session that follows the ideate page. The check is the page's first step, and nothing holds a session that skips the page.
- Holds for what the session says it holds. A command run inside a session is shown no tool list, so a wrong statement by the agent gives a wrong answer about the legs.
- Holds for the three legs and the needs the page gives them today. A leg that changes later changes the table in the same change.
- Holds with ideate optional, as it is: the check is never asked of a capture or of a draft intent.
- Gives no session a reader of web sources. Whether a session on abcd alone may hold one is itd-2611040344170206's.

## Acceptance Criteria

- **Given** the ideate page a session is handed, **when** an agent reads it, **then** above the legs it finds what each one needs: for the research leg, a reader of web sources for every claim whose primary source is a web address, and none for a claim that rests on a filed source or a record id; for the record grill, reading and searching the checkout's record and this machine's private verdicts; for the adversarial review, a way to start two agents, each with a context of its own. It also finds the one command to run before the first leg. The table is generated from the binary, as the page's command facts are, so a reader of the page and a reader of `abcd explain ideate` see the same needs, and the sync check fails when they differ.
- **Given** a session that says it holds a reader of web sources and can start sub-agents, **when** the agent runs the check, **then** it reads each leg named as able to run, the exit is 0, and nothing is written.
- **Given** a session that names no reader of web sources, **when** the agent runs the check, **then** the exit is not 0 and the answer it reads, plain and under `--json`, names the research leg as unable to run and why; names the record grill and the adversarial review as able to; says the research leg can still take a claim whose primary source is a filed source or a record id; says no verdict is recorded from fewer than three legs; and gives the next step in one line, the whole gauntlet in a session that holds what is missing. It promises no reader to a session on abcd alone, and nothing is written.
- **Given** a session that cannot start a sub-agent, **when** the agent runs the check, **then** it reads the adversarial review named as unable to run, since each reviewer must be an agent with a fresh context, and the exit is not 0.
- **Given** a check run with no statement of what the session holds, or one naming something the verb does not know, **when** it runs, **then** it is refused with exit 2, the line names the words it accepts, and no leg is called able to run on a guess.
- **Given** the check has named a leg as unable to run, **when** the agent goes on under the page, **then** the page tells it to start no leg so named, to relay the answer to the person in the check's own words with the parts on offer, and, where no person is there to choose (the away mode), to run none of them and leave the answer in its report. A reader of the session's transcript finds no leg started after such an answer without the person's choice.
- **Given** a part on offer is run by itself, the record grill for one, **when** it ends, **then** the person reads its findings in the session, and no verdict record and no decision is written. `abcd ideate record` is unchanged: its tests pass untouched, and it still refuses a verdict whose legs are missing or out of order.
- **Given** a capture, a draft intent, or `abcd ideate record` run with no check before it, **when** it runs, **then** nothing asks for the check, warns that it was skipped, or refuses for it.

## Open Questions

- **Deferred** Whether a session on abcd alone may hold a reader of web sources is itd-2611040344170206's to settle, as iss-2611251828573245 says. Nothing here needs the answer: this reports what a session holds and gives it nothing.

## Decisions

1. The product thinker's, on 2026-11-25: this is an intent, it matters, and it is built in their six-day unattended run. They gave no wording for a criterion. The criteria, the Mechanism, and the scope conditions are the writing session's, written from the remedy on iss-2611251828573245, for their review on return; a change then is made through the record.
2. The writing session's, no question put: the session says what it holds, and abcd takes its word. A command run inside a session is shown no tool list. To reverse: read the list abcd itself handed a session it started, once the start command of itd-2611251725203320 has landed.
3. The writing session's, for review: a part run by itself records no verdict, and the recorder is unchanged. A verdict stands so that an idea is not argued again, and one whose research leg could not open a source would stand on a missing tool. To reverse: give the verdict a field naming a leg not run, and let the recorder accept it.
4. The writing session's, for review: a shell is not counted as a reader of web sources. This is the issue's own reading, since the list it checked holds a shell and was found to hold no reader.
5. The writing session's, no question put: the name of the check and the words a session uses to say what it holds are the spec's, since a name is a technical choice.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent: a run is told, before its first leg, which legs this session can run. No trust rule: what a session on abcd alone may hold is not touched, and stays with adr-2610222222326104 and itd-2611040344170206. No new stance: that ideate is optional and never a gate is itd-104's, kept by the last criterion. The table of needs, the check, and the generated block are plumbing, for the spec and for the brief's ideate chapter (`04-surfaces/18-ideate.md`). Typed links: `refines` itd-104, adding to its gauntlet what each leg needs and a check before the first; `related_issues` iss-2611251828573245, which the change that ships this resolves. No reversal is flagged. Verdict: FILE-AS-IS, the routing for a person to confirm.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
