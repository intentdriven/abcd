---
id: spc-2610161949559050
slug: a-product-thinker-who-wants-to-know
intent: itd-2610161733316518
origin: researcher-authored
production_mode: dictated-and-formatted
---
# a-product-thinker-who-wants-to-know

## Summary

Delivers itd-2610161733316518: one task (a code feature, a written doc, or a check such
as doc fidelity) done many times across a declared matrix of models, from one frozen
commit, judged the way the task allows, merged nowhere, and reported with a record from
which the run can be started again and its drift shown. The design record is the
intent's Decisions 1–33; this spec says how they are built.

**Shape.** A new core package, `internal/core/matrix`, owns the run: the matrix
declaration, the plan and consent, the cells, the trial records, the judging ingest and
the report. It drives existing pieces rather than copying them:

- **Run as a lab procedure** (decision 4): `internal/core/lab` pins the frozen commit and
  the binary digest; the matrix adds a model pin per cell. Each copy is an isolated clone
  of the frozen commit in the machine-scoped store, keyed intent/cell/repeat, bundled and
  deleted once scored.
- **Trial mode of the build loop** (decision 4): `internal/core/implement/loop` gains a
  mode that takes no reservation, lands nothing, and switches off host fallback inside a
  cell; a runner failure is the cell's failure.
- **Routing per copy** (decision 14): the existing per-invocation route override, extended
  to name a runner and its model (iss-2610161801358637), on the person's own layer only.
- **Copy production per task kind** (decision 29): code through the runners; a
  self-contained writer through the provider adapter once on the self-contained list;
  a file-reading check through the runners with a pinned reviewer prompt. Fallback off on
  both paths.
- **Recording** (itd-166, its blocker): every call's asked and served sides, unpinnable
  fields marked; the run record names the frozen commit, binary digest and version,
  prompt and tool-definition hashes, the render template, machine, concurrency and load
  cap. The trial record maps onto a Harbor trial result and an Inspect sample (the seam
  for path 2 of sota-per-intent).
- **Judging** (decisions 21–24, 26–27, 32; adr-2610161733400551):
  - code: hidden tests outside the clone, reviewer models as self-contained blinded
    requests assembled through the reading surface's include table and hashed manifest,
    scored on known-good and seeded-defect copies;
  - writing: an abcdesign.app judging session (itd-2610091016262335, session kinds
    iss-2610161928115039); calibration, ranked subsets of at most eight, good-enough
    marks, no author guess; verdicts read back as an untrusted file the run validates;
  - checks: findings kept per copy (never `docfidelity.Record`, never the ledger), matched
    by exact key then a validated matcher, judged in two phases in the same session kind.
- **Statistics** (decisions 8, 26): pilot then power-sized repeats; pass@1 with interval
  and pass^k; regularised Bradley–Terry with bootstrapped intervals; α with interval or
  "not estimable"; floors below three judges.
- **Safety and load** (decisions 12, 17; technical criteria): consent screen with cost
  against a named cap for metered cells; secret scan of each provider-bound clone; one
  local inference server at a time as an owned process group under a consented core cap;
  every network call inside the consented run; copies rendered inert.
- **Report** (decisions 16, 28): one row set per task kind, the limitations and suggested
  improvements of the run, and a re-run that reports drift field by field.

**How each acceptance criterion is met.** Person-visible: (1) plan and consent — step 2;
(2) code report — steps 4 and 8; (3) writing judged — step 6; (4) check compared — step 7;
(5) power floor — step 8; (6) re-run with drift — step 8; (7) nothing merged or filed —
steps 3, 5, 7; (8) limitations — step 8. Technical: no host fallback, same-pin retry,
isolated clones, secret scan, compliance check — steps 3–4; blinded reviewers and the
author-guess for models — step 5; matching findings — step 7; recording and run record —
step 1 with itd-166; one local server — step 3; inert rendering and validated verdicts —
step 6; gate receipts untouched — step 7; Harbor/Inspect mapping — step 1.

## Footprint

