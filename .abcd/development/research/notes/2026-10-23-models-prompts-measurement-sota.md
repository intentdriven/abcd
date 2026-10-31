# Model choice, prompts and measurement: state of the art

Dated 2026-10-23; recency is assessed against that date. A state-of-the-art
pass over the cluster of open intents that decide which model an agent gets,
how its prompt is versioned and audited, and how the work is measured:
[itd-2609170822093401](../../intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md)
(abcd proposes the model each agent deserves),
[itd-2609221009495079](../../intents/shipped/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md)
(typed judgements through a decision adapter, measured in a lab),
[itd-14](../../intents/drafts/itd-14-prompt-registry-versioning.md) (prompts are
versioned like code),
[itd-15](../../intents/drafts/itd-15-self-dogfooded-sota-audit.md) (abcd audits
its own prompts on every self-disembark),
[itd-64](../../intents/drafts/itd-64-benchmark-driven-config-optimization.md)
(benchmark-driven configuration optimisation),
[itd-98](../../intents/drafts/itd-98-solo-vs-duo-is-measured-not-debated.md)
(solo versus duo is measured) and
[itd-2609292107351737](../../intents/drafts/itd-2609292107351737-every-development-session-records-what.md)
(a session's token cost against its spec's size class). The researcher is the
proposer here; the independent fit-challenge that
[prefer-sota](../../principles/prefer-sota.md) requires is still owed at plan
time.

Four earlier notes already cover much of this ground and are reused rather than
redone: the [model-matrix SOTA note](2026-10-16-model-matrix-sota.md) (repeats
and power, served-versus-asked, the nondeterminism floor, review panels), the
[Jev and model-routing pass](2026-09-22-jev-and-model-routing-sota.md) (routers,
cascades, what Jev is), the [token-metering gap](2026-08-30-development-token-metering-gap.md)
with its siblings on [size classes](2026-08-30-spec-size-classification-gap.md)
and [review-round cost](2026-08-30-review-round-cost-calibration.md), and the
[judge-calibration survey](2026-07-12-judge-calibration-sota.md). The
[prompting baseline](../prompting/01-general-best-practices.md) holds the 2026
versioning consensus this note checks against practice. Only the gap was
researched. Every source linked below was opened in this pass; a source that
could not be opened is named in words under "Not verified" and carries no link.

