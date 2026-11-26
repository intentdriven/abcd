---
schema_version: 1
id: "iss-2611260602175153"
slug: "an-intent-s-open-interview-questions"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "examining a stop-hook loop, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/startcheck.go"
remedy: "While the away mode holds, have the open-questions row and the record's next move name the away route instead of the interview: a technical question is decided with abcd decide and marked resolved in the record's Decisions citing that decision, and a question only the person may answer is queued with abcd mode queue while the session takes the next ready item, so no question stops work. Prove it with a start-check test that the away remedy names decide and queue and never the interview, and one that the remedy stays the interview while the person is present."
refines: [iss-2611211700231699]
resolution: "An intent's open planning question no longer stops autonomous work while the person is away: the build's start check, intent ready, the record page and the away block name one remedy (intent.OpenQuestionsRoute) that, while away, decides a technical or product question with abcd decide, queues one only the person may answer with abcd mode queue, and takes the next ready item meanwhile; dcn-2611260629037745."
impact: fix
---

With the person away, an intent's open interview questions stop autonomous work: the build's start check refuses an intent that still asks a question, and the only remedy it and the record description name is the planning interview, a human-session act, so a session holds the intent for the person's return even when the question is technical and the away mode and the facilitator on auto tell it to decide and record such questions itself. The person reports it has happened before (2026-11-26) and asks that interview questions never block work while they are away again.

## Grounds

- pursued: we expect an away session meeting an intent with open questions to decide or queue them and move on rather than stop; a session that halts on an open question while away would show it wrong
