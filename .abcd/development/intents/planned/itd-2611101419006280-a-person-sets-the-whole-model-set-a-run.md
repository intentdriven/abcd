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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the strongest model beside the everyday model is expected to catch more applied findings per round than the earlier pair did; shown wrong if, after thirty recorded rounds at the new pair, the research-class model alone cited no more applied findings per round than the old middle-class model alone did at the earlier pair
