# Autonomous run A, 2026-09-23 to 2026-09-24: what it did, what it cost, and what a days-long run needs

Dated 2026-09-24. Composed at the run's end from three sources: the run log,
which lives in the machine store (`~/.abcd/runs/<root-sha>/`, one JSON line
per event, two files for the two UTC days) and is never committed; the
orchestrator's handover notes in the local tier; and the observations the
interviewing session recorded while the product thinker ruled on what the
run could not decide alone. The run's charter was to implement every READY
intent and every fixable capture in abcd without a person at the terminal,
then cut the release. A second session (session B) joined for one window
and landed one lane.

## What the run did

The run opened at 05:48Z on 2026-09-23 and ended when v0.10.0 was published
at 06:33Z on 2026-09-24, 24 hours and 45 minutes later. About 16.9 of those
hours were work windows and 7.9 were four pauses. The pauses ended at 02:02Z
on the second day, when the product thinker ruled that the run works
continuously.

- **Pull requests.** Thirty merged, #661 to #693: 29 from the first session
  and one (#663) from session B. Three more (#672, #673, #676) went DIRTY
  after other merges and were closed and replaced by new pull requests.
- **Records.** Thirty intents entered `shipped/` and 37 specs closed. The
  first lane found 23 intents whose work was already on main with their
  specs still open, and #661 closed them. In the issue ledger 26 issues were
  resolved and 5 declined, while the run filed 39 new captures, so the open
  ledger grew from 490 to 508. Planned intents fell from 81 to 53.
- **The release.** v0.10.0, impact breaking, 43 records since v0.9.0: nine
  shipped intents and 34 resolved or declined issues. It is the first
  release whose plugin archive is pinned by digest in the marketplace
  catalog, and the first with a press release page, `RELEASE.md`.
- **What the lanes were.** Batch 0 routed the 490 open captures (249 to
  build, 203 for the product thinker, 38 to judge on their merits, 11
  duplicates) and audited the 79 READY intents (23 delivered, 9 partly, 47
  not). The run built its own coordination: two sessions sharing a run
  (#662), `abcd peers` (#666), and the RS005 delivery gate (#667). Nine pull
  requests recorded fidelity audits of shipped intents. Three cleared the
  majors that the release guard refused (#675, #678, #679). The features:
  the generated surface appendix (#671), the LOAD rule domain (#677), the
  versioned plugin archive (#681), the release page (#683), the load
  warning before abcd's own test lanes (#684), the guard's per-call working
  directory (#685), report and inbox (#687), the two-sided promote links
  (#689), count-based cost guards in the scanner tests (#690), and the
  skipped-record answer (#692). #668 wrote the interview's rulings into the
  record, and #693 was the cut.

### What was deferred, and why

- **Batches 1 to 5, mostly.** itd-60, itd-2609150819445595, the remainder of
  itd-2609211913453478, itd-2609212137116617 and the reading-corpus lanes
  (itd-34, itd-28, itd-42, itd-48, itd-53) stay planned. The run's hours went
  first to what the release guard refused (majors filed since v0.9.0,
  including ones the run itself found), then to the press release, which the
  product thinker made a condition of the tag and which needed a planning
  interview and three review rounds. The corpus lanes could only go through
  one at a time, since each lane recalibrates the reading windows at its
  merged tip.
- **The load check.** itd-2609231434459890 is built and merged as a warning
  (#684), but the intent stays planned through a remainder close. Review
  found that its stray rule cannot see foreign busy loops running between
  about 1.125 and 4 times the core count. iss-2609231947544298 (major) is
  deferred until the product thinker rules on how a stray is defined.
- **itd-7 and the unspecced drafts.** itd-7 waits on itd-6. itd-6 passes its
  readiness gate but cannot be built yet: it needs the implement verb's
  review stage, the configuration ladder, a CLI runner, and an MCP client
  dependency that needs sign-off. The run wrote planning briefs instead,
  for itd-7 and four drafts (itd-33, itd-2609151838312703,
  itd-2609150819440345, itd-2609151838327688).
- **The product thinker's queue.** Of the 203 captures routed to the product
  thinker, the 35 majors were ruled one by one. The rest took the
  interview's default and are deferred past v0.9.0.
- **Rulings owed at the end.** The stray definition above; whether the
  release page requires a told headline's quote (iss-2609232155567377); and
  whether report-back scrubs sender names that are common words or three
  letters or fewer (iss-2609240133237651).

## Pace, from the run log

`go run ./cmd/abcd implement report`, over both days of the log (805
events):

```
mode         windows wall min opened landed B land collisions backoff m  agent m  ceiling
unset              2    187.1      4      0      0          0       0.0      0.0      0.0
single             1    176.3      7      2      0          0       0.0    271.0     27.0
claim              3   1121.0     44     10      1          0       0.0   1605.0      0.0
```

The table undercounts the run, and the reason is itself a finding. The
report counts a lane as landed only when its `lane_close` line carries an
outcome, and it counts agent minutes only from `agent_end` lines. The
orchestrator that took over at the first rotation wrote neither: it wrote
`lane_close` lines carrying the pull request and merge sha, and no agent,
gate, review or ceiling-wait lines for the rest of the run. So the report
shows 11 lanes landed where 30 pull requests merged, and its agent minutes
stop at 16:08Z on the first day. Nothing flags the gap, because every line
parses (iss-2609240646555891). The figures below are derived from the log
directly, and each says which part of the run it covers.

**Sessions and lanes.** Merged pull requests per stretch of the run, with the lanes
opened beside them (after the first rotation, review and fix rounds were
also logged as lanes, so the later counts run high):

| Stretch (UTC) | Orchestrator | Merged | Lane opens |
| --- | --- | --- | --- |
| Window 1, 05:48–07:44 | first | 1 | 7 |
| Pause, to 09:40 | — | 1 | 1 |
| Window 2, 09:40–12:29 | first (+ session B) | 7 + 1 | 10 |
| Pause, to 14:26 | — | 2 | 0 |
| Window 3, 14:26–17:26 | first, then second from 16:15 | 2 | 9 |
| Pause, to 19:25 | — | 3 | 1 |
| Window 4, 19:25–22:26 | second | 3 | 12 |
| Pause, to 00:26 | — | 3 | 0 |
| Window 5 and the continuous tail, 00:26–06:33 | second, then third from 04:03 | 7 | 15 |

Nine of the thirty merges happened during pauses, because the merge queue
kept working while no agent did. Over the whole run that is about 1.2 merges
per wall-clock hour and 1.8 per working hour.

**Agent minutes by role** (69 agents that ended before 16:08Z on the first
day, the part of the run the log covers):

| Role | Agents | Minutes | Tokens |
| --- | --- | --- | --- |
| Implementer | 20 | 790 | 4.37 M |
| Fix implementer | 19 | 614 | 2.50 M |
| Ruthless reviewer | 16 | 225 | 1.65 M |
| Auditor | 6 | 161 | 1.19 M |
| Security reviewer | 6 | 61 | 0.57 M |
| ADR second lens | 2 | 25 | 0.17 M |
| **Total** | **69** | **1,876** | **10.45 M** |

By model, Opus 5.5 ran 52 agents (1,531 minutes) and Fable 5.1 ran 17
(345 minutes); the Fable agents were reviewers and auditors, one at a time.
Of the 26 review lines logged, 15 were FIX FIRST and two AMEND FIRST,
against seven SHIP, one of them with a medium finding fixed first. A fix
round after the first review was the rule, not the exception.

**What the ceiling cost.** The run had four ceilings: two implementers
and one reviewer at the start, then five agents at 09:09Z, six at 14:10Z,
and four from 02:02Z on the second day. Ceiling waits came to 27 minutes
in window 1 under the first ceiling, none in window 2 at five, and about
21 minutes in window 3 at six, counted from the waits' start and end
lines, when six lanes were open at once; after 16:08Z on the first day
none was logged. Nothing counted the agents alive, and the run went over its ceiling three times: self-forking agents at the
start, six alive against five at 10:28Z, and five against four during the
cut (iss-2609240646542516). After the first raise the machine limited the
run more than the slots did. The load recorder took 254 samples: mean
load 13.3 on 16 cores, above the core count in 56 of them, a peak of 134 at
20:04Z, and up to 12 preflights alive at once across every session on the
machine. A preflight that passed took 743 seconds on average (longest
1,199), and a pre-push preflight 909 (longest 1,313).

**Gate failures by gate** (66 gate runs logged before 16:08Z on the first
day: 50 preflights, 14 pre-push preflights and two format checks; the 13
failure lines below come from 11 preflights and 2 pre-push preflights):

| Cause | Failures | Notes |
| --- | --- | --- |
| Reading-window eval | 3 | a corpus lane's merged tip over its declared window; recalibrated |
| RS004 (an `iss-` named without a `Refs:` line) | 2 | fixed by adding the line |
| record-lint | 2 | a link to a moved intent; an auditor's prose citing a fixture's id |
| docs-lint | 2 (one run) | harness naming without the allow marker; an unqualified `go run` |
| Tests | 3 | one canonical-list test, one flake, one test that was not hermetic |
| External SIGTERM | 1 | a peer agent's pattern kill (iss-2609240646538696) |

The notes record further failures after the log's coverage ends: RS004 in
two pull-request bodies, the payload scan reading the CI machine's account
name as private (iss-236), RS005 on the load-check branch
(iss-2609240646533487), and the scanner-scaling flake under load.
**Fallbacks:** none. Every agent ran on the host.

## The long-running-session experiment

The product thinker made the run an experiment in keeping an orchestrator
alive for days rather than hours. Three orchestrator sessions ran it, with
two handovers. Five more reserve sessions were opened in advance and never
needed, because only a person can open a session and the product thinker
was away after the first day.

| Orchestrator | Context over its life | How it ended |
| --- | --- | --- |
| First (abcd-a6), 05:48Z–16:15Z | 16% after start-up reading; 27%, 45%, 55%, 65%, then 72% | handed over at 72%, above the ~60% target: its own figures were estimates, and the harness figure came from the product thinker |
| Second (abcd-5b), 16:15Z–04:03Z | 4% after reading the handover; 13%, 36%, 48% | handed over at 48%, before the cut, so the cut ran in one fresh window |
| Third (abcd-63), 04:03Z to the end | not logged | cut and published v0.10.0 |

No compaction was logged in any session.

- **Start-up reading cost a sixth of the first session's context.** The run
  file, the rules, the principles and the records the orchestrator read
  before its first lane took 16% of its context. The next run's prompt should hand
  the orchestrator a digest, not the sources.
- **A handover block is cheap to read.** Each block in the notes ran to
  about 650 words, and a fresh orchestrator resumed from one at 4% of its
  context. Writing it cost the outgoing session more: a whole pass over
  the run's state, hazards, owed items and queue, at the point where its
  context was scarcest.
- **Measurement did not survive the first handover.** The block said where
  the log was and not what to write in it, so the successor kept the lane,
  claim and pull-request lines and dropped the rest. What a session writes
  to the log is a habit, and habits do not hand over. The verb has to
  write the log itself (iss-2609240648533996).
- **Rulings live at the top of the notes file.** Each successor re-read
  every ruling in force (the pace, the ceiling, the agenda line that
  authorised the publish approval) from prose lines at the head of the run
  section, and a correction to one ruling had to be written as a second
  entry.

## The product thinker's interview

While the run was paused, a separate session interviewed the product thinker
on everything the run could not decide alone. It recorded 24 observations
for abcd to learn from. Grouped:

- **Voice and surface.** The owed items were written in the facilitator's
  register (record ids, mechanisms), so each question needed one to three
  lookups before it could be asked in product terms. Text above a question
  did not reach the product thinker; the same text inside the question's
  option previews got an immediate pick.
- **Defaults and consent.** What happens on silence changed from item to
  item and was stated for one item only. A default offered inside a
  question is not consent: severity, taken from silence, would have been
  wrong, and when asked the product thinker overrode it.
- **Where a ruling lives.** Authority counts only as a DECISIONS.md line, a
  tracked file the interviewing session was told not to touch, so the
  rulings sat in a gitignored scratch file until the run transcribed them.
  A crash in between would have lost them. The run also asked the
  interviewing session to commit for it, which no peer message can
  authorise.
- **Cost and triage.** No option carried a cost, although two rulings each
  added build work before the release. Routing 203 captures to a person one
  question at a time does not work; the product thinker chose to rule on the
  35 majors and take the defaults for the rest. "Decide later" had nowhere to
  put the reason that turns a deferral into a decision.
- **Timing.** The first four rulings came within four minutes of the
  product thinker joining the interview. Planning the press release took
  about fourteen questions and 35 minutes of the product thinker's time,
  after two reviews of about eight minutes each; three answers were not options at all ("show
  me", "explain with examples", and a counter-proposal that changed the
  build).
- **Evidence moves rulings.** The load-check ruling of the morning ("refuse
  when it is too high") became "warn, never refuse" once a review showed
  from the run's own load data that no threshold separates the crash from
  normal parallel work. The record shows it as a change, in a DECISIONS
  entry of its own.
- **Who is running this now.** The pace changed mid-interview, and the
  interviewer heard it from the run, not from the product thinker. Each
  orchestrator rotation reached the interviewing session as a peer message
  naming a new address, and nothing the product thinker could see said
  which session held the run.

One observation resolves on inspection: `abcd mode product-thinker` printed
nothing because this machine has a status surface installed, and the set
form prints its line only where there is none.

## What the numbers say

- **Past a ceiling of five, slots were not the main limit.** Waits fell to
  none at five and returned at six only when six lanes ran at once, and the
  cost moved to gate time on a loaded machine: twelve-minute preflights, fifteen-minute pre-push runs, and one
  in five load samples above the core count. A higher ceiling on the same
  machine buys concurrency and pays for it in gate time.
- **The serial points are structural.** Three things forced lanes into
  single file whatever the ceiling: the reading-window recalibration (one
  corpus lane in the queue at a time), the forge ignoring the union merge
  driver on DECISIONS.md (every records pull request went DIRTY when
  another merged first, iss-2609240646538011), and regenerated surface
  chapters (three pull requests replaced, iss-2609211105010877).
- **Review is the lane's longest leg.** Most lanes took a review, a fix
  round, and a review of the fix. Reviewers one at a time on one model made
  review the queue the lanes waited in, as the pilot found at a ceiling of
  two.
- **A days-long run is carried by its records, not by its sessions.** The
  run survived two orchestrator handovers, a run file that disappeared from
  where it was kept and was rebuilt from a transcript, and an unexplained batch of killed shells
  (iss-2609240646530590), because the notes and the committed records held
  the state. What did not carry over is what lived only in a session's
  habits: the measurement discipline, and which rulings were in force. The
  implement verb (itd-2609201916151817) should own both, and the seed filed
  at the end of this run (iss-2609240648533996) lists what else it needs.
- **The run's frictions are in the ledger.** Thirteen new records and four
  appended ones, all with `found_during: "autonomous run 2026-09-23"`.
