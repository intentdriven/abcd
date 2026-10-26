---
id: itd-2610240922461281
slug: a-person-sets-how-many-models-review
spec_id: spc-2610240951109420
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609201925079472, itd-2609201916056194]
severity: minor
refines: [iss-2610140652230444]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The review set is a setting, in model classes

## Press Release

> A person sets how many models review each change and which, in vendor-neutral model classes, instead of the loop running a fixed review set. By default two reviewers run side by side on the same commit, the two model classes the first paired reviews used, and the stricter verdict decides; a person can set one instead, or other classes, and a third model, once it has been measured, may strike a finding it rules false when the two disagree. Each vendor's models map onto the classes in configuration, never in the rule. Every review records its findings per reviewer (which found what, where they overlap and where they differ, and whether each was acted on), so the default is chosen from counted catches, not assumed.

## Why This Matters

Today the build loop runs one `ruthless-reviewer` and one `security-reviewer` per round
(`internal/core/implement/loop/validate.go`, the validator roles) and leaves the model to
whoever starts the agent. The pairing the product thinker ruled on 2026-10-14 (a
middle-class and a below-class reviewer on the same copy, the stricter verdict deciding)
therefore lived only in one session's local memory, reaching no other agent or
contributor, and no review recorded which reviewer found what. The seven paired reviews
of 2026-10-16 in iss-2610140652230444 had to be reconstructed by hand from session
reports.

## Mechanism

We expect the per-reviewer record to make the review set a counted choice, because every
round at the default pair writes, for each reviewer, its class, the model asked for and
the model reported where a runner reports one, its verdict, and each finding with whether
the fix round applied or rejected it, so "the class below catches what the middle class
misses" becomes a count of applied findings only one of them cited. Shown wrong if, after
thirty recorded rounds at the default pair, no applied finding was cited by one class
alone, which would mean the second reviewer adds cost and no catch.

## Scope Conditions

