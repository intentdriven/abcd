---
id: itd-109
slug: a-two-part-verification-suite-for-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Acceptance Criteria Verify Themselves, And The Manual Rest Renders For A Human

## Press Release

> **abcd now runs a record's acceptance criteria as a two-part verification
> suite.** `abcd verify <record-id>` executes every machine-checkable assertion
> a record's criteria carry — guard-checked, toolchain-agnostic, in the
> environment class each assertion declares — and writes a receipt keyed to the
> content it verified. What cannot be automated is not faked: the remaining
> criteria render into a step-by-step page (`--html` builds and opens it) with
> their commands inline, a human walks them, and their verdicts land in the
> same receipt through the same verb. A record's checklist is always a view —
> boxes tick themselves only while the receipt's sha still matches the
> criteria, so an edited criterion loses its tick by construction. Shipping an
> intent, completing a plan, or cutting a release now reads these receipts the
> way the release gate already reads docs-currency: a fail or a pending manual
> verdict refuses the move unless the maintainer records a disposition.
>
> "The § 4 install gate was a hand-written page I had to keep honest by hand,"
> said Alice, a maintainer. "Now the criteria are the page. When I edit one,
> its tick vanishes until something — or someone — proves it again."

## Why This Matters

The Cut A install experience produced the exemplar: a hand-authored § 4
verification gate page (see the install-experience plan, § 4) whose five
scorecard rows restate acceptance criteria that already exist in itd-105 and
the iss-204..208 wave — connected to them only by prose, kept current only by
discipline. That page also proved why the manual half is irreducible: the
harness's parallel hook execution and plugin cache do not exist in CI, so some
assertions are only checkable by a human on a real machine. Meanwhile the
repo's other gates (record-lint, docs-currency, the release receipt gate) are
sha-keyed and deterministic — but the one thing they cannot see is whether
delivered behaviour still matches the criteria that justified shipping it.

Without this intent, every future cut re-invents the § 4 page by hand, every
checklist is a drift risk against its criteria, and a green box has no
provenance. With it, verification is derived — never separately authored — and
a tick is a claim with a sha and a receipt behind it.

## Design Decisions (grilled 2026-08-15)

1. **Criteria are the single source.** The testing script is a rendered view of
   a record's own Given-When-Then acceptance criteria, never a separately
   authored artifact. A test step that fits no criterion is evidence of a
   missing criterion.
2. **Assertions live inline.** An auto-tagged criterion carries a fenced,
   typed assertion block: plain shell, exit-code semantics, optional
   expected-output pattern; `abcd --json` probes preferred where they exist.
   Record-lint: auto without assertion block = blocker; manual with one =
   warning. The format is host-agnostic — assertions in a Python or TypeScript
   repo invoke that repo's own toolchain commands; nothing assumes Go.
3. **Results are receipts, never state on the record.** A run writes a receipt
   (record id, criteria content-sha, per-criterion pass/fail/manual-pending,
   timestamp, sha256 of evidence files). Raw evidence stays in
   `.abcd/.work.local/` and is cited by hash only. Ticked boxes are a
   render-time join of record × latest sha-matching receipt; a criterion edit
   makes its tick vanish (stale, never green).
4. **One verb, action as default.** `abcd verify <record-id>` runs part (a),
   stamps the receipt, and — on a TTY — walks any pending manual criteria
   (pass/fail/skip + evidence path) into the same receipt. `--html` renders
   and opens the generated page (disposable artefact, never committed); other
   formats may follow. Non-TTY runs report pending and never prompt, never
   default (the iss-221 lesson). Ships on both doors: CLI verb and
   `/abcd:verify` plugin page.
5. **Protocol derives from criteria; record types set defaults.** A record
   needs part (b) iff it has ≥1 manual criterion. Defaults as lint
   expectations: issues default all-auto (manual criterion on an issue warns);
   intents mix freely; a plan whose criteria are all auto must state why in
   one line — skipping (b) is a stated decision, never an omission.
