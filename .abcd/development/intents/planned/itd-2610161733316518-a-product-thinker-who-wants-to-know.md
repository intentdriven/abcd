---
id: itd-2610161733316518
slug: a-product-thinker-who-wants-to-know
spec_id: spc-2610161949559050
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609212137128014, itd-2609201916056194, itd-2609170822093401, itd-2610091016262335]
severity: minor
origin: researcher-authored
production_mode: dictated-and-formatted
blocked_by: [itd-166]
related_issues: [iss-2610161709506207, iss-2610161734413178, iss-2610161734428291, iss-2610161801358637, iss-2610140652230444, iss-2610161928115039, iss-2610161938458308]
related_adrs: [adr-2610161733400551]
impact: additive
---

# One task done across a model matrix, re-runnably

## Press Release

> **Moving a role to a cheaper or a local model becomes a measured decision, not a
> bet.** A product thinker who wonders whether a cheaper model could build their
> features, write their docs or check those docs against the code, or whether a local
> model reviews as well as the one they pay for, names the task, the models and how many
> of each, and starts a model-matrix run.
>
> abcd does that one task many times from the same starting point, repeating each model
> often enough to tell a real difference from luck. The judging fits the task: for code,
> tests no builder saw and reviewer models that are not told who wrote each copy; for
> writing, the product thinker or a team of them, reading the copies without knowing
> which model wrote which; for a check such as doc fidelity, a side-by-side view of where
> the copies overlap and differ. Nothing is merged. Afterwards they read one table: for
> each model, how often its build passed tests it never saw and how sure that number is,
> what the reviewers caught, what a passing build cost and how long it took. Anyone who
> doubts the table can start the same run again from its record and see where it
> drifted.
>
> "I'd been paying for the big model to do everything because I had no evidence
> otherwise," said Alice, a solo founder. "The table showed the local model passed nearly
> as often on this kind of feature, and the reviewer I'd doubted caught more than I
> expected. I moved two roles the same day."

## Why This Matters

Which model builds and which reviews is today a judgement written into a routing table.
Moving a role to a cheaper model, or from a hosted model to a local one, is a bet nobody
measured. A comparison only helps if it can be trusted and repeated: the same feature,
the same starting point, the same harness, enough repeats to see past run-to-run noise,
reviewers who are not told whose work they judge, and a record exact enough that someone
else can run it again and see where it drifted.

## Mechanism

We expect the table to change routing choices because every copy differs from its
neighbours only in the model (and, across sources, the harness), is judged by something
that never saw who made it (hidden tests for code, people who haven't read it before for writing, a matched
comparison of findings for a check), and is repeated to a count sized for the smallest
difference that would change a choice; shown wrong if, at that count, repeats of one cell
spread as widely as the cells differ, so the table cannot separate the models it
compares.

## Scope Conditions

- The run happens on the person's own machine, through model routes from their own <!-- cond: cond-2610161949554962 -->
  configuration. Its record names every route but does not launch one: another person
  re-runs it by setting up the same routes on their machine.
- Every task is fixed before any run and has not been done anywhere public: a code task <!-- cond: cond-2610161949553740 -->
  has a public interface in its spec that tests written in advance can exercise from
  outside; a writing task names its source material and the judges' criteria; a check
  names the material it checks.
- Pinning is unequal by source. A local model can be named down to its weight file; an <!-- cond: cond-2610161949552168 -->
  OpenRouter model down to its provider and settings; a host model only as far as the
  host reports, with the rest marked unpinned.
- Model and harness travel together by source. Host models run in the host's harness and <!-- cond: cond-2610161949555623 -->
  every other model in the opencode runner, so a comparison across sources compares model
  and harness together; a comparison within one source compares models.
- The machine serves one local model at a time, under a load cap below its core count <!-- cond: cond-2610161949556950 -->
  that the person consents to when the run starts.
- Judges take part through an abcdesign.app session the person hosts. Hosted on the <!-- cond: cond-2610161949559968 -->
  person's own machine it reaches only devices on their private network; judges
  elsewhere join through the team's relay.