- The reviewers are the build loop's two code-review roles, `ruthless-reviewer` and <!-- cond: cond-2610240951106671 -->
  `security-reviewer`. The `intent-auditor` keeps its single run, and a review started
  by hand outside the loop is outside this claim (the loop is the one seam abcd controls;
  the rules-loader route to hand-started reviews is itd-83's).
- A runner-run reviewer is handed its model and reports the model it ran; a host-run <!-- cond: cond-2610240951104584 -->
  reviewer is asked for a class, and the record says the model was asked for, not
  confirmed.
- Two reviewers per role at most, plus one judge, and the judge only once it has cleared <!-- cond: cond-2610240951107153 -->
  itd-81's calibration gate.
- Overlap between two reviewers' findings is matched only where both cite the same file <!-- cond: cond-2610240951105102 -->
  and the same line, the single `file:line` each reviewer's definition asks it to cite;
  two reports of one defect a few lines apart are recorded as each reviewer's alone,
  never matched by meaning.

## Acceptance Criteria

- **Given** no review set in any config layer, **when** a lane reaches validation,
  **then** each reviewer role runs twice, once at the middle class and once at the class
  below it, the stricter verdict decides, and the round's record names for each reviewer
  the class, the model asked for and, where the runner reported one, the model it ran,
  with the bundled layer that set the pair.
- **Given** a review set of one reviewer, **when** a lane reaches validation, **then**
  each role runs once at the class it names, and the record says no comparison was made.
- **Given** a review set of two reviewers at two classes, **when** a lane reaches
  validation, **then** `implement step` hands out each role's two reviewers in
  consecutive steps within the run's sub-agent ceiling, on the same head, with briefs
  that differ only in their return path; each `awaiting` names its reviewer index, its
  class and the model the class resolves to; and each reviewer's return is kept under
  its own path in the round, so neither overwrites the other.
- **Given** a run state written before review sets existed, **when** a newer abcd reads
  it, **then** the state is migrated or refused by its schema version as the loop
  already does for older states, and never read as a review set it was not.
- **Given** two reviewers of one role, **when** the round closes, **then** they disagree
  exactly when one verdict passes and the other fails, by the loop's existing pass or
  fail of each verdict word; two failing verdicts (`BLOCK` and `NEEDS-INPUT`) are
  agreement, and the round fails as any failing validator fails it today.
- **Given** two reviewers who disagree and a judge in the review set, **when** the round
  closes, **then** the stricter verdict still decides, the judge is handed both returns
  and the head, and it may only strike from the fix brief a finding it rules false,
  each struck finding recorded with the judge's reason; it never passes a round a
  reviewer failed, and a round whose every failing finding is struck still goes to a fix
  round with an empty brief rather than passing.
- **Given** a judge in a review set whose agent has not cleared itd-81's calibration
  gate, **when** a run starts, **then** it is refused at the `review` stage naming the
  gate, and with no judge set the round runs as the stricter verdict alone decides.
- **Given** a review set in the repository's config, **when** a run starts, **then** the
  repository's count and classes apply only up to the machine's own ceiling on reviewers
  per role (two when the machine sets none), the class map is read from the machine's
  config or the bundled default alone, and a repository setting above the ceiling is
  clamped with one line on stderr naming both files.
- **Given** a class the class map does not carry, **when** a run starts, **then** it is
  refused at a `review` stage before anything is written, naming the class and the
  map's file.
- **Given** a reviewer's return, **when** it is ingested, **then** each `file:line` under
  its findings is recorded as one located finding of that reviewer; a return with none is
  recorded as zero located findings, not refused; and after the fix round each finding
  carries whether the fix report applied or rejected it.
- **Given** a closed round, **when** `implement record --json` is read, **then** each
  finding carries the reviewer that cited it, and a finding two reviewers cited at the
  same file and line is marked as cited by both.

## Open Questions

_All resolved: the default and the judge's authority by the product thinker on
2026-10-24 (decisions 1 and 7)._

## Decisions

Recorded 2026-10-24 by the facilitator on delegation (the product thinker: "you could
run the planning interview, then plan, etc.; you don't need to wait for me"), after an
adversarial review pair (Opus and Sonnet, design and record discipline, both FIX FIRST,
every finding applied here); the two open questions above are the product thinker's.

1. **Two reviewers by default, the pair of the first tests** (product thinker,
   2026-10-24, asked which strength the one default reviewer runs at, verbatim: "I want
   two reviewers by default: Frontier and middle class (exactly what the tests did, Opus
   & Sonnet, so that we can extend the tests to a full(er) measurement"). This reverses
   the 2026-10-16 addition's one-reviewer default. The models named settle the classes:
   Opus and Sonnet are the middle class and the class below it on the issue's four-class
   scale; the person's word "Frontier" for Opus is the oracle tiers' word for the
   strongest model, a clash of the two vocabularies recorded in decision 2.
2. **Classes beside tiers, read by one map.** The oracle tier (`internal/core/oracle/
   tier.go`) is "a class of model that survives model churn", but its vocabulary
   (`local`, `economy`, `frontier`, `host-decides`) mixes locality, cost and rank, and
   has no name for the middle class the rulings turn on. The four classes of the
   2026-10-14 ruling (top, middle, the class below, smallest) are one machine-layer class
   map, the same one itd-2610100704270254 reads for its sub-agent count, so there is one
   class vocabulary, not a third reading of tiers. The product thinker's own answer called
   Opus "Frontier": the class names the person sees must say which model each holds
   today wherever a class is shown, so the two vocabularies cannot be confused unseen. For the two reviewer roles a
   configured review set's class wins over the role's oracle row; without a review set
   the existing routing applies unchanged.
3. **The class map is the machine's.** Which model spends the person's key is theirs
   alone, as RN2 and ruling AA(b) keep a runner's route on the machine
   (`internal/core/runner/config.go`); a repository may name a count and classes, capped
   by the machine's ceiling. The bundled map is written as admitted
   `<provider>/<model>` routes for one vendor, a default and not part of the rule.
4. **A runner takes a model per request.** `runner.Request` carries no model today; one
   runner serving two reviewers at two classes needs a per-request model, admitted
   against the provider's allowlist as a route is. This is part of the work.
5. **Out loud, out of this intent.** Captured as iss-2610240946046901, not dropped: the
   three other top-class consult cases of 2026-10-14 (a third
   read of release-blocking security changes, a design-heavy plan before it locks, an
   executor failing twice alike) and the ruling that code is written by the middle
   class. Each is a seam outside the review round.
6. **The auditor stays single.** The rulings speak of reviews, "security or code"; the
   audit is a criterion-by-criterion verdict whose fan-out the matrix intent measures.
7. **The judge strikes false findings only** (product thinker, 2026-10-24, asked what the
   judge may do when two reviewers disagree: "Strike false findings only"). The stricter
   verdict still decides; the judge never passes a round a reviewer failed, and runs
   only once calibrated under itd-81.
8. **Pre-pass overlaps, each with one defensible answer, recorded not asked:** keep both
   with itd-2610100704270254 (one class map, two readers), with itd-2610161733316518
   (live rounds here, controlled repeats there), and with itd-83 (the rules loader there,
   the validate stage here); the judge refines itd-81, inheriting its calibration gate.

## Typed Links

- **refines `itd-2609170822093401`** (model routing): the class map sits beside the
  oracle rows and wins only for a configured review set.
- **refines `itd-81`** (judge calibration): the judge inherits its gate.
- **keeps both with `itd-2610100704270254`**, **`itd-2610161733316518`** and **`itd-83`**.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the product thinker, 2026-10-24: two reviewers by default, the pair the first tests used, "so that we can extend the tests to a full(er) measurement"; shown wrong if thirty recorded default-pair rounds show no applied finding cited by one class alone
