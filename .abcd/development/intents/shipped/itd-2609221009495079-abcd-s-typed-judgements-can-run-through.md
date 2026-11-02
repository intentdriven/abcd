---
id: itd-2609221009495079
supersedes: [itd-17]
slug: abcd-s-typed-judgements-can-run-through
spec_id: spc-2609221011151661
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212137128014, itd-2609081951381895, itd-2609170822093401]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-17, itd-2609212137116617, itd-82]
related_adrs: [adr-2609221009491186]
---

# abcd's typed judgements can run through a decision adapter, measured in a lab before it decides anything

## Press Release

> **A decision adapter answers abcd's closed-option judgements with a value and a calibrated probability; it runs in shadow as a lab first, and a research note turns it on.**
>
> "Half of what my agents ask a frontier model is a yes-or-no about one record," said a product thinker reading a run's cost. "A decision model answers those in half a second for a fraction of a cent. I do not trust it yet, so it runs beside the host, the lab records every pair, and when the note shows it agrees, I turn that judgement over."

## Why This Matters

The research pass of 2026-09-22 found Jev (TypeSafe AI) to be a typed-decision model, not a text model and not a router: choose one of N, score against a rubric, yes or no, each with a calibrated probability, in under a second at $0.042 per million tokens, 32K context, and unable to abstain. Independent evidence is one forty-case test; the vendor's numbers are vendor-graded. abcd's judgement-shaped steps are many: the remedy read, duplicate matching, unmeetable criteria, the eligibility residual, source classification and selection in the library, memory checks, ledger category and severity, the pre-pass's overlap, the docs lint's present-tense advisory, embark's lesson ranking. Ruled: an adapter behind one interface, the host as default, shadow mode as an abcd lab, a research note as the evidence a ruling turns a judgement type on with.

## Mechanism

We expect calibrated typed answers to agree with the host on closed-option judgements at a rate a lab can measure, because those judgements have a small option set and a state that fits the model's window; shown wrong if the shadow run's agreement is below what the note's threshold demands.

## Scope Conditions

- Holds for judgements whose option set is closed and whose state fits 32K tokens after retrieval; a judgement needing the whole corpus in view is out. <!-- cond: cond-2609221011150901 -->
- Holds where a refusal is not the right answer: the model cannot abstain, so a judgement type whose right answer is often "unknown" is not turned over. <!-- cond: cond-2609221011151505 -->
- Holds under adr-2609221009491186: the adapter's model is on its provider's allowlist. <!-- cond: cond-2609221011150032 -->

## What's In Scope

