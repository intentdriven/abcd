---
id: itd-2609211116005482
slug: abcd-build-next-picks-the-readiest
spec_id: spc-2609212015048113
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609201925079472, itd-50]
related_intents: [itd-78, itd-82]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# `abcd build next` picks the readiest planned intent itself, and writes down why

## Press Release

> **`abcd build next` chooses the next intent to build, says why on the record, and builds it.** It reads every planned intent that passes every refusal check `abcd build <itd-N>` would run (READY, no open question, no unanswered claim, not held, not blocked, no peer holding it), scores each on facts its own record holds — how clear its acceptance criteria are, whether there is an obvious way to test it, how small the change it asks for looks — takes the readiest, oldest among equals, and writes onto that intent a grounds entry marked as the run's: who else was considered, why each lost, and what would show the pick wrong. The entry is the lane's first commit, so the reason reaches `main` with the work and stays on the branch as history if the lane is discarded. Then it hands the intent to the same machinery `abcd build <itd-N>` uses and takes it to delivered. One intent per run, because "next" means one; `--until-empty` or `--max <n>` keeps it going under the pace rule, which is how `abcd drain` reads by default, because "drain" means all.
>
> "I used to open the run file every morning and pick by feel, then forget by Friday why Tuesday's pick was that one," said a product thinker who had planned forty-eight intents and could build one a day. "Now the run picks, the intent says why in its own record, and when a pick turns out wrong the reason is there to argue with."

## Why This Matters

