---
id: itd-2611101419006280
slug: a-person-sets-the-whole-model-set-a-run
spec_id: spc-2611101423523725
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610240922461281]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# The model set is the person's, in classes named for their use

## Press Release

> A person sets the whole model set a run uses: which model each class holds, how hard each class works, which classes review and which one judges. The four classes are named for what they are for (research, top, default and small), every class runs at high effort unless the person says otherwise, and with nothing configured two reviewers run side by side at the research class and the default class, with the top class named as the judge for when they disagree.

## Why This Matters

The review set of itd-2610240922461281 made the reviewers a setting, but three things
about the models stayed fixed or unclear. The class names (`top`, `middle`, `below`,
`smallest`) say where a model ranks and nothing about what it is for, and "the class
below" is not a name a person types without looking it up. How hard a model works on a
task (its effort) is left to whatever the harness defaults to: the run record has a
place for the effort asked for and writes "not configured" there every time, so two
machines running the same review set can run different reviews. And the bundled pair
leaves the strongest class out of every review, although a review is where a missed
defect costs most.

A person who wants a different arrangement should be able to say so in one place and
see, on every surface, which model and which effort each class stands for.

## Mechanism

We expect the strongest model beside the everyday model to catch more than the earlier
pair did, because the per-reviewer record of itd-2610240922461281 already counts which
reviewer alone cited each applied finding, and the widest spread between two reviewers
gives the least overlap in what they miss. The comparison is by the model each record
names, since the class labels changed. Shown wrong if, after thirty recorded rounds at
the new pair, the research-class model alone cited no more applied findings per round
than the model of the old middle class alone did in the rounds recorded at the earlier
pair. The earlier intent's own thirty-round count at its pair ends with this change;
the rounds recorded so far are the baseline, however many they are, and a baseline
under ten rounds is reported as too thin to decide on.

## Scope Conditions

- The classes are the one class map `models.classes` already is: four classes, each one <!-- cond: cond-2611101423520954 -->
  `<provider>/<model>` route, read from the machine's own configuration over the bundled
  map. A repository still names which classes review and how many, and never which model
  a class holds.
- Effort is a property of a class on this machine. It reaches a model only where abcd <!-- cond: cond-2611101423527751 -->
  launches a role at a class through a runner that can pass it. A reviewer the host
  session runs is asked for the class and its effort, and the record says asked, not
  confirmed.
- The judge is named by default and runs only once a judge agent has cleared itd-81's <!-- cond: cond-2611101423524336 -->
  calibration gate. Until then the default judge is recorded as waiting and the
  stricter verdict decides, as it does today.
- The bundled map names one vendor's models. Another vendor's models are matched to the <!-- cond: cond-2611101423527935 -->
  classes by the person, in their machine's map; abcd bundles no model names it cannot
  launch and test.
- The default pair costs more than the earlier one: every review round of every lane <!-- cond: cond-2611101423529482 -->
  now asks the strongest class once per review role, at high effort. On a subscription
  the classes share one usage window, so an unattended run reaches its limit sooner and
  waits there as it already does.
- Everyone working one repository reads its `review.reviewers` with the same names, so <!-- cond: cond-2611101423522510 -->
  a repository that names classes there is read only by an abcd from this change on.

## Acceptance Criteria

- **Given** no model or review configuration in any layer, **when** a run resolves its
  review set, **then** each review role runs two reviewers, at the `research` class and
  at the `default` class, the judge is the `top` class, and every surface that shows a
  class shows its model and its effort, `high`, with the bundled layer that set them.
- **Given** a review set shown by `abcd build`, `abcd implement` or a drive's start,
  **when** it is rendered, **then** each class reads as its name, its model and its
  effort together, so a class name is never shown alone, and a drive's start names the
  pair it will run before it asks to start.
- **Given** a default judge and no judge agent that has cleared itd-81's calibration
  gate, **when** a run starts, **then** it starts, its state records the judge as
  waiting, its record says so in one line, and for the whole of that run no judge is
  launched, awaited or counted in an estimate, whatever is calibrated meanwhile.
- **Given** a default judge and a judge agent that has cleared the gate, **when** a run
  starts, **then** its state records the judge as active and the start says a judge
  will run on a disagreement.
- **Given** `review.judge` set in a configuration file and no calibrated judge agent,
  **when** a run starts, **then** it is refused at the `review` stage as it is today.
- **Given** a machine configuration that maps `models.classes.research` to another
  model, **when** a run resolves its review set, **then** the research-class reviewer is
  asked for that model, and the record names the machine layer and its file.
- **Given** `models.effort` set to one level in the machine's configuration, **when** a
  runner that takes an effort launches a role at any class, **then** it is launched at
  that level, and the launch's record of what was asked carries the level.
- **Given** `models.effort` set as an object naming a level per class, **when** a role
  is launched at a class, **then** it runs at that class's level, and a class the
  object leaves out runs at `high`.
- **Given** an effort that is not one of `low`, `medium`, `high`, `xhigh` and `max`, or
  an effort object keyed by anything but the four classes, **when** the configuration
  is read, **then** the run is refused before anything is written, naming the key, the
  value and what is accepted.
- **Given** `models.effort` or `models.classes` in a repository's configuration,
  **when** a run resolves its review set, **then** the repository's value is skipped
  with one line naming the machine's file.