- A check runs where the check itself runs: doc fidelity, for instance, needs the <!-- cond: cond-2610161949559501 -->
  repository's surface chapters. A writing task's frozen commit predates the doc's own
  answer, so no copy can read it from history.
- A writing task judged by a few people supports choosing a copy, or a model for that <!-- cond: cond-2610161949552231 -->
  kind of page; a claim about models in general needs more pages and more judges than
  one run of a decision tool provides.

## SOTA

Surveyed and fit-challenged on 2026-10-16; sources, evidence tiers and the local checks
are in the [model-matrix SOTA note](../../research/notes/2026-10-16-model-matrix-sota.md).

- **Repeats.** Spread inside one model-and-harness pairing has been measured larger than
  the spread between pairings, and separating the gaps observed needed about 20–113 runs
  per arm (arXiv 2609.33812); size repeats by a power estimate from a pilot (Miller, arXiv
  2411.00640); report pass@1 with an interval and pass^k (τ-bench).
- **The cell is model plus harness.** Harness gains depend on the model (2609.33812,
  exploratory); SWE-bench holds its scaffold fixed for comparison.
- **Served, not only asked.** OpenRouter's generation record names the served model,
  provider, fallback attempts, tokens and cost; provider pinning exists; providers of one
  model differ in tool-call accuracy.
- **Nondeterminism floor.** Temperature zero with a seed does not reproduce hosted
  output; infrastructure alone moved Terminal-Bench by six points. The record's stance is
  [a model run is re-runnable, not reproducible](../../principles/a-model-run-is-re-runnable-not-reproducible.md).
- **Review.** Self-preference is contested; consensus underperforms the best single judge
  on code; reviewers are blinded and scored, never voted
  ([adr-2610161733400551](../../decisions/adrs/2610161733400551-a-reviewer-in-a-model-matrix-run-never.md)).
- **Outcome validity.** Weak tests mis-estimate performance (ABC checklist, arXiv
  2507.02825); reject rule-breaking runs before ranking (2609.33812).
- **Existing harnesses.** Harbor, Inspect AI with inspect_swe, HAL, the SWE-bench harness.
  All are Python, most need Docker, and the ones that run host agents do so on API keys,
  against host-only frontier models and the one-reader credential rule.
- **Path 2**, a native run on abcd's own loop and lab, with a real seam: one trial
  interface (frozen commit, intent, cell → trial record) whose record maps field by field
  onto a Harbor trial result and an Inspect sample, so a later move replaces only the
  scheduler.

## Acceptance Criteria

Seen by the person:

- **Given** a declared matrix (the tasks, how each is judged, the builder and reviewer
  models, how many of each, the repeats and, for a run with a metered cell, the spending
  cap), **when** the person starts
  the run, **then** abcd shows the number of copies, the expected cost by source against
  the cap, the load cap, the time it expects and the judges it will need, and starts
  nothing until the person consents.
- **Given** a finished code task, **when** the person opens its report, **then** each
  builder shows its pass rate on the hidden tests with an interval, its pass^k, the
  reviewers' confirmed findings, its cost per passing build stated per source with a note
  on what is comparable, and its model time excluding time spent waiting on gates.
- **Given** copies of a written doc, **when** a judge joins the run's judging session on
  abcdesign.app by its code, **then** the judge first calibrates on one good and one bad
  example, then sees the copies drawn for them (at most eight) under random codes, in an order drawn for that judge, rendered through one
  template, ranks them on accuracy against the source before style, marks each good
  enough to ship or not; the report gives each model a rating from all the judges' rankings
  with its spread, the judges' agreement and each copy's length. A judge's verdict is saved to the run's record on the host's machine as it
  is given, nothing is sent to GitHub, and a judge who leaves and rejoins finds their
  ranking where they left it.