Forty-eight planned intents were READY in this repository on 2026-09-20, and the file that routes them into batches was written by hand, ordered by a person's sense of what comes first, with no line saying why. Every autonomous coding platform surveyed on 2026-09-21 (Copilot's coding agent, OpenHands, Jules, Devin, Cursor's cloud agents, Claude Code and Codex routines) is triggered by a human assigning a task or a label; none publishes a policy for choosing among a backlog, so a run that is meant to be unattended still begins with a human's choice. Two things the same survey found do have evidence: readiness is predictable from the record alone (static features of the task predict an agent's success at 0.76 to 0.84 AUC before any code is written), and small, tidy work merges far more often than large features (84.7 per cent for clean-ups against 64.5 per cent for features across 878 agent pull requests in one large repository). A pick made from those facts, written where a person can later check it, is what turns the run file's hand-ordered batches into something the binary does.

The reason has to be checkable. Agent-written rationales, when audited, were factually wrong more than half the time in one repository's decision log; the survey found no evidence that recording a reason improves later review, only that an unrecorded one cannot be reviewed at all. So the record the run writes is computed, not composed: the candidate set, each candidate's score and the rule that placed it, and a falsifier the lane's own outcome can meet.

## Mechanism

We expect a pick made from the record's own facts, with its reason written as computed comparison and a falsifier, to choose intents that deliver and to be correctable when one does not, because the facts that predict an agent's success are in the acceptance criteria, the test path and the footprint before a lane starts, and a written falsifier is what a later reader checks the outcome against; shown wrong if the falsifier on a failed pick did not name what actually failed, or if picks scored readiest end unachievable (itd-50's verdict) more often than not over ten or more picks.

## Scope Conditions

- Holds for a repository abcd manages whose planned intents carry acceptance criteria in Given-When-Then form and a linked spec, so the readiness score has fields to read; an intent with a placeholder criteria section is not a candidate. <!-- cond: cond-2609212015041824 -->
- Holds where the intent store's grounds section accepts an appended entry the run writes beside a human's; the human's entry is never edited or displaced. <!-- cond: cond-2609212015048560 -->
- Holds where `blocked_by` and `builds_on` edges are declared on the records; an intent with an unshipped blocker is not a candidate, and the edges are not otherwise ranked on. <!-- cond: cond-2609212015045488 -->

## What's In Scope

- **The candidate set**: every intent in `planned/` that passes every refusal check `abcd build <itd-N>` runs before it starts (READY, no open question, no unanswered claim section, not held, no peer holding it, nothing unshipped in `blocked_by`, a superseded blocker read as the intent that replaced it, per decision 9); the pick never writes onto an intent the build would then refuse.
- **The score**, from the record alone, each component named in the reason: criteria clarity (count and shape of the Given-When-Then bullets), a test path (the spec's `## Footprint` section naming the tests, absent today on every spec), and the expected footprint (that section's package list); three parts at equal weight, the bundled default, revisable after ten picks; age is not a part of the score and breaks ties only. Until a spec carries the section, its test-path and footprint parts read zero and the reason says so.
- **The reason**, written onto the chosen intent's `## Grounds` section as one entry marked as the run's (`pursued:` prefixed with the run's identity and date), carrying the candidates considered with their scores, the rule that placed the winner, the runner-up and why it lost, and the falsifier: a lane outcome the state file can meet (fix rounds past the pace rule's count, or the lane handing back as unachievable under itd-50's stage). The marker ("picked by run <id> on <date>") goes inside the entry's text after the `pursued:` token, since the token vocabulary is closed. The entry is committed as the lane's first commit on the lane branch.
- **The hand-over**: the chosen intent goes to the implement machinery as `abcd build <itd-N>` would send it, with one difference: after the loop's branch step and before the implementer starts, the pick step commits the grounds entry as a record-only commit at the branch base, and the loop's receipt verifier counts the implementer's commits from after it.
- **The gate**: `abcd intent ready`'s grounds row skips a run-marked entry when it names the most recent conjecture, so the human's entry stays the one it reports.
- **The run record**: names each pick, its score table, and whether its falsifier was met.
- **The run count**: one by default; `--max <n>` and `--until-empty` continue under the pace rule, each further pick written the same way.
- **Refusals**: no candidate (says so, names each excluded intent and the check that excluded it, writes nothing); a tie the score cannot break beyond age (takes the oldest and says the tie was broken by age).
- **The spec template**: gains a `## Footprint` section (packages, tests) the score reads; `intent plan` seeds it empty.
- **The command page**: `commands/intent.md` (or the build page once it exists) says what `next` does, its two flags and its refusals.

## What's Out of Scope

- The build itself, the lane, the validators, the landing: `itd-2609201916151817`.
- The pace, the ceiling and the pause: `itd-2609201925079472`.
- A derived priority over the dependency graph: `itd-78`; this intent filters on edges and does not rank on them.
- The issue ledger: `abcd drain` (`itd-82`) is the run over issues, and it hands over to the same machinery.
- Any estimate the agent makes of its own confidence as an input to the pick.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed this draft:

1. **The verb is `build`, for people; `implement` is the machinery, for agents.** `abcd build <itd-N>` and `abcd build next` are what a person types; both hand over to `abcd implement`, whose step-level words (`step`, `receipt`) a driving host calls. The planned implement intent is amended to say so; which verbs the command list shows a person and which it shows an agent is a separate record (`iss-2609211119023345`).
2. **Readiest first, oldest among equals.** Not oldest-first, not most-important-first; the evidence for readiness as the predictor is the reason, and the order is declared a heuristic.
3. **The reason lives on the intent itself**, as a grounds entry marked as the run's, so opening the intent shows why it was picked and what would prove the pick wrong.
4. **One by default, many by flag**, and the defaults differ from `drain` on purpose: "next" reads one and "drain" reads all; `--max <n>` and `--until-empty` are the two flags both share.
5. **The reason is computed, never composed.** Candidates, scores, placing rule, falsifier; no prose the agent writes about its own choice.
6. **The reason is the lane's first commit** on the lane branch (ruled 2026-09-21 on the design review's finding): a record-only commit the pick step makes at the branch base before the implementer starts, so it reaches `main` with the work and stays on the branch if the lane is discarded; the receipt verifier counts the implementer's commits from after it.
8. **The ordering is declared a heuristic** with the evidence in Why This Matters; it is revised on the record, never silently.
7. **Equal weights at first**, declared as the bundled default and revisable after ten picks; age breaks ties and is not scored.
9. **A superseded blocker waits on its replacement** (amended 2026-09-29 by ruling BZ2 of 2026-09-29, decision [dcn-2609290000000007](../../../work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md) landing with lane recRulings; the ruling supersedes the reading this record carried until then, that every blocker outside `shipped/`, a superseded one included, blocks). The blocked check follows a blocker's `superseded_by` to the intent that replaced it, transitively, and the intent is a candidate only when the last intent of that chain has shipped. A chain that loops, ends at a record this checkout does not hold or at a decision (`adr-N`), or stops at a superseded record naming no successor blocks, and the refusal names the chain. **Amendment, 2026-09-30:** the product thinker's rulings CF1 and CF2 of 2026-09-30 ([dcn-2609300000000009](../../../work/decisions/dcn-2609300000000009-the-person-answered-eighteen-rulings-of.md), the entry recording the eighteen rulings of that day) settle two of those ends, as lane cfSettled built them in `followBlocker` (`internal/core/intent/startcheck.go`): a chain ending at a decision (`adr-N`) is settled when that ADR's status is `accepted` (CF1), and refuses naming the decision when it carries another status or no status, or when this checkout's decision store does not hold it; and a blocker, or the last intent of its chain, that sits in `disciplines/` is settled as a standing rule (CF2), as a shipped one is. A chain that loops, ends at an intent this checkout does not hold, or stops at a superseded record naming no successor still blocks, and a settled chain names the record that settled it.

## Open Questions

- **Whether a run-made grounds entry needs its own token.** RESOLVED: no. The product thinker ruled on 2026-09-29 (ruling BZ1 of autonomous run A's owed list, recorded in [dcn-2609290000000007](../../../work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md)) that a run-made entry keeps the shared `pursued:` kind with the run's marker opening its text, as built; a token of its own for a machine's conjecture was the alternative not taken.

## Acceptance Criteria

- **Given** planned intents of which some pass every refusal check of `abcd build <itd-N>` and some fail one (not READY, an open question, an unanswered claim, held, blocked, a peer holding it), **when** `abcd build next` runs, **then** only those that pass are candidates, and the refusal for an empty candidate set names each excluded intent and the check that excluded it, writing nothing.
- **Given** two or more candidates, **when** the pick is made, **then** the chosen intent's `## Grounds` section gains exactly one `pursued:` entry whose text opens with the run's marker, naming every candidate with its score, the rule that placed the winner, the runner-up and why it lost, and a falsifier the state file can meet; no existing entry is changed; the entry is the lane branch's first commit; and `abcd intent ready` still reports the human's entry as the most recent conjecture.
- **Given** candidates whose scores tie, **when** the pick is made, **then** the oldest is taken and the entry says the tie was broken by age.
- **Given** a pick, **when** the lane starts, **then** it is the lane `abcd build <itd-N>` would start for that intent, with the same brief, worktree, validators and landing, and one record-only commit at the branch base carrying the grounds entry, which the receipt verifier does not count as the implementer's work.
- **Given** the default run count, **when** the picked intent is delivered or its lane stops, **then** the run reports and exits without picking again; **given** `--max <n>` or `--until-empty`, **then** it picks again under the pace rule and writes each further reason the same way.
- **Given** a pick, **when** the lane's state file shows fix rounds past the pace rule's count or an unachievable hand-back, **then** the run record names the pick as falsified and the intent's entry is not edited.
- **Given** `abcd intent plan` mints a spec, **when** the spec stub is read, **then** it carries an empty `## Footprint` section (packages, tests) for the author to fill.
- **Given** a spec without a `## Footprint` section, **when** the score is computed, **then** its test-path and footprint parts read zero and the reason says the spec carries no footprint.
- **Given** the command page, **when** it is read, **then** it states what `next` does, `--max` and `--until-empty`, and every refusal.
- **Given** `--json`, **when** any of the above renders, **then** the candidate set, scores, the reason and every refusal are in the payload.

## Prior Art

- The run file of 2026-09-20 (`abcd-autonomous-run.md`), whose hand-ordered batches this pick replaces.
- `itd-78` (the dependency graph): the derived priority this intent filters on and does not build.
- The state-of-the-art pass of 2026-09-21 (in the record-families of the platforms surveyed and the psychometrics and dotnet/runtime evidence cited above).

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement machinery, `abcd build <itd-N>`): the pick hands over to it and adds nothing to the lane.
- **builds_on `itd-2609201925079472`** (the pace): `--until-empty` and `--max` run under it.
- **refines `itd-78`** (the dependency graph): this intent filters on `blocked_by` and does not derive a priority; that draft is where a derived one would come from.
- **related `itd-82`** (`abcd drain`): the sibling run over the issue ledger; the two share `--max` and `--until-empty` and the hand-over to `abcd implement`, and differ in their default count by the meaning of their names.
- **builds_on `itd-50`** (the audit-driven fix round): the unachievable verdict is one of the pick's falsifiers.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-c99b85cd869d -->
Fidelity review — receipt rcp-c99b85cd869d (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:9307d185086739eeb259935a6253f4140f9ba45e9d3fa9cc5a3e7e9d6710d123
Input attestations: diff:a09dd42b6de9a6556d5ddc49b8810f65b5064b61..1868cf1e3c1c4410ca55e3063268c5dd12c26bfc@sha256:c97dda675ecbfa3d7930648aab344083d9830ce13c2997c7b459116930a3c8db; log:.abcd/.work.local/run/run-2610110908001704/lane-1/validate/round-1/fix/dod.log (implementer's make preflight at 1868cf1e3, exit 0, receipt minted)@-; test:auditor's own run on a git-archive copy of 1868cf1e3: go test ./internal/core/intent/ -run 'Pick|Readiness|Choose'; ./internal/core/spec/ -run Footprint; ./internal/core/implement/loop/ -run 'Next|Series|Pick|Removed|Entry|Record|Falsif'; ./internal/surface/cli/ -run 'BuildNext|ImplementRecord' — all ok@-;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: candidates() judges every planned intent with the build's own check() rows and records the first failing check per intent, plus the run check; an empty set is refused at the pick naming each exclusion; nothing is written before the refusal (beginSeries judges the set before EnsureRealDirAll, the single pick before startPick). Loop and CLI tests assert the exclusions by check and the absent run tier; they pass on my copy.
  evidence: internal/core/implement/loop/next.go:144-156 — "chk, err := check(repoRoot, id, session, snap) ... set.Excluded = append(set.Excluded, Excluded{ID: id, Check: row.Name, Reason: row.Detail})"
  evidence: internal/core/implement/loop/next.go:266-275 — "no planned intent passes every check `abcd build <itd-N>` runs; %d excluded: %s"
  evidence: internal/core/implement/loop/check.go:29-36 — "CheckReady ... CheckOpenQuestions ... CheckClaimSections ... CheckHold ... CheckBlocked ... CheckSteps ... CheckPeers"
  evidence: internal/core/implement/loop/nextseries.go:152-155 — "if !ok { return NextResult{}, emptyRefusal(set) } if err := fsutil.EnsureRealDirAll("
  evidence: internal/core/implement/loop/next_test.go:75-128 — "TestNextCandidatesAreWhatTheBuildWouldStart ... runTierAbsent(t, repo.Root())"
  evidence: internal/surface/cli/build_next_surface_test.go:78-100 — "TestBuildNextRefusalCarriesEveryExclusion ... runDirAbsent(t, repo.Root())"
- ac-2 — MET: PickEntryText opens with the run marker and names every candidate with its score, the rule, the runner-up and why it lost, and the falsifier; the worktree stage appends it through intent.RecordGrounds and commits that one file; isPickCommit proves the commit's parent is the base and its record is byte for byte the base's plus one entry; ready.go skips run-pick entries when naming the most recent conjecture. next_test asserts exactly one appended entry, no changed line, first-and-only commit past the base, and the gate's detail.
  evidence: internal/core/intent/pick.go:232-252 — "b.WriteString(RunPickMarker + runID + " on " + ... "; candidates: ") ... "; rule: " ... "; runner-up: ... "; falsifier: ""
  evidence: internal/core/implement/loop/pickcommit.go:147-173 — "intent.RecordGrounds(wt, st.Intent, g) ... pickGit(wt, "commit", "-q", "-m", pickMessage(st), "--", rel)"
  evidence: internal/core/implement/loop/pickcommit.go:211-236 — "its record is not the base's with this run's one entry appended and nothing else changed"
  evidence: internal/core/intent/ready.go:349-364 — "if !isRunPick(g) { human = append(human, g) } ... most recent %s: %s ... run pick entry(ies) skipped"
  evidence: internal/core/implement/loop/next_test.go:181-214 — "the pick's entry is the branch's first and only commit past its base ... exactly one run-marked pursued entry is appended ... the gate still reports the person's entry"
- ac-3 — MET: PickLess orders by total then IDOlder; Choose sets TieBrokenByAge when the runner-up's total equals the chosen's, and the entry text then says the tie was broken by age; tested at the unit (ordinal id before timestamp ids) and in the series test (itd-11 over itd-12).
  evidence: internal/core/intent/pick.go:166-171 — "if a.Score.Total != b.Score.Total { return a.Score.Total > b.Score.Total } return IDOlder(a.ID, b.ID)"
  evidence: internal/core/intent/pick.go:212 — "p.TieBrokenByAge = ru.Score.Total == p.Chosen.Score.Total"
  evidence: internal/core/intent/pick.go:246 — "tied at %d and younger, so the tie was broken by age"
  evidence: internal/core/intent/pick_test.go:58-72 — "TestChooseTakesTheOldestOfATieAndSaysSo"
  evidence: internal/core/implement/loop/nextseries_test.go:111-113 — "second.Pick.Chosen.ID != "itd-11" || !second.Pick.TieBrokenByAge"
- ac-4 — MET: startPick calls the same start() that Start (build < itd-N>) wraps, with a RunPick beside it, so the run's stages, brief, worktree, validators and landing are the build's; the worktree stage of the pick's lane makes the record-only commit at the base and records PickSHA; the receipt verifier names a commit inside PickSHA as not the implementer's work and pickKept refuses a branch that dropped it. next_test asserts the same key/spec/lane/stage and the receipt refusal.
  evidence: internal/core/implement/loop/next.go:253 — "started, err := start(repoRoot, p.Chosen.ID, o, &RunPick{Pick: p, Excluded: set.Excluded})"
  evidence: internal/core/implement/loop/loop.go:282 — "func Start(repoRoot, key string, o Options) (StartResult, error) { return start(repoRoot, key, o, nil) }"
  evidence: internal/core/implement/loop/lane.go:228-235 — "if p := c.State.Pick; p != nil && p.Lane == lane.ID { sha, err := pickCommit(c, lane, lw.Path, lw.Branch, base)"
  evidence: internal/core/implement/loop/receipt.go:384 — "if err := pickKept(c.RepoRoot, lane, receiptRel); err != nil {"
  evidence: internal/core/implement/loop/receipt.go:517-532 — "case inPick: off = append(off, shortSHA(sha)+" (the pick's record-only commit, not the implementer's work)")"
  evidence: internal/core/implement/loop/next_test.go:169-172 — "the pick starts the build's own run"
  evidence: internal/core/implement/loop/next_test.go:226-234 — "the pick's commit is not the implementer's ... the implementer's own commit verifies"
- ac-5 — MET_WITH_CONCERNS: The default and --max 1 pick once and write no series (TestADefaultPickLeavesNoSeries); --max above 1 or --until-empty begins a series whose move reads the last pick's run (delivered, handed-back, removed, in-progress), honours the window clock and next_eligible_at under a fixed clock, excludes already-picked intents, and starts each further pick through the same startPick, so each reason is written the same way; the removed outcome and the lock contention now have tests, and every test passes on my copy. Concerns, named: (1) 'picks again' is one move per further `abcd build next` invocation, host-driven as the loop is, not the run continuing on its own; (2) the pace rule's window and pause are enforced across the series, but its ceiling and fix rounds are forwarded to each run as the series' typed flags and are not applied across the series.
  evidence: internal/core/implement/loop/nextseries.go:130 — "func (n NextOptions) many() bool { return n.UntilEmpty || n.Max > 1 }"
  evidence: internal/core/implement/loop/nextseries.go:9-10 — "The loop is driven by the host session, so a series is too: each `abcd build next` makes one move and exits."
  evidence: internal/core/implement/loop/nextseries.go:231-246 — "outcome, falsified, err := seriesOutcome(repoRoot, pk.RunID) ... if outcome == SeriesPickInProgress { ... return finish(changed) }"
  evidence: internal/core/implement/loop/nextseries.go:255-280 — "The window clock, as a run keeps it: before next_eligible_at nothing is picked"
  evidence: internal/core/implement/loop/nextseries.go:283-284 — "so.Pace, so.SubAgents, so.FixRounds = st.Flags.Pace, st.Flags.SubAgents, st.Flags.FixRounds"
  evidence: internal/core/implement/loop/nextseries.go:304-308 — "picked, err := startPick(repoRoot, so, set, p) ... st.Picks = append(st.Picks, SeriesPick{"
  evidence: internal/core/implement/loop/nextseries_test.go:67-79 — "TestADefaultPickLeavesNoSeries"
  evidence: internal/core/implement/loop/nextseries_test.go:86-155 — "TestMaxPicksAgainOnceThePickIsDelivered ... the further pick's reason is written the same way"
  evidence: internal/core/implement/loop/nextseries_test.go:202-244 — "TestASeriesPicksOnlyInsideItsWindow"
  evidence: internal/core/implement/loop/nextseries_test.go:324-392 — "TestARemovedPickIsPickedPast ... TestASeriesMoveBacksOffWhileAnotherHoldsItsLock"
- ac-6 — MET: The fix-round bound hands a lane back with Verdict unachievable and no Kind once the fix rounds taken reach the cap, so the state file shows both halves of the falsifier as one event; pickFalsifiedBy matches exactly that hand-back, handBackLane appends a `pick` record line naming the pick falsified, recordPick sets falsified/falsified_by, and `implement record` renders it. The new loop test drives a picked run through the real validate path with --fix-rounds 1 to a hand-back and asserts the intent on the lane branch and in its worktree is byte for byte the pick commit's, carrying the one entry; it passes on my copy.
  evidence: internal/core/implement/loop/validate.go:184-185 — "if taken, limit := lane.fixRoundsTaken(), c.State.FixRoundCap(); taken >= limit { hb := HandBack{Verdict: VerdictUnachievable, Round: cur.Round, FixRounds: limit"
  evidence: internal/core/implement/loop/record.go:93-96 — "p.Falsified = true; p.FalsifiedBy = fmt.Sprintf("%s was handed back as %s after %s""
  evidence: internal/core/implement/loop/record.go:103-110 — "if l.HandBack != nil && l.HandBack.Kind == "" { return l.ID, *l.HandBack, true }"
  evidence: internal/core/implement/loop/handback.go:31-35 — "the pick of %s is falsified: %s was handed back as %s after %s; the intent's grounds entry is left as it was written"
  evidence: internal/core/implement/loop/nextseries_test.go:432-486 — "TestAPickedRunHandedBackAfterItsFixRoundsLeavesTheEntryAlone ... the intent on the lane's branch is the pick commit's, unedited"
  evidence: internal/surface/cli/build_next_surface_test.go:190-228 — "TestImplementRecordCarriesThePick"
- ac-7 — MET: intent.Plan calls spec.Create, which writes renderSpecRecord; that renderer iterates stubSections, whose entries carry `## Footprint` with the empty body `- packages:\n- tests:` above `## Steps`; footprint_test asserts the minted stub carries it present and empty, and passes on my copy.
  evidence: internal/core/intent/lifecycle.go:291 — "sp, err = spec.Create(repoRoot, intentID, it.Slug, opts.ProductionMode)"
  evidence: internal/core/spec/store.go:222 — "[]byte(renderSpecRecord(id, slug, intentID, intents, bundle, stamp, steps))"
  evidence: internal/core/spec/spec.go:345 — "for _, sec := range stubSections(id, intentID, steps) {"
  evidence: internal/core/spec/spec.go:308-312 — "{heading: FootprintHeading, body: footprintStubBody}"
  evidence: internal/core/spec/footprint.go:35 — "const footprintStubBody = "- packages:\n- tests:""
  evidence: internal/core/spec/footprint_test.go:13-32 — "TestRenderSpecSeedsAnEmptyFootprintSection"
- ac-8 — MET: readinessWith sets NoFootprint and zero-point test-path and footprint parts whose detail reads 'the spec carries no footprint' when ReadFootprint finds no section, and candidateText appends 'its spec carries no footprint' to the reason; tested at the unit and passing on my copy.
  evidence: internal/core/intent/pick.go:104-109 — "case !fp.Present: s.NoFootprint = true; s.TestPath = ReadinessPart{Detail: "the spec carries no footprint"}"
  evidence: internal/core/intent/pick.go:259-261 — "if s.NoFootprint { t += ", its spec carries no footprint" }"
  evidence: internal/core/intent/pick_test.go:41-54 — "TestReadinessWithoutFootprintReadsZeroAndSaysSo"
- ac-9 — MET_WITH_CONCERNS: commands/build.md's 'Pick the next intent' section states what next does, the score, the entry and the commit; 'One pick, or a series' states --max and --until-empty and the series' moves; the refusals list now names the empty set, a negative --max, two counts, another count or pace on a live series, the three series contentions (exit 3), every next.json refusal readSeries raises, the grounds-gate refusal, a run already in progress, and the build's own refusals, matching the code. Concern, named: of the pick commit's worktree-stage refusals in pickcommit.go (the intent not planned in the lane's worktree, a branch whose first commit is not the pick's, a worktree dirty with other changes, no git identity) the page states only the git-identity one.
  evidence: commands/build.md:97-143 — "## Pick the next intent ... The pick starts the run `build <itd-N>` would start for that intent"
  evidence: commands/build.md:145-174 — "### One pick, or a series ... `--max <n>` (above 1) or `--until-empty` begins a **series**"
  evidence: commands/build.md:176-204 — "Refusals, each writing nothing: ... `.abcd/.work.local/run/next.json` that cannot be read as the series' state ... the pick's reason refused by the grounds writer's own gate"
  evidence: commands/build.md:206-207 — "The pick commit is made with the git identity the repository's commits are made with; with none configured the `worktree` stage is refused naming it."
  evidence: internal/core/implement/loop/nextseries.go:383-409 — "cannot be read as the series' state ... does not parse as the series' state ... names no run count a series takes ... names a pick with an outcome the series never writes"
  evidence: internal/core/implement/loop/pickcommit.go:120-167 — "is not planned in the lane's worktree ... is not the pick's record commit ... holds changes the pick did not make"
- ac-10 — MET: NextResult serialises candidates (each with its score), excluded, pick, entry, start, series and next; the empty-set refusal carries excluded in the refusal document; a count refusal and the lock contention render as refusal documents with stage, reason and contention; the run record's pick is in `implement record --json`. The CLI tests decode each and pass on my copy.
  evidence: internal/core/implement/loop/next.go:81-88 — "type NextResult struct { CandidateSet; Pick *intent.Pick `json:"pick"`; Entry string `json:"entry"`; Start ...; Series ...; Next ..."
  evidence: internal/core/implement/loop/next.go:237-242 — "ref.Excluded = set.Excluded"
  evidence: internal/surface/cli/build_next_surface_test.go:21-72 — "TestBuildNextPicksAndCarriesTheReasonInJSON"
  evidence: internal/surface/cli/build_next_surface_test.go:78-100 — "TestBuildNextRefusalCarriesEveryExclusion"
  evidence: internal/surface/cli/build_next_surface_test.go:116-174 — "TestBuildNextMaxCarriesTheSeriesInJSON"
  evidence: internal/surface/cli/build_next_surface_test.go:252-273 — "TestBuildNextBacksOffWhileTheSeriesIsLocked ... ref["contention"] != true"

Gap audit:
- honoured:
  - the candidate set is the build's own pre-start checks over planned/, and an empty set is refused naming each exclusion with nothing written
    evidence: internal/core/implement/loop/next.go:93-165 — "func candidates(repoRoot, session string) (CandidateSet, error)"
  - the reason is computed (candidates, scores, rule, runner-up, falsifier), written as one run-marked pursued entry and committed as the lane branch's first, record-only commit
    evidence: internal/core/implement/loop/pickcommit.go:110-182 — "func pickCommit(c Context, lane *Lane, wt, branch, base string) (string, error)"
  - readiest first, oldest among equals, with the tie named
    evidence: internal/core/intent/pick.go:164-171 — "PickLess is the pick order, the one statement of it"
  - the pick hands over to the same machinery as build < itd-N>, and the receipt verifier does not count the pick commit as the implementer's work
    evidence: internal/core/implement/loop/loop.go:282 — "func Start(repoRoot, key string, o Options) (StartResult, error) { return start(repoRoot, key, o, nil) }"
    evidence: internal/core/implement/loop/receipt.go:465-486 — "pickKept refuses a receipt over a lane branch that no longer carries the pick's record-only commit"
  - intent ready keeps reporting the person's entry as the most recent conjecture
    evidence: internal/core/intent/ready.go:346-364 — "the most recent conjecture the row names is a person's, so the pick never displaces it"
  - intent plan seeds an empty ## Footprint; an absent footprint reads zero and says so
    evidence: internal/core/spec/spec.go:310 — "{heading: FootprintHeading, body: footprintStubBody}"
    evidence: internal/core/intent/pick.go:106-109 — "case !fp.Present:"
  - the run record names each pick's score table and whether its falsifier was met, and the falsified pick's entry is not edited
    evidence: internal/core/implement/loop/record.go:82-98 — "func recordPick(st State) *RecordPick"
    evidence: internal/core/implement/loop/nextseries_test.go:473-482 — "the intent on the lane's branch is the pick commit's, unedited"
  - --max < n> and --until-empty keep picking under the pace rule's window, each further reason written the same way, until the cap or an empty set
    evidence: internal/core/implement/loop/nextseries.go:187-316 — "continueSeries is one move of the series in progress, under its lock: route the last pick, stop at the cap, honour the window clock, then pick again or end with nothing left."
  - the command page states what next does, its two flags and its refusals; --json carries the set, scores, reason and refusals
    evidence: commands/build.md:97-207 — "## Pick the next intent"
    evidence: internal/surface/cli/build_next_surface_test.go:21-273 — "the payload carries the candidate set with each score, the pick with its rule and runner-up, the reason's text"
- diverged:
  - `--until-empty` or `--max <n>` keeps it going under the pace rule: delivered as a host-driven series, one move per `abcd build next` invocation that the host must call again, rather than the run continuing on its own
    evidence: internal/core/implement/loop/nextseries.go:9-12 — "The loop is driven by the host session, so a series is too: each `abcd build next` makes one move and exits."
    evidence: commands/build.md:172-174 — "drive the run it starts to delivery or hand-back, then run `build next` again, and repeat until `series.complete` is true"
  - the pace rule across the series: the window and pause are enforced over the series, but the ceiling and fix rounds are forwarded to each run as the series' typed flags, not applied across the series
    evidence: internal/core/implement/loop/nextseries.go:283-284 — "so.Pace, so.SubAgents, so.FixRounds = st.Flags.Pace, st.Flags.SubAgents, st.Flags.FixRounds"
- missing:
  - the command page states the pick commit's worktree-stage refusals: of the four pickcommit.go raises (not planned in the lane's worktree, a foreign first commit, a dirty worktree, no git identity) only the git-identity one is on the page
    evidence: internal/core/implement/loop/pickcommit.go:120-167 — "is not planned in the lane's worktree ... is not the pick's record commit ... holds changes the pick did not make"
    evidence: commands/build.md:206-207 — "with none configured the `worktree` stage is refused naming it"

Scope-condition dispositions:
- cond-2609212015041824 — survived: a candidate must pass the build's ready check, whose acceptance_criteria row fails on a section with no top-level bullets and whose spec_link row fails without a linked spec (next_test excludes itd-14 with spec_id null at `ready`), and the score reads the Given-When-Then bullets and the spec's footprint, so the score has fields to read on every candidate
  evidence: internal/core/intent/ready.go:185-194 — "no top-level bullets in '## Acceptance Criteria' (itd-1 discipline)"
  evidence: internal/core/intent/ready.go:449 — "c := ReadyCheck{Name: CheckSpecLink}"
  evidence: internal/core/intent/pick.go:88-96 — "mdrecord.SectionLineRangeIn(lines, mask, acHeadingRe) ... givenRe.MatchString(text) && whenRe.MatchString(text) && thenRe.MatchString(text)"
  evidence: internal/core/implement/loop/next_test.go:102-103 — ""itd-14": CheckReady"
- cond-2609212015048560 — survived: the entry is appended through the intent store's grounds writer, the commit is verified as the base's record plus exactly one entry, the hand-back path leaves the record byte for byte as the pick commit wrote it, and the ready gate keeps reporting the person's entry as the most recent conjecture
  evidence: internal/core/implement/loop/pickcommit.go:184-205 — "the record at the lane's base with this run's entry appended, built as intent.RecordGrounds builds it"
  evidence: internal/core/intent/ready.go:346-348 — "it is counted, but the most recent conjecture the row names is a person's, so the pick never displaces it"
  evidence: internal/core/implement/loop/next_test.go:195-202 — "no existing line of the record changes ... exactly one run-marked pursued entry is appended"
  evidence: internal/core/implement/loop/nextseries_test.go:473-482 — "the intent on the lane's branch is the pick commit's, unedited"
- cond-2609212015045488 — narrowed: blocked_by filters and nothing ranks on either edge (PickLess reads score and id age only; builds_on is not read by the pick), but under decision 9's amendments a blocker excludes while its supersession chain is unsettled rather than while it is unshipped: a blocker, or the last record of its chain, that has shipped, sits in disciplines/, or is an accepted ADR releases the edge
  narrowing: holds for blockers whose supersession chain ends at a record that is neither shipped, in disciplines/, nor an accepted ADR; an unshipped blocker settled by one of those does not exclude the intent
  evidence: internal/core/intent/startcheck.go:150-166 — "An intent is settled when it has shipped or sits in disciplines/ ... A chain ending at a decision (adr-N) is settled when that ADR's status is `accepted`"
  evidence: internal/core/intent/startcheck.go:189-194 — "names no unsettled blocker ... a superseded blocker is settled by the record that replaced it"
  evidence: internal/core/intent/pick.go:166-171 — "if a.Score.Total != b.Score.Total { return a.Score.Total > b.Score.Total } return IDOlder(a.ID, b.ID)"
<!-- abcd-review-end receipt=rcp-c99b85cd869d -->

## Grounds

- pursued: the autonomous run's routing table is a person's hand-written file today; we expect the pick to be what replaces it after the run; shown wrong if the next run still starts from a hand-written table