- **Given** a role launched with no class, the implementer among them, or a runner
  whose own configured model happens to be a class's model, **when** it is launched,
  **then** no effort is passed and its record says none was configured.
- **Given** a role launched at a class through a runner that cannot pass an effort,
  **when** it is launched, **then** it runs, and the launch's record says the effort
  was not passed to that runner.
- **Given** a launch the harness refuses because of the effort it was asked for,
  **when** the launch fails, **then** the failure names the effort and the class, as
  any other failed launch names its cause.
- **Given** a configuration that names a class `middle`, `below` or `smallest`, in a
  class map, a review list, a judge or an effort object, **when** it is read, **then**
  it is refused naming the key, the file and the new name of each old class (`top` is
  now `research`, `middle` is `top`, `below` is `default`, `smallest` is `small`), and
  nothing is written.
- **Given** a machine class map that sets `top` and does not set `research`, **when**
  it is read, **then** it is refused, saying that `top` once named the strongest class
  and now names the second, and that a map which sets `top` sets `research` too, so
  the file shows which names it was written for.
- **Given** a review list or a judge that names `top` in a configuration file, and no
  configuration file that names `research`, `default` or `small` anywhere, **when** a
  run resolves its review set, **then** it resolves, and one line on stderr says which
  model `top` now stands for and that the strongest class is `research`.
- **Given** a run state written before the rename, **when** a newer abcd reads it,
  **then** it is migrated by its schema version: each stored class label takes its new
  name, each reviewer keeps the model it was resolved to, and the run goes on with the
  reviewers it started with.
- **Given** a run state written after the rename, **when** an earlier abcd reads it,
  **then** it is refused as a newer abcd's state by its schema version, as the loop
  already refuses any newer state.
- **Given** the command page for `abcd build`, **when** a person reads its model
  section, **then** it shows a worked class map for a second provider and the one line
  that sets a cheaper review.

## Open Questions

_None recorded: the product thinker asked for this to be decided within the run, and
the decisions below are recorded for review once it is live._

## Decisions

Recorded 2026-11-10 by the technical facilitator on delegation. The product thinker
asked for a fully customisable model set, the research and default classes as the
default pair with the class between them as the judge, the four new class names, and
high effort as the default, and asked that the remaining design decisions be made
within the autonomous run and recorded for review. An adversarial review pair at the
research and default classes read the first draft, both said fix first, and every
finding is applied here. Each decision is open to reversal at that review.

1. **The names map one for one, and the old names are refused, not aliased.** `top`
   becomes `research`, `middle` becomes `top`, `below` becomes `default`, `smallest`
   becomes `small`. The old `top` and the new `top` are different classes, so an alias
   would read a configuration as the wrong class without a word. This is why the
   impact is breaking.
2. **A reused `top` is caught where a file can show which names it was written for.**
   An old file can never hold `research`, so a class map that sets `top` without
   `research` is refused, and a person who means the new `top` writes both. A review
   list is only a list of names and cannot carry that proof, so a `top` there is read
   as the new class and announced in one line until some configuration file uses a new
   name. Refusing every `top` in a list was considered and left out: it would refuse a
   correct new configuration for ever. The name `top` was the product thinker's choice
   and is kept; a different name for the second class would remove this decision.
3. **Run state moves one schema version and is migrated on read.** The loop's state
   check holds every stored class to the current names, so an unversioned rename would
   refuse older runs and mislabel a stored `top`. The version says which names a state
   file uses, so the migration is exact, and an earlier abcd refuses the newer state
   the way it refuses any newer state. A waiting judge is part of the same version.
4. **The default pair is research and default; the default judge is top.** This
   reverses the bundled pair of itd-2610240922461281 decision 1 on the product
   thinker's instruction, and amends that intent's first criterion and its spec's first
   step, which name the earlier pair. The per-reviewer record that intent set up stays.
5. **A default judge waits, fixed at the run's start; a configured judge still
   refuses.** itd-81's calibration gate is not reopened: no judge runs until a judge
   agent has a passing calibration record. A bundled judge that refused every run
   would stop all work, so the bundled judge is stored with a status, waiting or
   active, decided once when the run starts. A waiting judge is never launched,
   awaited or counted, and a run never gains a judge part-way. Once an agent clears
   the gate, new runs start with the judge active and say so at the start: that is the
   default the product thinker asked for, arriving when the gate allows. A judge a
   person writes into configuration keeps today's refusal, since they asked for
   something the run cannot give. The open spec of itd-2610240922461281 launches a
   judge only where the stored status is active.
6. **Effort is the machine's, per class, default high.** `models.effort` takes one
   level for every class or an object naming a level per class. It is read from the
   machine's configuration or the bundled default alone, as the class map is, because
   a higher effort spends more of the person's budget and a repository must not be
   able to raise it. The levels are the five the harness takes. A per-repository
   effort was considered and left out for that reason; a repository that wants a
   cheaper review names a smaller class.
7. **Effort follows the class.** The reviewers and the judge are launched at a class
   and get its effort, recorded in the place the run record already keeps for the
   effort asked for. A role launched with no class keeps the harness's default, the
   implementer among them, so this intent raises what a review costs and leaves what
   an implementer run costs alone. A runner's own configured model is not a class,
   even where it names the same model.