- **Given** copies of a check, such as a doc-fidelity review, **when** the person judges
  them in the run's judging session, **then** they first give a verdict on each finding,
  shuffled, with no model named; once their verdicts are locked, one table shows each
  finding as a row and each model as a column, marks the findings only one model raised,
  shows the proposed verdict from a judge outside the matrix, and gives each model its
  precision, its unique valid findings, its recall on the seeded falsehoods and against
  the findings judged valid, and how much each pair of models overlaps.
- **Given** a comparison resting on fewer repeats than its power estimate asks for,
  **when** the report shows it, **then** it says the smallest difference it could have
  detected rather than ranking.
- **Given** a finished run's record, **when** the person starts it again on a machine
  with the same routes, **then** abcd re-runs it and reports, field by field, where the
  second run drifted from the first.
- **Given** a finished run, **when** it ends, **then** no copy is merged, no finding is
  filed as an issue, and the task's material is in the state it was in before.
- **Given** a finished run, **when** the person opens its report, **then** it lists the
  run's limitations (what it could not pin, examine or detect at its repeat count) and
  the improvements it suggests for the next run.

Technical, as one list:

- **Given** a runner that fails, refuses or answers unparsably inside a cell, **when** the
  copy is scored, **then** it counts as a failed attempt for that cell and never falls
  back to the host model; a copy carrying a fallback receipt is excluded and counted as a
  failure.
- **Given** an OpenRouter provider that is down before a copy starts, **when** abcd
  retries, **then** it retries on the same pin, records the attempt as not started, and
  never scores it as the model's failure.
- **Given** a copy, **when** it is made, **then** it is made in its own clone of the
  frozen commit outside every working tree, holding no local-tier file and no other
  copy's refs, and a code task's hidden tests sit outside that clone; for a copy made in
  the host's harness, the report says this isolation is asserted, not proven.
- **Given** a copy routed to a provider, **when** it is about to start, **then** the
  clone has passed the secret scan the launch bundle uses, and a hit refuses the cell.
- **Given** the copies are made, **when** they are ranked, **then** a compliance check
  has run first (tests edited, other copies read, the network policy kept); a copy that
  broke a rule is excluded and named, and a copy the check could not examine is marked
  unexamined and reported apart, never ranked as compliant.
- **Given** a reviewer model or a human judge, **when** it judges a copy, **then** it is
  told nothing about authorship as the blind-review decision sets out; a reviewer
  model's recall and true-negative rate are measured on known-good and seeded-defect
  copies, a reviewer model's guess at the author is scored, and same-family verdicts
  are reported apart; people are not asked to guess.
- **Given** the findings of check copies, **when** they are matched, **then** an exact
  key matches first (the consistency finding key; for doc fidelity, the document,
  chapter and quoted sentence, then overlapping lines), a matcher model takes only the
  leftovers, and the matcher's agreement with a human-labelled sample is reported.
- **Given** any call in the run, **when** it is recorded, **then** the record holds the
  asked and served sides of the run record, and every field a source cannot expose is
  marked unpinned or self-reported.
- **Given** the run, **when** its record is written, **then** it names the frozen commit,
  abcd's binary digest and version, the prompt and tool-definition hashes, the template
  doc copies are rendered through, the machine, the concurrency and the load cap.
- **Given** local cells, **when** they run, **then** one inference server serves at a
  time, started as one owned process group and stopped as one when its cells end.
- **Given** a copy's output, **when** it is shown to a judge, **then** it is rendered as
  inert text in an isolated frame, so nothing a model wrote can run in the judge's page.
- **Given** a check copy, **when** its clone is about to be deleted, **then** its findings
  have been read into the run's record first, the check ran in report mode, nothing it
  proposed was applied to the material, and the release gates' receipts are unchanged.
- **Given** a judging session's verdicts, **when** the run reads them, **then** it
  validates them as untrusted input, keeps one ranking per judge identity, and refuses
  the file with nothing recorded when it does not validate.
- **Given** a trial record, **when** it is mapped onto a Harbor trial result and an
  Inspect sample, **then** every field lands, checked by a test over a fixture of each.

