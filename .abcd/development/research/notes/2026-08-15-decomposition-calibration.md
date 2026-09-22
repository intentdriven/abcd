# Decomposition calibration — graded hand-runs of the itd-84 protocol

The calibration corpus for the
[itd-84](../../intents/disciplines/itd-84-intent-decomposition.md) hand-run
decomposition protocol (the four-piece table in the `/abcd:intent` surface
page). Every hand-run appends one graded entry; roughly **50 graded captures**
is the recorded threshold before the automated capture-time validator may be
built, and the grading follows the
[itd-81](../../intents/disciplines/itd-81-judge-calibration.md) recipe: the
initial routing is the prediction, the human-confirmed routing is the label.

## Entry format

Per hand-run, append:

- **Date, proposal** (one line), and the session context.
- **Initial routing table** — part | type | home, plus typed links and any
  reversal flags, exactly as proposed before the human ruled.
- **Confirmed routing** — what the human adopted, with edits marked.
- **Verdict** — FILE-AS-IS / SPLIT / HOLD, and whether the initial verdict
  survived confirmation (the graded outcome).
- **Notes** — over-flags, missed parts, taxonomy ambiguities (feeds the
  open-question on the enum).

## Graded entries

### 2026-07-13 — the auto-merge proposal (founding case, graded retrospectively)

- **Proposal:** a single "`--auto-merge` feature" intent.
- **Initial routing (as filed at the time):** one part | intent | intents/ —
  a monolith.
- **Confirmed routing (from the 2026-07-13 review):** four parts — the
  auto-merge *experience* | intent | intents/; the *trust rule* on what may
  merge unattended | ADR + brief invariant | decisions/adrs/ + brief; the
  additive-vs-editing *stance* | principle | principles/; the eligibility
  *plumbing* | brief | brief.
- **Verdict:** SPLIT. The initial (implicit FILE-AS-IS) routing did **not**
  survive review — the case that motivated itd-84.
- **Notes:** graded from the review record, not a live protocol run; counts
  toward the corpus as the canonical SPLIT exemplar.

### 2026-08-15 — itd-111 planning interview (first live hand-run)

- **Proposal:** itd-111 (staleness detection), an already-filed draft at its
  planning interview — the decomposition ran over the draft before promotion.
- **Initial routing:** five parts — staleness detection + refusal
  (capability | this intent); the network trichotomy (trust rule | ADR +
  brief invariant — flagged as system-binding: "no version-discovery request
  exists anywhere in abcd" outlives the feature); anti-wallpaper micro-prompt
  (capability seed | already extracted to iss-230); vintage-comparison seam
  (plumbing | brief via the spec); platform parity (verification | itd-109
  calibration set, `refines`). One advisory reversal flag: the SessionStart
  staleness notice vs the iss-206 skew-notice retirement (install-experience
  decision 7).
- **Confirmed routing:** adopted unchanged. The reversal flag was ruled a
  **scoped replacement** (`refines`, not `reverses`): steady-state machinery
  stays retired; itd-111 covers the non-steady states.
- **Verdict:** SPLIT (network posture → adr-38 + brief invariant 7),
  proposed and confirmed — the initial verdict survived confirmation.
- **Notes:** no over-flags; the one reversal candidate was genuinely
  ambiguous and worth the human ruling (advisory-only behaved as designed).
  Taxonomy: "trust rule → ADR + brief invariant" fit cleanly; no enum
  ambiguity encountered.

### 2026-08-16 — the multi-harness proposal (hand-run at user confirmation)

- **Proposal:** "make abcd available on further harnesses, as native as
  possible, mapping concepts where the host has them, MCP as the fallback,
  never double-writing per host" — plus a host-tier statement (MCP floor for
  any harness; the current plugin host as the assumed SOTA surface for the
  time being; an open-source harness as the eventual default).
- **Initial routing:** four parts — the host-tier policy incl. the
  default-flip gate (trust/strategy rule | ADR | decisions/adrs/); the shared
  adaptor machinery — host profile seam, ladder semantics, parity suite
  (capability | reworked itd-22, renamed `harness-portability`); the MCP
  front door (capability, the floor | new intent); per-host adoptions incl.
  concept mapping (capability | one intent per host, DEFERRED to explicit
  adoption decisions, nameless until then). Typed links: the 2026-08-15
  DECISIONS entry `reverses` the out-of-scope annotation on itd-22 (human
  had already confirmed that reversal); adr-39 `refines` that decision;
  itd-22 rework `supersedes` its own prior single-host framing (id kept).
- **Confirmed routing:** adopted structurally unchanged; two content edits at
  confirmation — the reference-host rationale reworded to "assumed SOTA
  surface for the time being", and the routing's own justification prose
  ordered kept OUT of the record.
- **Verdict:** SPLIT, proposed and confirmed — the initial verdict survived.
- **Notes:** the stance piece ("never double-write per host") routed into the
  ADR rather than a standalone principle — it restates one-canonical-primitive
  at the host boundary, so a new principle file would have been a second copy;
  flagged here for the ~50-capture calibration review. Rename executed with a
  retire-the-name ban (`retired-itd-22-slug`).

### 2026-08-16 — itd-114 collision-proof ids (hand-run at capture)

- **Proposal:** abcd mints collision-proof record ids across parallel agents —
  native time+content-hash default, optional GitHub forge backend for capture.