- **One interface**: `(state, typed question) → (value, probability)` with three question kinds (choose, score, yes/no); the host's own judgement implements it too and stays the default.
- **The judgement types** it may be configured for: the remedy read, duplicate matching, unmeetable criteria, the eligibility residual, source classification and selection over a shortlist, the memory checks, ledger category and severity, the pre-pass's overlap, the docs lint's present-tense advisory, embark's lesson ranking.
- **Shadow mode as a lab**: `abcd lab mint` opens a lab home; the adapter runs beside the host on the configured types, decides nothing, and each pair (host's answer, adapter's answer, probability, the state's hash) is recorded under the lab home in the machine-scoped store, never the repository.
- **The evidence**: `abcd lab harvest` produces agreement rates per type with counts and probability bands; the research note filed from it (dated, under the record's research notes) is the evidence a ruling turns a type on with.
- **Turning on**: per judgement type in configuration; unreachable falls back to the host with a receipt; the run record names which route judged.
- **The first candidate**: `typesafe/jev-1.13` through the API adapter, on OpenRouter's allowlist; a small text model scoring options by log-probability is the second, behind the same interface.

## What's Out of Scope

- Any generative step.
- A learned per-request router for lane models (itd-17, superseded).
- What to build next (a computed score, itd-2609211116005482; a rubric score may become one component later).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. Shadow mode only, as an abcd lab; a research note is the evidence; the host stays the default (ruled 2026-09-22).
2. The interface is the host's too, so the two are compared like for like.
3. Turned on per judgement type, never all at once.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [models, prompts and measurement SOTA note](../../research/notes/2026-10-23-models-prompts-measurement-sota.md);
what Jev is and why shadow first are in the
[Jev and model-routing pass](../../research/notes/2026-09-22-jev-and-model-routing-sota.md).

- **An agreement number is a property of the protocol.** Scale, retained
  cases, abstention handling and pooling determine what the figure estimates;
  protocol choice alone moved one benchmark's accuracy from 0.551 to 0.899 and
  kappa across zero with no verdict changed; for binary verdicts the common
  correlations are one statistic, and Cohen's kappa differs from them only by
  a marginal-mismatch factor, so reporting several adds nothing: report kappa
  with the protocol and the marginals (Rao and Callison-Burch, arXiv
  2606.00093, 2026).
- **Raw agreement overstates.** Across 21 judges and about 541,000
  judgements, exact match exceeded kappa by 33–41 points on MT-Bench, and two
  production judges with test–retest reliability above 0.95 still showed
  position bias above 0.10 (Norman, Rivera and Hughes, arXiv 2606.19544,
  2026).
- **Jev's calibration is asserted, not shown**: an unpublished training
  method, no reliability diagram, no ECE. The nearest outside test, option
  scoring on open Qwen2.5 models, found base models near-calibrated, instruct
  models overconfident, and temperature scaling on 600 labelled cases cutting
  ECE from 0.10 to 0.03 without changing accuracy; the speed advantage narrows
  as the state grows (Dibia, 2026).
- **A vendor-run comparison** (900 cases, September 2026) found Jev calibrated
  on closed classification (error 0.032 and 0.096), poorly calibrated on
  rubric scoring (0.284; lowest human correlation of six models) and seven
  points behind frontier models on 77-way routing — vendor evidence, and the
  page's method for the 0.284 figure is an unfinished TODO with a placeholder
  link to its test materials, so that figure is a hint only; the direction
  still says `choose` and yes/no types go first and `score` types last.
- **ECE is paired with Brier**, since ECE alone rewards reporting the base
  rate (2026 preprints, snippets only).
- **Contrary to nothing ruled**: decisions 1–3 stand; this only shapes what
  the harvest reports.

**Path 2**, as planned: the one interface is the seam, Jev the first adapter
behind it, a log-probability scorer the second. The harvest reports, per type,
kappa beside percentage agreement with the protocol stated, ECE over ten bins
and Brier, counts and probability bands, and for `choose` agreement under a
permuted option order; the shadow pairs double as the labelled set a
temperature scale for the second candidate is fitted on, held out from the
pairs it is scored on. No dependency. This pass came after the build was planned and had no fit-challenge before
the intent shipped, so it stands as notes for a later revision.

Questions this pass raised, kept as notes for a later revision of the shipped work:

- Whether the `score` kind is held back from the first
  shadow configuration altogether, given that rubric scoring is where the
  first candidate measured weakest, or run in shadow precisely to measure
  that.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a judgement type configured for shadow and a lab home minted, **when** the judgement runs, **then** the host decides, the adapter answers beside it, and the pair with its probability and the state's hash is written under the lab home; nothing is written into the repository.
- **Given** a lab home with recorded pairs, **when** `abcd lab harvest` runs, **then** it reports agreement per judgement type with counts and probability bands, and the note filed from it names the lab home and the count.
- **Given** a judgement type turned on in configuration, **when** it runs, **then** the adapter decides, the run record names the route, and an unreachable adapter falls back to the host with a receipt saying so.
- **Given** a configured model not on its provider's allowlist, **when** the configuration is read, **then** it is refused before any call (adr-2609221009491186).
- **Given** a judgement whose state exceeds the model's window, **when** it runs in shadow, **then** the pair is recorded as skipped with the size, and the host decides alone.
- **Given** the host judgement and the adapter, **when** either is called, **then** both go through the one interface, so a type can be switched without a caller changing.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-94dc11cd60c0 -->
Fidelity review — receipt rcp-94dc11cd60c0 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e3e6f09f32b7e8e7e88fe61b1eba98d010427ab25f328e370f57c3c0ff391e4f
Input attestations: diff:3034da71fd67a271a7914cd9d51374e125e7b4e4..165f8a3e5809ce9270c64b53017a7eda84ed32dc (lane-1, spec step 2, with round-1, round-2 and round-3 fixes)@sha256:132ed046dd0bb0bef686423dc4e1594a05b816bce47d2fdf2874050c0da8b8b0; diff:e533cf361743c6f23593b11d3fb5cfbd9c401db8..aa2f716515adaf4f8dec8adb8d408eac8c6601f6 (spec step 1, merge of #1245, an ancestor of the lane head)@sha256:637a3a1e2ce4a42d7acb61f4c956b7fdfd7e0b9198674b7527748f0982fad49b;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: In shadow, Router.Decide has the host decide, then asks the adapter and appends a pair to pairs.jsonl in the newest lab home under the machine-scoped store. The pair holds both answers, the probability and state_sha256, and a test checks that git status in the repository is unchanged. The second candidate is reached through the same router path via Load's candidate factory, so its shadow pairs go to the same place. The round-3 fixes do not touch this path. Two concerns from earlier rounds stand, unchanged on purpose. First, only two of the twelve registry types have a caller (eligibility-residual and embark-lesson-ranking); asked_test pins those two, now also failing on a dot import or a type built by conversion, and Load says a route on any other type runs nothing. Second, embark's shadow pass stops after the first failed pair or when its 2-minute budget runs out, and the lessons it does not reach keep no pair, though a note names them. The new drain test drives the second candidate end to end in on mode only (a bare route string defaults to on); no test drives a shadow route to the second candidate end to end, though the shadow code path is shared.
  evidence: internal/core/decide/judge/router.go:217 — "pair := lab.Pair{Type: string(q.Type), Kind: string(q.Kind), StateSHA256: rc.StateSHA256, StateTokens: size,"
  evidence: internal/core/lab/pairs.go:75 — "func RecordPair(repoRoot string, p Pair) (id, where string, err error) {"
  evidence: internal/core/decide/judge/judge_test.go:106 — "func TestShadowWritesThePairUnderTheLabHomeAndNothingInTheRepository(t *testing.T) {"
  evidence: internal/core/decide/judge/router.go:147 — "func candidate(cfg *oracle.APIConfig, creds credential.Source, t oracle.Target) Judge {"
  evidence: internal/core/implement/loop/drain_decide.go:124 — "_, rc, err := r.Decide(ctx, drainState(iss), drainQuestion(), decision.Answered(drainValue(j)))"
  evidence: internal/core/lifeboat/retrospectives.go:490 — "_, rc, err := r.Decide(cctx, lessonState(v.framing, l), q, judge.Answered(strconv.Itoa(scores[i])))"
  evidence: internal/core/decide/judge/judge.go:71 — "var asked = []Type{EligibilityResidual, EmbarkLessonRanking}"
  evidence: internal/core/decide/judge/asked_test.go:23 — "func TestAskedIsTheTypesTheCallersName(t *testing.T) {"
  evidence: internal/core/lifeboat/retrospectives.go:486 — "noteLessonsNotAsked(jd, v.lessons[i:], fmt.Sprintf("the shadow pass's time (%s) ran out", lessonShadowBudget))"
- ac-2 — MET_WITH_CONCERNS: lab harvest reports, for each type, the pairs, compared, agreed, skipped and rate, plus probability bands, in harvest.md, in --json and in the plain-text output, with a note line naming the lab home and the pair count. The round-3 fixes do not touch the harvest. The standing concern holds and the fix report rejects it again: abcd does not file the research note, it only prints the line a note should carry, so nothing checks that a filed note names the lab home and the count. The harvest counts by type, not by candidate, so commands/lab.md tells the person to compare the two candidates in separate labs.
  evidence: internal/core/lab/harvest.go:142 — "if res.Agreement, err = l.agreement(); err != nil {"
  evidence: internal/core/lab/pairs.go:219 — "Shadow agreement from the lab home %s: %d pairs (%d compared) over %d judgement type(s), harvested %s."
  evidence: internal/core/lab/pairs.go:311 — "For the research note that turns a type on:"
  evidence: internal/core/lab/pairs_test.go:74 — "func TestHarvestReportsAgreementPerTypeWithBands(t *testing.T) {"
  evidence: internal/surface/cli/lab.go:379 — "fmt.Fprintf(w, " by the adapter's probability (agreed/compared): %s\n", strings.Join(parts, ", "))"
  evidence: commands/lab.md:109 — "two candidates in separate labs."
- ac-3 — MET: When a type is turned on, Router.Decide asks the adapter and the receipt names the route and the target; if the adapter fails or the state is over the window, the host decides and the receipt carries a Fallback naming why. Round 3 adds a further failure the second candidate reports instead of guessing: a first token that is none of the labels is an error, whatever its alternatives name. The new drain test shows this on the drain's real question through the second candidate: a word answer leaves no judgement taken and a fallback receipt saying the host's request stands, and a letter answer is taken with its probability in the reason. Tests cover the unreachable case at the router.
  evidence: internal/core/decide/judge/router.go:203 — "rc.Fallback = fmt.Sprintf("%s did not answer (%s), so the host decided", rc.Target, detail(err))"
  evidence: internal/core/decide/judge/judge_test.go:231 — "func TestOnAnUnreachableAdapterFallsBackToTheHostWithAReceipt(t *testing.T) {"
  evidence: internal/core/implement/loop/drain_decide.go:157 — "return nil, &runner.FallbackReceipt{At: now, Role: RoleDrainJudge, Asked: asked, Reason: runner.ReasonFailed,"
  evidence: internal/core/decide/judge/logprob.go:200 — "if labelOf(ls, answered) < 0 {"
  evidence: internal/core/implement/loop/drain_decide_test.go:197 — "func TestTheDrainsQuestionThroughTheLogprobCandidate(t *testing.T) {"
  evidence: internal/core/decide/judge/logprob_test.go:157 — "func TestTheLogprobCandidateWithNoOptionTokenIsAFailure(t *testing.T) {"
  evidence: internal/core/decide/judge/logprob_test.go:174 — "func TestTheLogprobCandidateWithNoLogprobsIsAFailure(t *testing.T) {"
- ac-4 — MET: Every judgement route, shadow ones included, goes through Admit against the provider's models list and the denylist when the configuration is read. logprob_models may name only a model the block already lists, and Load refuses any other; ScoresByLogprob returns false for any target that is not admitted. Tests show refused routes never reach the fake provider. The round-3 fixes only document this (commands/ahoy.md now names logprob_models and its refusal) and do not change it.
  evidence: internal/core/oracle/config.go:585 — "if err := c.Admit(provider, model); err != nil {"
  evidence: internal/core/oracle/config.go:403 — "func readLogprobModels(models, marked []string) ([]string, error) {"
  evidence: internal/core/oracle/config.go:424 — "func (c *APIConfig) ScoresByLogprob(t Target) bool {"
  evidence: internal/core/decide/judge/judge_test.go:395 — "func TestLoadRefusesAnUnlistedModelBeforeAnyCall(t *testing.T) {"
  evidence: internal/core/decide/judge/logprob_test.go:191 — "func TestTheLogprobCandidateIsRefusedByTheAllowlistBeforeAnyCall(t *testing.T) {"
- ac-5 — MET: In shadow, a state whose estimated tokens exceed the window (32K by default) is never sent: the pair is recorded as skipped=window with state_tokens, and the host's answer stands alone. The window check runs before either candidate is asked, and a test shows the second candidate is not called for a state over the window. Unchanged by round 3.
  evidence: internal/core/decide/judge/router.go:40 — "const DefaultWindow = 32 << 10"
  evidence: internal/core/lab/pairs.go:41 — "SkippedWindow = "window""
  evidence: internal/core/decide/judge/judge_test.go:140 — "func TestShadowSkipsAStateLargerThanTheWindow(t *testing.T) {"
  evidence: internal/core/decide/judge/logprob_test.go:236 — "func TestTheLogprobCandidateIsNotSentAStateOverTheWindow(t *testing.T) {"
- ac-6 — MET: Host, Adapter and LogprobAdapter each implement the single Judge interface. Load's candidate factory picks between the two adapters from machine configuration alone, so no caller changes when a route moves between host, shadow, on or between candidates. The new drain test is direct evidence: the drain's unchanged caller, decideDrainJudgement with the real drainQuestion, reaches the second candidate by configuration alone. Another test drives one caller through all three modes, and another shows an unmarked route still reaches the first candidate.
  evidence: internal/core/decide/judge/judge.go:182 — "type Judge interface {"
  evidence: internal/core/decide/judge/logprob.go:132 — "func (a LogprobAdapter) Judge(ctx context.Context, s State, q Question) (Answer, error) {"
  evidence: internal/core/decide/judge/router.go:149 — "return LogprobAdapter{Calls: cfg, Creds: creds, Target: t}"
  evidence: internal/core/decide/judge/judge_test.go:259 — "func TestOneInterfaceSwitchesATypeWithoutTheCallerChanging(t *testing.T) {"
  evidence: internal/core/implement/loop/drain_decide_test.go:197 — "func TestTheDrainsQuestionThroughTheLogprobCandidate(t *testing.T) {"
  evidence: internal/core/decide/judge/logprob_test.go:274 — "func TestAnUnmarkedRouteIsTheFirstCandidate(t *testing.T) {"

Gap audit:
- honoured:
  - One interface, (state, typed question) to (value, probability), with three question kinds. The host implements it at probability 1 and stays the default.
    evidence: internal/core/decide/judge/judge.go:182 — "type Judge interface {"
  - The second candidate, a small text model scoring options by log-probability, behind the same interface. It is opt-in per model through logprob_models, asks for a one-token answer with top_logprobs, and normalises the label probabilities over the options. An answer with no option token, no log-probabilities, or (new in round 3) a first token that is none of the labels is a failure, never a guess. Tested against fakes. Residual, named by the fix report: the candidate sets no temperature, so on a sampling provider the answered label can differ from the most probable label it returns; both are labels, so it is never an off-option guess.
    evidence: internal/core/decide/judge/logprob.go:181 — "func fromLogprobs(q Question, ls []string, answered string, top []openaiapi.TokenLogprob) (Answer, error) {"
    evidence: internal/adapter/openaiapi/logprobs.go:57 — "func WithTopLogprobs(n int, sink LogprobSink) Option {"
    evidence: internal/core/decide/judge/logprob_test.go:106 — "func TestTheLogprobCandidatePicksTheMostProbableOptionNormalised(t *testing.T) {"
    evidence: internal/core/decide/judge/logprob_test.go:292 — "func TestTheLogprobCandidateRefusesAnOffLabelAnswerWhateverTheAlternatives(t *testing.T) {"
  - Round 3's audit finding (two answer formats in the second candidate's prompt for the drain's question) is answered. The drain's Ask still names the answers in words, which the host and the first candidate need. The candidate's prompt now says the label wins where the question's words name an answer, shows each option as letter and word, and ends with a reply line naming the letters. A word answer is refused and falls back to the host instead of being scored. A new test runs the drain's real question through the candidate against a fake endpoint and checks the prompt, the request's one-token and logprob settings, a letter answer taken and a word answer falling back.
    evidence: internal/core/decide/judge/logprob.go:94 — "Where the question names an"
    evidence: internal/core/decide/judge/logprob.go:112 — "func replyLine(q Question, ls []string) string {"
    evidence: internal/core/implement/loop/drain_decide.go:82 — "" when it changes a trust boundary (or both, or you cannot tell), and no only when it changes neither.","
    evidence: internal/core/implement/loop/drain_decide_test.go:197 — "func TestTheDrainsQuestionThroughTheLogprobCandidate(t *testing.T) {"
  - Shadow mode as a lab. Pairs are recorded under the newest lab home in the machine-scoped store, never in the repository.
    evidence: internal/core/decide/judge/judge_test.go:106 — "func TestShadowWritesThePairUnderTheLabHomeAndNothingInTheRepository(t *testing.T) {"
  - The harvest gives agreement per type with counts and probability bands, in harvest.md, in --json and in the plain-text output.
    evidence: internal/surface/cli/lab.go:379 — "by the adapter's probability (agreed/compared)"
  - A type is turned on per judgement type, from configuration only. The run record names the route, and a fallback leaves a receipt.
    evidence: internal/core/decide/judge/router.go:203 — "so the host decided"
  - The allowlist refusal runs when the configuration is read, before any call, for both candidates.
    evidence: internal/core/decide/judge/logprob_test.go:191 — "func TestTheLogprobCandidateIsRefusedByTheAllowlistBeforeAnyCall(t *testing.T) {"
  - Embark's lesson ranking is a judgement the host answers, with one 0-4 score question per lesson, routed through the adapter in shadow or turned on.
    evidence: internal/core/lifeboat/retrospectives.go:352 — "return judge.Question{Type: judge.EmbarkLessonRanking, Kind: judge.KindScore, Ask: lessonAsk,"
- diverged:
  - The scope lists twelve judgement types the adapter may be configured for, but only two have a caller: eligibility-residual and embark-lesson-ranking. A route on any of the other ten is kept but runs nothing. Load reports it in a diagnostic, and asked_test pins the two.
    evidence: internal/core/decide/judge/judge.go:71 — "var asked = []Type{EligibilityResidual, EmbarkLessonRanking}"
    evidence: internal/core/decide/judge/router.go:131 — "routes a type no step asks through the decision adapter yet;"
  - abcd does not file the research note from the harvest. It prints the line the note should carry, naming the lab home and the count, and leaves filing to a person with nothing to check it.
    evidence: internal/core/lab/pairs.go:311 — "For the research note that turns a type on:"
  - Both candidates exist in code and are tested only against fake endpoints. The live check of each is owed to a person under the paid-services ruling, recorded as a remedy on an open minor issue.
    evidence: .abcd/work/issues/open/iss-2609291925398978-the-live-check-of-the-decision-adapter.md:15 — "A person runs one typed judgement live through each of the decision adapter's two candidates"
    evidence: .abcd/development/brief/05-internals/02-adapters.md:293 — "The live paid check against either candidate is owed to a person"
  - The harvest counts agreement per type, not per candidate. Comparing the two candidates needs separate labs, and the docs say so.
    evidence: commands/lab.md:109 — "two candidates in separate labs."
- missing: (none)

Scope-condition dispositions:
- cond-2609221011150901 — survived: The type registry is closed, and every question is checked for its kind and bounds. The 32K window is enforced before either candidate is asked. The second candidate also refuses a choose question with more than 26 options and a score outside 0-9, so it only answers option sets small enough to label with one token each. Unchanged by round 3.
  evidence: internal/core/decide/judge/router.go:40 — "const DefaultWindow = 32 << 10"
  evidence: internal/core/decide/judge/judge.go:118 — "if !known(string(q.Type)) {"
  evidence: internal/core/decide/judge/logprob.go:66 — "if len(q.Options) > len(labelLetters) {"
  evidence: internal/core/lifeboat/retrospectives.go:274 — "const maxLessonFramingBytes = 16 << 10"
- cond-2609221011151505 — untested: Neither candidate can abstain: the first treats an off-choice answer as no answer, and the second now refuses any first token that is not a label and otherwise picks the most probable label. But the delivery turns no type over, and nothing checks before turning a type on whether its right answer is often 'unknown'. The condition is neither exercised nor contradicted.
- cond-2609221011150032 — survived: Every judgement route is admitted against its provider's models list and the denylist when the configuration is read. logprob_models can mark only a model already on that list, and ScoresByLogprob requires an admitted target. Both candidates therefore run only on a model the provider lists.
  evidence: internal/core/oracle/config.go:585 — "if err := c.Admit(provider, model); err != nil {"
  evidence: internal/core/oracle/config.go:403 — "func readLogprobModels(models, marked []string) ([]string, error) {"
<!-- abcd-review-end receipt=rcp-94dc11cd60c0 -->

## Grounds

- pursued: cost per judgement is a hundredfold lower and the evidence is vendor-graded; we expect the shadow lab to show agreement the note can put a number on; shown wrong if the agreement is below the note's threshold