## Decisions

Recorded 2026-10-16. Product-thinker rulings are restated in plain words; the rest are
taken by the facilitator on delegation, each on the grounds given, after two adversarial
reviews (design and record discipline, each an Opus and Sonnet pair) and an independent
SOTA fit-challenge.

1. **Five records** (product thinker): this intent is the run; recording what was asked
   and what served is itd-166, which this intent is blocked by; the plumbing is the
   captures listed under related issues; blind review is adr-2610161733400551; the
   reproducibility stance is the principle linked under SOTA.
2. **Where models come from** (product thinker): frontier models run on the host only and
   never through OpenRouter; OpenRouter serves cheaper models, to trial a model before
   moving it local or because it costs less than a host model; local models run on this
   machine. This keeps decision 3 of itd-2609081951381895 for frontier models and widens
   OpenRouter's use from decision models to cheaper builder and reviewer models, under
   the per-provider allowlist (adr-2609300107513982). That enforcement holds while the
   person's own allowlist leaves frontier models off an OpenRouter route, since no vendor
   list is bundled; the run's plan names each cell's source before consent, so a
   frontier model on OpenRouter is visible before anything is sent.
3. **No copy lands.** Picking the best of several copies by a noisy table selects on
   noise, and a landed copy contaminates any re-run of that feature. The feature is built
   afterwards through the ordinary loop. Grounds: 2609.33812; the confirmed split said
   unmerged.
4. **Run shape: a lab procedure first.** The run is a lab procedure that drives the build
   loop's stages in a trial mode which takes no reservation and lands nothing, one isolated
   clone per copy in the machine-scoped store, each copy keyed by intent, cell and
   repeat. A new verb waits for a second use. Grounds: script-first-mvp; a lab already
   pins the commit and binary; without a trial mode, copies of one intent collide on the
   reservation footprint and each would land. The trial mode is a change to
   itd-2609201916151817's loop, reachable from both the command line and the plugin
   surface. Inside a matrix cell it switches off the host fallback that the runners and
   the routing table otherwise take, because the cell is the route under test and a host
   answer would measure the wrong model; invariant 2's purpose, never to block, still
   holds, since a failed cell is scored as a failure and the run goes on. As a lab, the
   run follows the lab discipline (itd-2609251624540864): the person's route file and each
   runner's credential store are named reads, used in place and never copied into the
   lab; each copy's clone is bundled and its live tree deleted once scored, and a re-run
   needs the frozen commit and the run's record, which keeps the hidden tests, the
   seeded material and the render template by content.
5. **Sampling: each vendor's recommended settings, pinned per cell.** One fixed setting
   cannot apply: host models cannot be set, several reasoning models refuse a
   temperature, and open-weight cards warn against greedy decoding. Where a setting cannot
   be set, the record says "harness default" with the harness version.
6. **OpenRouter fallbacks off.** A fallback changes the cell (providers differ in
   tool-call accuracy); an outage is retried on the same pin and not scored.
7. **A model may build and review in one run**, with its same-family verdicts reported
   apart. Forbidding it would confound the reviewer comparison; hidden-test scoring makes
   any bias measurable.
8. **Repeats are sized, not picked.** A pilot of three to five runs per cell estimates the
   spread; the count is then sized by a power estimate for the smallest difference that
   would change a routing choice, and the report states it.
9. **Hidden tests** are written from the acceptance criteria before any run by a host
   model that is not a builder in that run, read by the facilitator, and kept outside
   every builder's reach.
10. **The harness confound is reported, not removed.** Comparisons within one source are
    reported as model comparisons, across sources as model-plus-harness; an anchor cell
    running one model through both harnesses waits until a run needs it.
11. **Siblings stay separate.** itd-98 (solo versus duo) is a different user moment that
    can reuse this run's protocol, with facilitator mode as one axis; it is not folded in.
    itd-64 (benchmark-driven configuration) would consume this run's tables. Both links
    are written when either is planned.