- **Initial routing:** four parts — the collision-proof minting capability
  (capability | intent itd-114); the native-scheme + forge-adapter seam (SOTA
  declaration path-2 | inside the intent's SOTA section); the
  "collision-proof-by-construction, native-default forge-optional" stance
  (stance | embodies existing basics-built-in / adapter-over-native-default /
  prefer-sota principles — no new principle); and the id-FORMAT change
  (architecture decision | a future ADR, only if adopted). One advisory reversal
  flag: a pure time+hash id reverses the human-sequential-readable property of
  today's iss-N.
- **Confirmed routing:** adopted (author-confirmed at capture). No separate
  principle or ADR filed now — both are conditional-on-adoption. Reversal flag
  handled by writing an acceptance criterion that FORBIDS the readability
  regression, leaving the exact scheme (time-sortable / git-style dual id /
  forge number) to the grill.
- **Verdict:** FILE-AS-IS with flags (not SPLIT — nothing separate to file yet).
  Initial verdict survived.
- **Notes:** typed link `refines` iss-80 (the resolved issue whose body deferred
  this exact "SOTA under research" scheme). The reversal flag was the valuable
  part — it forced the readability requirement into the ACs rather than letting
  a hash-only scheme ship. Filed while the peer was concurrently minting iss-N
  in a parallel session: a live instance of the collision class the intent
  addresses (no actual collision — different family).

### 2026-08-16 — itd-115 managed-repo merge policy (hand-run at capture)

- **Proposal:** abcd-managed repos merge PRs without the out-of-sync churn —
  merge queue by default, protocol-level auto-update fallback, strict preserved.
- **Initial routing:** four parts — the merge policy capability (intent
  itd-115); the adopt-merge-queue SOTA + merge_group CI trigger + rung-1 fallback
  (SOTA path-1 declaration | inside the intent); the "strict is the duplicate-id
  gate, never relaxed until ids are collision-proof" trust rule (already recorded
  as iss-172's invariant — RESPECT, do not re-declare); the ruleset/CI-trigger
  wiring (plumbing | itd-92/itd-106 onboarding surface).
- **Confirmed routing:** adopted. The trust-rule part is the interesting one — it
  is ALREADY a recorded invariant (iss-172), so the decomposition's job was to
  route the intent to RESPECT an existing record rather than file a new one. The
  first-pass design (relax strict) would have reversed that invariant; reading
  iss-172 before filing caught it and the design was corrected — the value of the
  decomposition's interdependency pass.
- **Verdict:** FILE-AS-IS with flags. NO reversal flag (the corrected design
  respects iss-172; the rejected relax-strict alternative would have reversed it).
  Initial verdict survived after the correction.
- **Notes:** typed links refines iss-172 / itd-92 / itd-106 / itd-114 (itd-114
  unlocks the future relax-strict option). A clean case of the interdependency
  pass preventing a record-contradicting design.

### 2026-08-16 — itd-119 capture promote (hand-run at planning, process plan intent 1)

- **Proposal:** `abcd capture promote <iss-N>` — an issue graduates into an
  intent without retyping; `promoted_to` stamped in the same invocation (the
  first walkability intent of the process-coherence plan).
- **Initial routing:** four parts — the promote capability (capability | this
  intent); the "capture now, decide later" stance (already embodied in the
  Which-ledger note + `PromotedTo` schema — cite, file nothing); the docs
  corrections (capture.md / 04-naming.md / 04-surfaces | ride the sweep); the
  intent-side reciprocal edge field (spec detail | spec body). Typed link:
  `refines` iss-245 (its `promoted_to` half). No reversal flags.
- **Confirmed routing:** adopted structurally unchanged. Two AC-level edits at
  the walk/grill: the proposal's only-open-issues restriction was struck
  (promotion ruled orthogonal to fix-status — the maintainer's challenge), and
  the seed moved from body-copy to a by-id pointer + first line (SSOT) after
  the maintainer floated the link alternative mid-grill.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** the interesting failure was an *over-restriction*, not an
  over-flag: the proposed AC invented a status gate the schema never implied.
  Grill surfaced a real unimplementability (two-store atomicity) that became
  the mint-first + `--intent` repair contract. Filed as itd-119/spc-24.

### 2026-08-16 — itd-120 resolve provenance (hand-run at planning, process plan intent 2)

- **Proposal:** `abcd capture resolve` writes `resolved_by` via `--intent` /
  `--spec` / `--commit` (the second walkability intent of the
  process-coherence plan).
- **Initial routing:** four parts — the provenance-write capability
  (capability | this intent); validation depth (spec detail | spec body);
  docs (capture.md + issues README | ride the sweep); the two-evidence-
  standards observation (motivation | press-release prose, no new record).
  Typed link: `refines` iss-245 (its `resolved_by` half). No reversal flags.
- **Confirmed routing:** adopted unchanged. Rulings at the walk: provenance
  optional (never defaulted, never demanded); existence-checked ids,
  shape-checked sha; backfill of already-resolved issues ruled OUT of scope.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** clean run; the only near-part was the backfill idea, which the
  grill surfaced and the maintainer scoped out rather than routed. Filed as
  itd-120/spc-25.

### 2026-08-16 — itd-121 record-id dispatch (hand-run at planning, process plan intent 3)

- **Proposal:** `abcd <id>` — dispatch on a record id, report what it is and
  the next move (the third walkability intent of the process-coherence plan).
- **Initial routing:** four parts — the dispatch capability (capability |
  this intent); the next-move mapping (spec detail | spec body, one Go
  table); the bare-vs-id mental model (already recorded in the plan — cite);
  docs (root surface page + 04-surfaces | ride the sweep). Duplicate check
  vs itd-86 (cold reading) and itd-112 (banner): negative. No reversal
  flags; SD001-safe by construction.
- **Confirmed routing:** adopted unchanged. Rulings at the walk: `adr-N`
  joins the dispatch read-only (a scope widening the proposal had left out);
  the next-move mapping gains an anti-drift test asserting every recommended
  verb resolves in the cobra tree.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** the maintainer widened scope (adr-N) where the proposal had
  narrowed to the plan's three families — the second time this session the
  human's edit was toward *less* restriction than proposed. Filed as
  itd-121/spc-26.

### 2026-08-16 — itd-122 sub-verb tables (hand-run at planning, process plan intent 4)

- **Proposal:** sub-verb tables in every `04-surfaces/` file plus an extended
  `surface_coverage` checking each row against registered sub-commands both
  ways (the coherence keystone of the process plan; adr-40 decision 6).
- **Initial routing:** five parts — the tables + extended check (capability |
  this intent); the population pass (same change, per adr-40's consequence);
  the four-bucket vocabulary (already adr-40 — cite); the bucket-enum
  registration in 04-naming.md (rides the sweep, VR001); the no-`partial`
  ruling (recorded in draft prose). Typed link: `refines` iss-246. No
  reversal flags — implements a recorded decision.
- **Confirmed routing:** adopted unchanged. Rulings at the walk/grill:
  exemptions are explicit config like `bare_command` (host-delegated
  surfaces, operator-internal verbs); three arguable bucketings pre-ruled
  binding (identity = audit, launch changelog guardrail = gate, guard check
  = gate) so the unattended build never faces the ambiguity STOP on a known
  case.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** the grill's value was schedule-shaped: pre-ruling the arguable
  buckets moves a predictable overnight STOP into the human session. The
  committed `surface.json` snapshot dissolved the expected layering problem
  (core/lint never imports the cobra tree). Filed as itd-122/spc-27.

### 2026-08-16 — itd-123 intent-audit rename (hand-run at planning, process plan intent 5)

- **Proposal:** `abcd intent review` → `abcd intent audit`, breaking, with
  the agent and task-class token moving (adr-40's first named rename).
- **Initial routing:** four parts — the rename sweep (capability, breaking |
  this intent); the vocabulary decision (already adr-40 — cite); the
  sub-verb table row flip (same change, gated by itd-122); the no-aliases
  stance (settled — cite, not reopened). No reversal flags — implements a
  recorded decision.
- **Confirmed routing:** adopted unchanged. Rulings at the walk: the agent
  becomes `intent-auditor` (maintainer's counter-proposal over the proposed
  `intent-fidelity-auditor` — the intent-grain auditor of the three audit
  grains, mirroring the verb); the sweep boundary excludes historical/dated
  records, which keep the old name as history.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived; one naming edit at confirmation.
- **Notes:** the live sweep enumerates to 23 files (the plan's ~37 counted
  historical records the boundary ruling excludes). Filed as itd-123/spc-28.

### 2026-08-16 — itd-124 audit→lint rename (hand-run at planning, process plan intent 6)

- **Proposal:** `abcd audit` → `abcd lint`, breaking; `/abcd:audit` returns
  to itd-16's reservation (adr-40's second named rename; name ruled in
  planning question 1).
- **Initial routing:** four parts — the rename sweep (capability, breaking |
  this intent); the reservation return (already recorded in 04-naming.md —
  the sweep reinstates it); the chosen name (ruled today — recorded in
  draft); no-aliases (settled — cite). No reversal flags.
- **Confirmed routing:** adopted unchanged. The grill's package question
  (`repolint` vs merging the two lint engines) went to adversarial review at
  the maintainer's direction: rename-only won — the engines have different
  rule models and blast radii, and adr-40 rules implementation orthogonal to
  bucket. The merge is captured as iss-251, a deliberate future intent.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** the maintainer's "adversarially review both" instruction is a
  useful grill pattern: the review moved the ruling from taste to facts
  (importer counts, existing cross-import, contract differences). Filed as
  itd-124/spc-29.

### 2026-08-16 — itd-125 disembark-review rename (hand-run at planning, process plan intent 7)

- **Proposal:** `disembark oracle` → `disembark review`, breaking — the
  seventh intent, created by the session's own investigation of planning
  question 2 (the plan had recommended keep-the-verb).
- **Initial routing:** three parts — the rename sweep incl. artefacts and
  agent (capability, breaking | this intent); the adr-40 §5 amendment
  (decision | home ruled by the maintainer: dated in-place amendment in
  adr-40 + a DECISIONS.md line); the oracle seam itself (adr-25 — cite,
  untouched). **One reversal flag, the session's first genuine one:**
  reverses adr-40 §5, investigated and confirmed by the maintainer before
  routing.
- **Confirmed routing:** adopted unchanged. Rulings: agent becomes
  `lifeboat-reviewer` and artefacts `review/review-<manifest12>.*`
  (target-grain + role, self-describing basenames); older lifeboats get
  clean replacement across the rename (the grill's AC — one manifest never
  holds two verdicts).
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived.
- **Notes:** the reversal path worked as itd-84 designed it: flagged
  advisory, investigated against the code (the verb never invokes the seam
  — compute-or-ingest only), ruled by the human, homed before filing. Filed
  as itd-125/spc-30.

### 2026-08-16 — itd-116 GitHub-issue adoption into the ledger (hand-run at capture)

- **Proposal:** a verb that selects validated GitHub issues (the bughunt's
  filings), reviews them, and captures them into the ledger — "bug-adoption
  owner" made a tool.
- **Initial routing:** four parts — the adoption capability (intent itd-116);
  the host-delegated select/review + fail-closed ingest trust rule (already
  adr-25 + the core boundary — spec constraint, no new ADR); the
  human-adoption-gate stance (already `verifier-selects-gates-decide` — cite,
  don't re-file); provenance/dedupe plumbing (spec detail; consumes itd-87
  later). Duplicate check against itd-82 (drain) came back negative: opposite
  pipeline directions (GitHub→ledger vs ledger→PRs).
- **Confirmed routing:** adopted, with one live design exchange at capture —
  the author floated `drain --github-issues` as an alternative surface; set
  aside (drain is an unbuilt draft already stubbing itd-46; different blast
  radius) in favour of a capture extension, recorded in the draft as
  considered-and-rejected. Mint stays capture-only (one-canonical-primitive).
- **Verdict:** FILE-AS-IS (no reversal flags). Initial verdict survived; the
  surface-spelling exchange narrowed open question 1 rather than changing the
  routing.
- **Notes:** typed links builds_on itd-4; refines the DECISIONS.md
  bughunt-hybrid line ("adoption is a downstream human/fix step"); prose
  cross-refs itd-82 (downstream consumer) and itd-87 (recurrence semantics).
  Filed the same day the bughunt's first validated issue (#250) landed —
  itself the first adoption candidate.

### 2026-08-16 — itd-76 sources corpus (hand-run at planning)

- **Proposal:** the existing itd-76 draft — personal corpus + provenance
  ledger + guards, team share/ingest, and paper reconstruction in one
  intent, with three open questions and no acceptance criteria.
- **Initial routing:** six parts — personal core verbs
  (capability | itd-76, narrowed); team share/ingest (capability | new
  draft); paper reconstruction (capability | new draft); "documents and
  ledgers never travel; citation requires both gates" (trust rule | ADR +
  brief invariant); consult-freely-cite-deliberately (stance | principle);
  store formats and folder-as-classification (plumbing | brief/spec).
  Typed links: the split drafts `refines` itd-76; itd-76 `builds_on`
  itd-74 (shipped) and itd-77 (draft, non-gating — default path works).
  No reversal flags.
- **Confirmed routing:** adopted unchanged, plus three interview rulings the
  proposal had surfaced as open questions: the share surface is the existing
  `references.csl.json` research store, not a second `.abcd/work` exchange
  file (one committed bibliography); the "Dogfood (already running)" claim
  was stale and is rewritten as target (no corpus exists on the development
  machine); multi-machine ledger ownership is explicitly deferred. Seven
  Given-When-Then criteria authored and accepted in full; persona corrected
  Maya→Alice per the intents discipline.
- **Verdict:** SPLIT, proposed and confirmed — the initial verdict survived.
- **Notes:** filed as itd-76/spc-31 (planned), itd-126 and itd-127 (drafts,
  seeded criteria marked unconfirmed), adr-41 (proposed), brief invariant 9,
  and the consult-freely-cite-deliberately principle. Both prior share/ingest
  open questions moved to itd-126 with a third (share vs the store's
  admission criteria) added at the walk.

### 2026-08-19 — the itd-92 capability-ladder extension — EXCLUDED from the corpus

- **Excluded from the calibration count:** the human directed the home before
  any table ran ("draft the extension to itd-92"), so there is no blind
  prediction to grade — the label preceded the prediction. Recorded for the
  protocol's history, not as a sample toward the ~50-capture threshold.
- **Proposal:** bring the collaborator fence (built by hand on this repo,
  2026-08-19) to abcd-managed repos gracefully for users without GitHub or
  an organisation — tiers, per-piece verdicts, loud degradation.
- **Initial routing (as first filed):** four parts — doctor + apply + gate
  (capability | itd-92 in place); never-mutate-a-remote-uninvited (stance |
  claimed as carried by loud-staging); verdict mechanics (plumbing | intent
  body); evidence (research | the field note). Two adversarial reviews the
  same day refuted the routing: loud-staging does not carry the
  remote-mutation rule (it is a trust rule, homeless as filed), and a second
  trust rule — identity from caller-local facts, already binding the shipped
  launch scanner (iss-283) — was never routed at all.
- **Corrected routing:** read-only doctor + drift + tier report (capability |
  itd-92, extended in place); apply-on-request (capability | future intent,
  named in itd-92's out-of-scope); both trust rules (ADR |
  [adr-44](../../decisions/adrs/0044-remote-mutation-and-caller-identity-trust-rules.md),
  proposed, brief invariant at adoption); gate refusal policy (deferred to
  its own decision record after doctor field experience — out of itd-92's
  press release entirely); verdict mechanics (plumbing | spec at planning);
  evidence (research | the field note). Record relations: itd-92 will close
  iss-277 when shipped; iss-281 and iss-282 remain open gaps the fence
  tracks; iss-283 is resolved and serves as evidence only. No reversal flags.
- **Verdict:** SPLIT (capability kept in itd-92; trust rules to adr-44; apply
  and gate descoped) — reached only after adversarial review overturned the
  initial FILE-AS-IS. Not a graded sample (see exclusion above), but the
  overturn itself is calibration signal: a pre-directed routing survived
  neither reviewer.
- **Notes:** acceptance criteria rebuilt against what a read-only probe can
  decide per caller; the draft stays in `drafts/` for the planning interview.

### 2026-08-19 — one canonical YAML scalar resolver (itd-128)

- **Proposal:** from the PR #294 commissioned review's F6 — the fix widened
  two of four independent YAML scalar decoders; extract one exported
  resolution helper and have every decoder delegate.
- **Initial routing (proposed before the human ruled):** capability (the
  exported resolver, all four decoders delegating) | intent | itd-128;
  trust rule (ledger-canonical issue store, per-field ownership — decided the
  same session but a distinct part) | decision log now, ADR at graduation;
  stance (fix the class, not the instance) | already carried by the existing
  one-canonical-primitive principle, no new filing; plumbing (where quoting
  normalisation lives, whether the dump-path null list folds in) | intent
  open questions, settled at planning. Typed links: itd-128 closes iss-285;
  iss-287 is evidence. No reversal flags.
- **Confirmed routing:** adopted unchanged (human confirmation 2026-08-19,
  after filing — the prediction preceded the label).
- **Verdict:** FILE-AS-IS — survived confirmation unedited.
- **Notes:** the stance slot resolving to an *existing* principle (no
  artefact filed) is a case the four-piece table handles but the entry format
  under-describes: "home" here is a citation, not a filing. Worth a line in
  the protocol page when the enum question is next revisited.

### 2026-08-20 — `abcd update` in one verb (itd-130)

- **Proposal:** from the update-mechanism investigation — automatic updates
  for abcd across its two install shapes (plugin, standalone CLI), covering
  the self-update verb, plugin-side delivery, Homebrew, and the bootstrap
  script's future.
- **Initial routing (proposed before the human ruled):** capability (the
  `abcd update` verb: fetch/verify/swap, dispatch, refusals, progress) |
  intent | itd-130; trust rules ("never touches a plugin-root binary",
  "never ambient") | citations of existing records, not filings — adr-38
  tier 3 and itd-108's one-cut coherence carry both, no new ADR; stance
  (notify-only-plus-explicit-verb over silent auto-update) | already carried
  by adr-38, no new principle; decision (Homebrew parked until the verb
  exists; install-channel refusal ships with the verb) | dated DECISIONS.md
  entry 2026-08-20; plumbing (bootstrap.sh demoted to cold-start trampoline)
  | itd-130 open question (sequencing + delegation hardening), brief
  internals when it lands; defect discovered en route (stranded PATH symlink
  classifies foreign) | iss-345, fixed test-first the same day, independent
  of the verb. Typed links: itd-130 builds_on itd-105, itd-108. No reversal
  flags.
- **Confirmed routing:** adopted unchanged (human confirmation 2026-08-20,
  in-session, after two adversarial reviews had already reshaped the draft's
  scope — the reviews moved the trampoline from scope commitment to open
  question before the human saw the table).
- **Verdict:** FILE-AS-IS — survived confirmation unedited.
- **Notes:** like itd-128's stance slot, three of the six parts resolved to
  citations of existing records rather than filings; the four-piece table
  keeps working when "home" means "already recorded there". The defect slot
  (a bug found while investigating a capability) is a fifth part-type the
  table absorbs under plumbing-adjacent routing but the protocol page does
  not name; second data point for the enum revisit.

### 2026-08-20 — itd-114 planning interview (re-run after two adversarial reviews)

- **Proposal:** itd-114 (collision-proof record ids) at its planning
  interview, after the two-reviewer prerequisite.
- **Initial routing (the 2026-08-16 table):** FILE-AS-IS with flags — one
  capability, format change deferred to "a future ADR at adoption", stance
  claimed as carried by existing principles.
- **Confirmed routing:** the initial verdict did **not** survive. Both
  reviewers overturned it on the adr-44 precedent: the id format and the
  forge network posture are trust rules extracted to adr-45 + brief
  invariant 11 AT planning; the stance citation was corrected (the claimed
  principle did not exist — it reduces to brief invariant 2 + prefer-sota);
  the forge option was reframed from store to allocator with a typed
  builds_on edge to itd-129, whose ledger-canonical decision the original
  draft had silently reversed. Maintainer confirmed the corrected routing at
  the interview and ruled the four open forks (timestamp-numeric; captures
  first then all families; loud native fallback; detectors kept).
- **Verdict:** SPLIT (capability in itd-114/spc-33; trust rules in
  adr-45 + invariant 11), reviewer-proposed and maintainer-confirmed — the
  initial FILE-AS-IS was overturned. Counts as a graded sample: the 08-16
  table was the blind prediction, the interview the label.
- **Notes:** second consecutive FILE-AS-IS overturned by the two-reviewer
  prerequisite (itd-92 was the first) — the prerequisite is earning its
  cost; the calibration question for the eventual automated pre-pass is
  whether "trust rule parked in intent prose" is mechanically detectable.

### 2026-08-21 — managed-repo identity gate, routine extension (itd-131)

- **Proposal:** from a session finding — ahoy/new-repo setup should establish
  the human git identity so autonomous-routine commits pass the attribution
  gate (a routine defaults to `Claude <noreply@anthropic.com>` and its PR
  reads as unmergeable). Captured as `iss-…367948`
  (ahoy-setup-routine-git-identity).
- **Initial routing (proposed before the human ruled):** capability (guarantee
  the human git identity on every commit incl. routines) | intent — but the
  grill found iss-62 (managed-repo-identity-gate) ALREADY owns this capability,
  so the routing was PROMOTE iss-62, not file-new; the new capture refines it
  (cloud-routine instance). Trust rule ("human is author of record; AI only in
  the trailer) | already a convention (AGENTS.md § Attribution + check-attribution),
  no new ADR. Stance (how AI is acknowledged) | itd-91, cited not duplicated.
  Plumbing (local attribution mirror; lint the pushed tree) | the two sibling
  captures filed the same session. Typed links: new capture refines iss-62;
  intent refines/relates itd-91; adr-44 adjacent.
- **Confirmed routing:** adopted (human confirmation 2026-08-21) — PROMOTE
  iss-62 to itd-131, fold in the routine refinement, cite itd-91.
- **Verdict:** SPLIT/HOLD reached, NOT file-as-is — the decomposition caught
  that a fresh intent would duplicate iss-62. This is the discipline working:
  the grill's "already owned by iss-62" is exactly the miss itd-84 exists to
  prevent, and it landed before an intent was minted, not after.
- **Notes:** third session data point (with itd-128, itd-130) where the grill
  redirected a would-be new intent to an existing owner — twice to a citation
  (itd-128, itd-130), once to a promote (itd-131). Worth a line in the protocol
  page: "promote an existing seed" is a distinct routing outcome from "file new"
  and "cite existing," and the four-piece table should name it.

### 2026-08-21 — hook binary to persistent data dir (itd-132)

- **Proposal:** from the plugin-update post-mortem (session 8db3dbd6) — the
  hook binary lives in the harness's re-cloned, GC'd plugin cache dir, so
  every update re-downloads it, an update-then-quit cancels the SessionEnd
  bootstrap and loses the transcript, and the pinned PATH symlink dangles
  when the orphaned dir is collected. Five ledger captures
  (iss-2608210934566221..225) preceded the routing.
- **Initial routing (proposed, human confirmed same session):** capability
  (binary + .binary-meta relocate to the harness persistent data dir; PATH
  symlink retargets; refresh only on release-tag change) | intent | itd-132,
  absorbing iss-…222; trust rule (persistence must not weaken the spc-21
  verification posture; SessionEnd performs no network work) | ADR + brief
  invariant, drafted alongside the spec; stance (store durable state where
  the platform documents it survives — never fight the harness lifecycle
  inside the cache dir) | principle candidate; plumbing (pluginBinaryPath
  consumers, hooks.json exec order, meta relocation, migration) | spec
  detail; independent defect (SessionEnd bootstraps at exit) | iss-…223,
  fixed test-first on its own branch, no intent; seeds (missed-capture
  recovery sweep, statusline verb) | iss-…224/225 held in the ledger.
  Typed links: itd-132 builds_on itd-105; supersedes spc-21's cache-dir
  fast-path contract (flagged for the human — spc-21's "update into fresh
  cache dir heals by re-fetch" AC is deliberately reversed to "update never
  re-fetches unless the release changed"). No other reversal flags.
- **Confirmed routing:** SPLIT, human-confirmed 2026-08-21 in-session.
  Sequencing caveat for calibration: confirmation preceded the two-reviewer
  prerequisite (reviews launched after, per the interview protocol) — the
  reverse of itd-130's order. The planning interview is the final label;
  relabel here if the reviewers or the interview overturn the table.
- **Verdict:** SPLIT (blind prediction, confirmed pre-review).
- **Notes:** the defect slot recurs (iss-…223 mirrors itd-130's iss-345:
  a bug found en route, fixed test-first, routed outside the intent) —
  third data point for naming that part-type in the protocol enum. The
  reversal flag here is of a shipped spec's acceptance criterion rather
  than an invariant; the table's supersedes link carried it naturally.
  **Interview label (2026-08-21, same day):** the SPLIT table survived the
  two-reviewer prerequisite and the interview unchanged — routing homes and
  typed links held; the reviews instead overturned the *capability's
  internal shape* (relocate-execution → download-cache + per-root copy +
  owned PATH regular file, maintainer-ruled) and forced the record fixes
  (builds_on written, supersession flag recorded, SPLIT parts captured as
  iss-2608210934566226/227 before planning). Counts as a graded sample:
  routing prediction correct; shape prediction wrong. Planned same session
  as itd-132/spc-35.

### 2026-08-21 — the visual-identity proposal (itd-133)

- **Proposal:** one visual identity for abcd — a block-pixel duckling mascot,
  an official logo spelling a-b-c-d in true maritime signal flags, and
  lifeboat art scoped to the lifeboat verbs — rendered on any CLI and as
  HTML/SVG (forge README and web). Live session; the artwork was iterated
  interactively before filing.
- **Initial routing:** three parts — the identity assets and their
  multi-surface rendering (capability | this intent, filed as itd-133); the
  pixel-grid single source of truth with ANSI/SVG generators (plumbing |
  spec body at planning); the role assignment duckling=mascot /
  flags=logo / lifeboat=lifeboat-verbs (intent content, not a separate
  record). Typed link: itd-133 `refines` itd-112 — the banner draft
  explicitly leaves its "small colour logo — an obvious object" open, and
  this supplies it. No reversal flags.
- **Confirmed routing:** adopted unchanged — the maintainer directed the
  filing ("record it as an intent") after choosing the three assets
  in-session.
- **Verdict:** FILE-AS-IS, proposed and confirmed — the initial verdict
  survived confirmation.
- **Notes:** first aesthetic/identity capability in the corpus; the
  stance-shaped part (role assignment) routed *into* the intent rather than
  to a principle because it is the capability's content, not a standing
  rule — a data point for the enum's stance boundary. Confirmation preceded
  the two-reviewer prerequisite (as with itd-132); the planning interview is
  the final label.
  **Interview label (2026-08-21, same day):** the two reviews
  (design/feasibility + record-discipline) overturned the routing in part:
  the terminal-rendering slice (ANSI, colour detection, fallbacks) was
  re-routed *out* of itd-133 to itd-112, which already claimed those
  obligations — so FILE-AS-IS was optimistic; the honest retro-label is
  SPLIT-at-interview. The role-assignment part additionally graduated to a
  decision-log entry (the reviewers' RD4: an "official logo" designation
  needs a durable home beyond the intent), refining the stance-boundary
  data point above. Typed links held but had to be made machine-visible
  (frontmatter `related_intents` written; prose-only `refines` was
  invisible to the lexical pass), and a missed relation to itd-102
  surfaced. A reversal-adjacent flag the initial table missed: itd-133
  forecloses itd-112's object-vs-text-logo open question
  (maintainer-confirmed at interview). Graded: routing prediction partly
  wrong (scope boundary and one link missed); the interview corrected it.
  Planned same session.
### 2026-08-22 — the abcdev.app website proposal

- **Proposal:** "give abcd a website generated from the repository" — the
  full 2026-08-21 site plan (landing page, record explorer, install.sh,
  README migration, `site` verb family, deploy pipeline) arriving as one
  body of investigation.
- **Initial routing table:** capability — the landing page | intent |
  itd-135 (umbrella); capability — the record explorer, split further per
  decompose-before-filing because one press release could not carry both
  the reading surface and the visual chart | intents | itd-136 (record
  pages, contributors, references) + itd-137 (relationship chart,
  genealogy), `builds_on` itd-135; capability — install.sh | intent |
  itd-138, `builds_on` itd-135; capability (gated) — the generic explorer
  on a second instance | intent | itd-139, `builds_on` itd-135/136, held
  in drafts pending its fixture demonstration; trust rules — the
  single-source rule + the adr-30 amendment ("never bundled, rendered
  read-only") | ADR | adr-47; trust rule — deploy-from-tag only | ADR |
  adr-48; stance — the generic/specific boundary (genericity demonstrated,
  never asserted; working-tier crossing by opt-in) | discipline | itd-140;
  plumbing — README migration + the `site` verb family | brief |
  05-internals/10-site.md. Typed links as recorded in the files; no
  reversal flags (the adr-30 boundary change was routed as an amendment
  recorded inside adr-47, not a reversal).
- **Confirmed routing:** the coarse routing (three website intents +
  plumbing-to-brief + two ADRs) was human-dictated in the facilitation
  instructions before the run — the label preceded the prediction for
  those parts. The explorer split (136/137), the discipline, and the gated
  adoption intent follow the two ideate verdicts
  (abcdev-site: survives; record-explorer-generalisation: reframed) and
  await the interview label.
- **Verdict:** SPLIT (eight parts across four record types), part-confirmed
  in advance; the session-added parts pending interview confirmation.
- **Notes:** first hand-run where an ideate reframing minted a discipline
  (the boundary stance came out of the adversarial leg, not the proposal);
  the "capability gated on demonstration" part-type (itd-139) is new to
  the enum — a capability whose filing is confirmed but whose planning is
  explicitly blocked on evidence.


### 2026-08-22 — itd-112 planning interview (post-grill hand-run)

- **Proposal:** itd-112 (the bare-abcd banner), already grilled 2026-08-21
  (scope split to itd-134 confirmed there), at its planning interview after
  two adversarial reviews (design/feasibility and record-discipline).
- **Initial routing:** five parts — the banner capability (this intent);
  the managed-repo generator (already split to itd-134 at the grill); the
  emission-discipline trust rules — TTY-only decoration plus the termsafe
  carve-out, one boundary — (trust rule | one ADR + one brief invariant,
  adr-49 + invariant 13, not two records); the colour ladder and TTY seam
  (plumbing, but a shared primitive with a declared second consumer —
  named in-scope as exported primitives, with itd-110 gaining
  `builds_on: [itd-112]` rather than minting its own record, per
  one-canonical-primitive); the identity bake (plumbing | spec). Typed
  links: itd-112 `builds_on` itd-133, `refines` itd-102; itd-134 carries
  its own `refines` itd-112 (the coined `refined-by` direction was struck —
  not in the enum; the corpus records the pair from the child's side).
- **Confirmed routing:** adopted; the maintainer additionally ruled the
  slug rename (retire-the-name ban on the old slug) and eight design forks
  (identity baked at build time; half-blocks on a painted panel; five-row
  shade-block mono; tagline-below layout; truecolor rung + pinned tables;
  root-local --no-color; dev-build version render; Windows scoped out by
  the release matrix).
- **Verdict:** SPLIT overall (the generator half left at the grill; the
  trust rules left at the interview) — proposed and confirmed in stages
  across the two sessions.
- **Notes:** first hand-run where the reviews moved parts the grill had
  already settled *in prose* into their record homes — evidence for the
  "grill settles, interview routes" division of labour. The
  shared-primitive part (colour ladder) deliberately did NOT get its own
  record: no user moment, so the routing is an in-scope export plus a
  dependency edge — a taxonomy data point for plumbing-with-two-consumers.

### 2026-08-22 — the brief-creation interview (hand-run at capture)

- **Proposal:** the brief-creation interview workstream from the 2026-08-22
  filing handover: staged elicitation (narrative → frontier rounds → options
  at conjectural questions → per-item confirm), four question regimes with an
  escalation rule, a hold register, the two-output rule, and one entry door
  for brownfield (probe-populated) and greenfield (all-blank coverage).
- **Initial routing:** four parts — the interview surface (capability |
  intent itd-142, command-shaped per the `05-internals/08-skills.md`
  boundary: it mutates state, and abcd ships zero skills); "framing traces
  are never committed and never visible to automated reviewers" (trust rule |
  adr-50 + brief invariant 14); "at conjectural questions the tool widens
  options, never recommends" (stance | principle
  `widen-options-never-recommend`); provenance stamps + the hold-register
  store (plumbing | brief, via the itd-142 spec). Typed link: itd-142
  `refines` itd-90 — the deterministic shortlist over `drafts/` surfaced the
  existing coverage-blanks interview before minting, so the overlap filed as
  a typed refinement instead of a duplicate. Open sign-offs (escalation
  rule; one-vs-three intents at the final round; a held working-principle at
  the final round) stayed in the intent as Open Questions — routed nowhere.
- **Confirmed routing:** maintainer confirmed 2026-08-22 (file all homes
  now; only the spec waits, until the collaborating prototype has run once).
- **Verdict:** SPLIT, proposed and confirmed.
- **Notes:** the candidate pass earned its keep — without it this session
  would have minted a second blanks-interview beside itd-90. The
  hold-register *home* question deliberately stayed unrouted (open ledger
  seed iss-2608220750029991 + the evidence chapter's open question), a
  taxonomy data point: a part whose home is itself the open question routes
  to a seed, not a record.

### 2026-08-22 — the framing chapter (hand-run at capture)

- **Proposal:** a framing section under `01-product/` — the macro-why home,
  also the destination for the interview's committed framing products.
- **Initial routing:** three parts — the framing home + its brief↔lifeboat
  mapping row (capability | intent itd-143); this repository's own framing
  content (plumbing | brief, written when the section ships); the
  glossary-as-deliberate-frame-surface note (docs | `brief/glossary/README.md`,
  filed in the same change).
- **Confirmed routing:** maintainer confirmed 2026-08-22.
- **Verdict:** SPLIT, proposed and confirmed.
- **Notes:** the mapping row makes this a code-touching intent (mapping.go +
  round-trip tests), not a docs-only change — the routing caught that early.

### 2026-08-22 — intent-shape additions (hand-run at capture)

- **Proposal:** intents gain a mechanism claim ("we expect this to work
  because…") and a scope-conditions section.
- **Initial routing:** two parts — the record-shape change (architecture
  decision | adr-51 + the `04-surfaces/05-intent.md` template); enforcement
  (discipline question | explicitly deferred, no record minted — if it
  comes, it files on the itd-84/itd-1 staged-gate pattern).
- **Confirmed routing:** maintainer confirmed 2026-08-22 (ADR + page now).
- **Verdict:** FILE-AS-IS (as the routed ADR), proposed and confirmed.
- **Notes:** both sections optional and unenforced; `enforcement-claims-are-facts`
  keeps the template honest about that.
### 2026-08-22 — the writing style guide (hand-run at capture)

- **Proposal:** one canonical writing style guide (maintainer request,
  2026-08-22): consolidate the scattered rules (British/US split, present
  tense, Diátaxis) plus new punctuation rules (no em dash in list items — use
  a colon; capital after a colon; lower case after a semicolon), enforce the
  machine-checkable subset, and record the loader pointer.
- **Initial routing:** four parts — the docs-lint enforcement of the
  machine-checkable subset (capability | intent itd-141); the canonical guide
  page (docs | `docs/reference/writing-style.md`, linked from
  CONTRIBUTING.md); the loader pointer (config plumbing | a DOCUMENTATION
  override in `.abcd/rules.json`, the point-don't-copy pattern); the Vale
  ruling and the adopt-an-open-licensed-guide exploration (SOTA | inside
  itd-141's SOTA section, per sota-per-intent — no separate record). No new
  principle: point-don't-copy restates the existing OPINIONS pattern; no ADR:
  no new rule family (maintainer pre-ruled).
- **Confirmed routing:** pre-confirmed — the maintainer's 2026-08-22 request
  arrived already decomposed into these homes, with the Vale ruling and the
  no-ADR condition explicit; filed as ruled.
- **Verdict:** SPLIT, proposed and confirmed (all parts filed in one change).
- **Notes:** first hand-run where the human's proposal arrived pre-routed;
  the protocol's value here was verifying no part was missing a home, not
  discovering the split. `enforcement-claims-are-facts` did real work: the
  guide labels the staged rules `review` until the lint ships.

### 2026-08-22 — the livery-placement and credit-enforcement captures

- **Proposal:** two threads surfaced by an end-of-session sweep — where the
  unwired livery marks belong (the lifeboat shown on `disembark` and mirrored
  on `embark`, the duckling as the harness mascot, the icon for the website),
  and how the acknowledgement convention stops depending on the author
  remembering.
- **Initial routing:** four parts. Mark placement across surfaces (capability
  | intent | itd-144); the forge/web logo decision itd-112 deferred (decision
  | resolved inside itd-144's planning, `refines` itd-112 — not a separate
  ADR: it is a positioning choice, not a trust rule); credit enforcement
  (capability | intent | itd-145, shaped after itd-141's lint-arms-a-
  convention precedent); the detector's heuristic — what counts as an
  uncredited citation (plumbing | itd-145's spec at planning). Typed links:
  itd-144 `builds_on` itd-133 and itd-112, `refines` itd-112; itd-145 carries
  none, the itd-141 kinship being a shape precedent rather than a
  supersedes/reverses/duplicates/refines edge. No reversal flags.
- **Confirmed routing:** adopted; the maintainer directed both filings and
  ruled the placement work explicitly non-urgent.
- **Verdict:** FILE-AS-IS for both, proposed and confirmed.
- **Notes:** itd-145 is the second instance of the "arm the convention that
  currently relies on vigilance" shape in two days (iss-2608220750029993, the
  session-presence detector, is the first) — both were surfaced by a live miss
  rather than by review, which is worth watching as the corpus grows: the
  protocol catches structure, but the missing-detector class keeps arriving
  through incidents. The closed typed-link enum had no value for a
  same-shape-different-subject sibling; recorded here rather than inventing a
  fifth link type.



### 2026-08-23 — the brief's surface chapters drift (hand-run at capture)

- **Proposal:** the brief's surface chapters become a generated reflection of
  the shipped surface, so shape claims cannot drift. Prompted by a full-tier
  `iss35-brief-surface-crosscheck` returning 125 discrepancies at the 0.6.2
  release gate (iss-2608231346137587).
- **Initial routing:** four parts — the generated chapters (capability |
  intent); shape claims are derived, never hand-authored (trust rule | ADR +
  brief invariant); the underlying stance (stance | none new,
  `one-canonical-primitive` already says it, so this is an application rather
  than a new rule); the generation mechanism and its seam (plumbing | brief).
- **Confirmed routing:** maintainer confirmed 2026-08-23, with the intent to be
  filed after the 0.6.2 cut rather than during it.
- **Verdict:** SPLIT, confirmed.
- **Notes:** the stance leg is the interesting one — the hand-run's first
  instinct was to mint a principle, and the four-piece table is what caught
  that `one-canonical-primitive` already covers it. Recording "none new, this is
  an application" is a routing outcome the table should keep making easy, since
  a fifth principle restating a fourth is the failure mode a stance leg invites.
  Evidence base preserved out-of-tree at
  `.abcd/.work.local/scratch/brief-drift-2026-08-23/`; it earns promotion to a
  dated research note when the intent is filed, and the issue says explicitly
  not to fix the 124 by hand before mining it.

### 2026-08-22 — the CLI verb taxonomy — EXCLUDED from the corpus

- **Excluded from the calibration count:** the routing was confirmed before the
  adversarial gate ran, and the gate then changed the payload the routing had
  been confirmed over. The prediction and the label therefore describe two
  different proposals, and grading the verdict enum across that break would
  book a correct prediction from a run whose content did not survive. Recorded
  for the protocol's history, not as a sample toward the ~50-capture threshold.
- **Proposal:** the top-level verb list keeps growing and risks becoming
  unmanageable, so regroup the verbs into a noun-verb hierarchy (`abcd quality
  banlist`; `memory` as a category), with the CLI tree decoupled from the flat
  plugin command surface.
- **Initial routing:** four parts, plus one held. The SOTA finding and the
  verdict (research | a dated note). The shippable capability, read at the time
  as grouped help *plus hiding the operator-internal verbs* (capability | one
  intent). Folding `changelog` under `launch` (plumbing | a capture). The
  design-time test for new verbs (stance | a capture). HELD: the
  CLI-as-engine-API ruling (structural rule | ADR), not filed because the
  maintainer had not ruled and an ADR records a ratified choice.
- **Confirmed routing:** the human confirmed the four homes as proposed, with
  the ADR left held. **Edits at the second confirmation, after leg 3:** the
  hiding half of part 2 was struck, leaving grouped help alone as the intent
  (filed as itd-146); part 3 was struck entirely, `changelog` surviving only as
  an open question inside itd-146; part 4 was confirmed unchanged (filed as
  iss-2608221310142705); part 1 was confirmed unchanged. No home changed at
  either confirmation. A fifth part was added at the second confirmation, not
  present in the initial table: an intent for review-finding disposition.
- **Verdict:** SPLIT, proposed and confirmed. Ungraded, per the exclusion above.
- **Notes: the routing was right and the payload was wrong, and the protocol
  measures only the first.** Every home held across both confirmations, which
  under the itd-81 recipe reads as a clean hit. What the table never saw is that
  two of the five parts were later withdrawn outright: the fifth part's central
  evidence was falsified by a second adversarial pass, and part 3 rested on a
  claim about a pending breaking cut that `iss-284` disproved. A protocol that
  grades destination and never content will score a run like this as a success.
- **Second note, on ordering:** the protocol has no step that re-opens a
  confirmed routing when a later gate changes what there is to file. Routing
  confirmed on a pre-adversarial proposal is provisional, and saying so at
  confirmation time is cheaper than discovering it at filing time.
- **Third note, on what the grill bought:** `iss-284` showed the adr-40 renames
  had already shipped in v0.6.0, which falsified the justification the
  `changelog`-folding alternative rested on. The citation-proving step forced
  that check. The limit of that same step is recorded in the verdict note's
  errata: it proves a cited id resolves and proves nothing about whether the
  claim beside it is true, and a claim marked `verified` in leg 1 of that run
  was false.

### 2026-08-26 — the worktree-first proposal (itd-148)

- **Proposal:** "automate the worktree step": primary checkout read-only,
  every change starts in a worktree, hard-block enforcement, sweep of stale
  worktrees, cross-worktree agent communication, scaffolded to all
  abcd-managed repos. Live interview session; two peer sessions active.
- **Initial routing:** seven parts — worktree-first discipline (capability |
  new intent); mutation block (mechanism | same intent); all-repos scaffolding
  (plumbing | the spec); merged-worktree removal (**duplicates itd-118** |
  routed there, consumed via `builds_on`); abandoned-worktree dossier sweep
  (capability | new intent, `builds_on: itd-118`); cross-worktree
  communication (**overlaps itd-33** | flagged for the human — fold, layer, or
  advance itd-33); timestamp mint (prerequisite | already
  iss-2608210737260468, linked as blocker, not filed). Bundle candidates
  surfaced from the ledger: iss-370, iss-2608230847432285 + iss-213,
  iss-2608230957104179, iss-2608210738378295.
- **Confirmed routing:** adopted unchanged, with the itd-33 overlap resolved
  as **layered** (announce-duty AC in the new intent; mechanism stays
  itd-33's) and all four bundle candidates confirmed into scope. One addition
  from the human at confirmation: the iss-370 seeding gains a four-category
  pre-population floor for the private banlist (home/user paths, machine
  hostnames, personal email, real surname — categories committed, values
  never).
- **Verdict:** SPLIT, proposed and confirmed — the initial verdict survived
  confirmation; filed as itd-148.
- **Notes:** no over-flags; no reversal candidates. The duplicate detection
  (itd-118) came from a ledger grep before the table was built — the scan
  step earns its place. The communication part shows the taxonomy gap for
  "requirement whose mechanism lives in another draft": neither `refines`
  nor `duplicates` fits; recorded here as *layered* pending the enum
  question.

### 2026-08-27 — the pilot-note automation proposal (hand-run at filing)

- **Proposal:** the security-advisory pilot's closing instruction — "abcd
  handles inbound security advisories and issues, and cuts the release, for
  every managed repo", citing F-A…F-W as acceptance criteria with F-U and
  F-Q load-bearing.
- **Initial routing:** four parts — the end-to-end loop (capability | new
  intent); the verified gate-arming trust rules F-N/F-Q/F-U (trust rule |
  ADR + brief invariant, held as iss-2608271817300337 until the ADR's own
  filing); the F-W authorship default — adopt an external contributor's
  commit, `Reported-by` is the fallback (stance | principle, held as
  iss-2608271817309715); receipt mechanics, F-S detector pairing, F-R
  re-roll classification and the S1–S6 intake stages (plumbing | brief at
  spec time). Typed links: **refines itd-93** (planned release-gate
  scaffolding), **refines itd-82** (draft ledger triage), **refines** the
  intake.md S1–S6 protocol, whose graduation is already recorded as
  iss-2608271702469341.
- **Confirmed routing:** adopted unchanged.
- **Verdict:** SPLIT, proposed and confirmed; filed as itd-149.
- **Notes:** no reversal candidates; the standing stances the loop relies on
  (evaluator-outside-the-loop, record-before-fix, sweep-the-pattern) are
  cited, not re-homed — only F-W's authorship default was a genuinely
  unhoused stance.

### 2026-08-28 — itd-141 at its planning interview (hand-run, graded)

- **Proposal:** plan itd-141 (the staged punctuation lint), after the em-dash
  core had shipped as a banned token in the same session's docs sweep.
- **Initial routing:** four parts — the casing lints with masking
  (capability | itd-141, KEEP); the shipped em-dash line rule (capability |
  already delivered by iss-2608280706531199, REMOVE from the promise, cite as
  prior art); the continuation-line residual (capability | itd-141, KEEP);
  the guide-adoption-via-`.abcd/rules.json` open question (capability, a
  different one | route out to a future record when live). Proposed by the
  record-discipline reviewer.
- **Confirmed routing:** the maintainer's interview verdict went further than
  the proposal: the design review's corpus evidence (0 true positives on
  every remaining rule; unclosable false-positive classes; promotion
  impossible under warn-then-promote) collapsed both KEEP parts — the casing
  rules re-route to a permanent `review` label with recorded reasons, the
  residual to an accepted non-build. Confirmed home: adr-54 supersedes the
  intent; the routed-out guide-adoption question is captured to the ledger.
- **Verdict:** SPLIT proposed, SUPERSEDE confirmed — the first hand-run where
  the interview's outcome was that the record under decomposition should not
  exist. The initial routing did not survive, and the miss was systematic:
  the shape-lens pass routed parts correctly but could not see yield; only
  the corpus-measuring pass could rule the parts unbuildable.
- **Notes:** calibration lesson for the automated rung — a decomposition
  pre-pass that routes without measuring corpus yield will confidently route
  dead capabilities; the design/feasibility lens is not optional for
  lint-shaped intents. `enforcement-claims-are-facts` did the closing work
  twice: it caught the draft's stale claim about the guide, and it forced the
  guide's forward-claim ("staged lints mask code spans") out in the same
  change.

### 2026-08-28 — the deterministic attribution machinery (hand-run at filing)

- **Proposal:** from the 2026-08-28 attribution review's closing
  recommendation — cited sources must resolve into the references, credit
  must surface transparently in the acknowledgements, and a source's licence
  is vetted before it is used, "deterministically in some way".
- **Initial routing:** five parts — the reference-closure and
  acknowledgements-mirror gate with its influence registry (capability |
  new intent); the licence-vetting admission check (capability | second
  new intent, since it needs the refresh verb, a baseline schema, and a
  licence backfill the first gate does not); "no source enters the record
  without a vetted licence; the gate never dials out" (trust rule | ADR +
  brief invariant, landed with the second intent); "every design-shaping
  influence is credited on the canonical surface" (stance | already
  half-stated in the acknowledgements header; principle line to be
  confirmed at planning); the acknowledgements backfill itself (not an
  intent | issue, filed as iss-2608280824478819, resolved by the change
  that arms the first gate). Typed links: itd-164 **builds_on** itd-163;
  itd-163 **refines itd-145** (the acknowledgement convention arming
  itself); both extend the spc-17 citation lint family rather than adding
  a parallel checker.
- **Confirmed routing:** the two-intent split and the issue-first order
  were proposed to the human and confirmed in-session before filing
  ("yes, capture the issue and draft the two intents"); the stance line's
  final home is left as a planning question.
- **Verdict:** SPLIT, proposed and confirmed; filed as itd-163 and
  itd-164.
- **Notes:** the ordering follows record-before-fix — the backfill issue
  is captured ahead of the gate that will refuse its absence, and day-one
  green is an acceptance criterion of itd-163 so the arming change and
  the backfill are forced into the same diff. Mint coordination: three
  peer sessions were consulted before minting (itd-150–162 were taken on
  queued PRs invisible to this checkout's tree); the binary's mint ledger
  agreed and produced 163/164 directly. A miss, caught late: the
  candidate scan over `drafts/` ran only after minting, when the brief's
  later-phase index refused the new ids — it surfaced itd-145 as an
  existing owner of the credit-enforcement aim. The overlap filed as
  itd-163 `refines` itd-145 with an advisory supersession flag left for
  the human at planning; had the scan run first, the routing might have
  been PROMOTE itd-145 rather than file-new. Fourth data point for the
  scan step earning its place, and the first where skipping it cost a
  mint. **Ruling (same day):** the maintainer ruled itd-145 superseded —
  the link is `supersedes`, itd-145 moved to `superseded/` with both
  sides stamped, and itd-163 is the canonical credit-enforcement intent.

### 2026-08-30 — claim typing and scope-condition identity (itd-177, hand-run at filing)

- **Proposal:** an intent's claims are typed and its scope conditions keep
  their identity across edits — the readiness gate refuses an intent that
  leaves a context claim unrecorded.
- **Initial routing:** three parts — the typing and the identity marker at the
  gate (capability | intent, itd-177); the gradient's rationale, nullity
  grammar and staging (standing rule | discipline, itd-190); the dispositions
  the identity makes attachable (capability | a second intent, itd-181).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, proposed and confirmed; filed as itd-177 with itd-190
  and itd-181 carrying the other two parts.
- **Notes:** the identity marker is stamped by `intent plan`, never hand-typed;
  adr-51 is consumed rather than reopened, so no ADR part arose.

### 2026-08-30 — origin and production-mode keys (itd-178, hand-run at filing)

- **Proposal:** every record written through a command carries its origin and
  its production mode, stamped by the command and never typed by hand.
- **Initial routing:** two parts — the two keys, resolver support, the
  command-side stamping on every write path and the hand-edit lint (capability
  | intent, itd-178); the three-term production-mode vocabulary (already ruled
  | the decision log, not re-minted here).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** the attribution seam is itd-91's `.abcd/config/identity.json`,
  extended rather than duplicated. Population is forward-only, so no backfill
  part exists to route — the absent stamp is information, not a gap.

### 2026-08-30 — grounds at conjecture granularity (itd-179, hand-run at filing)

- **Proposal:** readiness and triage record grounds for the conjecture being
  acted on, not only for the decision reached.
- **Initial routing:** one part — the grounds argument and its three-value
  vocabulary on `intent ready` and the capture triage routes, refusing on
  absence (capability | intent, itd-179). The ADR family's
  decision-granularity grounds are explicitly out of scope.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** routed as one part because the refusal and the vocabulary land on
  the same command surface. No ADR: the finer grain sits beside the existing
  grounds rather than redeciding them.

### 2026-08-30 — reading records and disposition records (itd-180, hand-run at filing)

- **Proposal:** a cold reading's findings land as reading records and the
  researcher's response is a separate disposition record — two acts, two
  writes, never collapsed.
- **Initial routing:** two workstream items collapsed to one part — one record
  type with a position-typed body plus the separately written disposition
  record (capability | intent, itd-180), `refines` itd-86. The reserved hold
  field's home question is not a part: it stays with iss-2608220750029991.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS, as one record rather than two.
- **Notes:** the interesting call is a merge, not a split — four record types
  were rejected for one envelope, giving one lint, one disposition surface and
  one identifier scheme.

### 2026-08-30 — scope-condition disposition (itd-181, hand-run at filing)

- **Proposal:** a shipped intent's scope conditions are dispositioned by the
  fidelity verdict — what was assumed ex ante and what survived are recorded
  as different things.
- **Initial routing:** one part — the four-value disposition surface keyed to
  condition identity, populated at verdict ingest (capability | intent,
  itd-181), depending on itd-177's identity marker.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, as the disposition half of the itd-177 proposal;
  confirmed.
- **Notes:** kept separate because it lands in `intent audit` ingest and the
  auditor contract, not the readiness gate — different surface, different
  owner, so a single record would have spanned two gates.

### 2026-08-30 — the lapse capture category (itd-182, hand-run at filing)

- **Proposal:** the record's own discipline failures are recorded — a lapse is
  a capture category, timestamped at the lapse rather than at write-up.
- **Initial routing:** two parts — the `lapse` value in capture's validated
  category list (capability | intent, itd-182, one enum line); the first three
  lapse entries (not an intent | ledger captures, written at the outset).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** the smallest record of the cycle, and the only one whose code part
  landed ahead of its intent — the enum line went in first so the three
  entries had a category to be filed under.

### 2026-08-30 — the cold-reading input assembler (itd-183, hand-run at filing)

- **Proposal:** the cold reading sees exactly what the assembler passes —
  positive inclusion, field projection, and a per-run manifest.
- **Initial routing:** four parts — the assembler with its per-position include
  table and field projection (capability | bundle-member intent, itd-183); the
  per-run manifest (same record, it is the assembler's own output); the
  read-block eval (capability | its own intent, itd-186); the amnesia eval
  (capability | its own intent, itd-187).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, proposed and confirmed — the two evals to their own
  drafts.
- **Notes:** the split is on falsifiability: an eval whose oracle reads the
  assembler's include table can only assert that table. The two assembler
  rules ruled the same day stay inside this record — they make the include
  list derivable rather than remembered.

### 2026-08-30 — the four cold-reading definitions (itd-184, hand-run at filing)

- **Proposal:** four cold-reading definitions over one blindness core — each
  position licenses a different output, and none may hold another's licence.
- **Initial routing:** one part — four agent definitions holding object,
  question, byte-identical core, regime value and item shape (capability |
  bundle-member intent, itd-184). The criteria the comparative definition
  consumes route out to itd-191 rather than into this record.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** the count of definitions and the count of contexts are different
  countings and are compatible — four instances within one detector context.
  The ruling is implemented as stated, not re-litigated at filing.

### 2026-08-30 — the cold-reading output contract (itd-185, hand-run at filing)

- **Proposal:** one ingest verb validates every cold-reading output, including
  what the reading was licensed to produce, not only what it saw.
- **Initial routing:** two workstream items collapsed to one part — the ingest
  verb with its strict schema and the supply-regime gate (capability |
  bundle-member intent, itd-185), so the read-block and the contract are
  written in one place.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS, as one record.
- **Notes:** the regime gate is the enforcement half of
  `widen-options-never-recommend`, whose promotion moves to the staged rung in
  the same filing. Whether the signatures lint cleanly in practice stays an
  open question inside the record rather than a routed-out part.

### 2026-08-30 — the read-block eval (itd-186, hand-run at filing)

- **Proposal:** planted warm content that reaches a reading fails the build
  loudly.
- **Initial routing:** one part — the sentinel fixture state, one plant per
  warm location class, and field-level absence assertions with an oracle
  independent of the include table (capability | intent, itd-186).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, as the first of the two instruments taken out of the
  itd-183 proposal; confirmed.
- **Notes:** separate because it carries its own claim and its own verdict.
  Folding it into the assembler would put the falsifier inside the thing it
  exists to falsify.

### 2026-08-30 — the amnesia eval (itd-187, hand-run at filing)

- **Proposal:** the same state assembled twice is byte-identical, so no case
  run is spent evidencing amnesia.
- **Initial routing:** one part — the double-assembly comparison with the
  manifest excluded, and the determinism preconditions it enforces on the
  assembler (capability | intent, itd-187).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, the second instrument out of the itd-183 proposal;
  confirmed.
- **Notes:** filing it as a repository eval is exactly what keeps amnesia off
  the closing case run's list of properties — the routing decision and the
  epistemic claim are the same decision here.

### 2026-08-30 — the scribe context (itd-188, hand-run at filing)

- **Proposal:** machine assistance in maintaining the ledger, without any
  context that holds both ledger content and a reading.
- **Initial routing:** three parts, all routed to one record — the scribe
  definition with the assembler's inverse access rule; the fidelity-flag
  permission and the contribution stamp; the hand-run protocol until the
  ingest verb lands (capability | intent, itd-188, the last part marked
  [HAND]).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** kept whole because each part qualifies one access rule rather than
  standing alone. The contribution stamp is explicitly the precursor of
  itd-178's keys and retires when they ship, which is why it is staging inside
  this record and not a record of its own.

### 2026-08-30 — step-2 admission records (itd-189, hand-run at filing)

- **Proposal:** what the widening reading proposes is admitted or declined on
  the record — grounds on admission, dispositions on declines, and surprises
  as their own entries.
- **Initial routing:** three shapes collapsed into one schema-only record —
  the admission-grounds record, the declined-proposal disposition, and the
  surprise entry (capability | intent, itd-189); schema this cycle, command
  enforcement next.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** [HAND] this iteration — no reading runs, so there is nothing to
  record yet. The surprise entry is deliberately a third shape rather than a
  disposition variant: the reading's output, the researcher's response, and
  the surprise that occasions abduction are three acts.

### 2026-08-30 — the claim recording gradient (itd-190, hand-run at filing)

- **Proposal:** an intent's three claim kinds carry three recording
  requirements, and the readiness gate holds them.
- **Initial routing:** one part — the gradient, the nullity grammar, the
  forward-only population rule and the discipline-kind exemption (standing rule
  | discipline record, itd-190), `builds_on` itd-1.
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** SPLIT, the rationale-and-staging half of the itd-177 proposal;
  confirmed.
- **Notes:** routed to a discipline rather than a principle because it imposes
  an acceptance gate every other intent inherits, which is the discipline
  family's own test. itd-1 already owns the criterion claim, so the gradient
  extends a rule rather than founding one.

### 2026-08-30 — the selection criteria (itd-191, hand-run at filing)

- **Proposal:** a candidate is characterised and selected against criteria the
  record states, never criteria supplied at invocation.
- **Initial routing:** one part — the six-criterion slate, amendable only by
  ordinary discipline amendment, consumed by the comparative definition
  (standing rule | discipline record, itd-191).
- **Confirmed routing:** confirmed by the orchestrator under the
  facilitator's standing authorisation, decision (20) of 2026-08-28, no
  per-run human confirmation — the autonomy is itself under observation this
  cycle.
- **Verdict:** FILE-AS-IS.
- **Notes:** a taxonomy ambiguity the record states about itself — these
  criteria govern selection rather than delivery, so the discipline family is
  the nearest home rather than an exact fit, filed there in preference to
  minting a record family for one record. Feeds the open question on the enum.

### 2026-08-30 — the exclusion floor (itd-194, hand-run at filing)

- **Proposal:** the cold-reading exclusion floor leaks, on ten records, and
  itd-183 shipped over the residue under a facilitator ruling that the floor be
  carried as its own intent.
- **Initial routing:** four parts. (1) The capability: the include table
  refuses what the floor cannot parse, so admission and comprehension describe
  one set (intent, itd-194), `builds_on` itd-183. (2) The trust rule: an
  exclusion control asserts only what it can prove, so a manifest may not claim
  a refusal the floor did not perform (ADR + brief invariant). (3) The stance:
  a control that declines to run must say so rather than report a clean pass.
  (4) The plumbing: the include table's narrowing is a scope decision it makes
  and the floor inherits unstated, which belongs in the brief.
- **Confirmed routing:** confirmed by the orchestrator under the facilitator's
  standing authorisation, decision (20) of 2026-08-28, no per-run human
  confirmation. The design fork underneath part (1) was NOT absorbed into the
  ceremony: iss-2608301450065320 states the remedy is a design question and
  declines to pick, so it was put to the facilitator, who ruled on 2026-08-30
  for narrowing admission rather than widening the floor's parse.
- **Verdict:** SPLIT, into (1) and (2); (3) files nowhere and (4) rides with
  the ADR.
- **Notes:** part (3) is the calibration's own result this run. The stance is
  already held by the `loud-staging` principle, whose letter covers a stage that
  no-ops or degrades saying so, and the floor declining to scan while the
  manifest reports exclusion is that principle's exact shape. Routing it to a
  new principle would have been a third copy of a rule the repository already
  states once, so the hand-run cites `loud-staging` instead of minting against
  it. That is the first time this table has retired a part by finding its home
  already occupied, and it is the outcome the one-canonical-primitive rule
  predicts should be common.

### 2026-08-30 — the review-round cost calibration (hand-run at recording)

- **Proposal:** the maximal review discipline run in cycle 1 was an experiment
  to measure verification overhead; the evidence is in, and what it says should
  bind the remainder of this phase and future phases in abcd and abcd-managed
  repos.
- **Initial routing:** three parts. (1) The stance — review effort scales with
  what CHANGED, not only with what is being crossed (principle). (2) The
  evidence — per-intent commits, rounds and diff sizes, the token accounting,
  and the delta-versus-full-diff waste (dated research note). (3) Any tooling
  that would scope a review to the delta mechanically (a future capability,
  not filed).
- **Confirmed routing:** confirmed by the orchestrator under the facilitator's
  standing authorisation, decision (20) of 2026-08-28. The facilitator set the
  frame explicitly: run it all-in first to see the overhead, then state what
  should be done with the evidence in hand.
- **Verdict:** SPLIT into (1) and (2), with (3) recorded as a future rung
  inside the note rather than filed.
- **Notes:** part (1) did NOT mint a principle. `adversarial-review-scales-
  with-blast-radius` already holds the stance for record crossings and simply
  said nothing about build rounds, so the run EXTENDED it with a build-round
  half rather than adding a near-synonym beside it. That is the second
  consecutive hand-run to close a part by finding the home already occupied
  (the previous one retired a stance into `loud-staging`), which is now enough
  of a pattern to say the table's most common non-trivial outcome is
  consolidation rather than minting.

### 2026-08-31 — a claim about behaviour is executable (itd-195, hand-run at filing)

- **Proposal:** prose stating how another part of the codebase behaves is backed
  by something that runs, or it is not written; and the existing corpus is swept
  for the shape.
- **Initial routing:** three parts. (1) The rule every change inherits, with a
  gate (discipline record, itd-195). (2) The codebase-wide sweep, which is the
  rule's own staging work rather than a separate capability. (3) The stance —
  checked against the principles shelf first, per the last two runs.
- **Confirmed routing:** confirmed by the orchestrator under the facilitator's
  standing authorisation; the facilitator asked for the filing directly
  ("record it as a future intent ... this probably becomes a discipline"), so
  the routing question put to them was the SHAPE, not whether to file.
- **Verdict:** FILE-AS-IS as a discipline, with the sweep carried inside it.
- **Notes:** part (3) is the interesting one and it did NOT close the way the
  last two did. The shelf holds two near neighbours and neither is the home.
  `enforcement-claims-are-facts` forbids describing a gate that does not run —
  a claim about EXISTENCE, judged true or false when written. This rule is about
  a claim STAYING true, which that one cannot reach, because a comment can be
  exactly right the day it is written and false six months later.
  `guards-prove-themselves` is the same instinct scoped to refusal paths. So the
  new record is the GENERAL form and the two adopted principles are its special
  cases; it is filed beside them with that relationship stated, and if adopted
  they should say they are instances of it. Three consecutive hand-runs have now
  turned on "is the home already occupied" — twice yes, once no — which is
  becoming the table's most load-bearing question rather than an afterthought.

### 2026-09-02 — the Iteration 2 cold-reading intents (hand-run at filing, eight proposals)

- **Proposal:** the eight capabilities the design documents require before
  Iteration 2 can open: the comparative channel, the admission and surprise
  verbs, the scribe verbs, the reading-occasioned origin, the calibrated
  presets, the reframe record, the knowledge-record extension, and the
  condition disposition from a reading run.
- **Initial routing:** one intent each, with every part named as capability.
- **Confirmed routing:** the record-discipline adversarial review, run before
  the interview under the maintainer's instruction to plan the eight into
  specs, split two trust rules, one ruling, one record-architecture decision
  and one piece of brief plumbing out of the intents: the
  comparative channel's positional exception to the prior-run exhaust (ADR plus
  brief invariant), the two-sessions property behind the scribe (ADR plus brief
  invariant, with the bundle-shape change as brief plumbing), the rule over
  every construal rewrite (ADR refining adr-55, or a discipline), and the
  principles family becoming a declared record store (ADR on the adr-30
  pattern). Each is captured as an issue in the same change and flagged in the
  intent it left. The scanner fix for absence as a class was routed out of the
  admission intent to its two existing issue records, which is a consolidation
  on the pattern of the three preceding runs rather than a split, so the
  admission intent counts as filed as-is. The maintainer confirmed
  the routing by instruction to file and plan; the reversal flags (itd-199's
  comparative-preset refusal withdrawn, itd-186's exhaust rule gaining an
  exception, adr-55 refined) stay advisory until adopted.
- **Verdict:** SPLIT on four of eight (comparative, scribe, reframe,
  knowledge); FILE-AS-IS on the other four (admission, origin, presets,
  condition), each with typed `refines` links to the record it extends.
- **Notes:** every split was the same shape, a trust rule riding inside a
  capability, which is the failure the table exists to catch and the first
  time it has caught four in one run. None of the eight found its home already
  occupied, unlike the three preceding runs; the record has no comparable
  capability for any of them, which is consistent with the design documents
  having scheduled them for a later iteration.

### 2026-09-08 — the OpenAI-compatible api oracle adapter (hand-run at filing, one proposal in two framings)

- **Proposal:** one capability offered in two framings across consecutive
  prompts — "an OpenRouter api adapter", then "an api adaptor for a local
  server (Gropius)". Both name the same record: abcd reaching a model over a
  direct HTTP API, adr-25's `api` oracle-backend shape, with the provider
  (cloud aggregator vs localhost MLX server) as configuration.
- **Initial routing:** the adapter as the intent (refining adr-25, sibling of
  itd-6's mcp-shape adapter); backend resolution via `oracle.backend` config
  as brief plumbing already designed; provider-credential handling as covered
  by existing scanner/redaction invariants (itd-152, itd-28) unless key
  storage becomes abcd's job; the host-delegated-default stance as already
  settled by adr-25. The ahoy-time "probe OpenRouter and offer it" discovery
  behaviour surfaced as a second part inside the proposal.
- **Confirmed routing:** the human confirmed the generic
  provider-agnostic intent (not a Gropius-only narrow one), and routed the
  OpenRouter discovery-and-offer behaviour OUT to a captured issue
  (iss-2609081951416843) rather than folding it into the intent's acceptance
  criteria. No split: every piece kept its initial home. The intent filed as
  itd-2609081951381895 with the first wired provider the local Gropius MLX
  server and OpenRouter named as config-only.
- **Verdict:** FILE-AS-IS, with one part (ahoy discovery-and-offer) captured
  separately at the human's choice.
- **Notes:** the same proposal arriving twice under two provider framings was
  itself the decomposition signal — the run turned on recognising that
  "OpenRouter" and "Gropius" are values of one configuration axis, not two
  capabilities, so the choice put to the human was generic-vs-narrow rather
  than capability-vs-trust-rule. Second run in a row where the home was
  already occupied (adr-25 names the shape; itd-6 is the sibling adapter) and
  the work was filling an existing slot, not minting a concept.

### 2026-09-09 — the lifeboat widens to narrower sources (hand-run at filing)

- **Proposal:** "expand the lifeboat mechanics to also cover (beside entire
  repos) a worktree, lab session, and otherwise abandoned feature
  test/experiment/implementation" — the widening the press release had
  recorded that morning under the not-yet-real marker.
- **Initial routing:** one capability intent (narrower sources); source-kind
  plumbing and the git-assuming acceptance rows to the brief's disembark
  chapter as a refinement of adr-35; the harness-name confinement on a pack
  from a lab home as already covered by adr-39 rule 6 and scan-before-write,
  flagged rather than filed; the graveyard's tier-0 reading of abandoned
  branches noted as existing behaviour (adr-35 §7). Typed links: refines
  itd-88 and adr-35; touches the 2026-08-31 lab convention, the unfiled
  capstone lab intent, itd-118 and itd-8. No reversal.
- **Confirmed routing:** the human SPLIT by source kind — git sources
  (worktree, branch, abandoned feature work) as one intent, the lab session
  home as another, because the second source is not a repository and its
  packer waits on the lab verb family. Filed as itd-2609090746410233 and
  itd-2609090746414083. The plumbing and trust parts kept their initial homes.
- **Verdict:** SPLIT (proposed FILE-AS-IS; the initial routing did not
  survive on the count of intents, survived on every other part).
- **Notes:** the split axis was the SOURCE's nature (git vs non-git), not a
  trust rule inside a capability — a new split shape for the corpus. Third run
  in a row where the home was already occupied: adr-35 owns the mechanics,
  itd-88 the coverage readout, and both intents fill slots those records left
  rather than minting a concept.
## 2026-09-09 — sub-agent transcript capture (itd-2609090559376002)

- **Proposal:** the transcript store keeps only the top-level session
  transcript, so every sub-agent transcript is missed. Measured on this
  machine's corpus, that is roughly three quarters of all recorded bytes.
- **Initial routing:** five parts. The capture capability to an intent; the
  lineage representation (explicit fields for the spawning session and agent
  kind, against the composite identifier already in hand-use) to an ADR
  refining adr-29; the migration of existing composite records to the intent's
  spec; a cross-repo ingest seam to its own record; and a candidate principle
  about a record being reachable from the identifier a reader holds.
- **Confirmed routing:** the product thinker adopted the SPLIT, then widened
  the intent twice in the same session: reconstruction of a session as one
  agent-readable artefact, and a telemetry file beside it. Both were folded
  into the intent rather than split out, because neither is a separable
  capability: an artefact nothing can read is not a reconstruction, and the
  telemetry is derived from the same bytes in the same pass. The cross-repo
  ingest seam, initially routed out as its own record, came back INTO the
  intent when the product thinker specified that reconstruction is repo-wide
  and covers the history already on disk. The candidate principle was not
  filed; it reads as the ADR's rationale rather than a standing stance.
- **Verdict:** SPLIT, but a narrower split than proposed. One intent
  (capture, ingest, reconstruction, telemetry), one ADR (lineage), one issue
  already filed as the source record. Typed links: `refines itd-59`,
  `refines adr-29`.
- **Notes:** the first run where the initial routing was too aggressive rather
  than too coarse. Three of the five parts were re-merged into the intent by
  the human, and the re-merge was correct each time: the parts were stages of
  one user-visible capability, not separable capabilities, and splitting them
  would have produced records that could not ship independently. The table
  caught the one part that genuinely was a different record type (the lineage
  decision), which is what it exists for. A reversal flag was raised and stands
  advisory: itd-59 records interactive-session capture as already solved, which
  the measurement contradicts for everything below the main thread. The orphan
  case (transcripts whose repository no longer exists, about a quarter of the
  projects here) surfaced only during the interview and not in the initial
  routing, which is a gap in the pre-pass worth noting: the table asks where
  each part lives, not what the part cannot represent.

### 2026-09-11 — lifecycle symmetry across record families

- **Proposal:** all artefacts must be consistent where possible: issues, specs
  and intents should close the same way, and since issues are minted to avoid
  conflicts, specs, intents, ADRs and everything else should be minted the same
  way.
- **Table:**

  | Part | Type | Home |
  | --- | --- | --- |
  | Every write-side family mints through one allocator | already shipped | verify only (adr-45, `recordid.Minter`) |
  | `CLAUDE.md` states ADRs keep a hand-numbered ordinal | defect | issue `iss-2609111002410678` |
  | Terminal moves are asymmetric across families | capability | intent `itd-2609111003026787` |
  | Supersede has no verb for intents or ADRs | capability | same intent (maintainer's routing) |

- **Links:** `related_adrs: adr-45`, cited by analogy. The first routing said
  `refines`, which is wrong twice: `refines` is not a field the schema knows
  (`internal/core/lint/schema.go:113-115`), and adr-45's five rulings are about id
  allocation, which this narrows none of. `related_intents: itd-34` was MISSED at
  routing and is the run's blocking defect — see the correction below.
- **Corrected after the two adversarial reviews (2026-09-11).** The run's
  routing was WRONG in one part and the table's own method produced two false
  numbers. Recorded here because the calibration corpus is worth less if it only
  records the runs that went well.

  **The blocking miss:** the table asserted "Supersede has no verb for intents or
  ADRs". Half wrong. `itd-34` sits in `planned/` and already specifies
  `/abcd:intent reclassify --kind superseded --by`, with a richer contract
  (`kind_at_supersession`) than the draft proposed. The run's own stated lesson —
  check whether each part is already shipped before routing it — was applied
  against SHIPPED CODE only, never against PLANNED RECORDS, and that is exactly
  the blind spot that produced the duplicate. The lesson generalises further than
  it was written: check the record store, not just the tree.

  **Why nothing caught it:** the draft omitted `## Prior Art`, which
  `intents/README.md:206-209` makes required, and which exists precisely to
  surface this. Four of the five minted-era drafts carry it; this was the only one
  without.

  **Two false numbers**, both from counting directory entries rather than
  records: "7 superseded intents" (6 records plus a `README.md`) and "60 ADRs
  carry a superseded edge" (5 carry an edge; 60 was the file count, and every ADR
  carries `superseded_by:` as a null-initialised schema field written at mint by
  `decide.go:222`). A count of files is not a count of acts.

- **Verdict:** SPLIT, confirmed by the maintainer, who chose one intent plus one
  issue over three offered alternatives (a single intent carrying the minting
  audit as a criterion; two intents split by act; hold and file nothing).
- **Notes:** the first run where a whole half of the proposal was found ALREADY
  SHIPPED. The minting half needed no record: every write-side family already
  holds a `recordid.Minter` and names its family tag, `core/decide` last on the
  2026-09-01 ruling. The corpus reads as mixed (295 sequential ids against 34
  minted) only because minting is forward-only, which is a property of ids as
  citations rather than evidence of a split surface — a distinction a table that
  counted filenames would have got backwards.

  That finding was only reachable by measuring the tree and reading the package
  map before routing. A decomposition run from the proposal's own words would
  have filed an intent to build what exists. Worth generalising into the
  protocol: check whether each part is already shipped BEFORE routing it, not
  after.

  The run also turned up a defect the proposal did not mention and could not
  have: `CLAUDE.md` asserts the exact opposite of the shipped minting behaviour,
  in the file the rules loader puts before every session. The table caught it
  because verifying "already shipped" meant reading both the code and the rule
  that describes it, and they disagreed.

## 2026-09-15 — four release-cycle deferrals elevated to intents

- **Proposal:** four findings deferred at the v0.9.0 cut as capabilities rather
  than defects: the release flow for a non-plugin managed artefact, an errata
  disposition for durable records, a surface counting owed fidelity reviews, and
  a claim record for which session holds which worktree.
- **Initial routing:** all four to intents, each promoted from its issue so the
  back-edges are stamped rather than narrated.
- **Confirmed routing:** the product thinker confirmed all four as intents in one
  instruction, after reviewing the deferrals classed by the reason each was not
  fixed. No part was split out: none carries a trust rule or a stance separable
  from the capability, and the plumbing each needs lives inside its own spec.
- **Verdict:** FILE-AS-IS on all four.
- **Notes:** the first run in this corpus where the routing question was asked
  of a whole class at once rather than per proposal, and the class had been drawn
  by the reason for deferral rather than by subject. The routing held because the
  class was already "capability wearing a defect's clothes": the classification
  did the decomposition's work before the table was drawn. Worth noting as a
  pattern: a deferral classed honestly is most of a routing.

## 2026-09-15 — append-only logs conflict on every merge (iss-2609100507439414)

- **Proposal:** a managed repository's `.abcd/work/DECISIONS.md` conflicted on
  the first two of 27 branch merges in a single day's autonomous run, and
  `CHANGELOG.md`'s `[Unreleased]` section conflicted on four of the eight merges
  that carried an entry. abcd had answered both halves for itself — a union
  merge attribute for the log, a derived changelog for the cut — and neither
  answer travelled to a repository it manages.
- **Initial routing:** one capability, framed as propagation. Scaffold the union
  attribute into a managed repository at adoption time, and give the changelog a
  per-change fragment directory; the shape change to the decisions log was listed
  as an alternative in the source record's own "wanted" paragraph, costed as "a
  new id family" and not routed anywhere.
- **Confirmed routing:** the product thinker reframed the question before
  routing it. Not "how does abcd propagate its two workarounds" but "why do these
  two files conflict when five record families did not" — and the answer is in
  the run's own positive finding, iss-2609100508570803: the two that conflicted
  are the two single append-to-the-bottom files, and the five that did not are
  all one file per entry. That makes the capability a shape change (the decisions
  log becomes a folder of minted records with an assembled index, in abcd and in
  every managed repository) and lifts a shape rule out of it (a decision is a
  record, minted like every other, stored one per file; the index is derived and
  never appended to). Capability → itd-2609151138388536; shape rule →
  adr-2609151138420062. The changelog half was NOT folded in: union is the wrong
  remedy there rather than an unapplied one, and abcd's own answer to it —
  derive the changelog from records at the cut — is a capability a managed
  repository cannot yet use, which is itd-2609150819432059, already filed.
- **Verdict:** SPLIT. One intent, one ADR, the source issue and its positive
  sibling as the evidence, and an existing intent named as the companion that
  closes the other class. Typed links: `related_adrs` adr-2609151138420062,
  `refines` adr-45 (a sixth caller of the id seam, unchanged), `promoted_from`
  iss-2609100507439414.
- **Notes:** the routing changed because the question changed, and the table did
  not catch it — the table asks where each part lives, and every part of the
  proposal as framed lived in one capability. What surfaced the better question
  was the deferral reason already written on the record at the v0.8.0 cut, which
  said in as many words that scaffolding a merge attribute into somebody's
  repository unasked is the act this project's own rule about user-owned space
  tells it to think twice about. A deferral classed honestly named the objection
  to the workaround, and the objection is what made the shape option visible. The
  2026-09-15 run above noted that a deferral classed honestly is most of a
  routing; this is the sharper form of the same observation — it can also be most
  of a reframe. Second note: the evidence for the split was a positive finding.
  iss-2609100508570803 was filed as an observation, not a defect, and it is the
  measurement the whole decision rests on; had it not been recorded, the
  comparison between the two shapes would have had to be reconstructed from
  memory.

## 2026-09-15 — closing a spec ships its intent (iss-2609100508566552)

- **Proposal:** `abcd spec close` moves the spec to `closed/` and, as its
  close-hook, the intent from `planned/` to `shipped/`. A session in an
  autonomous run met a spec that was complete against an intent whose acceptance
  criteria were roughly half met, and stopped rather than close — correctly, since
  every move available to it would have produced a false record.
- **Initial routing:** three options, all of them capabilities, all of them
  changes to what the verb does. Close the spec while leaving the intent planned,
  with the reason recorded on the intent; or split the intent at close, minting a
  successor intent for the unmet criteria and shipping the delivered half; or,
  failing both, make the close refuse — or demand an explicit acknowledgement —
  when the criteria are visibly unmet. The first two are the source record's own
  "wanted" paragraph; the third is its fallback.
- **Confirmed routing:** the product thinker reframed the question before routing
  it. Not "what should the verb do when an intent is half delivered" but "what is
  the relationship between an intent and its specs" — and the answer is 1:n. An
  intent that has been thought through stands as written, so a spec that delivers
  part of it is closed on its own terms and a new spec is minted for the
  remainder and attached to the same intent; the intent ships when its last spec
  closes. That makes every one of the three initial options wrong in the same
  way: each keeps the 1:1 link and buys honesty by narrowing something — the
  intent, the shipped claim, or the operator's attention. Because it changes what
  closing a spec means everywhere rather than adding a capability, the confirmed
  routing is a DECISION RECORD plus one brief invariant and no intent:
  adr-2609151513118583, invariant 17. The source record stays open, annotated
  with the ruling, because the build is owed.
- **Verdict:** RULE, NOT CAPABILITY. One ADR, one invariant, zero intents. Typed
  links: `related_intents` itd-80 (the record that shipped the close-hook this
  narrows), `related_adrs` adr-26 and adr-31; the motivating issue and its
  adjacent finding iss-2609091732329046 are cited in prose, the shape an ADR
  already uses for an issue, since no frontmatter field carries that edge.
- **Notes:** the table did not catch this and could not have, which is worth
  recording precisely. It asks where each part lives, and all three proposed
  parts lived in one verb — they differ only in what `spec close` does next, so
  every decomposition of the proposal as framed returns the same home. What
  surfaced the better question was asking what the RECORD should be able to
  represent rather than what the verb should do: the state the field session was
  actually in — work finished, capability not yet whole — had no representation
  at all, and once that is the question the cardinality is the answer and the
  verb behaviour falls out of it. Second note: the rule arrived with a cost that
  a pure-capability routing would have hidden. Three readers assume one spec per
  intent, and one of them is the release cut's stale-intent refusal, which under
  the new rule would refuse a cut during exactly the partial delivery the rule
  exists to permit. Naming that in the ADR's consequences is what stops the
  invariant reading as free.

## 2026-09-15 — the committed banlist cannot exist when a repo most needs it (iss-2609100506269348)

- **Proposal:** on a fresh PUBLIC repository the committed banned-names layer
  cannot be created, because the visibility fence narrows only on positive
  evidence of tracked record files and the fence is what prevents that evidence
  from ever existing. The window in which the layer is unavailable is exactly the
  window in which a repository is being set up to ban a name.
- **Initial routing:** none, and that is the finding's own state. Three candidate
  reconciliations were left for a human to pick between in iss-176 — move the
  config outside the fenced namespace, carve a single un-ignore into a table with
  no exceptions, or declare the committed layer private-visibility-only — and the
  intent the adjacent record iss-223 was promoted into, itd-159, has sat as an
  unfilled skeleton since. The proposal as it reached the routing question was
  therefore "pick one of three", with the fourth shape (a committed-record
  declaration that suppresses the fence) stated in iss-223's own title and routed
  nowhere.
- **Confirmed routing:** the product thinker took the fourth shape and added a
  second capability the tree does not have. (a) A committed declaration lifts the
  fence, so the evidence the narrowing waits on becomes something a repository can
  give on its first commit. (b) A machine-global private banned-names list in the
  user-level home, applying across every repository on the machine — their own
  framing, from the observation that the home already exists and holds the other
  machine-scoped stores. It was made explicit before the ruling that a home list
  cannot be seen by CI and therefore cannot rescue the committed layer; they chose
  BOTH halves with that limit stated, which is what makes them one record rather
  than two. Capability → intent promoted from the issue, itd-2609151516525843,
  `impact: additive`, left in drafts. A shared or multi-user home was held OUT of
  scope as the product thinker's own open question.
- **Verdict:** FILE-AS-IS, one intent carrying both halves. No rule was split
  out: the trust rule this case sits under is already ruled — adr-56, an
  exclusion control asserts only what it can prove — and the intent `refines` it
  rather than restating it, since what changes is that the control is given a
  fact it CAN prove, not what it may assert. Typed links: `builds_on` itd-74 (the
  shipped two-layer banlist), `refines` adr-56, `promoted_from`
  iss-2609100506269348.
- **Notes:** the run's most useful signal is a variant of the table's
  "is the home already occupied" question, which has now turned three consecutive
  hand-runs. Here the home was occupied by a SKELETON: itd-159 exists, names the
  right shape in its title, and contains nothing — no press release, no criteria,
  no mechanism. An occupied home that is a skeleton reads as coverage to a
  scanner and as nothing to a builder, and the table as drawn cannot tell the two
  apart. Second note: the widening again came from the product thinker rather
  than from the decomposition, as it did on 2026-09-09, and again it was correct
  to fold rather than split — the two halves are one user moment (declare a name
  you must never publish, and have the declaration take effect), and the second
  half cannot be stated honestly without the first half's limit beside it. Third:
  this is the first entry in the corpus where the initial routing was not a
  proposed decomposition at all but an unmade choice left in a record, which is
  its own failure mode — three candidates and no owner is indistinguishable, from
  the outside, from a question nobody has asked.
### 2026-09-15 — interview context shown before every question (hand-run at filing)

- **Proposal:** whenever a user is interviewed, provide the appropriate
  context: before asking whether acceptance criteria stand, show them; for
  every interaction during the interview.
### 2026-09-15 — messaging between sessions on one machine and one local network (hand-run at filing)

- **Proposal:** the best way for a local setup that lets agents communicate
  across user accounts and machines on the same network; whether Hermes or
  OpenClaw answer it; whether abcd can carry it, given that abcd comes with
  basic functionality and an external dependency brings full power.
- **Table:**

  | Part | Type | Home |
  | --- | --- | --- |
  | The material under decision is shown in full before each question, at every interview step | capability | intent `itd-2609151541116052` |
  | "Never ask about a text the reader cannot see" as a standing rule | stance | GRILL domain in `.abcd/rules.json`; the bundled default when itd-201 ships |
  | The `/abcd:intent` interview steps say "summarise back" and "walk every bullet"; make "show it first" explicit | plumbing | `commands/intent.md` |

- **Links:** `refines itd-201` (the draft that governs how a question is asked:
  one at a time, one sentence of context, an example per option). No reversal.
- **Verdict:** proposed SPLIT-or-AMEND with four options offered (new intent
  refining itd-201; amend itd-201's draft; GRILL rule only; decide later); the
  maintainer chose a NEW INTENT. Routing survived as one of the offered options,
  not as the table's first line, which had left the choice open.
- **Notes:** the proposal came from the maintainer's own experience of being
  asked "does this criterion stand?" with the criterion out of view. The stance
  and plumbing parts are not filed with the intent; they follow when it is
  planned, because a rule written before the intent it enforces is the shape
  itd-201's own filing avoided.

### 2026-09-15 — advisories publish with the release that fixes them (hand-run at filing)

- **Proposal:** a process so that security advisories are captured, resolved,
  and automatically published once a new release is cut with their fixes;
  perhaps captured as an intent.
- **Table:**

  | Part | Type | Home |
  | --- | --- | --- |
  | The release cut publishes each draft advisory whose fixing record shipped in the cut, and closes one whose record was closed as won't-fix | capability | intent `itd-2609151658486398` |
  | A security record names its advisory in a typed field, not only in its slug and prose | plumbing (schema) | issue `iss-2609151703088754` (split out at the routing) |
  | Publishing needs the forge token; `launch ship` never publishes, so the act belongs to the release workflow | trust rule, already recorded | unchanged; a scope condition on the intent |
  | An advisory's specifics stay out of the committed record until it publishes | stance, already in the pilot note of 2026-08-27 | unchanged |

- **Links:** `builds_on itd-70` (release retention at ship time); the pilot
  note of 2026-08-27 names this publication step as its target and is cited by
  the intent. No reversal.
- **Verdict:** four options offered (one intent; SPLIT into intent plus schema
  issue; HOLD for one more hand-run; decide later); the maintainer chose SPLIT,
  so the typed field can land first on its own. The table had folded the field
  into the intent's spec; the routing moved it out.
- **Notes:** the trigger was a concrete gap found the same day: GHSA-gx3m is
  fixed in v0.8.0 and was still an unpublished draft five days after the tag,
  published by hand at the maintainer's word during this run. The three
  advisories triaged today ended in three different states (fixed and released;
  fix in progress; not a vulnerability, won't-fix), which is the case split the
  intent's criteria have to cover.
  | A shared-directory mailbox between sessions, across accounts and mounts, delivered by the prompt hook | capability | intent `itd-2609151838312703` |
  | An opt-in adapter to an embedded broker for push delivery and cross-machine reach | capability | intent `itd-2609151838327688`, sequenced after the mailbox |
  | "abcd comes with basic functionality; an external dependency brings full power" | stance | principle `basics-built-in-adapters-bring-power` |
  | The survey with its sources and rankings | reference | research note `2026-09-15-agent-messaging-on-a-local-network` |
  | A message from another account is untrusted input: signed, size-capped, sanitised, injected only from a mailbox the recipient owns | trust rule | an ADR at the mailbox's planning; not filed yet |

- **Links:** the adapter `builds_on` the mailbox; both refine the
  host-delegated-by-default boundary in the conventions. No reversal.
- **Verdict:** four options offered (SPLIT into two intents plus principle
  and note; one intent covering both halves; HOLD with the note and the
  principle only; decide later); the maintainer chose SPLIT. The table's
  routing survived as offered.
- **Notes:** two constraints arrived mid-run and reshaped the survey before
  it returned: local machine and local network only, and the
  basics-plus-adapter stance. The first removed the harness's own
  cross-machine path (vendor-relayed) from consideration; the second turned
  a ranking of transports into a pair of rungs. The named tools both turned
  out to be agent runtimes with their own peer protocols, not message paths
  for a foreign harness, which is the finding that made "build it into abcd"
  the answer rather than "adopt one of them".

## 2026-09-18 — per-task oracle routing (itd-2609170822093401, promoted from iss-2609170818061083)

- **Proposal:** configure the oracle choice by task (local model for some,
  harness decides for some, frontier or economy for others) and configure
  whether the harness may use sub-agents and how many. Filed as an issue,
  promoted to a draft, then two independent adversarial reviews
  (design/feasibility; record discipline) before the interview.
- **Table (reviewer-proposed, then confirmed):**

  | Part | Type | Home | Typed link |
  | --- | --- | --- | --- |
  | Routing rows: tier and fan-out per agent, layered bundled → machine → repo → `--route` | capability | this intent | refines adr-25; builds on itd-2, itd-2609081951381895 |
  | Every ingested payload names its model and agent count; refuse silence | capability | its own intent, `itd-2609180517121254` | this intent builds on it |
  | Fan-out ceiling declared on the agent contract, a row may only tighten | trust rule | agent contract (`capability_scope`); enforcement rides the provenance intent | refines adr-2609090636172016 |
  | Refuse-vs-fall-back on an unwired backend | standing stance | loud-staging, applied: fall back to the harness, announced on stderr and the receipt | — |
  | Which settings a provider accepts; refuse an unsupported one | plumbing of the adapter | `itd-2609081951381895` | — |
  | Learned routing across repos | capability | itd-17 refines this (declared rows are the floor) | refines |

- **Verdict:** SPLIT (provenance out to its own intent; fan-out enforcement
  and provider settings to their homes; learning stays itd-17). The human
  adopted the split, then reframed the capability twice in the same
  interview: from operator-written rows the host is asked to honour, to a
  proposal abcd ships and applies on consent with best-effort facilitation
  through configured providers and the harness as the loud fallback; then a
  per-invocation `--route` override for autonomous measurement runs, and
  provider settings layered connection → row → flag.
- **Routing survived?** Partly. The initial key (task class) fell to the
  design review (uneven distribution, no binary schema, proactive agents
  unroutable) and was re-taken twice — verb, then agent — as the frame
  moved from operator-written to abcd-proposed. The near-duplicate with
  itd-17 was resolved as `itd-17 refines this`. The fallback ruling
  reversed mid-interview (refuse → fall back loudly) when "harness always
  as fallback" was stated. The adr-25 reversal flag was ruled a refinement
  on the ground that acceptance of the proposal is the opt-in.
- **Notes:** the first run where the two reviews converged on the same
  finding (the binary has no enforcement site on a host-delegated step; the
  honest product is provenance) and the human's reframing dissolved it by
  adding a leg the binary does own. Reviewer evidence that the repo-wide
  `oracle.backend` is read by nothing was decisive and would not have
  surfaced from the record alone.

## 2026-09-20 — the implement verb (itd-2609201916151817)

- **Proposal:** "a new verb (implement) as a fully autonomous implementation
  run for a single intent, verifying that all design decisions have been
  made, then plan, implement, and validate autonomously using sub-agents
  (configurable), incl. for orchestration, implementation, validation;
  multiple agents using either claude CLI and/or opencode/openrouter models."
- **Table (as proposed):**

  | Part | Type | Home | Link |
  | --- | --- | --- | --- |
  | One verb runs a READY intent to delivered through orchestrator, implementer and validator sub-agents, configurable | capability | this intent | refines itd-29, itd-50, itd-2 |
  | "All design decisions made" as a precondition: no open question, no unanswered claim, not held | gate | refines the readiness gate; the answer verb and the held state are iss-2609200830002711 and iss-2609200830076665 | refines |
  | Validator independent of implementer | stance | evaluator-outside-the-loop, already a principle | cited, nothing new |
  | Agents run through the claude CLI or opencode/openrouter | capability | refines itd-2609170822093401 (per-agent model tier); the opt-in cli oracle rung | refines |
  | "Plan … autonomously" | reversal | the intent page rules `plan` never unattended; flagged for the human | reverses (flagged) |

- **Verdict:** SPLIT. The human ruled the reversal a flagged path (`--plan`
  allowed for a draft whose decisions are all recorded, default refuses,
  ADR owed before that path ships), routed the provider part to its own
  draft (itd-2609201916056194) that this one builds on, and adopted the
  four refinement links.
- **Routing survived?** Yes, with one widening: the proposal's "plan"
  became a flagged path rather than being struck or adopted whole, which
  neither the table's "reverses" nor a plain "refines" would have named;
  the three-way question (never / always / behind a flag) was what let the
  human answer it. The provider part went to a separate draft rather than
  into the planned model-tier intent, on the ground that a remainder spec
  on a planned intent widens what a human already signed off.
- **Notes:** the first run where the field evidence preceded the filing:
  four managed-repo sessions had hand-built the verb's shape (coordinator,
  implementer, three reviewers, adjudicator, pause-and-resume) in the four
  days before, and one of them asked for the recipe outright. The
  decomposition was faster for it; every part had a record to land on.

### Run: `abcd build next` and `abcd drain` (2026-09-21, product thinker's interview)

- **Proposal (as received):** "abcd implement next and abcd implement drain
  as two customised runs: 'next' asks the agent to pick the next intent to
  implement autonomously (and to record why it was picked) and 'drain' drains
  the ledger of issues that do not require a design decision. Conduct SOTA as
  part of the plan to write the intent."
- **Table (as proposed):**

  | Part | Type | Home | Link |
  | --- | --- | --- | --- |
  | A run that picks the next READY intent itself and records why | capability | new draft (itd-2609211116005482) | builds_on itd-2609201916151817 |
  | The ordering the pick uses | capability | itd-78 already claims the derivation | builds_on or refines itd-78 |
  | A run that drains the ledger of no-decision issues | capability | itd-82 (draft) is this idea; supersede or revive | supersedes itd-82 (flagged) |
  | The rule for "needs no design decision" | trust rule | ADR + brief invariant, owed before the path ships | refines the human-owns-decisions boundary |
  | "A machine-made choice records a falsifiable rationale" | stance | principle | new |
  | `implement next` already names the step interface | plumbing | the implement spec: rename the step or the run | edit |

- **Verdict:** SPLIT. The human revived itd-82 rather than superseding it
  (so the flagged supersession became an edit), filed the pick as a new
  draft that refines itd-78 (filters on edges, ranks on none), kept the
  trust rule as a decision record owed before ship (recorded as the drain
  draft's decision 4, the `--auto-plan` pattern), and folded the stance into
  the new draft's decision 5 ("computed, never composed") with no principle
  file written. The plumbing part became a ruling on the planned implement
  intent (decision 8): `build` for people, `implement` for the machinery,
  the step word renamed from `next` to `step`.
- **Routing survived?** Mostly. Two changes the table did not anticipate:
  the human renamed the whole family (`build` / `drain` / `implement`) from
  a naming question the table had filed as plumbing, and the drain's
  hand-back was routed by kind on the human's own objection that an issue
  is not always a user moment, which turned "promote to intent" into the
  itd-84 routing itself applied at run time. The principle part did not get
  a file; whether it deserves one is left to a later recall.
- **Notes:** the SOTA pass ran independently before the interview
  (evaluator outside the loop) and changed two answers the human would
  otherwise have been asked cold: the ordering (readiest-first, on the
  psychometrics and the dotnet/runtime evidence, over oldest- or
  severity-first) and the rationale's form (computed facts, on the
  audited-ADR failure rate). Questions asked: seven, one at a time; two
  answered with a clarification instead of an option, both taken as the
  answer. One capture filed for a part outside the proposal (the
  person/agent verb split, iss-2609211119023345).

### Run: the fourteen specless planned intents and the two run drafts (2026-09-21, product thinker's interview)

- **Proposal (as received):** "interview me on all remaining intents so that
  the autonomous run can include them, too" — scoped by the human to the
  fourteen planned intents with no spec plus the two drafts of the same
  morning (`abcd build next`, `abcd drain`).
- **Table (as run, one row per record; the routing was the first question
  each time):**

  | Record | Verdict | Home |
  | --- | --- | --- |
  | itd-24 reflect | FILE-AS-IS (re-planned) | intent; two open questions ruled |
  | itd-27 grill | RETIRE | superseded by itd-94 (the planning interview) |
  | itd-28 reviews | SPLIT | the pin and the staleness view kept as the intent; the store, the redaction and the verifier dropped to what the record already holds |
  | itd-34 kinds | SPLIT | the shipped kinds read, not built; the bundle command and the reclassify verb kept |
  | itd-36 memory | CLOSE AS DELIVERED | three remainders to the ledger |
  | itd-42 coherence | FILE-AS-IS (re-scoped) | the itd-84 pre-pass |
  | itd-48 fidelity roles | SPLIT | consistency pass kept; shape role to itd-34's lint; overlap to itd-42 |
  | itd-50 loop | FILE-AS-IS (re-scoped) | build's last stage |
  | itd-53 owed drainer | FILE-AS-IS (re-scoped) | one bounded command; the gate to itd-2609150819445595 |
  | itd-58 verdict | FOLD | an invariant on the build intent (decision 9) |
  | itd-6 RepoPrompt | FILE-AS-IS (re-filed) | one opt-in reviewer adapter |
  | itd-7 RP workspace | HOLD | waits on itd-6 |
  | itd-63 wizard | FILE-AS-IS | intent; three open questions ruled |
  | itd-69 lockstep | CLOSE AS DELIVERED | nothing owed |
  | build next | FILE-AS-IS after two reviews | intent; three rulings on the reviews' findings |
  | drain | SPLIT after two reviews | intent; the issue key to the build intent (decision 10); the eligibility rule to a decision record the spec mints |

- **Verdict:** nine re-planned and READY, two closed as delivered, two
  retired (one folded), one held, two planned after review. A question the
  human raised mid-interview (whether phases survive bundles) was routed to
  a new draft (itd-2609211913453478) rather than answered in passing.
- **Routing survived?** Largely. Three rows moved from where the table
  first put them: itd-58 went from "keep as its own record" to a fold, on
  the observation that the build intent lacked the invariant in so many
  words; itd-53 went from "retire" to "one command", on the human's want
  for the run's batch 0 to have a verb; drain's lane went from "the loop
  stays intent-only" to "the loop grows an issue key", on the design
  review's first finding, which the human ruled rather than the table.
- **Notes:** the two adversarial reviews on the run drafts changed the
  records more than the interview did (seventeen findings applied); the
  design lens caught false premises (a field that does not exist; a lane
  keyed to the wrong record family) that no question to the human would
  have surfaced. The "close as delivered" row is a shape the run file's
  batch 0 hunts for among READY intents and cannot reach among specless
  ones; iss-2609211738504433 records the gap. Fourteen records took three
  hours of the human's time at one question per turn.

### Run: the vocabulary rulings and eight new records (2026-09-21, evening, product thinker's interview)

- **Proposal (as received, in turns):** "do we need new intents for what we
  just decided?"; then CLI discoverability for humans and agents; the help
  sentence per verb; whether 53 verbs all make sense; the website for any
  managed repo; the status-line badge; doubles at filing; the abcd labs;
  never "the maintainer".
- **Table (as run):**

  | Part | Type | Home | Link |
  | --- | --- | --- | --- |
  | Retire phase, milestone, roadmap; two axes; issues have no spec; batch is internal | decision | adr-2609212115255771 supersedes adr-9 | reverses (confirmed) |
  | The map page, the glossary entries, the lint | capability | itd-2609211913453478 (planned) | builds_on itd-34 |
  | Now / Next / Later status block | capability | new itd-2609212103568351 | builds_on the map, the pick |
  | Steps below a spec | capability | new itd-2609212103565953 | builds_on the build machinery |
  | target_release | capability | new itd-2609212103572513 | builds_on the map |
  | Grouped help, agents block | capability | itd-146 widened and planned | related consolidation, sentence |
  | One sentence per verb | capability | new itd-2609212113220149 | builds_on itd-146 |
  | Verb consolidation | capability, breaking | new itd-2609212130136102 | related the four renames |
  | The badge guarded and reset | capability | new itd-2609212130146198 | builds_on itd-200 |
  | One verb to a live site | capability | itd-2609061543533170 written and planned | builds_on the status block |
  | Doubles linked at filing | capability | new itd-2609212137116617 | builds_on itd-84, itd-42 |
  | Never "the maintainer" | capability + lint | new itd-2609212137129937 (captured for the run, no sweep) | builds_on the badge |
  | abcd lab | capability | new itd-2609212137128014 from the capstone draft | builds_on itd-22 |
  | Four lab findings | issues | two captured (guard workdir; short banlist fragment), two already fixed | |

- **Verdict:** SPLIT throughout; one reversal (adr-9) confirmed by the
  human at the question, not inferred.
- **Routing survived?** Two moves against the first table: Now/Next/Later
  was first offered as a roadmap bucket and became a status block on the
  research pass's own challenge (a second name for the shelves); the badge
  was first offered as event-derived and the human ruled verb-set,
  guard-enforced, hook-reset. Steps were offered three ways and named by
  the human for the loop's word.
- **Notes:** every ruling was taken one question at a time; where the
  human answered with a clarification rather than an option, the
  clarification was taken as the answer and read back. The maintainer
  sweep is the first record of the day captured explicitly *not* to be
  fixed in the session that ruled it.

### Run: Jev, OpenRouter and the allowlist (2026-09-22, product thinker's interview after a research pass)

- **Proposal (as received):** "consider Jev as a new routing model (e.g. via
  openrouter); SOTA on which tasks Jev could help automate"; then "how to
  integrate openrouter and ensure only certain models are accessed via it";
  then classification and library selection as use cases; then the one-time
  setup at `ahoy` with the key kept out of the harness.
- **Table (as run):**

  | Part | Type | Home | Link |
  | --- | --- | --- | --- |
  | Provider adapters serve listed models only, under a vendor denylist | trust rule | adr-2609221009491186 + brief invariant | refines adr-25 |
  | OpenRouter integration | capability | itd-2609081951381895 written and planned | builds_on the tier |
  | The decision adapter, shadow as a lab, note as evidence | capability | new itd-2609221009495079 | builds_on the lab, the adapter, the tier; supersedes itd-17 |
  | Escalation on a failed fix round | rule | a decision on itd-2609170822093401 | |
  | The one-time setup with three key homes | capability | criteria on itd-2609081951381895 through itd-63's mode | builds_on itd-63 |
  | Model per role, the pick, consistency | not routing | no record | |
  | itd-17 (a learned router) | RETIRE | superseded by the decision adapter | |

- **Verdict:** SPLIT; the research pass changed the proposal's own framing
  (Jev is a typed-decision model, not a router) before any question was
  asked, and the human's questions widened the adapter's judgement list to
  the library and the memory store.
- **Routing survived?** Yes, with one move: the shadow run's evidence was
  first offered as a run-record rate and the human routed it through the lab
  verb filed the night before, which made the research note the evidence
  path rather than a new one.
- **Notes:** the human corrected the vendor mid-pass ("from typescript"), and
  the pass resolved it as a near-homophone with a primary source rather than
  guessing; a wrong vendor would have routed the whole proposal wrongly.