8. **The cost is stated, not capped.** The strongest class on every round at high
   effort is what was asked for. The start of a run names the pair, and the command
   page gives the one line for a cheaper review. No new limit is added: the run's
   existing pace and its wait at a usage limit already bound an unattended run.
9. **No bundled models for other vendors.** The class names are vendor-neutral and the
   machine's map points any class at any configured provider's model. Bundling model
   names for vendors abcd does not launch in its own tests would ship names that go
   stale unseen. The command page for `abcd build` and the brief's configuration page
   show a worked map for a second provider.
10. **Open records that use the old names are revised in the same change; closed ones
    are not.** The planned intent and open spec of the review set, the open spec that
    names the old class constants, and the draft that counts sub-agents in classes are
    brought to the new names. Shipped intents, closed specs, resolved issues, decision
    records, review receipts and research notes are history and keep their words.
11. **Process only: the review of this change runs at the arrangement it sets.** Its
    code reviews run at the research and default classes, and the top class is asked
    only where the two disagree, by hand, since no calibrated judge exists.

## SOTA

Checked 2026-11-10 against each project's own documentation.

| Alternative | What it does | Maturity |
| --- | --- | --- |
| [Aider](https://aider.chat/docs/config/adv-model-settings.html) | names a weak model and an editor model beside the main one, per model entry; effort is a per-model setting | mature |
| [opencode](https://opencode.ai/docs/models/) | one `provider/model` route, a model per agent, and a variant suffix that selects the reasoning effort | mature, changing fast |
| [Claude Code](https://code.claude.com/docs/en/model-config) | vendor-tier aliases, each remappable by the person, and an effort level saved per model in the person's settings | mature |
| [Codex CLI](https://learn.chatgpt.com/docs/config-file/config-reference) | a reasoning-effort key and a separate review model in the person's file; a project file cannot set provider keys | mature |
| [LiteLLM](https://docs.litellm.ai/docs/routing) | an alias table from a model name to a real route | mature |

Every tool read names its models by function or by one vendor's tiers; none has
vendor-neutral classes, and none pairs reviewers. The nearest seams are an alias table
the person owns, which the class map already is. Effort levels differ by vendor
([Anthropic](https://platform.claude.com/docs/en/build-with-claude/effort),
[OpenAI](https://developers.openai.com/api/docs/guides/reasoning)) and by model, and a
harness may lower a level a model does not take. On judges, one study finds a model
judge prefers text like its own ([arXiv 2410.21819](https://arxiv.org/abs/2410.21819));
no measured study of a two-class review pair with a judge on disagreement was found.

Path taken: 2, a native floor with a seam. The class map stays a thin alias table in
the person's file, and effort is passed to the runner as the runner spells it. Two
adjustments were weighed. Lowering an effort a model does not take is left to the
harness, which already does it, and the launch record's reported side shows what ran.
A judge from a different model family than both reviewers is the hedge the study
suggests; the bundled map is one vendor's, so it cannot give one, and a person who
maps the `top` class to another vendor gets it. That is noted for the judge's
calibration under itd-81 and not required here.

## Typed Links

- builds_on: itd-2610240922461281 (the review set and the class map this renames and
  re-defaults).
- reverses (advisory, for the product thinker to confirm at review): decision 1 of
  itd-2610240922461281, the bundled pair.
- refines: iss-2610140652230444 (the ruling that set the four classes).

Records that read differently under the new names and are revised with this change:
itd-2610100704270254 (sub-agent counts by class) and spc-2610301925429250.
itd-2609170822093401 is where a per-invocation effort was first recorded.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0bb4426c4246 -->
Fidelity review — receipt rcp-0bb4426c4246 (verifier intent-auditor claude-fable-5-2).

Provenance: intent-auditor@claude-fable-5-2 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:99726740d88782ef11c251d06c293b7af84169182d6aae1c0bfb666be7b33bf9
Input attestations: diff:3499ff56279cc9b1ba921f9060be1f8254012d29..e6a7d4b70986a4311bc65cf90c259c7cb9041db7 (merge of pull request #1487 against its first parent; every file:line below is at e6a7d4b7; tests were read, not run)@-;

Acceptance rollup: MET 15 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The bundled pair is research and default, the bundled judge is top, and every class resolves effort high from the bundled layer; the text renders show name, model and effort and say 'efforts from bundled'. Concern: the layer that set a class's model (map_layer) is carried only in the JSON and the state, never in the text line, and the drive's start line (Pair) and the reviewer phrases show no layer at all, so 'every surface ... with the bundled layer that set them' holds for the effort in the run's review line and for both only in JSON.
  evidence: internal/core/implement/loop/reviewset.go:108 — "var BundledReviewers = []string{ClassResearch, ClassDefault}"
  evidence: internal/core/implement/loop/reviewset.go:113 — "const BundledJudge = ClassTop"
  evidence: internal/core/implement/loop/reviewset.go:834 — "rc.Effort, rc.EffortLayer, rc.EffortOrigin = BundledEffort, layered.Bundled.String(), layered.Bundled.String()"
  evidence: internal/core/implement/loop/reviewset.go:325 — "parts = append(parts, "efforts from "+o)"
  evidence: internal/core/implement/loop/reviewset.go:312 — "p := fmt.Sprintf("%s %s (%s)", r.Role, strings.Join(cs, " and "), describeLayer(r.Layer, r.Origin))"
  evidence: internal/core/implement/loop/modelset_test.go:372 — "want: "research=high@bundled default=high@bundled research=high@bundled default=high@bundled judge top=high@bundled"},"
  evidence: internal/surface/cli/build_reviewset_surface_test.go:49 — "if c.Model == "" || c.Effort != "high" || c.EffortLayer != "bundled" {"
- ac-2 — MET: One phrase function renders every class as name, model and effort; build, implement and the run record render the set through it, and the drive's consent prints the pair on a 'review:' line before 'Start? [y/N]'. A test asserts no class name stands alone in either render.
  evidence: internal/core/implement/loop/reviewset.go:232 — "return class + " (" + model + ", effort " + effort + ")""
  evidence: internal/surface/cli/build.go:588 — "fmt.Fprintf(w, " review: %s\n", termsafe.Sanitize(rs.String()))"
  evidence: internal/surface/cli/drain_drive.go:406 — "fmt.Fprintf(w, " review: %s\n", termsafe.Sanitize(review))"
  evidence: internal/surface/cli/drain_drive.go:613 — "yes, err := driveConsent(cmd, asked, f, lanes, runs, review.Pair(), p)"
  evidence: internal/surface/cli/drain_drive_reviewset_test.go:33 — "if strings.Index(text, " review:") > strings.Index(text, "Start? [y/N]") {"
  evidence: internal/core/implement/loop/modelset_test.go:564 — "if a := aloneIn(text); len(a) != 0 {"
- ac-3 — MET: With the gate uncleared the bundled judge is stored waiting and the run starts; the start appends one record line saying it waits; a resume under a calibrated fixture keeps it waiting; the estimate counts only the implementer and the reviewers, and no code path in the delivered tree launches or awaits a review judge.
  evidence: internal/core/implement/loop/reviewset.go:551 — "status = JudgeWaiting"
  evidence: internal/core/implement/loop/loop.go:740 — "st.Record = append(st.Record, Entry{At: now, Stage: StageReview, Note: review.Judge.note()})"
  evidence: internal/core/implement/loop/reviewset.go:336 — "return fmt.Sprintf("the judge, %s, waits for the whole run: no judge agent has cleared itd-81's calibration gate, "+"
  evidence: internal/core/implement/loop/modelset_test.go:297 — "if err != nil || !again.Resumed || again.Review.Judge.Status != JudgeWaiting {"
  evidence: internal/core/implement/loop/modelset_test.go:320 — "if runs != want {"
  evidence: internal/core/implement/loop/budget.go:65 — "func estimateSharesFor(cfg *runner.Config, steps int, audits, autoPlan bool, review *ReviewSet) []budgetShare {"
- ac-4 — MET_WITH_CONCERNS: With the gate cleared the bundled judge is stored active and the run record says a judge will run where a role's two reviewers disagree. Concerns: the production gate answers false unconditionally, so active is reachable only through the test seam Options.judgeCalibrated; and nothing in the delivered code launches a judge on a disagreement (Active() is read only by the note), so the start's statement is a promise the open spec spc-2610240951109420 still has to keep.
  evidence: internal/core/implement/loop/reviewset.go:540 — "status := JudgeActive"
  evidence: internal/core/implement/loop/reviewset.go:334 — "return fmt.Sprintf("the judge, %s, is active: a judge will run where a role's two reviewers disagree", j.phrase())"
  evidence: internal/core/implement/loop/modelset_test.go:334 — "if j := res.Review.Judge; j == nil || j.Class != ClassTop || j.Status != JudgeActive || !j.Active() {"
  evidence: internal/core/implement/loop/reviewset.go:433 — "return false, "no judge agent has a passing calibration record under itd-81's calibration gate " +"
  evidence: internal/core/implement/loop/reviewset.go:203 — "func (j *ReviewJudge) Active() bool { return j != nil && j.Status == JudgeActive }"
  evidence: .abcd/development/specs/open/spc-2610240951109420-a-person-sets-how-many-models-review.md:59 — "stored judge status is `active`: the bundled `top` judge is stored `waiting` until a"
- ac-5 — MET: A judge from any layer but the bundled one, with the gate uncleared, is refused at the review stage with nothing written; the existing refusal test covers a repository's review.judge.
  evidence: internal/core/implement/loop/reviewset.go:545 — "if j.Layer != layered.Bundled {"
  evidence: internal/core/implement/loop/reviewset.go:546 — "return ReviewSet{}, nil, refuse(StageReview, "", "", fmt.Sprintf("%s.%s is %q in %s (%s layer), but a judge runs only once its agent "+"
  evidence: internal/core/implement/loop/reviewset_test.go:223 — "{name: "uncalibrated judge", repo: `{"review": {"judge": "research"}}`,"
- ac-6 — MET_WITH_CONCERNS: A machine models.classes.research replaces the bundled route, the reviewer's await carries that model, and the stored review set names the machine layer and its file for the class (map_layer, map_origin). Concern: only the structured record names them; the run record's text line 'review set ...' shows the model but neither the map's layer nor its file.
  evidence: internal/core/implement/loop/reviewset.go:812 — "rc = ReviewClass{Class: c, Model: route, MapLayer: f.Layer.String(), MapOrigin: f.Origin}"
  evidence: internal/core/implement/loop/reviewset_test.go:119 — "if c := ruth.Reviewers[0]; c.Model != "acme/big-1" || c.MapLayer != "machine" || c.MapOrigin != abcdhome.Display("config.json") {"
  evidence: internal/core/implement/loop/reviewers.go:58 — "out = append(out, ValidatorRun{Role: role, Reviewer: i, Class: c.Class, Model: c.Model, Effort: c.Effort})"
  evidence: internal/core/implement/loop/reviewset.go:305 — "func (rs ReviewSet) String() string {"
- ac-7 — MET: One level in the machine's file is set on all four classes; the await's effort is put on the runner request; the claude runner passes --effort=< level> and records it as the effort asked. A loop-level test asserts both the argv and the route record.
  evidence: internal/core/implement/loop/reviewset.go:877 — "for _, c := range Classes {"
  evidence: internal/core/implement/loop/runner_pause.go:490 — "Effort: aw.Effort,"
  evidence: internal/core/runner/claude.go:199 — "return append(a, "--effort="+req.Effort)"
  evidence: internal/core/runner/claude.go:366 — "ans.EffortAsked = req.Effort"
  evidence: internal/core/implement/loop/reviewers_test.go:573 — "if !slices.Contains(env.argv(t, "claude"), "--effort="+BundledEffort) || routed.Route.EffortAsked != BundledEffort {"
  evidence: internal/core/implement/loop/modelset_test.go:374 — "want: "research=max@machine default=max@machine research=max@machine default=max@machine judge top=max@machine"},"
- ac-8 — MET: Every class is first set to the bundled high, then an effort object sets only the classes it names; the test shows research xhigh, default low and the unnamed top at high from the bundled layer.
  evidence: internal/core/implement/loop/reviewset.go:869 — "set(c, per[c])"
  evidence: internal/core/implement/loop/reviewset.go:145 — "const BundledEffort = "high""
  evidence: internal/core/implement/loop/modelset_test.go:376 — "want: "research=xhigh@machine default=low@machine research=xhigh@machine default=low@machine judge top=high@bundled"},"
- ac-9 — MET: A level outside the five, a non-level value, and an object keyed by anything but the four classes are each refused at the review stage with the key, the value and the accepted form, ending 'nothing was written'; the test asserts the run tier is absent afterwards.
  evidence: internal/core/implement/loop/reviewset.go:852 — "return refuse(StageReview, "", "", fmt.Sprintf("%s (%s layer): %s is %s: %s", f.Origin, f.Layer, key, value, why),"
  evidence: internal/core/implement/loop/reviewset.go:863 — "return nil, bad(effortKey, layered.BoundKey(raw), fmt.Sprintf("it names %q, which is no class; an effort object is keyed by %s","
  evidence: internal/core/implement/loop/reviewset.go:875 — "return nil, bad(effortKey, layered.BoundKey(raw), "an effort is one of "+effortLevelsPhrase)"
  evidence: internal/core/implement/loop/modelset_test.go:438 — "runTierAbsent(t, repo.Root())"
- ac-10 — MET: A repository's models.effort and a repository's models.classes are each skipped with one diagnostic naming the machine's file, and the front door prints the diagnostics on stderr.
  evidence: internal/core/implement/loop/reviewset.go:842 — "diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: a higher effort spends more of the person's budget, so how hard "+"
  evidence: internal/core/implement/loop/reviewset.go:817 — "diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: which model a class runs spends the person's key, so "+"
  evidence: internal/core/implement/loop/modelset_test.go:456 — "if len(res.Diagnostics) != 1 || !strings.Contains(res.Diagnostics[0], "models.effort") ||"
  evidence: internal/surface/cli/build.go:278 — "printReviewDiagnostics(cmd.ErrOrStderr(), res.Diagnostics)"
- ac-11 — MET: The effort flag is added only where the request carries an effort, which only a review set's reviewer does; a request with no class through a runner whose own model is a class's model carries no flag, and its record lists the asked effort as not configured.
  evidence: internal/core/runner/claude.go:195 — "func effortFlag(a []string, req Request) []string {"
  evidence: internal/core/runner/sides.go:154 — "un.Add(oracle.AskedEffort, oracle.NotConfigured)"
  evidence: internal/core/runner/effort_test.go:54 — "ans, _, err := newClaude("anthropic/claude-sonnet-5-5").Run(context.Background(), f.request("implementer"))"
  evidence: internal/core/runner/effort_test.go:61 — "if ans.EffortAsked != "" || reasons(ans.Unpinned)[oracle.AskedEffort] != oracle.NotConfigured {"
- ac-12 — MET: The opencode runner launches the role without an effort flag and its record lists the asked effort as not_passed, which survives the stored route record's round trip.
  evidence: internal/core/runner/opencode.go:156 — "unpassed := req.Effort != """
  evidence: internal/core/runner/sides.go:152 — "un.Add(oracle.AskedEffort, oracle.NotPassed)"
  evidence: internal/core/oracle/sides.go:100 — "NotPassed UnpinnedReason = "not_passed""
  evidence: internal/core/runner/effort_check_test.go:67 — "if got := reasons(back.Unpinned)[oracle.AskedEffort]; got != oracle.NotPassed {"
- ac-13 — MET: Every failed claude launch at an effort has the effort and the class appended to the failure's detail, on both failure paths, beside the cause the failure already names.
  evidence: internal/core/runner/claude.go:206 — "fl.Detail += fmt.Sprintf("; it was launched at effort %s, the %s class's", req.Effort, req.Class)"
  evidence: internal/core/runner/claude.go:347 — "effortNamed(fl, req)"
  evidence: internal/core/runner/effort_test.go:96 — "if !strings.Contains(err.Error(), "effort max") || !strings.Contains(err.Error(), "research class") {"
- ac-14 — MET: Every class name a file holds, in a class map, an effort object, a review list (plain or per role) and a judge, is collected, and middle, below or smallest is refused at the review stage naming the key, the file, its layer and the whole mapping, with nothing written; tests cover each of the five places.
  evidence: internal/core/implement/loop/reviewset.go:726 — "return refuse(StageReview, "", "", fmt.Sprintf("%s in %s (%s layer) names class %q, a name no class takes any more: %s","
  evidence: internal/core/implement/loop/reviewset.go:91 — "const renameNote = "the classes were renamed for what each is for: `top` is now `research`, `middle` is `top`, `below` is `default` " +"
  evidence: internal/core/implement/loop/modelset_test.go:75 — "for _, w := range append(tc.want, renameMapping...) {"
  evidence: internal/core/implement/loop/modelset_test.go:420 — "{name: "an old class", machine: `{"models": {"effort": {"middle": "high"}}}`,"
- ac-15 — MET: A machine class map that sets top and not research is refused, saying top once named the strongest class and now names the second and that a map setting top sets research too; a map naming both resolves silently.
  evidence: internal/core/implement/loop/reviewset.go:744 — "return refuse(StageReview, "", "", fmt.Sprintf("%s.%s in %s (machine layer) is set and %s.%s is not: `top` once named the strongest class "+"
  evidence: internal/core/implement/loop/reviewset.go:747 — "fmt.Sprintf("a map that sets %s.%s sets %s.%s too, so the file shows which names it was written for: set %s.%s in %s to the strongest "+"
  evidence: internal/core/implement/loop/modelset_test.go:93 — "for _, w := range []string{"models.classes.top", "models.classes.research", abcdhome.Display("config.json"), "strongest", "second"} {"
- ac-16 — MET_WITH_CONCERNS: A review list or judge naming top, with no file naming research, default or small, resolves with one diagnostic naming the model top stands for and research as the strongest class, printed on stderr. Concern: the line is also suppressed by any models.effort object, even one keyed only by top, so a list naming top beside {"effort": {"top": ...}} gets no line although no file names research, default or small; the criterion states no such exception. The carve-out is deliberate and tested.
  evidence: internal/core/implement/loop/reviewset.go:774 — "return fmt.Sprintf("review: %s names `top`, which now stands for %s, the second class; the strongest class is now `research` (%s). "+"
  evidence: internal/core/implement/loop/modelset_test.go:124 — "if len(res.Diagnostics) != 1 {"
  evidence: internal/core/implement/loop/reviewset.go:764 — "if strings.HasPrefix(n.key, effortKey+".") || (n.name != ClassTop && slices.Contains(Classes, n.name)) {"
  evidence: internal/core/implement/loop/modelset_paths_test.go:72 — "if len(res.Diagnostics) != 0 {"
- ac-17 — MET: The schema version moves to 25; a file of version 24 or lower has every stored class label renamed in the review set, the judge, each await and each validator, leaving the stored models untouched, and is read at the current version; the test steps the migrated run.
  evidence: internal/core/implement/loop/state.go:316 — "const SchemaVersion = 25"
  evidence: internal/core/implement/loop/state.go:1418 — "if !st.renameClasses() {"
  evidence: internal/core/implement/loop/reviewset.go:621 — "rename(&rs.Roles[i].Reviewers[k].Class)"
  evidence: internal/core/implement/loop/modelset_test.go:208 — "if c.Class != was.Class || c.Model != was.Model {"
  evidence: internal/core/implement/loop/modelset_test.go:217 — "if _, err := advance(root, id, DefaultStages(), Options{}); err != nil {"
- ac-18 — MET: The new state is written at version 25, and the code before this change (the merge's first parent, version 24) already refuses any higher version as a newer abcd's; the same refusal is asserted for version 26 against the new reader.
  evidence: internal/core/implement/loop/state.go:316 — "const SchemaVersion = 25"
  evidence: internal/core/implement/loop/state.go:1543 — "return State{}, refuse("state", "", "", fmt.Sprintf("is schema version %d, which a newer abcd wrote; this abcd reads versions %d to %d","
  evidence: internal/core/implement/loop/modelset_test.go:260 — "if r := mustRefusal(t, err); !strings.Contains(r.Reason, "a newer abcd wrote") {"
- ac-19 — MET: The build command page's review set section carries the one line for a cheaper review and a worked class map pointing all four classes at a second provider's models; a test reads the page for both.
  evidence: commands/build.md:581 — "{"review": {"reviewers": ["default", "small"]}}"
  evidence: commands/build.md:584 — "A class map for a second provider points each class at that provider's models"
  evidence: commands/build.md:594 — ""research": "openrouter/qwen/qwen3-235b-a22b","
  evidence: internal/surface/cli/build_modelset_page_test.go:37 — ""A class map for a second provider","

Gap audit:
- honoured:
  - The four classes are named research, top, default and small, one for one, and the old names are refused with the whole mapping, never aliased.
    evidence: internal/core/implement/loop/reviewset.go:79 — "var Classes = []string{ClassResearch, ClassTop, ClassDefault, ClassSmall}"
    evidence: internal/core/implement/loop/reviewset.go:85 — "var renamedClasses = map[string]string{"top": ClassResearch, "middle": ClassTop, "below": ClassDefault, "smallest": ClassSmall}"
  - With nothing configured two reviewers run at the research and default classes, with the top class named as a judge that waits.
    evidence: internal/core/implement/loop/reviewset.go:108 — "var BundledReviewers = []string{ClassResearch, ClassDefault}"
    evidence: internal/core/implement/loop/reviewset.go:551 — "status = JudgeWaiting"
  - Every class runs at high effort unless the person says otherwise, set per class or for all, in the machine's file alone.
    evidence: internal/core/implement/loop/reviewset.go:145 — "const BundledEffort = "high""
    evidence: internal/core/implement/loop/reviewset.go:841 — "if f.Layer != layered.Machine {"
  - The effort reaches the harness where a runner can pass it, and the launch record says what was asked, what was not passed and what was not configured.
    evidence: internal/core/runner/claude.go:199 — "return append(a, "--effort="+req.Effort)"
    evidence: internal/core/runner/sides.go:152 — "un.Add(oracle.AskedEffort, oracle.NotPassed)"
  - Run state moves one schema version and older states are migrated on read, each reviewer keeping its model.
    evidence: internal/core/implement/loop/state.go:316 — "const SchemaVersion = 25"
    evidence: internal/core/implement/loop/reviewset.go:605 — "func (s *State) renameClasses() bool {"
  - A drive's start names the pair it will run, with models and efforts, before it asks.
    evidence: internal/surface/cli/drain_drive.go:406 — "fmt.Fprintf(w, " review: %s\n", termsafe.Sanitize(review))"
  - The cost is stated and a cheaper review is one line; a worked map for a second provider is on the build page and the brief's configuration page.
    evidence: commands/build.md:573 — "Every review round of every lane asks the strongest class once per review role,"
    evidence: .abcd/development/brief/05-internals/03-configuration.md:644 — ""research": "openrouter/qwen/qwen3-235b-a22b","
- diverged:
  - Every surface that shows a class shows the layer that set its model and effort: the text surfaces name the layer of the list and of the efforts, but the class map's layer and file appear only in JSON and the state, and the drive's start line and the reviewer phrases name no layer.
    evidence: internal/core/implement/loop/reviewset.go:312 — "p := fmt.Sprintf("%s %s (%s)", r.Role, strings.Join(cs, " and "), describeLayer(r.Layer, r.Origin))"
    evidence: internal/core/implement/loop/reviewset.go:276 — "func (rs ReviewSet) Pair() string {"
  - The lone-top line is promised whenever no configuration file names research, default or small; it is also withheld when any file holds an effort object, even one keyed only by top.
    evidence: internal/core/implement/loop/reviewset.go:764 — "if strings.HasPrefix(n.key, effortKey+".") || (n.name != ClassTop && slices.Contains(Classes, n.name)) {"
    evidence: internal/core/implement/loop/modelset_paths_test.go:58 — "machineConfig(t, `{"models": {"effort": {"top": "max"}}}`)"
  - An active judge's start line says a judge will run on a disagreement, but the delivery launches no judge; the launch is left to the open spec of itd-2610240922461281, and the production gate never answers calibrated.
    evidence: internal/core/implement/loop/reviewset.go:334 — "return fmt.Sprintf("the judge, %s, is active: a judge will run where a role's two reviewers disagree", j.phrase())"
    evidence: internal/core/implement/loop/reviewset.go:433 — "return false, "no judge agent has a passing calibration record under itd-81's calibration gate " +"
  - Open records are brought to the new names (decision 10): the planned intent's criterion and decisions carry dated amendments, but its Mechanism still reads in the old names with no amendment.
    evidence: .abcd/development/intents/planned/itd-2610240922461281-a-person-sets-how-many-models-review.md:38 — "the fix round applied or rejected it, so "the class below catches what the middle class"
  - A repository that names classes in review.reviewers is read only by an abcd from this change on: a list naming only top is read by an earlier abcd too, as the strongest class, and by this one as the second class with one stderr line.
    evidence: internal/core/implement/loop/reviewset.go:767 — "if at == "" && n.name == ClassTop && strings.HasPrefix(n.key, reviewNamespace+".") {"
- missing:
  - The Mechanism's test (applied findings per round by the model each record names, against the rounds recorded at the earlier pair, with a baseline under ten rounds reported as too thin) has no delivered count or report; the per-verdict record carries the model and the effort, and nothing compares them.
    evidence: .abcd/development/intents/shipped/itd-2611101419006280-a-person-sets-the-whole-model-set-a-run.md:42 — "names, since the class labels changed. Shown wrong if, after thirty recorded rounds at"
    evidence: internal/core/implement/loop/record.go:173 — "Effort string `json:"effort,omitempty"`"
  - Decision 11 (this change's own code reviews ran at the research and default classes, the top class asked by hand on a disagreement) leaves no artefact in the delivered range: no review receipt or record of it is in the diff, and the pull request carries no review.
    evidence: .abcd/development/intents/shipped/itd-2611101419006280-a-person-sets-the-whole-model-set-a-run.md:215 — "11. **Process only: the review of this change runs at the arrangement it sets.** Its"
  - The live check that each bundled class launches at its effort exists behind a build tag nothing runs by default, and the delivery records no run of it.
    evidence: internal/core/runner/live_effort_test.go:1 — "//go:build live"

Scope-condition dispositions:
- cond-2611101423520954 — survived: The class map is still the one map of four classes, each a < provider>/< model> route read from the machine's file over the bundled map; a repository's map is skipped with a warning, and a repository still names the review list.
  evidence: internal/core/implement/loop/reviewset.go:801 — "if f.Layer != layered.Machine {"
  evidence: internal/core/implement/loop/reviewset.go:817 — "diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: which model a class runs spends the person's key, so "+"
- cond-2611101423527751 — survived: The effort is resolved per class from the machine or the bundled layer, passed only by the claude runner for a request at a class, recorded as not passed by opencode, and a host-run reviewer's move says the effort was asked, not confirmed.
  evidence: internal/core/runner/claude.go:199 — "return append(a, "--effort="+req.Effort)"
  evidence: internal/core/runner/opencode.go:156 — "unpassed := req.Effort != """
  evidence: internal/core/implement/loop/loop.go:1518 — "at += fmt.Sprintf(", at effort %s, the class's, asked, not confirmed", a.Effort)"
- cond-2611101423524336 — survived: The bundled judge is named by default and stored waiting while the gate is uncleared, and the record line says the stricter verdict decides; the gate itself is unchanged and answers uncalibrated for every agent abcd ships.
  evidence: internal/core/implement/loop/reviewset.go:551 — "status = JudgeWaiting"
  evidence: internal/core/implement/loop/reviewset.go:337 — ""so where a role's two reviewers disagree the stricter verdict decides, and no judge is launched, awaited or counted", j.phrase())"
- cond-2611101423527935 — narrowed: The bundled map names one vendor's four models and none for another vendor, and the pages tell the person to match another vendor's models in the machine's map; but that each bundled name can be launched and tested rests on a live-tagged check that nothing runs by default, and the delivery records no run of it.
  narrowing: holds for the bundled map naming one vendor's models and none for others; that each bundled name was launched and tested is shown only by a tagged live check that exists and is not run by default, with no recorded run in the delivery
  evidence: internal/core/implement/loop/reviewset.go:98 — "ClassResearch: "anthropic/claude-fable-5","
  evidence: internal/core/runner/live_effort_test.go:40 — "func TestEachBundledClassLaunchesAtItsEffort(t *testing.T) {"
  evidence: commands/build.md:584 — "A class map for a second provider points each class at that provider's models"
- cond-2611101423529482 — survived: The bundled list of each review role names the research class once, at the bundled high effort, so every round asks the strongest class once per role; the build page states the cost and the shared usage window. How soon an unattended run reaches a limit is not exercised by the delivery, and no new limit was added.
  evidence: internal/core/implement/loop/reviewset.go:108 — "var BundledReviewers = []string{ClassResearch, ClassDefault}"
  evidence: internal/core/implement/loop/reviewset.go:145 — "const BundledEffort = "high""
  evidence: commands/build.md:575 — "classes. On a subscription the classes share one usage window, and an"
- cond-2611101423522510 — narrowed: A repository list naming research, default or small is refused by an earlier abcd as an unknown class, and one naming middle, below or smallest is refused by this abcd, so those are read by one side only. A list naming only top is read by both, as the strongest class by an earlier abcd and as the second class by this one, which says so in one stderr line; the two do not read it with the same names.
  narrowing: holds for a review.reviewers list that names research, default or small (or a retired name); a list naming only top is read by an earlier abcd and by this one as different classes
  evidence: internal/core/implement/loop/reviewset.go:774 — "return fmt.Sprintf("review: %s names `top`, which now stands for %s, the second class; the strongest class is now `research` (%s). "+"
  evidence: internal/core/implement/loop/reviewset.go:782 — "return refuse(StageReview, "", "", fmt.Sprintf("%s.%s in %s (%s layer) names class %q, which the class map does not carry: "+"
  evidence: internal/core/implement/loop/modelset_test.go:121 — "if r := reviewRole(t, res.Review, RoleRuthless); strings.Join(classesOf(r), ",") != ClassTop {"
<!-- abcd-review-end receipt=rcp-0bb4426c4246 -->

## Grounds

- pursued: the strongest model beside the everyday model is expected to catch more applied findings per round than the earlier pair did; shown wrong if, after thirty recorded rounds at the new pair, the research-class model alone cited no more applied findings per round than the old middle-class model alone did at the earlier pair