12. **Load.** Local cells run one server at a time, as one owned process group, under a
    cap below the core count the person consents to at the start; a run never starts
    local inference without that consent. Every network call (provider calls, the
    served-side readback, retries) happens inside the run the person started, never
    implicitly (invariant 7).
13. **Impact: additive**, a new capability, stamped at planning.
14. **Each copy is routed with the existing per-invocation route override**, extended to
    name a runner and its model, rather than a second override (one-canonical-primitive);
    iss-2610161801358637 is that extension. itd-166's asked and served sides extend the
    route receipt that override already writes.
15. **What the run is for, in stages** (product thinker, 2026-10-16): first a routing
    decision tool that improves abcd's own choice of which model does which job; then the
    same comparison for any repository abcd manages, so each can tune its own routing;
    and, if the results warrant it, a publishable research claim that helps others pick
    the combination for their use case. The first run is sized as a decision tool, and
    its record is kept to the standard a publishable claim would need, so a later stage
    reuses it rather than starting again.
16. **Everything about a run is configurable** (product thinker, 2026-10-16): the person
    declares which features, which builder and reviewer models and how many of each, and
    the repeats, from a single feature with two models up to a full matrix, so the same
    run serves a quick routing check and an automated research loop. Every run also
    records its own limitations and the improvements it suggests, so the next run, or an
    automated research loop, starts from them.
17. **A run with a metered cell needs a spending cap the person named**, with no default
    abcd supplies: a figure in the person's own configuration, or one given for the run,
    which overrides it. The plan shows the expected cost against the cap before consent,
    the run stops when the cap is reached and reports what it finished, and a run of only
    local or host cells needs none. Not asked: decision 16 makes it configurable, and
    the consent screen shows the figure every time, so no money is spent on a number the
    person has not seen.
18. **Any task, judged the way the task allows** (product thinker, 2026-10-16): the run
    applies beyond building code. For writing, such as docs, the product thinker or a
    team of product thinkers judges the copies by hand; for a check such as doc fidelity,
    the copies' overlaps and differences are drawn out side by side, for example to spot
    inconsistencies; for code, hidden tests and reviewer models judge. A human judge is
    told no more about authorship than a reviewer model is: the blind-review decision
    applies to people as it does to models (not asked, as the same grounds hold).
19. **The first delivery covers all three task kinds** (product thinker, 2026-10-16):
    code, doc writing and doc fidelity, with the human judging screen and the side-by-side
    difference view built for it, rather than code first.
20. **Press release confirmed** (product thinker, 2026-10-16): the opening rewritten for
    any task, the table paragraph and the founder's quote, as they stand above.
21. **People rank, and mark what is good enough** (settled by the evidence, not asked):
    each judge ranks the copies drawn for them alone (at most eight, overlapping so every
    pair of models is compared), in an order drawn for them, after calibrating on one
    good and one bad example, judging accuracy against the source before style, and
    marks each copy good enough to ship or not, so the run answers both which is best and
    whether a cheaper model is good enough. Rankings are combined by a Bradley–Terry
    model, never by vote; agreement is reported as Krippendorff's alpha with plain
    percentage agreement; every copy's length is shown beside its rank, because length
    sways people more than any other style signal. Grounds: the model-matrix SOTA note,
    addendum.
22. **Findings are compared, not filed** (settled by the evidence, not asked): each check
    copy's findings are kept in the run's own record, nothing reaches the issue ledger;
    a model is repeated before a finding only it raised counts as unique; matching takes
    exact keys first and a validated matcher for the rest; a judge outside the matrix
    proposes a verdict on each finding and the person decides, as verifier-selects-gates-
    decide holds.
23. **People judge in an abcdesign.app session** (product thinker, 2026-10-16, replacing
    the dashboard answer given minutes earlier): judging is a new kind of abcdesign.app
    session beside brief editing. The person hosts it from their own machine or their
    team's relay, as abcdesign.app's access decision sets out (adr-2610091016270711);
    judges join by its QR or short code from any device, and team members sign in with
    GitHub. This is the first step of abcdesign.app growing beyond brief editing; later
    kinds, such as comparing design mocks two at a time, are captured as
    iss-2610161928115039.