Tiers: **[EVIDENCE]** an experiment, eval or large dataset, named;
**[CONSENSUS]** several independent practitioners or primary documents agree;
**[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** a single report or a vendor's own figures.

## itd-2609170822093401 — the model each agent deserves

The routing question itself was settled by the 2026-09-22 pass (a tier table
per role, no learned router, escalation inside a lane as a rule) and the
matrix note settled how a route is measured. What was left to check is the
Mechanism's first claim, that the harness exposes model selection to a
sub-agent request, and whether the escalation rule has evidence behind it.

- **The host resolves a sub-agent's model in a fixed order**: the per-invocation
  `model` parameter first, then the agent definition's `model` frontmatter
  (`sonnet`, `opus`, `haiku`, `fable`, a full id, or `inherit`), then a
  `CLAUDE_CODE_SUBAGENT_MODEL` environment default, then the main
  conversation's model. A second variable, `CLAUDE_CODE_SUBAGENT_MODEL_FORCE=1`,
  makes the environment value override both the frontmatter and the
  per-invocation parameter; a mod's `agent.spawn` hook can also replace the
  per-invocation value. Before v2.1.251 the environment variable came first.
  `effort` is settable per definition and per invocation on the same terms,
  with `CLAUDE_CODE_EFFORT_LEVEL` in the environment beating both (a second
  silent override beside the force variable), and `/tasks` names the model a
  running sub-agent is on. [CONSENSUS]
  (primary documentation) — [Claude Code sub-agents](https://code.claude.com/docs/en/subagents).
  Consequence: the Mechanism's claim holds on the harness leg, but a host-level
  force switch can silently override a row, so the receipt's "model reported"
  beside "tier asked" is the only check, as the third scope condition already
  says.
- **Two-model cascades pay; deeper ones rarely do.** Bouchard (arXiv
  2605.06350, v2 October 2026) derives optimality conditions for cascade depth
  and thresholds: two-model cascades improved mean accuracy over single-model
  selection by 2.1 to 8.2 points, allowing more than two models did not
  improve mean accuracy in 118 of 135 comparisons across scorers, datasets
  and depth caps and added at most 0.43 points, and simulated-confidence
  experiments show depth beyond two helps (on four of five benchmarks) only
  when the deferral signal reflects whether the current model is wrong rather
  than difficulty shared across models. [EVIDENCE] —
  [Is Escalation Worth It?](https://arxiv.org/abs/2605.06350). The paper
  studies single-query cascades, not fix loops with gate feedback. Whether a
  failed fix round is a current-model-error signal or a shared-difficulty
  signal decides whether a second step pays; it is a plausible instance of the
  first, not a tested one. Decision 1's one-tier-per-failed-round rule is
  consistent with this, and the matrix run is where a second step is tested.
- **Router comparison is now benchmarked, with accuracy against cost.**
  RouterArena (Lu et al., arXiv 2510.00202) is an open leaderboard over routers
  with difficulty levels per domain and an automated update framework; a
  search-result snapshot placed commercial routers mid-table. [EVIDENCE] for
  the platform's existence; the ranking figures were read from search results
  only — [RouterArena](https://arxiv.org/abs/2510.00202). Nothing here
  reopens decision 3 (no learned per-request router).
- **Served versus asked** and the nondeterminism floor: unchanged from the
  matrix note; a routing receipt records the served model, provider and
  fallback attempts where the source exposes them.

**Options weighed.** (a) Adopt the host's own frontmatter `model` field as
the routing table — rejected by the record already (the table must survive a
host change and name connections the host does not know). (b) The ruled
native table, resolved onto the host's per-invocation `model` and `effort`
parameters on the harness leg and onto a connection on the provider leg.
(c) A learned router — superseded.

**Recommendation: Path 2**, as planned. The seam is the harness request block:
a row's tier and effort map one-to-one onto the host's per-invocation `model`
and `effort` parameters, so a host that changes its vocabulary changes one
mapping. Two additions from this pass: the receipt compares the model reported
against the tier asked and flags a host force override by name; and the
escalation rule stays at one step up, with the matrix run as the place to test
whether a second step ever pays. No dependency.

**Local fit risks.** Host-delegated by default (the host's own force switch is
a legitimate operator choice that abcd must report, never fight); one
canonical primitive (the per-invocation override is the one route surface, and
the matrix intent already extends it rather than adding a second).

## itd-2609221009495079 — typed judgements through a decision adapter

The decision pass of 2026-09-22 established what Jev is and that it runs in
shadow first. The gap is how the shadow lab's harvest should report agreement
and calibration so the note that turns a judgement type on is evidence rather
than a number.

- **The agreement number is a property of the protocol, not of the judge.**
  Rao and Callison-Burch (arXiv 2606.00093, May–July 2026) show that the
  judgement scale, which cases are retained, how abstentions and invalid
  outputs are handled, and how verdicts are pooled determine what an agreement
  figure estimates; on one rubric benchmark protocol choice alone moved
  accuracy from 0.551 to 0.899 and moved kappa across zero with no verdict
  changed. For binary verdicts Pearson, Spearman, Kendall, phi and MCC are one
  statistic, and Cohen's kappa "differs from them only through a
  marginal-mismatch factor in (0,1]", so reporting several adds nothing: report
  kappa with the protocol and the marginals. They distil a reporting
  checklist. [EVIDENCE] —
  [Agreement Metrics for LLM-as-Judge Evaluation](https://arxiv.org/abs/2606.00093).
- **Raw agreement overstates; kappa deflation is universal.** Norman, Rivera
  and Hughes (arXiv 2606.19544, June 2026) evaluate 21 judges from nine
  providers over about 541,000 judgements: the gap between exact match and
  Cohen's kappa is 33–41 points on MT-Bench, two production judges combine
  test–retest reliability above 0.95 with position bias above 0.10, and judge
  rankings move by up to 14 places across benchmarks. They propose a minimum
  validation protocol covering agreement, consistency and a bias audit.
  [EVIDENCE] — [Reliability without Validity](https://arxiv.org/abs/2606.19544).
- **Jev's calibration is asserted, not shown.** The vendor describes training
  by an unpublished "RLCD" method and publishes no reliability diagram or
  ECE; the closest outside test is an option-scoring experiment on open Qwen2.5
  models, where base models were near-calibrated (ECE 0.02 for the 7B at ten
  options), instruct models were overconfident (95 % confidence, 64 % right at
  77 options), and temperature scaling fitted on 600 labelled messages cut the
  7B instruct ECE from 0.10 to 0.03 without changing accuracy; a one-letter
  readout stayed within about two points of full-option scoring up to 25
  options; the speed advantage narrows as the state grows (3.5× at 256 tokens,
  1.4× at 4,096). [EVIDENCE] for the open-model experiment, [ANECDOTE/MARKETING]
  for anything about Jev itself —
  [How Jev works: calibrated decisions](https://newsletter.victordibia.com/p/how-jev-works-calibrated-decision).
- **A vendor-run comparison** (an API aggregator that sells access to every
  model tested; 3 × 300 cases, September 2026): Jev's confidence tracked
  accuracy on two classification tasks (calibration error 0.032 and 0.096;
  on routing, right 93.9 % of the time at confidence ≥ 0.9, 12 of 197 such
  answers wrong), was poorly calibrated on answer rating (0.284, Spearman
  0.394 with human ratings, lowest of six models), and trailed the frontier
  models by about seven points on 77-way routing. The page's method for the
  0.284 figure is an unfinished TODO and its test-materials link a
  placeholder, so that figure is a hint only. [ANECDOTE/MARKETING]
  — [What is Jev? Tested against LLMs](https://aimlapi.com/blog/what-is-jev).
  The direction (good on closed classification, weak on scoring against a
  rubric) matches the second scope condition and should narrow which
  judgement types go first.
- **Calibration metrics.** ECE alone can look good for a model that always
  reports the base rate, so it is paired with Brier or a failure-prediction
  measure; ECE is conventionally ten equal-width bins. [CONSENSUS] (read from
  2026 preprints in search results; none opened in full, so no link).
- **Sample size for kappa.** Tables for two raters and two categories exist
  (Cantor 1996; Flack's width-bounded method; Sim and Wright 2005) but none
  was opened in this pass; the matrix note's power rule (size from a pilot)
  applies.

**Options weighed.** (a) Report raw agreement per type — refuted above.
(b) Report kappa with percentage agreement, the protocol stated, ECE with
bins and Brier, per judgement type, with counts and probability bands, plus a
position/order permutation for `choose`. (c) A full bias audit per type —
right for a published claim, more than a turn-on ruling needs.

**Recommendation: Path 2**, as planned (the one interface is the seam; the
Jev adapter behind it is the first candidate, the log-probability scorer the
second). The harvest reports, per judgement type: Cohen's kappa beside plain
percentage agreement; the protocol (option set, cases retained, pairs skipped
as oversize, how the host's own abstentions were handled); ECE over ten bins
and the Brier score; and, for `choose`, agreement under a permuted option
order. For the second candidate the shadow pairs double as the labelled set a
temperature scale is fitted on, held out from the pairs it is scored on. The
research note that turns a type on states the protocol with the number, as
the checklist asks. No dependency.

**Local fit risks.** Evaluator outside the loop (the host's own answer is the
label; where the host is also the judge of the harvest, say so);
prefer-the-experiment-to-the-inference (the vendor comparison is a hint, not
the lab).

## itd-14 — prompts are versioned like code

A local fact first: the substance of the first two acceptance criteria has
shipped in a different shape. The `agent_contract` lint family
(`internal/core/lint/agentcontract.go`) requires a semver `prompt_version` on
every prompt, refuses a prompt changed in a diff without a bump at preflight
and in CI rather than in the pre-commit hook the criterion names, and demands
a changelog entry per bump in one log kept outside `agents/` because a harness
loads that tree whole (iss-110), not the per-agent `## Changelog` section the
criterion names. Criterion 5 has no `Last SOTA Audit` footers left to convert
(none exist in the tree). What this intent still owes is the golden diff on a
bump (3), the revert verb (4), and the `audit:` entry shape itd-15 writes (6).

- **Versions are immutable, deployment is a pointer, rollback moves the
  pointer.** Langfuse assigns each saved prompt a version id; labels
  (`production`, `staging`, `latest`) are movable pointers; a request for a
  missing label gets a 404 and "never silently falls back"; rollback is
  "setting the `production` label to that previous version". Experiments
  compare versions on a dataset before a label moves. [CONSENSUS] (primary
  documentation of one widely used tool) —
  [Langfuse prompt version control](https://langfuse.com/docs/prompt-management/features/prompt-version-control).
- **Prompt regression runs on the pull request and posts a diff.** promptfoo's
  GitHub Action runs "on every pull request that modifies a prompt", evaluates
  edited prompts against their previous versions, posts the result link as a
  PR comment, and caches model calls; its page documents no pass/fail
  threshold or gating parameter. [CONSENSUS] —
  [promptfoo GitHub Action](https://www.promptfoo.dev/docs/integrations/github-action/).
- **A prompt asset carries its model and parameters.** Prompty's frontmatter
  holds `model` (`id`, `provider`, a connection, `options` such as
  temperature), typed `inputs`, `tools` and the template format, and each run
  writes a trace file. [CONSENSUS] — [microsoft/prompty](https://github.com/microsoft/prompty).
- **An optimised prompt is a build artefact.** DSPy's GEPA saves the optimised
  program as JSON holding the improved instructions per module; it needs a
  trainset, a validation set, a metric that can return text feedback, and a
  reflection model; `auto="light"` evaluates about six candidates. [CONSENSUS]
  — [DSPy GEPA guide](https://dspy.ai/current/getting-started/gepa-optimization/).
- **"Only formatting" is a behaviour change.** Sclar et al. (ICLR 2024)
  measure spreads of up to 76 accuracy points from formatting alone on
  LLaMA-2-13B, persisting with scale, more examples and instruction tuning,
  and recommend reporting a range across plausible formats. [EVIDENCE] —
  [Quantifying sensitivity to spurious features in prompt design](https://arxiv.org/abs/2310.11324).
  Consequence: the bump grammar's PATCH ("edit") is not safe to exempt from
  the golden run.
- **The same prompt changes behaviour when the model behind it does.** Chen,
  Zaharia and Zou (2023): GPT-4's prime-versus-composite accuracy fell from
  84 % to 51 % between March and June on the same questions; they call for
  continuous monitoring. [EVIDENCE] —
  [How is ChatGPT's behavior changing over time?](https://arxiv.org/abs/2307.09009).
  Consequence: a version record names the model the golden run used.
- **The repository's own baseline** already says this: capture body, model id,
  parameters, rationale and eval result per version; per-agent in place rather
  than a central library — [prompting baseline §5 and §Versioning](../prompting/01-general-best-practices.md).
- **Contrarian.** For eighteen markdown prompts in one repository, a registry
  is indirection: git is the store, the lint is the gate, and Langfuse's own
  model reduces to "a pointer and a diff". The value left in this intent is
  the golden diff and the revert that writes its own entry, not a registry.

**Options weighed.** (a) Adopt a prompt-management service — a hosted
dependency against local-first and host-delegated. (b) A pre-commit hook that
runs golden fixtures — fixtures need a model, so a deterministic hook cannot
run them; the hook could only refuse. (c) Keep the lint as the gate (shipped),
run the golden diff host-delegated in the review that a prompt change triggers,
and record the diff as a receipt the changelog entry cites.

**Recommendation: Path 2.** Native, shaped to Langfuse's version-and-pointer
model and promptfoo's PR-time before/after: the version lives where it already
does (frontmatter); a changelog entry gains the model id and the fixture-set
hash of the golden run that accompanied the bump, and an `audit:` event type;
`abcd prompts revert <agent> --to <version>` rewrites the file from git and
writes the entry (a pointer move, never a silent fallback); the golden diff is
a host-run review, surfaced before the commit completes only where the host
offers a hook that can wait, and recorded in the local tier otherwise. The
seam: the changelog entry and the golden-diff receipt are shaped so one
promptfoo eval result or one Langfuse version record maps onto them field by
field. No dependency.

**Local fit risks.** Host-delegated by default (a golden run is model work; a
commit hook cannot do it); loud staging (a PATCH that skips the golden run must
say so); one canonical primitive (the changelog outside `agents/` is already
the one log — do not add a per-agent `CHANGELOG.md` back inside the tree the
acceptance criteria still name).

## itd-15 — abcd audits its own prompts on every self-disembark

Mostly internal plumbing (self-detection, recursion depth, where findings
land). The one outside question is whether an automated "audit against a
research baseline" has evidence behind it.

- **Reflective rewriting works when it has a metric and traces to read.**
  GEPA (Agrawal et al., ICLR 2026 oral; arXiv 2507.19457 v2) beats GRPO by
  6 % on average and up to 20 % with up to 35× fewer rollouts, and MIPROv2 by
  over 10 %, by reading execution traces and proposing textual prompt edits
  along a Pareto frontier of candidates. [EVIDENCE] —
  [GEPA](https://arxiv.org/abs/2507.19457). The precondition is the point: it
  optimises against a metric over a trainset; an audit with neither is a
  model's opinion of prose against prose.
- **Prompt advice is model-generation-specific.** The vendor's current
  prompting reference is organised as model-specific guidance first, then
  shared techniques, then migration notes between generations. [CONSENSUS]
  (primary documentation) —
  [Claude prompting best practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices).
  Consequence: "drift from the baseline" has two sources, the prompt and the
  model the baseline assumed; the finding must say which.
- **Structured rubrics make judges harsher; over-flagging is the default
  failure** — the [judge-calibration survey](2026-07-12-judge-calibration-sota.md)
  §1 and §4, and the baseline's own note that rubric detail shifts scores.
  An auditor whose output is a list of findings per agent will find something
  in every agent; a clean verdict must be an admissible result.
- **Formatting sensitivity** (Sclar et al., above) cuts both ways: a
  "low-risk" footer or typo fix the auditor self-applies is a behaviour change
  until the golden fixtures say otherwise.
- **The host already ships a prompt audit.** `/doctor prompt-audit` checks
  CLAUDE.md, CLAUDE.local.md and AGENTS.md plus the rules, skills, commands,
  subagents and output styles under `.claude/` and `~/.claude/` for
  "instructions written for older models, references to files or commands
  that don't exist, and files that contradict each other", returns "a report
  of findings with proposed edits", and "nothing in your files changes until
  you ask Claude to apply them"; a path narrows it to one file or directory.
  [EVIDENCE] (vendor documentation) —
  [How Claude remembers your project](https://code.claude.com/docs/en/memory).
  It is the nearest shipped analogue of the model-generation drift above:
  abcd's `agents/*.md` are outside its default scope unless a path is given,
  its findings carry no baseline section and no fixture gate, and whether
  abcd's auditor defers to it for instruction files is a plan-time question.

**Options weighed.** (a) A free-form auditor agent writing prose findings —
cheap, unfalsifiable. (b) An auditor that emits findings keyed to a baseline
section, each with the model generation the baseline assumed, with
self-application gated on the itd-14 golden run. (c) A GEPA-style optimiser
over the prompts — needs a metric and trainset per agent that only the
reviewer agents (itd-81 corpora) have today.

**Recommendation: Path 2.** Native auditor (b), host-delegated, with one seam:
the findings record is shaped as a candidate-plus-feedback pair (prompt
section, textual feedback, baseline citation, model generation) so a later
optimiser can consume it as GEPA consumes a metric's text feedback. Self-applied
changes go through itd-14's bump-and-golden path, never around it. No
dependency. Self-detection and recursion need no outside practice.

**Local fit risks.** Guidance carries its evidence and its purpose (a finding
that cites no baseline section is noise); abcd fixes what it adds (an auditor
shipped without fixtures of its own is an unmeasured judge under itd-81).

## itd-64 — benchmark-driven configuration optimisation

- **Logged runs are observational; comparison needs the experiment.** One
  vendor's guide (MLflow, dated 17 July 2026) ranks A/B testing high on causal
  validity and offline evaluation as a screening gate over data "collected
  under a previous policy", required to go online when a change explores
  actions the logs do not cover. [ANECDOTE/MARKETING] (a single vendor
  article; the observational-versus-experimental line is this note's
  inference from it, not a controlled-experiment methodology source) —
  [MLflow: what is online evaluation](https://mlflow.org/articles/what-is-online-evaluation-in-ml-a-2026-guide/).
  The matrix note's power finding (20–113 runs per arm to separate observed
  gaps) says what an experiment costs; a correlation harvested from
  heterogeneous runs under different configurations is a hypothesis for a
  matrix cell, not a recommendation.
- **Optimisers need a metric with feedback, a trainset and a budget.** DSPy's
  GEPA and the DSPy optimiser family take a metric (GEPA's may return text), a
  training and a validation set, and a budget (`light` ≈ six candidates), and
  save the result as a JSON artefact; GEPA's paper gains are on prompt-shaped
  problems. [EVIDENCE] / [CONSENSUS] —
  [GEPA](https://arxiv.org/abs/2507.19457), [DSPy GEPA guide](https://dspy.ai/current/getting-started/gepa-optimization/).
  abcd has that metric only where itd-81's corpora exist.
- **Keep the configuration space small.** The evidence is the matrix note's
  power finding: at 20–113 runs per arm, every added dimension multiplies the
  runs owed. Bouchard's cascade result (above, about per-query deferral depth,
  not configuration dimensions) and HAL's finding in the matrix note (higher
  reasoning effort lowered accuracy in most runs) are analogies that point the
  same way. [EVIDENCE] for the power finding; analogy for the rest.
- **Attribute names for the store exist; the standard does not.** The
  OpenTelemetry GenAI registry names `gen_ai.request.model`,
  `gen_ai.response.model`, `gen_ai.provider.name` (which "may differ from the
  actual model provider") and `gen_ai.usage.input_tokens`, which "SHOULD include
  all types of input tokens, including cached tokens", with
  `gen_ai.usage.cache_read.input_tokens`, `cache_creation.input_tokens` and
  `reasoning.output_tokens` as included sub-counts; the page marks the
  conventions as moved to a separate repository, whose README carries no
  stability statement and a schema URL of "TODO". [CONSENSUS] on the names,
  no release to adopt —
  [GenAI attribute registry](https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/),
  [semantic-conventions-genai repository](https://github.com/open-telemetry/semantic-conventions-genai).
- **Outcome definition.** The judge-calibration note's binary-labels finding
  and the matrix note's pass^k apply: "better" is a pass rate on hidden tests
  with an interval, a true-negative rate for reviewers, and cost per passing
  build, never a composite score chosen after the data are in.
- **The matrix intent already decided the relationship**: itd-2610161733316518's
  decision 11 says itd-64 consumes that run's tables.

**Options weighed.** (a) A learning layer over all run signal — confounded by
construction. (b) A store keyed on the matrix run's trial record, with
analysis restricted to comparisons inside a declared cell set and observational
correlations shown as candidate cells. (c) A DSPy/GEPA optimiser — a Python
dependency and a metric abcd mostly lacks; sign-off would be required and it
is premature.

**Recommendation: Path 2.** Native store whose record is the matrix run's trial
record plus the configuration fields (route, review bar, thresholds, prompt
versions from itd-14), named with OpenTelemetry GenAI attribute names where one
exists; analysis reports within-cell comparisons with intervals and lists
observational correlations as proposed cells for the next run; proposals to
change a default cite the run and the smallest difference it could detect. The
seam is the trial record itself (already mapped field by field onto an Inspect
sample and a Harbor trial), so a later optimiser reads the same store. No
dependency now; a GEPA-style optimiser later would be a new dependency and
needs explicit sign-off.

**Local fit risks.** Prefer-the-experiment-to-the-inference (the whole
observational half is inference); less-but-better (the knob set must be
chosen before the store exists, not grown from whatever is logged);
verifier-selects-gates-decide (a tuned default is a proposal).

## itd-98 — solo versus duo is measured, not debated

- **Pair versus solo has a thirty-year answer: it depends, and effort is the
  metric that flips it.** The meta-analytic summary of fifteen pair-programming
  experiments (12 to 295 subjects each) finds pairing "moderately contributes
  to quality", "moderately reduces the duration" and "moderately increases
  effort", with the extent depending "on the group's composition and the tasks'
  characteristics" — complex tasks favour the pair on quality, simple tasks on
  duration at a quality cost. [EVIDENCE] —
  [Dybå, Arisholm, Sjøberg, Hannay and Shull, "Are two heads better than one?" (slides)](https://cs.huji.ac.il/w~feit/sem/se11/PairProgramming.pdf);
  the 2009 journal meta-analysis is Hannay et al., Information and Software
  Technology 51(7) 1110–1122 ([SINTEF record](https://www.sintef.no/en/publications/publication/500724/),
  citation only; the abstract was not opened). Consequence for this intent:
  duo's human minutes are effort and must be counted, or the comparison is
  duration against duration with half of one arm's cost missing.
- **The best human-plus-AI trial measured the opposite of what participants
  felt.** METR's 2025 RCT: 16 experienced maintainers, 246 real issues
  randomised per issue, screen-recorded and self-timed; AI-allowed issues took
  19 % longer, against a predicted 24 % speedup and a felt 20 % speedup
  afterwards; the blog post opened states its confidence intervals as
  forthcoming and gives none. [EVIDENCE] —
  [METR, early-2025 study](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/).
- **The follow-up could not staff the no-AI arm.** METR's February 2026 update
  calls its late-2025 data "only very weak evidence": developers declined to
  work without AI, 30–50 % withheld tasks from the AI-disallowed condition, pay
  fell from $150 to $50 an hour, and self-timing broke when people ran several
  agents at once; returning developers (10) showed −18 % time (−38 % to +9 %),
  new recruits (47) −4 % (−15 % to +9 %), neither interval excluding zero, over
  57 developers, 143 repositories and 800+ tasks. Design alternatives listed:
  shorter intensive experiments, fixed-task experiments, developer-level
  randomisation (lower power, worse developer selection), observational data.
  [EVIDENCE] — [METR, uplift update](https://metr.org/blog/2026-02-24-uplift-update/).
  Consequence: the "who plays duo's human" question is the experiment's
  binding constraint, as METR found; a scoped human budget per arm is the
  honest form.
- **The vendor RCT is one task.** Peng et al. (2023): a treatment group with
  the AI pair programmer built an HTTP server 55.8 % faster; one task, one
  setting. [EVIDENCE], narrow —
  [The Impact of AI on Developer Productivity](https://arxiv.org/abs/2302.06590).
- **Running the same items twice is a crossover, and carryover is the threat
  nobody analyses.** Frattini, Fucci and Vegas review 67 crossover experiments
  in software engineering (136 publications, 2015–2024): 29.5 % of validity
  threats properly addressed, carryover modelled in about 3 % of cases.
  [EVIDENCE] — [Crossover designs in SE experiments](https://arxiv.org/abs/2408.07594).
  Consequence for the matched-backlog question: the same item twice inherits
  knowledge through the record; the design that avoids it is randomising
  items to arms (METR's unit) with twinned pairs only where an item cannot be
  split, and the report names carryover wherever an item did run twice.
- **Escalation quality has a literature, and it is fragmented.** The
  human-AI reliance field distinguishes appropriate reliance from trust, and
  a 2026 review finds its constructs "still fragmented", presents three views
  (Traditional, Appropriateness, Dominance) and argues for a consensus on
  objective metrics; scoring over-reliance (accepting a wrong recommendation)
  and under-reliance (rejecting a right one) separately is the field's common
  objective form, not a finding of that abstract. [CONTESTED] —
  [From Trust to Appropriate Reliance](https://arxiv.org/abs/2604.23896).
  Consequence: escalation quality is a two-by-two (escalated or not, should
  have or not) with both error cells reported, never a single rate.
- **Repeats and power**: the matrix note's finding stands — three-run
  comparisons rank the weaker arm first in 28–44 % of resamples; the report
  states the smallest difference it could have detected (already an
  acceptance criterion of the matrix intent).

**Options weighed.** (a) An ad-hoc paired run with hand-counted metrics — the
present draft. (b) Reuse the matrix run's protocol with mode as the cell axis
(the matrix intent's decision 11 names exactly this), items randomised to arms,
human minutes and tokens as effort, the escalation two-by-two, and the matrix
report's floors. (c) A proper RCT with several maintainers — not available to
a single-maintainer repository.

**Recommendation: Path 2**, via (b): the paired run is a matrix run whose cell
is the mode, reusing its trial record, repeats-sizing, blind review and
per-item divergence listing; two metrics added to the trial record for this
use, human minutes per arm (a declared budget, counted) and tokens per arm
(from itd-2609292107351737's metering); escalation scored as a two-by-two with
a judge outside the run proposing the "should have" column. The seam is the
trial record's mapping onto an Inspect sample and a Harbor trial. No
dependency.

**Local fit risks.** The user's time is theirs (the human budget is an
experimental control the maintainer consents to, never a quota abcd sets);
loud staging (an arm whose human minutes were not recorded is a loud blank);
evaluator outside the loop (the maintainer who played duo's human cannot be
the escalation judge).

## itd-2609292107351737 — a session's token cost against its size class

- **The transcript already carries what the intent wants to keep.** This
  repository's own stream fixture shows each assistant message with
  `usage.input_tokens`, `cache_creation_input_tokens` (split by 5-minute and
  1-hour lifetime), `cache_read_input_tokens`, `output_tokens` and a `model`;
  the result record adds `total_cost_usd`, per-model `modelUsage` with
  `costUSD`, `costBasis: "list"`, `contextWindow` and thinking tokens, and
  `subagent_stats`. [EVIDENCE] (a captured stream) —
  [claude-stream-2.1.302.jsonl](../../../../internal/core/runner/testdata/claude-stream-2.1.302.jsonl).
- **The host's own figure is computed locally, at list price, and is an
  estimate.** `/usage` "computes the dollar figure locally from token counts at
  list price, unless a `modelPricing` table is in effect", is "an estimate, so
  for authoritative billing see the Usage page", and for subscribers "the
  session cost figure isn't relevant for billing purposes"; sub-agents' and
  background requests draw on the same usage, and totals reset on `/clear`.
  [CONSENSUS] (primary documentation) —
  [Claude Code: manage costs](https://code.claude.com/docs/en/costs). The
  intent's "API-equivalent estimate" label is the host's own framing.
- **Two conventions disagree on what "input" means.** OpenTelemetry's
  `gen_ai.usage.input_tokens` "SHOULD include all types of input tokens,
  including cached tokens"; Claude Code's exported `claude_code.token.usage`
  with `type="input"` "excludes tokens read from or written to the prompt
  cache", with `cacheRead` and `cacheCreation` as separate types, plus `model`,
  `query_source` (`main`, `subagent`, `auxiliary`), `speed`, `effort` and
  `agent.name` attributes; its `api_request` event carries `cost_usd` as
  "Estimated cost in USD". [CONSENSUS] on each, [CONTESTED] between them —
  [GenAI attribute registry](https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/),
  [Claude Code monitoring](https://code.claude.com/docs/en/monitoring-usage).
  Consequence: store the four raw counts and derive every total in the report,
  naming the convention it used.
- **Community meters do exactly this, priced from one community table.**
  ccusage reads the host's local usage data, reports by day, month, session and
  billing block, "tracks and displays cache creation and cache read tokens
  separately", prices from LiteLLM's table (embedded at build; per-model
  overrides in a config file) and runs `--offline` on pre-cached prices.
  LiteLLM's `model_prices_and_context_window.json` is "a community maintained
  list" at a stable raw URL, with a `LITELLM_LOCAL_MODEL_COST_MAP` switch whose
  cost is that "you will need to upgrade to get updated pricing, and newer
  models". [CONSENSUS] — [ccusage](https://github.com/ryoppippi/ccusage),
  [LiteLLM token usage](https://docs.litellm.ai/docs/completion/token_usage).
  A third-party claim that ccusage undercounts heavy users and skips sub-agent
  files was seen in search results only. [ANECDOTE/MARKETING]
- **A price is four rates and three modifiers.** The vendor's table lists base
  input, 5-minute cache write (1.25×), 1-hour cache write (2×), cache read
  (0.1×, 0.025× or 0.05× on the newest models) and output per model; batch is
  50 % off, data residency 1.1×; "for the most current pricing information,
  visit" the pricing site. The same page lists two cases a flat four-rate row
  cannot hold: a long-context tier (one model "is priced by prompt length: a
  request whose prompt is over 100,000 tokens pays higher prices") and a
  fast-mode premium that "applies across the full context window" and stacks
  with the other modifiers; the stream records `speed`, so a fast-mode session
  is detectable. The table either carries both or the report marks such a
  session as priced at base rates. [CONSENSUS] (primary documentation) —
  [Anthropic pricing](https://platform.claude.com/docs/en/about-claude/pricing).
  The OpenRouter `/api/v1/models` listing opened in this pass returned
  `max_input_tokens` and `max_tokens` per model with no pricing object; the
  per-call generation record in the matrix note carries `total_cost`.
- **Relative size estimates are weak predictors, and estimators are
  overconfident.** Tawosi, Moussa and Sarro (ESEM 2022): over 37,440 stories
  in 37 projects, story points correlated strongly with development time in
  7 % of projects, medium in 58 %, low in 35 %, and estimators misestimated in
  78 % of cases. [EVIDENCE] —
  [Story points and development effort](https://conf.researchr.org/details/esem-2022/esem-2022-technical-track/6/On-the-Relationship-Between-Story-Points-and-Development-Effort-in-Agile-Open-Source-).
  Jørgensen and Escott (ESEM journal-first 2022; 102 developers, then 20
  tracked): relative estimates "were less accurate than the absolute
  estimates", took longer to produce, and relative effort was "far from
  developer-independent". [EVIDENCE] —
  [Relative estimates of software development effort](https://conf.researchr.org/details/esem-2022/esem-2022-journal-first/4/Relative-estimates-of-software-development-effort-Are-they-more-accurate-or-less-tim).
  Jørgensen's 2004 review finds expert estimation the most used strategy with
  no substantial evidence for formal models over it, and supports assessing
  estimation uncertainty as a practice. [EVIDENCE] —
  [Review of studies on expert estimation (EBSE record)](https://ebse.webspace.durham.ac.uk/?p=167).
  Consequence: a size class is a distribution to be checked by hit rate, not a
  point to be right about, which is what the parked note asked for.
- **Task-level difficulty can be predicted from task features, but not
  cost.** Agent psychometrics (Ge et al., arXiv 2604.00594, 2026) extends item
  response theory with features from issue text, repository context, solution
  and tests, splits ability into model and scaffold, and predicts task-level
  success on unseen benchmarks; it predicts no tokens or dollars. [EVIDENCE] —
  [Agent psychometrics](https://arxiv.org/abs/2604.00594). A search-result
  snippet claims tool-call counts predict accuracy better than lines changed;
  not opened.
- **The repository's own corpus** says the same from the other side: rounds
  track difficulty and lines do not ([size-class note](2026-08-30-spec-size-classification-gap.md)),
  and review cost is dominated by probing, not reading
  ([cost calibration](2026-08-30-review-round-cost-calibration.md)).

**Options weighed.** (a) Adopt ccusage — a Node tool over the host's files,
reading what abcd's history store already captures; a second reader of the
transcript store against the one-reader rule. (b) Native metering at capture
with a price table in LiteLLM's JSON shape. (c) Classes denominated in tokens
— refuted by the corpus and by the story-point evidence.

**Recommendation: Path 2.** At capture, keep the four raw counts, the model
id and the sub-agent split per message, aggregated into metadata; the report
derives totals and states the convention (cached tokens shown inside or beside
input). The price table is a user-level file in LiteLLM's per-model field
shape (four rates, provider, context window) so a refresh can import that
community file on explicit ask and a row can be hand-edited; money is labelled
an API-equivalent estimate at list price, as the host labels its own. Size
classes are denominated in expected review rounds (the corpus's variable),
declared before the build, and reported as a hit rate per class beside the
measured tokens and rounds, so a wrong class shows as a miss and the scheme
survives being wrong. Attribution of a session to a spec: the capture stamp
first (the host's `session_id` is in the stream; `agent.name` is an attribute
of the host's telemetry export only, which the history store does not read,
and the fixture carries no agent-name field), the branch as fallback. itd-98 consumes this measurement rather than counting its own.
No dependency; adopting LiteLLM's file is a format, not a library.

**Local fit risks.** Durable state lives where the platform says it survives
(the history store, never the repository); the user's directory is theirs
(the price table is theirs to edit; no fetch without the ask, adr-38); one
canonical primitive (one reader of the transcript store — no second meter
beside the history store); host-agnostic prose (the report names token types,
not a host's attribute names).

## Local fit risks, collected

- **Host-delegated by default.** Golden runs (itd-14), the prompt audit
  (itd-15) and every judge are model work the host runs; nothing here may
  assume an API key abcd holds, except through the provider adapter already
  decided.
- **Minimal dependencies.** Nothing recommended adds a Go dependency. The two
  things that would (a DSPy/GEPA optimiser for itd-64 or itd-15; a hosted
  prompt registry for itd-14) are named as later moves needing sign-off, and
  both are Python or hosted, against local-first.
- **One canonical primitive.** The matrix run's trial record is proposed as
  the record for itd-64's store and itd-98's comparison; the per-invocation
  route override stays the one route surface; the history store stays the one
  transcript reader.
- **Prefer the experiment to the inference.** itd-64's observational half is
  inference by construction and is confined to proposing cells.
- **The user's directory and time are theirs.** The price table, the human
  budget in itd-98 and the accepted routing table are the person's, edited and
  consented to, never fetched or set.
- **Loud staging.** Every blank named: a PATCH that skipped its golden run, an
  arm with no human minutes, a session with no usage fields, a judgement pair
  skipped as oversize.

## Not worth adopting

- A hosted prompt-management service for eighteen in-repo prompts (Langfuse's
  model is borrowed; the service is not).
- A pre-commit hook that runs golden fixtures: it cannot, deterministically.
- A per-request learned router (the 2026-09-22 pass). A second step up is not
  refused: Bouchard finds depth pays when the deferral signal is a
  current-model error, and the matrix run is where that is tested.
- ccusage or another second meter over the transcript store.
- OpenTelemetry GenAI conventions as a schema to conform to: no release, a
  "TODO" schema URL; attribute names only, as the matrix note already ruled.
- A composite "better outcome" score for itd-64 chosen after the runs.
- Token-denominated size classes.
- A single agreement rate, or raw percentage agreement alone, as the evidence
  that turns a judgement type on.
- Re-running the same backlog item in both arms as the default design for
  itd-98.

## Not verified

- Hannay, Dybå, Arisholm and Sjøberg (2009), the journal meta-analysis: only
  its citation record was opened; effect sizes above are from the authors'
  2007 slide summary of the same fifteen studies.
- Jørgensen's overconfidence studies (90 % effort intervals hitting about
  64 % of the time): seen in search snippets; the publisher's PDF sat behind a
  verification page and was not read.
- Sample-size tables for Cohen's kappa (Cantor 1996; Flack et al.; Sim and
  Wright 2005): named in search results, none opened.
- The expected-calibration-error and Brier conventions: read from 2026
  preprint snippets, none opened in full.
- RouterArena's leaderboard figures and its ICLR 2026 acceptance: the arXiv
  page opened names no venue; figures came from search results.
- The claim that tool-call counts predict agent accuracy better than lines
  changed (arXiv 2610.10610): snippet only.
- The third-party claim that ccusage undercounts heavy users and skips
  sub-agent files: snippet only.
- The vendor's prompt-improver page redirected to the general prompting
  reference; whatever it says about automated prompt rewriting was not read.
- The METR 2025 study's five contributing factors: shown only as an image on
  the page opened.