- packages: internal/core/matrix/doc.go, internal/core/matrix/declaration.go, internal/core/matrix/run.go, internal/core/matrix/trial.go, internal/core/matrix/export.go, internal/core/matrix/declaration_test.go, internal/core/matrix/run_test.go, internal/core/matrix/trial_test.go, internal/core/matrix/export_test.go, internal/core/matrix/testdata/inspect-eval-sample.json, internal/core/matrix/testdata/harbor-trial-result.json, internal/core/lab/procedure.go, internal/core/lab/procedure_test.go, internal/core/lab/preflight.go, internal/README.md, ACKNOWLEDGEMENTS.md, internal/core/oracle/store.go, internal/core/matrix/plan.go, internal/core/matrix/plan_test.go, internal/core/matrix/cap.go, internal/core/matrix/cap_test.go, internal/core/matrix/consent.go, internal/core/matrix/consent_test.go, internal/core/matrix/testdata/declaration-plan.json, internal/surface/cli/lab_matrix.go, internal/surface/cli/lab_matrix_test.go, internal/surface/cli/lab.go, internal/surface/cli/lab_surface_test.go, internal/core/surface/examples.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
- tests: the matrix plan refuses to start without consent and a named cap for metered cells; a failing runner cell is scored failed with no host fallback; trial mode takes no reservation and lands nothing; a provider-bound clone with a secret-shaped file is refused; reviewer requests carry no authorship field; a malformed verdict file is refused with nothing recorded; a check run leaves the doc-fidelity receipts untouched; Bradley–Terry and α floors; the trial record round-trips through a Harbor and an Inspect fixture
- The packages line names step 1's files. Each later step's files join it
  when that step begins, and `abcd build` then reserves the widened
  footprint (dcn-2610230129331232). Generated output is reserved by no one.

## Steps

1. The run record and the trial seam
   - packages: internal/core/matrix, internal/core/lab
   - tests: the matrix declaration parses and refuses unknown keys; the run record names commit, binary digest, hashes, template, machine, concurrency and cap; the trial record maps onto a Harbor trial result and an Inspect sample fixture
   - waits on: itd-166 (the asked and served sides)
2. Plan, cost and consent
   - packages: internal/core/matrix, internal/surface/cli, commands
   - tests: the plan shows copies, cost by source against the cap, load cap, time and judges; nothing starts without consent; a metered run without a named cap is refused; a local- or host-only run needs no cap; a matrix refusal printed by the front door carries a control or direction character escaped, never raw; a scanner kind naming a repository's own rule id is printed sanitised; the role echoed in a pin refusal is quoted
   - The front door passes every matrix error through the terminal sanitiser
     before printing it, because the core returns raw text. The run record's
     refusal prints the scanner's finding kind sanitised, since an opted-in
     augmenter's kind carries the repository's own rule id.
   - When a trial writer is added, it calls the run record's screen on every
     trial before writing it, so a trial's reason and score detail are
     screened for credentials as the run record is.
   - The plan's cost and time follow decisions 34-36: no copy starts past the
     cap's remainder, the time waits for the pilot, and the expected cost
     comes only from prices the person declares.
3. Trial mode, isolated clones and routing
   - packages: internal/core/implement/loop, internal/core/runner, internal/core/launch, internal/core/matrix
   - tests: trial mode takes no reservation and lands nothing; a runner failure in a cell is the cell's failure with no host fallback; an OpenRouter outage is retried on the same pin and not scored; each copy is a clone outside every working tree with no local tier; a provider-bound clone passes the secret scan or is refused; one local server at a time as an owned process group
   - waits on: iss-2610161801358637, iss-2610161734413178, iss-2610161734428291, iss-2610161709506207
4. Code tasks: hidden tests and compliance
   - packages: internal/core/matrix
   - tests: hidden tests sit outside the clone; the compliance check excludes a copy that edited tests or read another copy, and marks an unexaminable copy unexamined; host-harness isolation is reported as asserted
5. Blinded reviewer models
   - packages: internal/core/matrix, internal/core/reading, internal/core/oracle
   - tests: a reviewer request is assembled through the include table with no trailer, author, branch or run metadata; recall and true-negative rate on known-good and seeded-defect copies; the author-guess score; same-family verdicts reported apart
6. Writing tasks judged in an abcdesign.app session
   - packages: internal/core/matrix, internal/core/oracle
   - tests: a judge's subset holds at most eight copies and every pair of models is compared; copies render as inert text; a verdict file is validated and refused whole when malformed; one ranking per judge identity; guest verdicts and the host's self-blinded verdicts reported apart
   - waits on: itd-2610091016262335 and iss-2610161928115039 (the session and its kinds)
7. Checks compared in two phases
   - packages: internal/core/matrix, internal/core/docfidelity, internal/core/record/findingkey
   - tests: a check copy's findings are read before its clone is deleted; the doc-fidelity receipts and the issue ledger are unchanged by a run; exact-key matching first, the matcher's agreement on a labelled sample reported; the overlap table appears only after the person's verdicts are locked; recall on seeded falsehoods
   - waits on: itd-2610091016262335 and iss-2610161928115039
8. Statistics, report and re-run
   - packages: internal/core/matrix, internal/surface/cli
   - tests: pass@1 interval and pass^k; regularised Bradley–Terry with bootstrapped intervals and "not separable"; α with interval or "not estimable"; floors below three judges; a comparison under its power estimate states the smallest detectable difference; the report lists limitations and suggested improvements; a re-run reports drift field by field