24. **Verdicts are run data, not changes to the project** (not asked: one defensible
    answer): a judging session collects each verdict for the host while it is open, and
    the run reads the session's verdicts into its record on the host's machine as an
    untrusted file it validates, the way abcd reads every agent's answer; abcd opens no
    listener of its own (invariant 7). Nothing goes to GitHub, and a verdict is never a
    suggestion awaiting approval, unlike a brief edit. abcdesign.app's access decision
    names GitHub as its only landing route, so it gains a second rule for judging when
    its session kinds are planned (iss-2610161928115039); the product thinker accepts
    that amendment then. On the relay
    route the relay keeps nothing once the session closes, as abcdesign.app's access
    decision sets out, so the host's record is the only copy. Who may join is the host's
    call under that decision; whether guests admitted by code may judge is a setting of
    each judging session (decision 16), and guests' verdicts are reported apart. A judge
    is identified by their GitHub sign-in, or for a guest by a token issued at joining,
    one ranking per identity, so one person on two devices counts once. The copy-to-model
    mapping never reaches the session or the relay; where the host also judges, abcd
    cannot prove the host did not look, so the host's verdicts are reported apart as
    self-blinded.
25. **The first delivery waits on abcdesign.app's session** (not asked: it follows from
    decisions 19 and 23): judging cannot ship before abcdesign.app runs a session, which
    is planned but not built, and before it has session kinds
    (iss-2610161928115039). The run waits on that intent for its judging step and its
    shipping, not for its other steps, so it carries no blocking edge (one would stop
    every step from building; the missing step-scoped edge is iss-2610161938458308): its spec lists steps that land on their own, code comparison first,
    so the code run is usable before judging lands and the intent ships when all three
    kinds are in.
26. **Statistics have floors** (settled by the evidence, not asked): ratings come from a
    regularised Bradley–Terry fit with intervals bootstrapped over copies and judges,
    and a model that cannot be separated is reported as such; agreement is reported with
    its interval or as not estimable. With fewer than three judges the report gives each
    judge's ranking and the good-enough counts only. A human-judged task is sized over
    judges times copies, not copies alone: each judge ranks a random subset of at most
    eight copies, overlapping so every pair of models is still compared.
27. **A check's comparison is judged in the same session kind, in two phases** (not
    asked: one canonical primitive): first the person judges the findings alone,
    shuffled, with no model named and no proposed verdict; once their verdicts are
    locked, the proposed verdict and the overlap table appear. Recall is measured against
    falsehoods seeded into the frozen material on purpose, as the reviewer copies are,
    and against the findings judged valid, never called true recall.
28. **Each task kind gets its own report row** (not asked): code as above; a written doc:
    its rating, how often it was good enough, cost per good-enough copy and time, plus
    any machine check the task has (the release changelog's citation check runs beside
    the judges); a check: precision, unique valid findings, recall on the seeded falsehoods
    and against the findings judged valid, cost and time. The confirmed press release's table paragraph describes
    the code row; the writing and check rows are their own views of the same report.
29. **Each task kind has its own way to make a copy, with no fallback** (not asked): code
    through the runners, as decision 4 sets out; a self-contained writer, such as the
    release changelog composer, through the provider adapter, once added to the
    self-contained list with the test that list requires; a check that reads files, such
    as doc fidelity, through the runners with a pinned reviewer prompt. A failure on
    either path is the cell's failure, never the host's answer.
30. **A check copy never becomes a release gate's receipt** (not asked): its verdict is
    validated without being recorded where the release gates read, kept under its cell
    and repeat in the run's record, and a test proves the gate's receipts are untouched
    by a run. Positions for matching are computed from the frozen material.
31. **Mechanism confirmed** (product thinker, 2026-10-16) with one change: writing is
    judged by people who have not read it before, in place of the facilitator's wording.