6. **Gating.** A current, fully-passing receipt is required to move an intent
   to shipped/, to claim plan completion, and at `launch ship` for every
   shipping intent in the cut. Override only via an explicit disposition
   recorded on the receipt (the iss-192 precedent): shipping over a fail is
   allowed, silently is not. Issue resolve: warn-only, ratchet later if the
   warning proves toothless.
7. **Three environment classes; machine folds into (b).** `repo` (read-only
   probes, runnable anywhere), `scratch` (isolated workspace the runner
   creates; what "workspace" means is a per-repo toolchain config key), and
   `machine` (state that cannot be conjured locally — fresh install, no-Go).
   Machine-class assertions are never executed locally; they render into the
   (b) page with commands inline and return as human verdicts.
8. **Execution is a trust boundary.** Every assertion passes the guard
   registry before running; a refusal is a loud lint finding, not a silent
   skip. No network unless declared. The implementation plan carries a
   mandatory security review before code lands.

## SOTA

Anchors: Cucumber/Gherkin (executable BDD criteria bound to step definitions)
and Go's `testscript` format (command + expected output as one document).
**Declared path: 2 — native floor with a real seam.** Adoption fails the
fit-challenge: a step-definition registry reintroduces exactly the
criteria-to-script drift decision 1 closes, and it would be a new required
dependency. The seam is kept honest: criteria stay strict Given-When-Then
(already linted), the receipt is a documented JSON schema, and the assertion
block borrows `testscript`'s conventions rather than inventing syntax — an
exporter to either anchor stays writable against stable interfaces. The
independent adversarial fit-challenge of this declaration runs at plan time
(evaluator-outside-the-loop).

## Rollout (ratchet, not big-bang)

- **Phase 1 — calibration corpus.** Annotate ~12–15 shipped intents and
  resolved issues with crisp criteria (itd-89, itd-105, the iss-171/204..208
  install wave). Assertions derive from criteria as written, never from
  observed behaviour. Run verify; triage every failure as regression, criteria
  drift, or harness defect — the failures are the product, and this phase is
  itd-109's own acceptance evidence.
- **Phase 2 — forward gating.** New and touched records get annotations at
  authoring time; the decision-6 gates arm.
- **Phase 3 — opportunistic backfill.** Open issues gain annotations on-touch,
  when picked up for fixing — never as a standalone slog.

## Acceptance Criteria

- Given a record whose criteria carry auto assertions, when `abcd verify
  <record-id>` runs, then each repo- and scratch-class assertion executes only
  after a guard-registry pass and a receipt records per-criterion outcomes
  keyed to the criteria content-sha.
- Given a record with at least one manual criterion, when `abcd verify
  <record-id> --html` runs, then a generated page opens showing sha-matched
  auto results ticked, manual criteria with their commands inline, and the
  page artefact is not committed.
- Given pending manual criteria on a TTY, when the default action finishes
  part (a), then the verb walks each pending criterion (pass/fail/skip plus
  evidence path) into the same receipt; given no TTY, then pending criteria
  are reported pending and no prompt is defaulted.
- Given a criterion edited after its receipt was written, when the record
  renders, then that box shows stale/unticked, never a match against the old
  sha.
- Given an intent with a failing or pending receipt and no recorded
  disposition, when a move to shipped/ is attempted, then the move is refused
  and the refusal names the criteria that block it.
- Given a machine-class assertion, when verify runs on any local machine, then
  the assertion is not executed and appears on the rendered (b) page instead.
- Given the phase-1 calibration corpus, when verify runs across it, then every
  record yields a receipt and every failure carries a triage label
  (regression | criteria-drift | harness-defect) before phase 2 arms.

## Open Questions

- Receipt home and naming: a new working-tier ledger (e.g.
  `.abcd/work/verification/`) vs beside specs — naming decision for the plan
  interview.
- Assertion-block syntax details (fence info-string, expected-pattern grammar,
  `net` declaration shape) — borrow from `testscript`, settle at spec time.
- The per-repo `scratch` toolchain config key shape.
- Disposition record schema (who, why, expiry?) shared with the iss-192-style
  release dispositions.
- Whether itd-84 decomposition splits runner, renderer, and gates into
  separate specs under one intent.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