32. **People judging writing do not guess the author** (product thinker, 2026-10-16):
    the guess stays a leak check for reviewer models only. How far people were kept
    blind is therefore unmeasured, and the report lists that among the run's
    limitations.
33. **Signed off** (product thinker, 2026-10-16): the press release, mechanism, scope
    conditions and every acceptance criterion approved; plan with impact additive
    (decision 13).
34. **The cap is never overshot** (the technical facilitator on the product thinker's
    delegation, 2026-10-22, from decision 17): no copy starts unless the cap's remainder
    covers its expected cost, and copies still running when the cap is reached are
    stopped and reported unfinished. The plan says so, because decision 17 spends nothing
    on a figure the person has not seen.
35. **The time estimate waits for the pilot** (the technical facilitator on the product
    thinker's delegation, 2026-10-22, from decision 8): until the pilot has run, the plan
    says the time is known after the pilot, with a bound when the person declares
    minutes per copy.
36. **The expected cost comes from the person's own prices** (the technical facilitator
    on the product thinker's delegation, 2026-10-22, for review on return): the
    declaration may give a price, or an expected cost per copy, for each metered cell,
    and the plan reckons the expected cost from it. A metered cell with no price shows
    "not estimated; bounded by your cap". Prices are never fetched for the plan, because
    decision 12 keeps every network call inside the run, after consent. Decision 17's
    cap stays the real bound either way.

## Open Questions

_None open: the three open at the first pre-pass are settled, 15 and 16 by the product
thinker and 17 on its one defensible answer._

## Typed Links

- **blocked by `itd-166`**: the run's record is that record's asked and served sides.
- **builds on `itd-2609201916151817`** (the build loop): its stages run each copy, in a
  trial mode this intent adds.
- **builds on `itd-2609212137128014`** (labs): the run is a lab procedure, adding a model
  pin to the commit and binary pins.
- **builds on `itd-2609170822093401`** (model routing): its per-invocation route
  override routes each copy and its receipt carries the record (decision 14).
- **refines `itd-2609251624540864`** (the lab discipline): the run is a lab procedure
  under it, with the named reads and clone handling of decision 4.
- **keeps both with `itd-2609221009495079`** (typed judgements through a decision
  adapter, run in shadow as a lab): live shadow agreement on judgements, not repeated
  trials; both share the lab driver and the reporting of counts and uncertainty.
- **keeps both with `itd-2610040822131032`** (a lab run from a product thinker's
  question): this run is the first concrete case that draft generalises, so its driver
  is built once.
- **waits on `itd-2610091016262335` for the judging step only** (abcdesign.app
  sessions): judging is a new session kind beside brief editing, under its access
  decision (decisions 23 to 25); the other steps build without it, and the intent ships
  only when its last spec closes, after that step lands;
  that intent's own prerequisites, such as admitting a live session listener under the
  network decision adr-2610032150581128, stay its own.
- **builds on `itd-2609201916056194`** (command-line runners): the only path for
  OpenRouter and local models into tool-using roles; its live proof is owed in
  iss-2609301519558538.
- **measures what `iss-2610140652230444` asks for**: each review pair's catch rate,
  computed from per-reviewer scores.
- **prerequisites filed as captures**: iss-2610161709506207 (the opencode runner reports
  the asked model), iss-2610161734413178 (OpenRouter pin and served-side readback),
  iss-2610161734428291 (local model pin and probe), iss-2610161801358637 (a runner model
  per invocation).
- **not blocking, consumed when they ship**: itd-2609292107351737 (session cost),
  itd-2609180517121254 (a payload names its model), itd-14 (versioned prompts; until then
  prompts are pinned by hash), itd-2610100704270254 (the sub-agent budget).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: some roles can move to a cheaper or local model without losing quality; shown wrong if no role's comparison favours moving
- pursued: a local model is ready for at least the review role on this kind of work; shown wrong if local reviewers catch clearly less than paid ones
