# DECISIONS

Append-only, one line per decision, newest last. Date-prefixed. Architecture-shaping
decisions graduate to an ADR under [`../development/decisions/adrs/`](../development/decisions/adrs/).
Graduate this file to per-file `decisions/<date>--<slug>.md` if size or
parallel-agent merge contention bites.
The DA001-DA004 gates (`scripts/check-decisions-append.sh`) refuse any commit
that inserts, rewrites, or deletes a line below this header — the last entry
included — or introduces a NUL byte: correct an old entry by appending a new
dated entry, never by editing the old one in place. Redaction and the
graduation above are deliberate gate-edit-and-review changes: the pull request
that deletes lines also adjusts or retires the gate in the same diff, reviewed
together (the script's header says why there is no escape hatch).

- 2026-07-06 — Rebuild abcd from scratch in Go, no external tools (specstory,
  RepoPrompt, flow-next, Ralph, codex); ship an MVP, extend via the companion harness then
  Claude Code.
- 2026-07-06 — Transport-agnostic Go core; CLI is the reliable default front door;
  MCP is an additive front door on the same core, added later.
- 2026-07-06 — Peer with the companion harness via conventions + MCP; no Go dependency either way.
- 2026-07-06 — LLM work host-delegated by default; native/CLI/API/MCP oracles are
  opt-in adapters.
- 2026-07-06 — Spec/task layer native-minimal; the companion harness `ccpm` the primary deeper
  backend; flow-next dropped. Autonomous run not a Ralph port (host orchestrators).
- 2026-07-06 — Single repo, curated release (no dev→public mirror); the repo is the
  marketplace. Private companion repo deferred (trigger: shared transcripts).
- 2026-07-06 — Three-tier `.abcd/` layout: development (durable) / work (shared) /
  .work.local (local). `docs/` user-facing only.
- 2026-07-06 — Module path `github.com/REPPL/abcd-cli`; Cobra approved as the CLI
  framework (matches ferry and the companion harness).
- 2026-07-08 — Confidential sources: global user-level corpus (CSL-JSON + grep
  corpus, local no-remote git), append-only JSONL influence ledger per repo,
  banlist patterns generated from confidential entries into the itd-74 private
  guard; convention + skill first, `abcd source` verbs deferred (itd-76). Quarto
  chosen for eventual paper reconstruction; RAG rejected at this scale.
- 2026-07-08 — Personas in any scenario are always Alice, Bob, Carol (in that
  order); the user is they/them. Recorded as a principle.
- 2026-07-08 — Consume-model interview: `spec_id` is SCALAR (never a list) —
  split-the-intent is doctrine (itd-67/72 precedent); task decomposition lives
  inside the spec. Principles/disciplines get a promotion path: enforced
  principle ⇒ discipline-kind intent (personas principle promotes when its
  registry lint ships). Coverage vocabulary (uncovered / covered-shallow /
  covered-deep / orphaned / unwanted) lands as itd-53's gate reporting
  language; "done" = covered-deep AND the intent's own criteria MET.
- 2026-07-08 — persona_registry lint shipped (record-lint blocker: quote
  attributions must name registry personas); the personas principle promoted
  to discipline itd-79 the same change, per the promotion path — first test
  case of enforced-principle ⇒ discipline. principles/ file retired.
- 2026-07-08 — Persona SSOTs reconciled: `personas.json` is the single registry
  (13, expandable, alphabetical sequence); selection is BY ROLE, the role's
  registered name is used, never a name picked directly; all personas and the
  real user are they/them. Principle file updated to point at the registry;
  registry-membership lint is the intended gate.
- 2026-07-08 — Intents gain a required `## Prior Art` section (positions the
  intent against corpus + outside work; ≥1 resolvable reference or an explicit
  "none found — searched X"). Coherence stays at promotion (itd-42), whose
  Tier 2 now also loads `principles/`; capture stays severity + edges.
- 2026-07-08 — Edges stay one-way (dependent-authored), reverse views derived
  only; itd-78 lint rejects hand-authored reverse fields; edges gain optional
  content fingerprints (`itd-N@hash`) so a target's change marks inbound edges
  suspect. Intent doneness = spec closed AND the intent's own itd-1 criteria
  MET (never inferred from spec close) — full consume-vocabulary decision
  deferred to a follow-up interview.
- 2026-07-08 — itd-76 grilled: leak guard promises literal strings only
  (paraphrase risk stated, handled behaviourally + review); citation is a
  two-level AND (source permission_status AND per-line cited_publicly); author
  bans default on with per-source ban_authors opt-out; standalone `source`
  domain (itd-16 a possible backend, not a dependency); pre-commit auto-
  refreshes the generated banlist; public render proven by structural filter
  AND post-render lint; team share of citation data via committed
  `.abcd/work/references.json` (share/ingest); durability = machine backup +
  git bundle, multi-machine deferred.
- 2026-07-08 — `~/.abcd/` blessed as abcd's user-level home (fourth tier,
  additive to repo `.abcd/`), path configurable; relocation wizard recorded as
  itd-77.
- 2026-07-08 — Author bans FLIPPED to opt-in (`ban_authors: true`), superseding
  today's default-on decision: the actual corpus population (own submitted
  work, purchased reports, private repos) makes author bans near-pure false
  positives — they would ban the user's own name — while title/alias patterns
  carry the real protection.
- 2026-07-08 — Corpus restructured to class-segregated per-source folders
  (confidential/<key>/, public/<key>/): confidentiality is declared at
  ingestion and LOCATION is its single source of truth (flag mirrors, tooling
  refuses on mismatch); derived artifacts inherit by location; declassification
  is a visible git mv.
- 2026-07-08 — Severity ≠ priority (records an earlier-session decision):
  intents declare `severity` (capture-ledger enum) and edges (`blocked_by`,
  `builds_on`); effective priority is DERIVED via priority inheritance (max of
  own severity and severity of everything transitively blocked) and never
  stored — a minor blocker of a major intent jumps the queue while staying
  minor. Phases keep sequencing authority (adr-9); lint makes contradictory
  schedules fail. Recorded as itd-78; piloted on itd-76/77.
- 2026-07-08 — Predecessor spc-N artefacts inside intents (do-not-implement
  banners, implementation-complete AC tables) are demoted to Prior Art design
  input per the delivery-state provenance doctrine — never implementation
  authority, never a delivery claim (iss-16 itd-66, iss-17 itd-50); their
  deltas become spec-time Open Questions.
- 2026-07-08 — itd-37's itd-36 edge downgraded blocked_by → builds_on: the
  capture + enforcement half ships independently (Phase 0 registration) and
  only extraction-to-memory waits on itd-36 (iss-18); the launch deepenings'
  unscheduled state is recorded in the phase index pointing at adr-33
  (iss-20); itd-6 stays planned/ — ADR-25 superseded its framing only, and
  scheduled implies planned per adr-34 (iss-22).
- 2026-07-08 — Post-review recording follows fix-the-detector: findings are
  captured as clustered issues (iss-29..49), each naming the detector (gate,
  lint rule, or test convention) that catches its class and carrying its
  instances as the detector's acceptance corpus; instances drain behind the
  armed detector, never hand-fixed ahead of it. Ten principles recorded from
  the 2026-07-08 multi-agent review; distillation in research/notes.
- 2026-07-10 — The practice/MVP/tool trichotomy lands as an amendment to the
  principles README promotion path (one canonical three-rung ladder:
  principle -> enabling convention/script/format -> discipline-kind intent or
  core absorption), never as a third doctrine file — the adversarial review
  found standalone adoption would duplicate and contradict existing doctrine.
  Intake rules kept verbatim: articulate the full ladder for every candidate;
  never fabricate an absent rung (research/notes/2026-07-09).
- 2026-07-10 — Doctrine grows on observed need: the 31 deferred medium
  proposals from the extraction stay parked in the 2026-07-09 research note
  until a live instance arises; calibrate-the-judge deliberately waits for
  the first live LLM gate (its measured-agreement requirement is already
  recorded in verifier-selects-gates-decide's promotion path).
- 2026-07-10 — Public sources whose titles collide with locally-banned
  private names are cited by author + arXiv/DOI identifier, never by title
  or corpus key, in committed artifacts; the corpus ledger carries the real
  key. First instance: Tan et al. (UCL, 2026; arXiv 2604.09581).
- 2026-07-10 — AI-generated-only ("tainted") proposals are recorded as
  hypotheses and never adopted until independently verified against a
  citable source — the manual form of tier-travels-with-the-source (iss-52).
- 2026-07-10 — CONTEXT.md goes status-free: it keeps orientation and the
  live sharp-edges list only; hand-written phase/status claims are banned
  (extending adr-5's no-status-in-design-docs rule to the work tier) and a
  record-lint rule on .abcd/work/CONTEXT.md is the detector, armed before
  the rewrite per fix-the-detector. The content rewrite rides with iss-35's
  brief-vs-surface reconciliation. Rejected: deleting the file (loses the
  only committed shared home for sharp edges); generating it (a committed
  generated file is its own drift problem).

- 2026-07-10: Repo preparation is a plugin skill (`/abcd:prepare-this-repo`),
  superseding the external scaffold-repo script's entry point. Grilled rulings:
  the committed AGENTS.md working-conventions section is full-inline and
  NAMELESS (a pre-public repo name never lands in target repos) between dated
  markers for later tooling; the skill hard-refuses not-owned repos (no audit,
  no local layer — we don't impose our principles on others' repos); legacy
  root `.work/` layouts migrate propose-then-sign-off, never leaving two
  working-state homes; no re-run/update machinery now — the CLI will own
  managed-repo migration (gaps seeded as iss-84/iss-85, originally minted as
  duplicate iss-56/iss-57). Rejected: a standalone
  handover prompt file (drifts, unversioned); naming abcd in private-only
  target repos (two-class rule someone eventually gets wrong).
- 2026-07-11 — iss-35's brief↔surface cross-check is **bidirectional, but only
  the structural half is deterministically lintable**: Direction B (every
  `commands/`+`skills/` entry has a brief home) is a coverage lint like
  `directory_coverage`; Direction A (brief claims match *binary behaviour* —
  flags, exit codes, schema fields, counts) is irreducibly semantic and stays an
  LLM/agent job (encoding binary facts into the linter just moves the drift).
  So "graduate the detector to a record-lint rule" is a *reshaping* (extract the
  deterministic half; keep the semantic half as a periodic/agent check), not a
  port. The graduation is a design gate held for maintainer sign-off — options in
  `.abcd/development/plans/2026-07-11-iss35-record-lint-graduation.md` (recommend
  Option A, structural `surface_coverage` rule) — and it is **blocked** until the
  docs/history surface-taxonomy adjudication is decided (a coverage rule fires on
  the three chapterless shipped verbs the moment it is armed).
- 2026-07-11 — iss-35 graduation SIGNED OFF (maintainer, 4 decisions):
  (1) **Graduation = Option C (hybrid)** — build the deterministic
  `surface_coverage` record-lint rule AND wire the LLM cross-check as a standing
  release gate for semantic (Direction-A) drift.
  (2) **docs/history/version = user-facing surfaces** — each gets a
  `04-surfaces/` chapter + README row (resolves adjudication item 5).
  (3) **consult/ingest/prepare-this-repo reclassify skills → commands via
  relabel** — they stay host-delegated markdown workflows (no Go verbs; the
  "host-delegated by default" boundary holds), but the brief calls them commands
  with command-shaped homes; the read-only skill boundary rule is kept as-is — the
  skill *classification* was what gave (resolves adjudication item 6). abcd ships
  zero skills again.
  (4) **Push/merge policy** — the run's blanket "never push" was an
  unattended-safety override, not the standing rule; normal repo policy resumes
  when the maintainer is driving (docs/chore direct-to-main OK; feat/fix via PR
  awaiting their merge). Main pushed to origin; `auto/context-status-lint` opened
  as PR #12 (awaits maintainer merge; no auto-merge on a feat).
- 2026-07-11 — itd-3 rules-loader hook is **Go**, not Python. abcd is Go-only, so
  the `UserPromptSubmit` router is a Go subcommand invoked by `hooks/hooks.json` —
  the intent's `hooks/prompt_router_hook.py` is a stale pre-Go-rebuild detail and
  is superseded. No Python is added for the loader.
- 2026-07-11 — itd-3 rules-loader **design signed off** (plan
  `2026-07-11-itd-3-rules-loader.md`, prefer-sota verdict). Surviving shape: a
  transport-agnostic `internal/core/rules` capability with two front doors
  (`abcd rules [domain]` verb + `abcd hook prompt-router`), **not** an adapter
  seam. Four intent deltas approved: **D1** event-driven refresh on
  `SessionStart(compact)` (fixed-N demoted to a ~15–20 backstop, not primary);
  **D2** keep the shipped `{schema_version,disabled,domains{}}` shape (legacy
  `extends`/`overrides` sketch superseded); **D3** zero model-facing tokens on
  no-match + out-of-band diagnostic log (supersedes the "<200-token header"
  acceptance criterion); **D4** `.abcdignore` rejected for v1. Build proceeds
  phased/TDD from Phase 1 (`internal/core/rules`).
- 2026-07-11 — itd-3 **shipped manually** ahead of the intent-lifecycle pipeline.
  Moved `planned/ → shipped/` by hand with `spec_id: spc-1` (reserved — the future
  native spec store adopts spc-1 for itd-3, never re-mints it) and a hand-authored
  `## Audit Notes` (the `intent-fidelity-reviewer` agent does not exist yet; judge
  = Claude Opus 4.8). Rollup 3 MET / 1 MET_WITH_CONCERNS / 1 INCONCLUSIVE / 1
  NOT_MET; every divergence is a signed-off D1–D4 delta, the one gap is the AC6
  legacy-harvest completeness. Inbound links repointed planned→shipped by hand —
  the link-drift-on-move the future reconcile pass automates.
- 2026-07-11 — Intent-lifecycle slice 1 (build sign-off given): the pipeline is
  **dogfooded** — itd-3 stays shipped as the reference fixture (option b), and a
  new tightly-scoped intent **itd-80-intent-lifecycle-automation** (ACs = the
  steel thread) is the pipeline's first real payload, driven drafts→planned→
  shipped through the machinery it specifies. Slice scope: minimal native spec
  store (`internal/core/spec`, directory-as-truth open/closed, `intent:` link),
  `abcd intent` (plan/link/review-ingest + bare render) and `abcd spec`
  (close + bare render) verbs, deterministic reconcile inside `spec close`
  (no vendor event), host-delegated `intent-fidelity-reviewer` markdown agent
  (Role 1 only) + async outbox/inbox verdict ingest to `## Audit Notes`.
- 2026-07-11 — `spc-N` minting rule for slice 1: `max(N over spec-store files ∪
  N over every intent's spec_id) + 1`, so the first mint is spc-2 (itd-3's
  reserved spc-1 is respected without a backing spec file). Reconciling the
  store's sequential minting with the brief's aspirational spc-numbering is
  deferred to the richer spec-store slice. Reviewer roles 2/3 (itd-48),
  loop-to-acceptance (itd-50), bundle/discipline lifecycles, and the spec
  dependency graph are all explicitly deferred.
- 2026-07-12 — clean-slate hardening run STEP 0 triage. A fresh adversarial
  sweep (15 ruthless + 9 security reviewers over current `main`, every finding
  independently verified) returned 34 real findings (19 CONFIRMED, 15 PLAUSIBLE,
  0 REJECTED; full corpus `.abcd/.work.local/logs/clean-slate-run/sweep-findings.json`).
  Key result: the sweep INDEPENDENTLY RE-CONFIRMED the 2026-07-08 review's
  code-defect backlog (iss-29/30/32/33/34) is real and still unfixed — prior runs
  deferred those code fixes for docs-reconciliation (iss-35/36) and itd-80 feature
  work. Draining them is this run. Two BLOCKs found: the scanner serialises a
  finding's snippet masking only its own token, leaking sibling secrets on the
  same line (iss-65). Triage disposition: newer-package findings (scanner, rules,
  intent, spec, frontmatter, lint receipt-gate, capture concurrency, core) minted
  as iss-65..72; older-package findings map to existing homes — memory ingest
  (C12/C13/P11)→iss-30, atomic-write/fsutil (P1/P6/seed2)→iss-32, ahoy install
  (C2/C3)→iss-33, launch glob panic (C11)→iss-34, identity fail-open (C8/P12)→iss-63,
  history redaction (C6/C7/P2)→iss-29. iss-70's C16 fix adds `policy.detector` to
  the receipt-JSON schema — a record-lint CONTRACT change flagged for maintainer
  sign-off before landing (a STOP-adjacent design surface). Ledger triage committed
  to `main` as a `chore:` record commit (matches prior record-to-main practice;
  keeps the fix branches clean); each code fix lands on its own `auto/*` branch + PR.
- 2026-07-12 — iss-66 rules-loader trust boundary. Fixed the two mechanical items:
  the Load Lstat→ReadFile TOCTOU on `.abcd/rules.json` (now open-once O_NOFOLLOW +
  fstat, C19) and the session-state dir moved off the world-writable shared /tmp to
  the per-user cache dir (P14). **P15 document-accepted, NOT changed:** a per-repo
  `.abcd/rules.json` can set a default domain dormant and flip the global kill
  switch (Merge is intentionally per-field + sticky-kill-switch). Rationale: rules
  are an *opt-in, opinionated-but-overridable* config layer; `.abcd/rules.json` is a
  committed file (editing it needs repo write access, like any committed guardrail),
  and the real enforcement of dangerous actions is harness-level (git-guardrails
  hooks, the iss-62 identity gate, pre-commit), not the injected advisory prose.
  Silencing a domain removes prose, not a hard gate. **Deferred design alternative
  (surfaced, not taken):** introduce a protected "guardrail" domain class that a
  per-repo override cannot set dormant and that the kill switch cannot silence —
  this adds a new protected-domain concept to the rules contract, a maintainer
  decision, not an autonomous change.
- 2026-07-12 — iss-30 (memory ingest boundary) partially resolved: the fetch/read
  subset — C12 (HTTP status), P11 (SSRF NAT64/6to4), C13 (local size cap), the
  ~user tilde mangle — landed in PR #38. iss-30 stays OPEN for its remaining
  instances (the larger "ingest test-suite" effort): the --keep-original
  partial-failure reporting, CRLF parser-parity (parseFrontmatter vs
  splitFileFrontmatter), and broader URL-ingest/content-type/PDF path coverage.
- 2026-07-12 — /abcd:auto-loop design recorded (plans/2026-07-12-abcd-auto-loop-skill.md,
  pending sign-off, not built). SOTA pass (sota-researcher, primary sources) backs the
  design: durable handoff + fresh-context resume over compaction/RAG (Anthropic
  long-running-harnesses — compaction "isn't sufficient"); delegate reads/reviews but
  keep implementation in ONE agent (Cognition "Don't Build Multi-Agents" + Anthropic
  multi-agent — converging read/write boundary); reviewers must be a SEPARATE fresh-
  context lens, not intrinsic self-review (Huang et al. 2310.01798; CriticGPT
  2407.00215); gate irreversible actions on action-class not self-confidence (RLHF
  miscalibration); attempt-journal lineage = Reflexion (NeurIPS 2023) + database WAL.
  Rejected: parallel multi-agent implementation, compaction-as-primary-continuity,
  RAG-over-ledger at single-milestone scale.
- 2026-07-12 — autonomous-run surface named /abcd:run, taking itd-29's reserved
  name as the host-delegated realization of its operator surface (not a parallel
  /abcd:loop). Discovery: itd-29 (autonomous-run-resilience, planned) already owns
  this surface over the ADR-27 run seam, already scopes out-of-band-merge/chain
  reconciliation (host-owns-git MVP → future read-only `abcd run reconcile --json`)
  and 429/quota (spc-35), and is deliberately deferred pending real evidence
  (revisit trigger #5: two end-to-end autonomous runs). Sequence C→A: run the loop
  as a plan+protocol under the harness loop now to dogfood + generate that evidence;
  formalize commands/abcd/run.md + brief row + surface_coverage, reconciled into
  itd-29, after 1-2 successful runs. Binary operator verbs (budget preflight, rewind,
  ship, run reconcile) stay deferred in itd-29.
- 2026-07-12 — Judge calibration captured as a DISCIPLINE (itd-81), not a standalone
  intent: verdict-rendering agents are plumbing (no user moment), and itd-5 is the
  precedent for a cross-cutting rule over agent prompts. Core rule: no judge ships
  unmeasured — a labelled corpus with known-good cases ≥40%, scored on true-negative
  rate as a first-class metric alongside recall, with a declared TNR floor gating the
  prompt lock. Evidence: LLM code judges systematically over-flag and ~1/3 of their
  errors are hallucinated code (2603.00539); judges over-rate LLM-written and
  under-rate human-written code (2507.16587); ground truth is manufactured by
  injecting defects (CriticGPT 2407.00215). CORRECTS itd-5: its pre-flight tiebreak
  ("passes goldens AND >10% shorter") selects for the brevity bias that ACE
  (2510.04618) identifies as destroying instruction quality — struck; the gate is the
  corpus score. CONSTRAINS itd-64: reviewer verdicts are not ground truth (the
  reviewer is the instrument under measurement) and its tuning loop must stay
  human-gated — unattended proxy-optimisation reward-hacks at 73.8% (OpenReview
  ikrQWGgxYg). Rejected: judge panels/juries (nine judges → 2.18 effective votes,
  correlated errors, no better than the single best judge — 2605.29800); 1-5 severity
  scores (middle-drift, position bias); reasoning inside a JSON schema (2408.02442).
- 2026-07-12 — itd-5 AMENDED (not superseded) per itd-81, two rules: (a) the v1.0.0
  pre-flight's "shorter by >10%" tiebreak is STRUCK — length selects for ACE's brevity
  bias, and it selected against goldens that never measured false positives; the gate
  is now the calibration-corpus score, ties to the candidate. (b) `1.0.0` now MEANS
  measured — an agent stays in the `0.x` band until it clears a corpus, because
  stamping 1.0.0 on an unmeasured prompt asserts a lock that never ran. All five
  shipped agents are `0.1.0`.
- 2026-07-12 — The four personal reviewer agents (ruthless, security, docs-currency,
  sota-researcher) MOVED from the machine-global `~/.claude/agents/` into abcd's
  plugin `agents/` and deleted at source; they now resolve as `abcd:<name>` in every
  repo with the plugin enabled, versioned in-repo and reviewable by PR. Frontmatter
  key is `prompt_version` (itd-5's name), not `version` — intent-fidelity-reviewer
  renamed. Colour encodes the DOMAIN EXAMINED, never rank or taste: red=trust
  boundary, orange=code correctness, blue=documentation truth, green=the record,
  purple=external evidence; cyan reserved for artefact-producing (non-verdict) agents.
  Accepted cost: the reviewers no longer resolve in repos without abcd installed.
- 2026-07-13 — Auto-merge is permitted ONLY to a non-protected trunk, gated on a SHIP
  review *verdict* (not merely green CI) + lint/smoke + an audit entry; never to `main`
  (explicit human `abcd spec ship` promotes). A bounded, opt-in reversal of the standing
  "a human merges" default — safe because the merge target is staging, not the protected
  branch, and the gate is a verdict, not a checkmark (green CI shipped a real leak during
  the 2026-07-12 drain; a security review then HELD it). Record homes: experience → itd-29
  (already scoped, deferred v2); enforceable form → a brief invariant + an ADR *when built*,
  not now (capture-now-build-later). SOTA is itd-29's (GitHub-native auto-merge, host-owns-
  git, no new dep); the ADR inherits it. Surfaced the `facilitator-default-thinker-optional`
  principle.
- 2026-07-13 — `abcd audit` (itd-85): a new read-only repo-conformance verb, distinct from
  `ahoy doctor` (doctor = tool-setup health, audit = does-the-repo-conform). Bespoke on
  `internal/core/lint` (adapt repolinter's rule-schema vocabulary + Conftest severity/exit
  codes + SARIF as an optional export), zero new deps → no dependency gate. v1 = five rules
  (three-tier-layout, conventions-router, decision-durability, docs-currency, privacy-hygiene);
  SARIF deferred to P3; wires into `prepare-this-repo` Phase 2, closing `iss-86`. SOTA-researched
  in plan `2026-07-13-abcd-audit-verb.md`.

- 2026-07-13 (itd-85 M1): kept `core.exists` (bool-only, swallows errors) and
  `ahoy.fileExists` (regular-file-only) as-is rather than folding all three
  `exists` copies into `fsutil.Exists`. Chose partial consolidation over the
  plan's full-consolidation because the other two hold different contracts;
  merging them would smuggle a behaviour change into a behaviour-preserving
  refactor. Only `lint.fileExists` (identical fail-closed contract) migrated.
- 2026-07-13 (itd-85, carry to M3): `gitutil.CheckIgnored` fails OPEN — git
  absent or not-a-repo returns "nothing ignored". The `three-tier-layout` rule
  MUST treat an empty result as "cannot tell", never as "compliant", or a repo
  with git unavailable silently passes the "is `.abcd/.work.local/` gitignored"
  assertion. Security review flagged this as the one consumer-side spec note.
- 2026-07-13 (itd-85 M2): audit engine uses severity vocabulary error|warn|off
  (repolinter/Conftest), NOT the record-lint engine's blocker|warn, because it
  maps directly onto the tri-state exit code (error->2, warn->1) and reads right
  in a human render. Reused docs-lint findings (blocker|warn) get mapped to
  error|warn at the docs-currency rule boundary in M3, not in the engine.
- 2026-07-13 (itd-85 M3): privacy-hygiene uses a deterministic, identity-INDEPENDENT
  absolute-path regex, NOT the identity-aware scanner (internal/adapter/scanner).
  Rejected the scanner because its home-path detection is identity-PARAMETERISED
  (kindHomeSelf=hardfail vs kindHomeOther=warn) — machine-dependent severity —
  whereas AC3's contract is "ANY absolute local path is an error", deterministic
  across machines. The scanner also scans the release BUNDLE (a curated allowlist
  excluding tests); audit scans all tracked files, a scope the scanner was not
  built for. Flagged for future consolidation: absolute-home-path detection now
  lives in two predicates (scanner identity matchers + audit regex); a later phase
  should extract a shared identity-independent path matcher.
- 2026-07-13 (itd-85 M3): docs-currency emits every finding at warn, downgrading
  docs-lint blockers, because audit is an advisory conformance surface and the
  authoritative docs gate is `abcd docs lint` (still exits 2 on a blocker).
  Re-raising a docs blocker as an audit error would double-gate the same check.
- 2026-07-13 (itd-85 M3): three-tier-layout does NOT require .abcd/.work.local/ to
  be present (diverges from the plan's literal "present and gitignored") — it is
  created on demand and a fresh clone has none; requiring presence would flag every
  clean checkout. The load-bearing assertion is "if present, gitignored". Mechanics
  revision, premise intact.
- 2026-07-13 (itd-85 M3): privacy-hygiene reads tracked files through os.OpenRoot
  (repo-root containment), not os.ReadFile. A leaf-only O_NOFOLLOW is insufficient
  — a symlinked INTERMEDIATE directory still escapes; security review PoC-confirmed
  an out-of-repo arbitrary read. os.Root refuses any escaping component. Plus
  O_NONBLOCK (FIFO/device non-blocking open) + IsRegular skip + 4 MiB size cap.
  Requires go 1.24+ (repo is 1.25); no new dependency.
- 2026-07-13 (itd-85 M7): acknowledged repolinter (rule schema) and Conftest
  (severity/exit vocabulary) in ACKNOWLEDGEMENTS now, since both are actually
  adapted in the shipped audit engine. DEFERRED the SARIF acknowledgement to P3:
  the serializer seam is shaped for SARIF but no SARIF is emitted yet, and the
  convention is to credit a pattern in the change that lands it, never ahead. Add
  the SARIF entry when the --format sarif serializer ships.

- 2026-07-13 (sensemaking method): recorded the ABCD method (cold reading / warm
  ledger / disposition) as a research note — the parent that itd-27, itd-42,
  itd-55, itd-86 and itd-87 had all been accumulating under without one. Minted
  exactly ONE principle (recurrence-is-signal) rather than one per method element:
  the cold/warm split is already stated by evaluator-outside-the-loop and
  verifier-selects-gates-decide, and one-canonical-primitive forbids a third
  near-copy. Recurrence was the only element with no counterpart in the record.
  REJECTED minting a `read-it-cold` principle for the same reason.
- 2026-07-13 (itd-86/87): recorded the two intents TOGETHER because they are
  coupled, not merely related — a blind cold reading re-raises old tensions by
  design, so pointing it at a ledger that dedupes them yields a detector fighting
  its own store. itd-87 is the precondition that makes itd-86's re-raising useful.
- 2026-07-13 (attribution): DEFERRED the ACKNOWLEDGEMENTS entry crediting the cold
  reading to abcd's co-author, pending confirmation of how they wish to be
  credited. Held loudly (stated in the method note), not silently; it must land
  before itd-86 ships. Do NOT guess the credit line.

- 2026-07-14 — The lifeboat is built as a COVERAGE EXPERIMENT, not a feature
  (adr-35, itd-88/spc-3). Probe before pack: `disembark probe <repo>` produces a
  cross-repo coverage aggregate BEFORE a packer exists, because the brief's
  structure is an untested assumption and building the packer first assumes the
  answer. The headline number is the delta in section coverage between a
  rich-record repo and a git-only one — that is what the record is worth, and if
  half the brief is permanently blank everywhere, the structure is wrong and we
  learned it for one milestone instead of a phase. Phase 6's "depends on every
  prior substrate being native" rationale was checked against the BINARY and found
  mostly false (spec engine ships; reviews are committed markdown; backgrounding is
  a host affordance; the itd-2 host-delegation seam already ships twice — memory's
  `--pages-json` and `intent review ingest --verdict-json`). The ONE real
  dependency is data, not code: `~/.abcd/` does not exist, `history.Capture` is
  called by nothing, and Pass B's corpus cannot be obtained retroactively — the
  only permanent, compounding cost on the board, which is why the transcript hook
  ships ahead of any lifeboat code. Rejected: building the packer first (assumes
  the answer); amending adr-4 in place (two of its three operative claims change —
  a replacement, not a clarification).
- 2026-07-14 — adr-4 SUPERSEDED by adr-35 and pruned per the ADR convention
  (superseded ADRs are pruned; git preserves the text; the successor carries the
  transition rationale). What survives is restated in adr-35: the lifeboat is
  regenerable output, and the `lifeboat`(noun)/`voyage`(verb) distinction is
  load-bearing. What changes: disembark is READ-ONLY and OUT-OF-TREE (a test hashes
  the source tree before and after), and `voyage/` moves to the OPERATOR level
  (`~/.abcd/voyage/<source-root-sha>/`, keyed like the history store). The voyage
  move is not cosmetic — voyage records absolute source paths, and the
  `privacy-hygiene` audit rule (itd-85) flags those in committed files, so abcd
  would have failed its OWN audit. adr-4's overwrite-with-`.bak` model is replaced
  by a destination safety gate (never overwrite a directory abcd did not produce);
  its `shared_with` field is dropped (nothing produces it, and an empty field is a
  lie in a schema); and its hash chain — asserted but never defined — is pinned.
  Nine inbound references repointed by hand (2 links, 7 prose/frontmatter).
- 2026-07-14 — The brief↔lifeboat mapping table now EXISTS. `00-meta.md` has always
  called it "the contract" while no such table existed anywhere (found by the
  2026-07-06 plan-consistency review). It lands as Go — `internal/core/lifeboat/
  mapping.go` is the single source of truth — and is rendered into `00-meta.md`
  between generated markers, with a test asserting the two agree so the document
  cannot drift from the code. It is framed as the experiment's HYPOTHESIS, stating
  the best status each brief section could reach at each source tier, in the SAME
  three-valued vocabulary the probe reports (`grounded`/`partial`/`blank`) so
  prediction and evidence are directly comparable. M2 is expected to revise it.
  A monotonicity test (a richer tier can never ground a section worse than a poorer
  one — tiers are CUMULATIVE) caught a real error in the first draft of the table.
- 2026-07-14 — Vocabulary registered in `02-constraints/04-naming.md`, fulfilling a
  claim adr-4 made and never kept (`voyage/`, `manifest_sha256`, `_provenance.json`,
  `history.jsonl` were absent from the registry). Added with them: `coverage.json`,
  `graveyard/`, and two new controlled enums — coverage `status ∈ {grounded, partial,
  blank}` and source `tier ∈ {git, conventions, abcd-native}`, both with the Go enum
  named as the machine-readable source of truth. The brief's `"sufficient"` oracle
  verdict — a member of NO registered enum — is retired in favour of the registered
  `{SHIP, NEEDS_WORK, MAJOR_RETHINK}`; no third verdict family is minted (four
  brief locations).
- 2026-07-14 — adr-35's blast radius across the record was FAR wider than the plan
  anticipated, and the line drawn is: **the brief, glossary and roadmap are
  reconciled; the intent corpus is NOT.** An adversarial review (four hostile lenses,
  every finding independently verified) found the first pass had rewritten the
  vocabulary registry to the new model while ~14 other files still asserted the old
  one as fact — including an INVARIANT (`03-invariants.md` #6), the product's own
  press release, the verification matrix (which encoded adr-4's `.bak` overwrite as a
  TEST GATE), and the lint-enforced glossary SSOT. A registry contradicting an accepted
  ADR is drift of exactly the kind iss-35 exists to prevent, so all of it was swept.
  The INTENTS (itd-2/8/9/10/13/15/19/22/24) were deliberately left alone and tracked as
  iss-94: an intent is a proposal with its own lifecycle, and silently rewriting nine of
  them inside an unrelated change is worse than recording the drift — each reconciles
  when it is next planned. Where adr-35 genuinely does not settle a question (where
  `embark scan` searches now that destinations are operator-chosen; what the `/abcd`
  status board reads now that there is no in-tree lifeboat to stat), the text carries an
  explicit `Open question (adr-35)` note rather than an invented answer.
- 2026-07-14 — iss-93: adr-35 promises disembark is READ-ONLY over the source (a test
  hashes the tree before and after), but two paths in the design still write into it —
  Pass-0 dev-sync (`.abcd/work/reviews/`, `.abcd/memory/`, `.abcd/work/issues/`) and the
  backgrounded-execution checkpoint (`.abcd/logbook/disembark/<ts>/_state.json`). Either
  they move out-of-tree (under `<dest>` or the operator-level voyage) or they leave the
  disembark path entirely. adr-35 does not settle it; the decision is owed before the
  packer ships, and the read-only test is what will force it.
- 2026-07-14 (M1, itd-89/spc-4) — Transcript capture is wired to `SessionEnd`, NOT the
  `Stop` the plan specified. The plan's letter was wrong on a matter of harness fact:
  `Stop` fires once per assistant TURN, and Claude Code's transcript file grows through a
  session, so a `Stop`-wired capture stores a fresh, larger superset every turn — proven
  by live test (one session, 4 turns → 4 records; a 100-turn session → 100 records and
  O(N²) bytes). `history.Capture`'s sha256 dedup only collapses byte-IDENTICAL
  re-captures, which never happens on a live transcript, so the plan's "re-capture is
  idempotent" acceptance is false under `Stop`. `SessionEnd` fires once at termination and
  by contract ignores exit code + stdout — a perfect fit for a fail-closed, non-blocking
  side-effect hook. Verified against the harness docs (code.claude.com/docs/en/hooks).
  Accepted cost, recorded not hidden: `SessionEnd` does not fire on a hard crash/SIGKILL,
  so an uncleanly-killed session is not captured; the `Stop`-with-session_id-dedup
  alternative that would recover that case needs a change to shipped core dedup semantics
  and is deferred. This is the M1 deviation the loop is required to surface.
- 2026-07-14 (M1) — iss-95: wiring the hook does NOT by itself start the clock. `history.
  Capture` requires `~/.abcd/history/<root-sha>/transcripts/` to already exist and
  deliberately never creates it (the `ownedDirsReal` symlink-safety discipline); `ahoy
  install` bootstraps it. On a machine where install has not run — INCLUDING THIS ONE,
  where `~/.abcd/` does not exist — `hook session-end` fails closed, logs to stderr, exits
  0, and captures nothing, silently. That is exactly itd-89's failure mode (a hook that
  looks wired while the corpus never accrues). Decision owed: hook self-bootstraps (changes
  Capture's precondition and has the hook create dirs, which the symlink discipline avoids)
  vs. `ahoy install` stays the sanctioned bootstrap and the not-installed case is made LOUD
  (ahoy doctor already flags `history.bootstrap_missing`). iss-96 records the adjacent point:
  automatic capture makes the scanner's secret-pattern coverage load-bearing — it catches
  anchored tokens (AKIA…, ghp_, sk-ant-) and home paths but not unanchored high-entropy
  values (a bare 40-char AWS secret, a prefixless token), so consider entropy detection or
  the gitleaks adapter for the transcript path.
- 2026-07-14 (M1, iss-95 — maintainer decision) — The store-not-bootstrapped case
  is made LOUD, not self-bootstrapped by the hook (rejects having `hook session-end`
  create `~/.abcd/history/`, which would put a dir-creating trust-boundary act inside
  a fail-closed hook and contradict the `ownedDirsReal` symlink discipline). Reality
  check: `ahoy install` ALREADY bootstraps the store (`bootstrapHistory`, plus the
  per-repo transcripts dir), and detection ALREADY emits `history.bootstrap_missing`
  as a required gap that bare `abcd`, `ahoy`, and `ahoy doctor` surface — so an
  installed user is never in the silent state. The only genuinely silent path is the
  `SessionEnd` hook itself, which by harness contract has NO output channel (its exit
  code and stdout are ignored), so it cannot speak at session end. "Loud" therefore
  lives where a channel exists: a SessionStart notice (SessionStart hook output is
  surfaced) that warns once when the store is absent, pointing at `/abcd:ahoy install`.
  Scoped as an M1 follow-up; keeps the hook fail-closed-silent and moves the loudness
  to the one event that can be heard.
- 2026-07-15 (M2 gate — maintainer-approved) — The lifeboat coverage experiment's
  cross-repo readout is in. Corpus (private repos anonymised): abcd-cli
  (git+conventions+abcd-native, 21/2/0 grounded/partial/blank), test repo 1 and
  test repo 2 (abcd-native scaffolding but no authored brief, 4/8/11 and 2/6/15),
  test repo 3 (git+conventions, no abcd, 3→4/8/11), and a git-only floor (0/2/21).
  Headline finding: **scaffolding is not a record** — test repos 1 and 2 carry
  `.abcd/` directories yet ground barely more than the record-less test repo 3,
  because their `.abcd/development/` has no `brief/`, no ADRs, no issue ledger; the
  native adapter is honest and grounds only authored prose. The
  brief structure holds (excluding the dogfood repo, 9 of 23 sections are blank across
  the messy corpus, not half). Decisions: (1) `product/personas` is demoted to a
  human-answered question in the lifeboat brief — the corpus confirms the M0 prediction
  that it is not derivable from a repository. (2) The other 8 always-blank sections stay
  in the brief but split: `product/mental-model`, `delivery/verification-matrix`,
  `delivery/out-of-scope` become human-answered questions; `evidence/what-didnt`,
  `evidence/open-questions`, `constraints/naming`, `glossary`, `internals` are blank more
  from thin adapters than genuine non-derivability and get adapter work before M3 decides
  (iss-98, iss-99, iss-100). (3) The dependency-manifest adapter under-detected Python/
  Ruby/PHP packaging (test repo 3's pyproject.toml+uv.lock read as blank) — fixed now, so
  test repo 3's `constraints/dependencies` grounds. M3 (the packer) builds to this list:
  grounded/partial sections extracted-and-cited, the human-question sections surfaced as
  the blanks-with-questions the coverage report already produces.
- 2026-07-15 (post-M2 gate design) — Graduated to adr-36 and itd-90. The coverage
  gate raised a lifecycle question the plan implied but never named: the coverage
  report knows what to ask, but nothing said who answers a blank, when, or where —
  and the person with the tool (facilitator) is rarely the person with the answer
  (product thinker). Decision (adr-36): a blank is a durable, fillable object, not
  a fill-now-or-lose snapshot; answering is decoupled from disembark/embark and runs
  as its own async, environment-agnostic step over the coverage JSON; blanks carry a
  `kind` (`extractable` = coverage debt abcd can fix, vs `human-owned` = personas/
  mental-model, never derivable and framed as a prompt not a failure — the durable
  form of the "personas is manual" gate call); and a filled blank is marked
  `authored-by` (a person + date), structurally distinct from a grounded section's
  `extracted-from` (a file), so an opinion never launders into a fact. Coverage
  schema grows to v2 (fillable object); mapping.go gains a per-section `Kind`. itd-90
  specifies the product-thinker-facing interview (draft). Boundary: distinct from
  itd-86 cold-reading (which reviews for contradictions, denied context; the interview
  answers questions, fed context — opposite direction).
- 2026-07-16 — M5 round-trip closure re-scoped: the plan's literal "re-pack of an
  embarked repo reproduces the same manifest hash" is structurally unmeetable
  (coverage/brief/archaeology are identity- and git-derived); ratified instead as
  (P1) record-derived sub-manifest closure — RecordManifestSHA256 over ADRs,
  issues, intents, specs, abandoned.json, recorded in provenance as
  record_manifest_sha256 — plus (P2) literal self-closure into a byte-copy of the
  source; the packer now carries specs (rescue/specs/) so the spec.Load
  round-trip assertion is real.
- 2026-07-16 — M6 synthesis, two plan amendments: the oracle audit is keyed by the
  lifeboat's manifest hash (audit/oracle-<manifest12>.json), not the plan's
  wall-clock <ts> (no timestamp ever enters a lifeboat artifact); and
  _provenance.json is never mutated post-pack — each synthesis artifact
  self-records its mode (deterministic|delegated) instead of the plan's
  "_provenance records which" (the commit marker stays immutable). Synthesis
  outputs (principles, press-release, audit/) join the lessons files outside
  manifest_sha256.
- 2026-07-17 — Burst 2 (run test B), two mechanics decisions: (1) an
  already-planned intent with spec_id null (itd-40, created before the
  lifecycle verbs existed) is routed through a transient git mv planned->drafts
  so `intent plan` can mint+link its spec fail-closed — there is no standalone
  spec-create verb, and hand-authoring a spec file would bypass the mint lock's
  id allocation; net record churn is the spec_id write. (2) Implementation was
  delegated to an Opus 4.8 worker (recorded protocol deviation, manual test B);
  the orchestrator re-ran the gate on the output, and commits carry the trailer
  of the model that authored them (worker code: claude-opus-4-8; orchestrator
  record work: claude-fable-5).
- 2026-07-17 — Burst 3 (M3, itd-4/spc-6), three record adjudications: (1) AC2's
  resolve note lives as the structured frontmatter scalar `resolution:` (the
  live setScalarField design), not body-appended prose as the AC letter says —
  recorded as intentional design evolution, not a gap. (2) AC3 (promote) is a
  genuine BLOCKED gap: skill-orchestrated by design but uncompletable with
  today's verbs (no intent-create until itd-46; no engine-backed related_intents
  back-link write; hand-editing frontmatter from markdown would violate the
  engine-backed convention) — spc-6 stays OPEN and itd-4 stays planned on it.
  (3) AC4 migration recorded satisfied-by-history (source absent, ledger
  populated iss-1..iss-103); no dead migration code built.
- 2026-07-17 — Burst 4 (M4, itd-46/spc-7): quoted-text create shipped; the
  seeded-draft shape spc-7 defines is canonical (the AC's "byte-identical to
  intent new" clause is historical — no such Go verb existed). Two intent
  scope bullets named old-system files with no native counterpart
  (commands/abcd/intent.md, docs/reference/commands.md) — adjudicated moot in
  the spec; the missing intent plugin-markdown surface is ledgered as iss-105
  rather than silently absorbed. Typo-guard asymmetry vs capture ledgered as
  iss-104, not fixed (ACs don't require it).
- 2026-07-17 — Burst 5 (M5, itd-43/spc-8): GL002 enforces a deliberate subset
  (['epic']) of the glossary's forbidden_synonyms — the others are common
  English words whose false-positive rate would sink the gate; each becomes
  opt-in via .abcd/record-lint.json as the corpus is readied. Two sweep hits
  inside internal self-quotes (itd-48 quoting a working-log line and spc-12's
  overview) were swept, not exempted: abcd's own records carry the canonical
  word even when quoting themselves. spc-8 stays OPEN and itd-43 planned on
  AC3 (spec-review token), blocked by itd-28's maintainer-gated dependency.
- 2026-07-17 — Release burst (maintainer-directed): adr-37 adopts
  changelog-driven releases (rolling Unreleased -> dated heading in a reviewed
  PR IS the release decision; auto-release.yml tags exactly that commit and
  calls release.yml as a reusable workflow; idempotent, GITHUB_TOKEN-only).
  Extends adr-31, does not replace it: number derivation stays itd-73's; the
  interim check is maintainer review of the roll PR. The detect step tolerates
  the historical [v0.1.0] heading style; new headings use the plain
  Keep-a-Changelog form. v0.2.0 rolls in the same PR as the port — the
  automation's first firing is its own acceptance test.
- 2026-07-18 — The iss-35 semantic release gate was self-referential
  (armed against the tagged commit, read receipts from that commit's own tree)
  and fail-closed the first public release; fixed in PR #99 by arming with the
  reviewed content commit (HEAD^2^ / HEAD^) from a full-history checkout, plus a
  check-reviews.sh RD001 exemption for sha-keyed receipt dirs. The gate is
  abcd-cli's OWN CI: it is NOT shipped or scaffolded to managed repos
  (launch-payload.json excludes .github/; ahoy/launch write no CI; lifeboat only
  reads .github/workflows as a grounding signal), so the flaw had no managed-repo
  reach — the iss-108 capture's "systemic" framing was corrected on resolve. Any
  future release-scaffolding intent should scaffold the fixed two-commit
  (roll -> receipts) pattern, not the original self-referential one.
- 2026-07-18 — Acceptance-criteria sign-off is implicit in a human running
  `abcd intent plan` (itd-94): no `ac_confirmed` frontmatter field, keeping
  directory-as-truth pure and adding no forgeable schema surface. The gate
  (`abcd intent ready`) distinguishes only drafts vs planned; agents are barred
  from unattended planning at the protocol layer (run-protocol step 0 +
  `/abcd:intent`'s interview script: `plan` is never run without the human's
  explicit in-session confirmation). Escalation path if violations appear: an
  `ac_confirmed_by:` field is lint-legal today and slots in as a fifth check.
- 2026-07-21 — itd-66 (`launch-payload-render-parity`) is DEFERRED to a follow-up
  after the derived-versioning + auto-changelog + distribution programme. itd-67's
  own AC frames its installability smoke as "light … later upgraded to call"
  itd-66's deep tier, so the split is clean: this programme builds the light tier
  and positions the surface-resolution seam so itd-66's deep render, `.abcd/**`
  leak-proof assertion, symlink resolution, parity diff, and isolated-subprocess
  deep smoke slot in as a drop-in upgrade rather than a rewrite. Ordering: this
  programme -> itd-66. Recorded in spc-11's "itd-66 is deferred" section.
- 2026-07-21 — itd-67's "a phase completed since the last launch -> minor" bump
  heuristic is SUPERSEDED by itd-73's `impact` derivation (spc-10). The bump falls
  out of the records' declared `impact`, not out of phase membership; spc-11
  consumes the derived number and never computes one. Recorded so the two intents
  do not both claim version selection.
- 2026-07-21 — `launch ship` does not tag. It writes the dated `## [X.Y.Z] - <date>`
  heading in the reviewed ship PR; the unchanged `auto-release.yml` greps it on
  merge and creates the tag. ADR-37 is preserved, not superseded: the reviewed ship
  IS the release decision, and the bot-on-main alternative stays rejected.
- 2026-07-21 — `impact` is a KNOWN property of the issue ledger schema, and
  `internal/core/capture` validates it against the shared enum in
  `internal/core/changelog` rather than a private copy. The back-fill added
  `impact:` to every resolved issue, which `validateStrict`'s
  additionalProperties:false allow-list rejected — `abcd capture` reported
  "resolved 0" and skipped all 57 records as malformed. Accepting the field
  without validating it was rejected: severity/category/source are all
  enum-checked on read, and a third definition of the impact enum is exactly what
  spc-10 exists to prevent. capture -> changelog is the import direction (no
  cycle: changelog imports launch/frontmatter/gitutil only).

- 2026-07-21 — The mapping table's per-tier status columns are a **ceiling**,
  not merely a prediction. Every conventions adapter already honours its row
  (`convGlossarySource` returns partial where the row says partial;
  `convPlatformSource` returns grounded where it says grounded), so itd-95's
  and itd-96's three new adapters cap at `StatusPartial` — the value all three
  rows predict — and carry signal strength in `Confidence` instead. Rejected:
  returning `StatusGrounded` for a dedicated `NAMING.md` or a prose-bearing
  `ARCHITECTURE.md`, which would have made the rendered brief table wrong and
  required editing `mapping.go`, the brief-to-lifeboat contract both intents
  put out of scope. Every acceptance bar asks only for "non-blank", so the
  ceiling satisfies them. Revisit only by amending the mapping row first.

- 2026-07-21 — A probe walk of a foreign tree must be bounded in **three**
  dimensions, not one. itd-95 shipped `WalkFiles` with a regular-file cap; an
  independent security review of the itd-96 branch showed both remaining
  dimensions were exploitable — a tree of directories holding no regular file
  never reaches a file cap, and `os.Root` re-resolves each directory from the
  containment root one component at a time, making a directory chain quadratic
  in its depth (depth-1500 did not finish in two minutes). Directories are now
  counted against the same cap and descent is capped at `maxWalkDepth`. The
  general rule: any new whole-tree traversal states which of {entries, depth,
  aggregate bytes} bounds it, because a per-item cap and a count cap multiply
  and their product is not a bound.

- 2026-07-22 — A release is a release regardless of path: the clean-cutover
  manual roll follows the SAME two-commit release-branch shape as a derived
  ship (the content commit, then a receipts commit carrying the sha-keyed
  PROMOTE receipts for docs-currency-reviewer and iss35-brief-surface-crosscheck).
  The v0.4.0 roll landed as one commit with no receipts; the receipt gate
  refused fail-closed — its first genuine firing, and correct. Recovery uses
  the workflow's own escape hatch (tag on the receipts commit; content = tag^).
  Rejected: weakening the required-gates list to unblock the tag — that edits
  the release contract the whole programme was built to preserve.

- 2026-07-22 — Detector findings are triaged adversarially BEFORE fixing, and
  the numbers justify it: the full-depth iss35 crosscheck returned 102
  discrepancies; independent refuters confirmed 95 and killed 7 (two
  cross-direction duplicates, two wrong-reality, three legitimately
  staged/exempt). 93% precision is high enough to trust the detector and low
  enough that unfiltered fixing would have written seven falsehoods into the
  record. The refutation criteria are the detector's own exemptions plus
  adr-5; the refuters re-probe the binary rather than trusting the finding.

- 2026-07-22 — When a brief is agreed upfront, it is a TARGET document; it
  becomes the state document claim-by-claim, at ship time, because shipping
  includes the brief row edit (spec-moves-with-the-surface). Unbuilt design
  lives in intents or under an explicit staged marking — never as unmarked
  present-tense brief prose. The 95 confirmed discrepancies are that ratchet
  skipped at scale; the crosscheck is the measuring instrument; promoting the
  principle to a mechanical discipline is the open follow-up (iss-121, iss-122).

- 2026-07-24 — iss-35 resolves: everything it stayed open for is verified
  shipped (surface_coverage armed as blocker, 16 surface chapters incl.
  docs/history/version, skills→commands relabel, machine-checked staged Status
  column, crosscheck wired as release gate). The only residue is iss-122.
  Decided in a maintainer grill interview this date.

- 2026-07-24 — iss-122 design: the crosscheck gate gets a committed input
  manifest (doc list, directions, checker count, prompt hash) with TIERED
  depth — full depth for feature/breaking releases, Direction-B-only shallow
  pass for patch. The receipt must echo the manifest hash AND the tier;
  receipt_gate refuses a receipt whose tier mismatches the release's declared
  impact. Automated refusal is PROCEDURAL ONLY (manifest/tier mismatch,
  undispositioned findings); confirmed findings go to the maintainer, whose
  PROMOTE with recorded dispositions is the gate (verifier-selects-gates-
  decide). Rejected: hard-blocking on confirmed majors (a stochastic LLM
  triage could block a release); never-worse ratchet (drift persists; tiers
  make counts incomparable).

- 2026-07-24 — itd-93 design settled in maintainer grill: (a) surface is a
  `launch` sub-verb (extends 04-launch; rejected: new top-level verb, ahoy
  install step, embark-time family); (b) SELF-SCAFFOLD PARITY — abcd-cli's own
  release workflows are regenerated from the shipped template and a test
  asserts parity, so the proven pattern and the template are one artifact
  (rejected: lockstep diff test between two artifacts; frozen verbatim copy);
  (c) built-in workflow_dispatch REHEARSAL mode — arms the full gate against a
  simulated roll, publishes nothing; a green rehearsal is the runbook
  precondition for the first real release; (d) the changelog dated-heading
  format IS the itd-73 seam — `launch ship` is one optional producer.
  Promotes standalone / severity minor, PRD synthesised from the interview
  (no grandfathering); the 5 seeded ACs stand, amended for rehearsal mode.

- 2026-07-24 — itd-28 gitleaks sign-off: Stage 2 runs through the scanner
  seam — native patterns are the default engine, gitleaks is the stronger
  opt-in adapter when the binary is present, and the hook reports which
  engine ran (loud staging). CI's gitleaks pass stays the authoritative
  backstop; iss-96 pattern parity folds into the spec. No new hard
  dependency — adr-22 holds. Rejected: gitleaks as hard local dependency
  (first adr-22 exception, per-machine install burden); CI-only Stage 2
  (secrets reach pushed remote history before CI sees them).

- 2026-07-24 — Next-run queue reshaped by the grill: the small consequences
  fold into Track 1 (resolve iss-35; implement iss-122 pinning; amend +
  promote itd-93 — promotion only, implementation later as its own focused
  run); the spine stays friction fixes → itd-94 → walk fixes → itd-88.
  itd-28 implements in the following run against the adapter decision.
  Queue: plans/2026-07-24-next-run-queue.md (supersedes the 2026-07-18
  queue file's pick-up role).

- 2026-07-25 — Record-id collisions across parallel branches (iss-115, iss-120)
  fixed as a class: one canonical allocator primitive (recordid.MaxAcrossRefs)
  mints max+1 over the union of the working tree AND every git ref, so a
  committed id on another branch is seen; git-unreadable-over-a-repo degrades
  loudly to tree-only, a non-repo mints quietly (no refs to collide). Detection
  completes the class — a new spec_id_unique record-lint rule via the shared
  validateIDUnique primitive (ADR ids left out of scope: this directive is
  iss-/itd-/spc- only). Rejected: id-range leases per programme, mint-at-merge,
  and non-sequential/hash ids — all trade away human-readable sequential ids
  without a maintainer mandate. The residual uncommitted-mint window (both
  branches mint before either commits) is accepted behind the armed detectors on
  the merged PR union.
- 2026-07-26 — itd-100 grill settled: crosswalk is a mapping not a registry (no canonical native definitions; gloss-only rows, docs/** links only, pending iss-40); alphabetical + thematic mini-index; footnote citations with per-line docs-lint allow for vendor names; British English. Positions: A2A and AP2/payments both WATCHING (AP2 via new capture — record silent, REJECTS never invented; x402 + OpenAI/Stripe ACP folded into the AP2 footnote); policy engine folded into policy-as-code row; agent skills is the sole REJECTS (brief 08-skills commands-only rule). Rejected-term admissions blessed (agentic workflow, ANP, AGNTCY, Verifiable Intent, harness engineering, ISO 22989-as-HITL-anchor). LinkedIn anecdote omitted from itd-100. Two OpenAI harness URLs await maintainer read before ship.

- 2026-07-14 (review): REJECTED a "super-reviewer" verb gated on a second model
  backend. Its motivating rationale does not survive: self-preference under genuine
  authorship is weak or absent (arXiv 2606.20093, gap -5.1pp, CI crosses zero), and
  the bias that DOES exist on code is a context-framing artefact, not model identity
  (arXiv 2603.04582). CHOSEN instead: a fresh-context, off-policy, same-model
  reviewer — re-present the diff in a new session as an artefact of unknown
  authorship. Captures the only measured debiasing effect, costs nothing, needs no
  second subscription, and removes the two-tier UX problem entirely. A second model,
  when present, is used for disagreement-as-triage per adr-25 asymmetric trust and
  itd-81 — NEVER a panel or a vote (nine judges = 2.18 effective votes; on code,
  consensus underperforms the naive baseline — the "popularity trap", arXiv 2510.21513).
- 2026-07-14 (review): REJECTED per-commit review as the unit. No serious tool does it
  (all per-PR); findings on WIP commits are false positives by construction; cost is
  $75-500 per PR-equivalent at published frontier rates. Unit of review is the BRANCH
  DIFF. Also rejected: spec-conformance as a BLOCKING gate — best agent scores 44.4%
  on the easier adjacent task (SpecBench) and no precision/recall for
  implementation-vs-spec is published anywhere. It ships advisory only, per
  verifier-selects-gates-decide. Full evidence:
  .abcd/development/research/notes/2026-07-14-cross-model-review.md
- 2026-07-14 (research): REJECTED building an authoritative benchmark from abcd's own
  data. Three independently fatal defects: (1) the corpus is n=2 labelled triples, both
  graded by the model that wrote the code, with exactly one NOT_MET; (2) circularity —
  the system would generate the spec, write the code, judge conformance AND supply the
  labels, and self-preference appears perplexity-driven so it SURVIVES swapping model
  family; (3) N=1 has no methodological remedy (Epoch attacks SWE-bench Verified for
  concentration at 12 repos; we have one). Arithmetic also fails: ~500 oracle-backed
  tasks needed to resolve model differences, in-situ A/B needs 2,200-15,500 paired
  observations. CHOSEN instead: a task-quality instrument + methodology, validated
  against an EXTERNAL reference — OpenAI's 731-task SWE-bench Pro audit. Ground truth
  must originate outside the tool (executable test or retrospective event oracle, never
  an LLM judge). Template is Aider polyglot: the tool supplies the harness and
  distribution; ground truth comes from outside (225 external Exercism exercises).
- 2026-07-14 (research): ADOPTED pre-registration of acceptance criteria — hash +
  timestamp at spec time, BEFORE any code. The novelty claim is that abcd writes the
  specification before the code exists, which is exactly the defect that killed
  SWE-bench Verified (35.5% of audited tasks demanded implementation details never
  stated in the problem) and SWE-bench Pro (retracted 2026-07-08). Without a commitment
  artefact the claim is unprovable and CANNOT be retrofitted — every intent shipped
  without it is lost evidence. The primitive exists (receipt_id already hashes the AC
  section, excluding Audit Notes).
- 2026-07-14 (review): ADOPTED splitting the conformance reviewer into a terse binary
  verdict call and a SEPARATE explanation call. Asking for verdict+explanation+fix in
  one call drives GPT-4o's false-negative rate from 26.2% to 73.2% (HumanEval) and
  35.9% to 87.9% (MBPP) — it does not find more bugs, it REJECTS CORRECT CODE
  (arXiv 2603.00539, Springer Automated Software Engineering). All five abcd agents
  currently do the forbidden thing. Largest measured effect size in the research and
  free to apply. Also adopted: fix-guided verification filter (execute the proposed fix
  as a counterfactual; if no test outcome changes, the rejection was hallucinated —
  FNR 54.8% -> 16.3%). Full evidence:
  .abcd/development/research/notes/2026-07-14-research-platform-benchmarks.md
- 2026-07-14 (research, UNVERIFIED): the SWE-bench Pro audit figures (27.4% automated /
  34.1% human) and the SWE-bench Verified figures (>=59.4% flawed tests / 35.5% hidden
  implementation details) come from SECONDARY sources — openai.com 403s to the research
  fetcher. These numbers are the reference standard the proposed paper is scored
  against. VERIFY AGAINST THE PRIMARY before writing anything. Do not cite the decimals
  until read first-hand.
- 2026-07-14 (research, CORRECTION — supersedes the two entries above): the SWE-bench
  figures are now VERIFIED against OpenAI's primary posts, and one claim is WITHDRAWN.
  The "35.5% of audited tasks demanded implementation details never stated" figure DOES
  NOT EXIST in the primary — it came from secondary reporting. OpenAI's real taxonomy
  for SWE-bench Pro (% of full dataset, agent-flagged / human-flagged, read per-bar from
  the chart's aria-labels): overly strict tests 14.4/17.8; low-coverage tests 4.1/9.4;
  misleading prompt 6.3/7.5; miscellaneous 1.9/1.2; UNDERSPECIFIED PROMPT 0.6/0.8.
  Underspecification is the SMALLEST category (~1%); "overly strict tests" is the largest
  by ~20x. CONSEQUENCE: the pitch "abcd's specs-before-code prevents the underspecification
  that killed these benchmarks" is aimed at a 1% defect and is DEAD. The surviving claim is
  stronger and is what the work now defends: SWE-bench's oracle (a merged PR's tests) is
  authored independently of, and after, the task statement, so it can demand what the spec
  never said — whereas in abcd THE ACCEPTANCE CRITERIA ARE THE ORACLE, so the oracle cannot
  exceed the spec. abcd therefore cannot OVER-reject; it can only UNDER-check (the
  low-coverage failure, 4-9%), which compounds with judges over-accepting AI-written code
  by up to 1.91x. Claim to defend: "a spec-derived oracle cannot over-reject; it can only
  under-check — and under-checking is measurable." Verified figures: SWE-bench Verified
  (2026-02-23) 27.6% subset audited, >=59.4% flawed tests, 138 problems x >=6 engineers;
  SWE-bench Pro (2026-07-08, retracted) 731-task split, 200 (27.4%) agent-flagged vs 249
  (34.1%) human-flagged — the two audits disagree by ~7 points, i.e. automated task-quality
  auditing UNDER-DETECTS relative to humans.
- 2026-07-14 (process): TWICE this session a research pass reported a paper as saying
  something it did not (a fabricated cross-family finding; the non-existent 35.5% figure;
  and SpecBench described as "deferring" conformance when it scopes code out permanently).
  RULE: for any number that will be load-bearing in a design doc or paper, open the PRIMARY
  before citing it. Secondary reporting of benchmark figures has been wrong every time it
  was checked in this session.
- 2026-07-14 (layout): ADOPTED "default to the local tier when in doubt". An artefact
  whose home is unclear (tool exports, oracle/review output, traces, intermediate
  analysis) goes to .abcd/.work.local/scratch/ or logs/ FIRST — never the repo root,
  never a tracked directory on a guess. Promotion to .abcd/work/ or
  .abcd/development/ is cheap and always available later; demotion is not, because a
  wrongly-committed artefact is already in the history. Guessing upward is
  irreversible; guessing downward costs nothing. Prompted by a RepoPrompt oracle export
  landing in a top-level prompt-exports/ during this session (moved to
  .abcd/.work.local/scratch/prompt-exports/). Recorded in AGENTS.md § Working-tree
  layout, the canonical home, rather than a new doc.
- 2026-07-27 (salvage): the four 2026-07-13/14 ideation-session entries above are appended out of chronological order — recovered from an unmerged ideation branch during branch cleanup, together with their research/plan docs; the /abcd:ideate seed from that session is re-recorded as itd-104 (its branch-local iss-93 capture id had collided with main's iss-93 and is retired unused).
- 2026-07-27 — DEFERRED the row-has-footnote structural docs-lint rule (spc-15 out-of-scope): no existing rule shape covers table-row-to-footnote structure, so it would need a bespoke check; the grill left it optional and the citation-gate intent (itd-101) is now its natural home — implement it there or not at all. Recorded per spc-15's deferral-is-recorded clause; surfaced by the itd-100 fidelity audit's gap check.
- 2026-07-27 — ADOPTED the canonical tagline: "A host-agnostic configuration layer for intent-driven development." Canonical identity home is the brief product chapter (01-product README, Identity section: title / tagline / pitch); README strapline, plugin manifest description, and AGENTS.md opening render from it. README's former strapline ("An opinionated, intent-driven development framework for product thinkers") is retired as a surface line; the product-thinker framing lives on in the README body. Rejected wordings: "for agent harnesses" (object shift, host/harness redundancy, plural over-promise), "over any agent harness" (drops the domain). Resolves iss-143; itd-102 generalises the drift check for managed repos.
- 2026-07-27 — itd-93 parity tension resolved: Branch A. The scaffold template is the single source for release.yml/auto-release.yml; abcd-cli's live workflows are regenerated from it; the only sanctioned live diff is the additive workflow_dispatch rehearsal (trigger + non-publishing dry-run job). Rejected: substitution-gating the rehearsal off for abcd (weakens the parity guarantee on the one novel component).
- 2026-07-27 — GRILLED itd-101/102/103/104 to settlement (six decisions, each now in its intent's Grill Settlements + acceptance criteria): itd-103 guard fails open loudly, blocker/warn tiers with committed-only overrides, shell-token-aware command-position matching with an itd-81-style TNR-floored corpus; itd-101 refresh is manual-with-nagging (scheduled CI later, own sign-off), staleness warns at 180d and blocks releases at 365d, human verifications age on the same clock; itd-102 identity lives in a parseable markdown block (config points at it), check is warn-tier and never rewrites; itd-104 ideate is an optional verb, never a pre-capture gate, with a fresh-context off-policy adversary. All four drafts are grill-settled and await intent plan.
- 2026-07-27: itd-101 spec minted as spc-16 collided with unmerged #156 (iss-80 class, both spc-N and iss-N allocators); renumbered to spc-17 by hand, duplicate capture folded into iss-80 — ids on this branch deliberately skip 16.
- 2026-07-27 — itd-101 part (a) settled four choices spc-17 left open. (1) The committed baseline lives at `.abcd/citations-baseline.json`, alongside docs-lint.json/record-lint.json/rules.json — it is config the gate reads every commit, not development record; rejected `.abcd/development/` (wrong tier) and `.abcd/work/` (not a session artefact). (2) A table is a CROSSWALK when its nearest preceding heading matches `(?i)crosswalk` (configurable) — the narrowest heuristic that selects docs/reference/terminology.md's table and leaves every ordinary directory-map/comparison table alone; rejected column-header and any-table-in-docs identification (both sweep up README.md and docs/README.md tables that carry no citations by design). (3) A page's CITATION CORPUS is its footnote definitions, including wrapped continuation lines — not its prose; a URL in a body paragraph stays links_resolve's business, which is what keeps the syntax and source-policy rules off ordinary text. (4) `refused_domains` ships EMPTY: the repo states its admission rule in prose ("no single-author coinages, no aggregators", docs/reference/terminology.md + ACKNOWLEDGEMENTS.md) but nowhere names a domain, and the gate must not invent an editorial blacklist the project never agreed. Also: staleness is measured from `last_checked` for automatic and manual entries alike (AC 3's "same clock"), and the 365-day threshold surfaces as a distinct rule id `citation_baseline_overdue` at warn — the commit gate never calendar-blocks, so promotion to blocker is the release gate's job in part (b).
- 2026-07-27 — itd-101 part (b) settled seven choices spc-17 left open, and consolidated one primitive. (1) The SSRF fetch guard moved from `internal/core/memory` to `internal/urlguard` rather than being copied for the second fetch path; its address predicate became a parameter so a fetch path can be exercised against an httptest server (which binds loopback) without the shipped policy ever being relaxed. (2) BLOCKED-FOR-AUTOMATION is classified by status code ONLY — 401, 403, 406, 429 — with no body sniffing for challenge-page markers: a heuristic over page text would be unreliable AND a reason to start reading content, and the challenge pages that matter answer with one of those codes anyway. Everything else non-2xx is broken. (3) One GET per URL, body never read and never retried: liveness is the status line, so a citation to a huge file costs a response header; a retry loop would make a run's duration a function of how many links are failing. (4) A blocked URL with NO prior entry writes NOTHING — the gate then reports it as unreceipted, which is exactly true; recording it broken would put a lie in a committed record the gate enforces. (5) A CURRENT manual receipt is preserved verbatim and not even re-requested; a STALE one is re-checked (AC 3's one clock) and, if the source still blocks, KEPT rather than deleted so the gate keeps warning honestly. Rejected: refetching every manual entry every run (downgrades a human's receipt to a robot's failure), and never refetching them (buys a permanent exemption from ageing). (6) `confirm` records ALIVE only — it is "I looked and it is there", never a channel for recording a link dead — and takes URLs positionally or a receipt file, both assembling ONE schema so the later generated checklist page is a different producer, not a second pathway. (7) `refresh` exits ZERO after recording broken links: it records, the gate decides; a verb that failed on a dead link could never write the record that reports it. Also: the release-gate promotion reuses `ArmReceiptGate`'s shape (`lint.ArmCitationOverdue`, armed by `abcd docs lint --release-gate`) with the FLAG as trust root, and the `abcd launch --dry-run` citation gate takes its measurement from the CLI as data because `core/lint` imports `core/launch` for its semver — the same shape the cobra-tree walk already uses.
- 2026-07-27 — itd-101 part (b) review round settled five more. (1) The committed `citation_baseline.baseline` config value is CONTAINED at one choke point (`lint.CitationPolicy`, which now returns an error): a `../../..` or absolute path is REFUSED, never silently normalised, because `SaveBaseline` MkdirAll's its parent and a contributor controls that file — this was a real arbitrary-file-write primitive found by the security review, reproduced against a built binary. Both writers and both readers resolve through the one check. (2) A refresh run in which NOTHING succeeded is REFUSED rather than committed, and only when a prior baseline existed: `Check` collapses DNS failure, refused connection and timeout into the same `broken` as a 404, so a run behind a captive portal would otherwise rewrite every entry as broken — stale human receipts included, since those are re-checked and so not covered by the blocked branch — and the operator would find the gate blocking every commit. A first run over genuinely dead citations has nothing to protect and still writes. (3) An unrecognised `Status` from a `Checker` is an ERROR, not a fall-through: the seam is exported for AC 5's adapter, and falling through would drop the URL from BOTH the baseline and the queue — the one outcome nothing downstream could detect. (4) The missing-entry lint message names BOTH verbs: the only state that produces a missing entry is a source refusing automated fetchers, so naming `refresh` alone sent the maintainer round a loop that cannot terminate — only `confirm` clears it. (5) A docs-lint config that arms the citation rule but fails to PARSE is reported to the release preflight as `Unreadable` and REFUSES, distinct from the nil "not armed" state — rendering a broken gate as an absent one would wave a release through on a false statement. Also consolidated: `daysBetween` is now the exported `lint.DaysBetween`, called by the verb and the gate alike, because two copies of the staleness boundary is two chances for the verb to call an entry current on the day the gate calls it overdue; and the CLI's ad-hoc `citeErrorDetail` was dropped for the canonical `scrubPaths`, which preserves the `*PathError` type the redactor needs (an absolute developer path was reaching machine output).
- 2026-07-28 — itd-101 part (b) second review round closed three holes the first round's fixes left open, all reproduced end-to-end by the reviewers. (1) The baseline-path containment check was purely LEXICAL, so a committed symlinked directory (`.abcd/evil -> /outside`) plus a lexically-innocent `"baseline": ".abcd/evil/x.json"` reopened the very arbitrary-file-write primitive the first fix claimed to close — `WriteFileAtomic` MkdirAll's *through* the link. Containment now also RESOLVES: `EvalSymlinks` on the deepest existing ancestor of the target (it usually does not exist yet on a first run), compared against an `EvalSymlinks`'d repo root, reusing the shape `internal/core/launch/bundle.go` already uses for symlinked bundle entries rather than adding a third copy. `CitationPolicy` now takes the repo root and returns a `CitationPolicySet` carrying both the repo-relative form (for messages, so no absolute path is ever rendered) and the absolute form (for I/O). (2) `cfg.Roots` had NO containment, and while reading outside the repo was inert for a reporting lint, this intent made the collector feed a live fetcher whose results are persisted into a baseline the workflow expects to be committed and pushed — `"roots": ["../private"]` therefore fetched and PUBLISHED every URL in a sibling directory. The collector (not the pre-existing lint walk) now refuses an escaping or symlinked root. (3) The wholesale-failure guard consulted `res.Preserved == 0`, but a preserved receipt was never fetched and so is no evidence the network worked: any repo that had ever run `confirm` — the designed steady state for robot-refusing sources — silently lost the protection entirely. The guard now reads a new `CheckOutcome.Answered` bit that the CHECKER sets when a host returned any status line at all. That is the bit `StatusBroken` was destroying (a DNS failure, a refused connection and a genuine 404 all landed there), and recovering it from a proxy was wrong in both directions — the old condition also made a sole genuine 404 permanently unrecordable, refusing forever with a false "check connectivity". RULE affirmed: when a guard needs a fact, carry the fact, never infer it downstream from something correlated.
- 2026-07-28 — itd-101 part (b) third review round closed the other half of the roots containment and recorded one accepted narrowing. (1) Containing the configured ROOT string is NOT enough: `WalkDir` yields a symlinked `.md` as an ordinary file and `os.ReadFile` follows it, so a committed `docs/leak.md -> ../private/notes.md` sits inside a perfectly contained root and still drags an outside file's citations into the live fetch and then into the committed, pushed baseline. Every collected FILE is now resolved-contained, not just its root; an in-repo symlink (the `CLAUDE.md -> AGENTS.md` bridge shape) still resolves, because containment is about where a path LANDS. RULE affirmed: a containment check on a directory says nothing about the files found under it. (2) ACCEPTED NARROWING in the refresh's wholesale-failure guard: keying it on `answered` (a host returned any status line) rather than on success means an intercepting proxy or DNS sinkhole that answers EVERY request with a non-2xx outside the blocked set — 503, 407, or a default-vhost 404 — no longer trips it, and a whole corpus would be rewritten `broken`. That is strictly narrower than the condition it replaced, but the replaced one was wrong in two commoner ways (any repo that had ever run `confirm` lost the protection entirely, and a sole genuine 404 could never be recorded). It ships because refresh is operator-initiated and its transcript prints the `broken:` count before anything is pushed; the guard covers failures that produce no status line at all, and nothing more. (3) A CHANGELOG entry must describe the tree AT ITS COMMIT, never the state it will reach once a human acts: rewording the citation bullet to the post-confirmation state asserted "every URL carries a receipt" while the repo's own gate printed the counter-evidence. Reverted to the true count.
- 2026-07-28 — itd-101 part (b) closed two robustness notes from the approving security review, and flagged two it left. (1) A page read is now BOUNDED and regular-file-checked (`fsutil.ReadGuarded`, 8 MiB), because containment answers "does this path land inside the repo" and says nothing about how big what it lands on is — and this collector's output is fetched over the network. It opens the RESOLVED path rather than the literal one: `O_NOFOLLOW` on the literal path refuses every symlink leaf, which would break the legitimate in-repo bridge (`CLAUDE.md -> AGENTS.md`) that containment has just approved. So `containedRealPath` returns the resolved path alongside its verdict, and the reader reads exactly the thing containment judged. (2) `StatusOK` and `StatusBlocked` now ENTAIL an answer in the wholesale-failure guard: a 2xx cannot exist without a host replying, and "blocked" is defined by a status code, so only `StatusBroken` is genuinely ambiguous. That is what those statuses MEAN, not inference from a proxy, and it removes a footgun where a third-party adapter omitting `Answered` would refuse a run in which every check succeeded. The `Checker` seam's doc comment now states both obligations an implementer owes. NOT DONE, deliberately: (a) `urlguard.BlockedIP` still omits CGNAT `100.64.0.0/10` and benchmark `198.18.0.0/15` — adding them would change `memory ingest` behaviour, and that extraction was landed as a behaviour-preserving port, so it belongs in its own change; (b) a narrow availability path remains where a repo whose entire baseline is current-manual plus one PR-added citation to an unroutable host refuses the whole refresh with a misleading "check connectivity" — the reviewer could not make it bite on a realistic corpus, and the guard's limits are already recorded above.
- 2026-07-27: itd-103 guard overrides live in dedicated committed .abcd/guard.json, not a rules.json domain — structured entry schema, and the rules kill switch must not silently disable a safety guard (spc-16).
- 2026-07-27: itd-103 guard wiring — `abcd guard hook` is a sub-verb of the USER-facing `guard` verb, not a fifth entrypoint under the hidden `hook` subtree. Reason: guard health has to be legible (AC 1), and a hidden adapter documents nothing; the other `hook` entrypoints are injection transport with no user-facing counterpart, this one is the same decision the user can also ask for by hand. Consequence: it appears in the generated CLI reference and must stay host-agnostic in its prose.
- 2026-07-27: itd-103 guard wiring — the two front doors disagree on ONE case by design. A guard that cannot be evaluated (unparsable command, registry that will not load) is a FAULT for `guard check` (exit 2: a script must never read silence as clearance) and fail-OPEN for `guard hook` (exit 0 + loud stderr: a broken guard must never stop a session). Rejected: one shared behaviour, which would either brick sessions or teach scripts that silence means safe.
- 2026-07-27: itd-103 guard wiring — the fail-open-loud shim is a shell wrapper in the plugin's `hooks/hooks.json` PreToolUse entry (pass through only the binary's own 0/2; everything else warns UNGUARDED and allows), not Go code. The failure it guards against is the Go binary not running at all, so it cannot live in Go. Tested by executing the committed manifest string under /bin/sh against a fake plugin root.
- 2026-07-28: itd-104 spec minted as spc-16 (third live iss-80 instance this run: 16 collided with #156, 17 with the itd-101 branch); renumbered to spc-18 by hand.
- 2026-07-28: itd-102 spec minted as spc-16 (fourth live iss-80 instance this run); renumbered to spc-19 by hand.
- 2026-07-28 — itd-102 implementation: repo positioning lives in a NEW `internal/core/positioning` package, not `internal/core/identity` — that package is the git commit-author gate (`.abcd/config/identity.json` pin, pre-commit hook) and shares nothing with repo self-description but the English word; folding them would put two unrelated concerns behind one name. The user-facing verb stays `abcd identity` (spc-19). For the same collision the registry is `.abcd/positioning.json` at the `.abcd/` top level (beside docs-lint.json / record-lint.json / rules.json), NOT `.abcd/config/identity.json`. Rejected: extending the commit-identity package; naming the verb `abcd positioning` (spc-19 fixes the verb).
- 2026-07-28 — itd-102: surface comparison is normalised CONTAINMENT of each required block field (markup, dashes, wrapping, case folded; trailing sentence punctuation trimmed from the needle), not line equality. Equality would flag abcd's own three conforming surfaces, each of which carries the tagline in a different rendering (inside `<p>`, concatenated with the pitch in the manifest, bolded mid-sentence and line-wrapped in AGENTS.md); containment catches all three iss-143 variants while accepting all three current ones. Consequence: `identity render` fires only on drift, and its template render is a proposal to adopt by hand, not a byte-exact reproduction of conforming prose.
- 2026-07-28 — itd-102: no scaffold entry point existed in ahoy/prepare for a repo record block, so the verb family gained a minimal `identity init` (write path: block + pointer, atomic, adopts an existing block, refuses to repoint an adopted registry). `abcd launch scaffold` is release-machinery-specific and `ahoy install` writes only abcd's own plumbing; extending either would have widened its remit.
- 2026-07-29 — privacy-leak follow-up: illustrative machine identifiers in anything committed or published always come from reserved documentation ranges (RFC 5737 IPv4, RFC 3849 IPv6, RFC 2606 domains, RFC 7042 MACs; hostnames derived from the persona registry, e.g. alice-laptop) — the itd-79 persona rule applied to infrastructure. Recorded as principle `examples-use-reserved-identifiers`; enforcement is the iss-154 allowlist-inversion lint (flag any identifier OUTSIDE the reserved ranges), whose shipping promotes the principle to a discipline. No new intent: real private identifiers are itd-74's banlist territory — the incident evidence extends itd-74's scope to machine identifiers (iss-158) rather than founding a second primitive. Incident issue text deliberately names no repo and no values (shape only), so the ledger itself cannot re-leak.
- 2026-07-29 — v0.5.0 scoped as "security & consistency" (plan: plans/2026-07-29-v0.5.0-security-and-consistency.md): the security half closes the NEXT.md leak class end-to-end (itd-74/spc-20; the atomic iss-154+157+125+153 detector item; iss-155/156; guard batch iss-159+144+148; boundary fixes iss-30/34), the consistency half retires the record-currency majors (iss-37..44, iss-80) — maintainer added the latter rather than deferring them. Version is derived by launch ship, not declared; 0.5.0 is the prediction given itd-74 additive. Deferred majors listed explicitly in the plan (largest: iss-124); the 2026-07-24 queue's pick-up role is superseded.
- 2026-07-29 — v0.5.0 cycle runs as a scheduled cloud loop (one item per 5-hour round, plan order): auto-merge authorised through the strict gate only (CI green + two adversarial MERGE verdicts; security lens mandatory on workstream A/B diffs); itd-74 is in autonomous scope, multi-round via PR resume, bounded by spc-20 and STOP condition 1. Rejected: human-merges-everything (stalls the ordered pipeline at merge cadence) and auto-merge-C/D-only (slows exactly the security half the release is for).
- 2026-07-29 — the A2 network-identifier detector lands as one atomic change (iss-154+157+125+153): an allowlist inversion built once in the scanner's canonical pattern set, folded into `DefaultPatterns` so Stage-1 redaction and the launch/lifeboat scan inherit it, and consulted directly by the audit privacy-hygiene rule so the two surfaces cannot disagree about what a leak is. Maintainer disposition on the first repo-wide run (28 findings, plan STOP 2): the exempt set is "values that name no individual host" rather than the reserved documentation ranges alone — loopback, unspecified, netmasks, masked CIDR prefixes, and the IANA special-use ranges (IPv4 link-local, multicast, benchmarking, protocol assignments; IPv6 link-local, multicast, NAT64 well-known prefix, benchmarking). What identifies private topology stays flagged, which is the incident class: RFC 1918, CGNAT/tailnet, IPv6 unique-local, and 6to4 (it embeds a routable address, so it names a host) — consistency of the rationale wins over convenience of the residue. Twelve deliberately illustrative lines take the sanctioned per-line waiver; the repo audits clean on privacy-hygiene. The plan's STOP threshold is clarified to count findings, not distinct identifiers. Also here: `/Users/Shared` and `/Users/Guest` stop reading as usernames (iss-153), in both the audit rule and the scanner's identity matcher.
- 2026-07-29 — iss-155: `three-tier-layout` gains the placement half of the tier convention — local-tier artefacts (`NEXT.md`, `scratch/`, `logs/`) found directly in a committed tier are now errors, each finding carrying a move-to-`.abcd/.work.local/` fix. The rule verified tier presence and the `.work.local` gitignore but never that local ephemera were ABSENT from `.abcd/work/` and `.abcd/development/`, which is exactly how a handover file carrying host infrastructure detail reached a public repo unflagged. Presence is checked on the filesystem, matching the tier checks themselves: an untracked NEXT.md in a committed tier is one `git add -A` from history. The existing rule extends rather than minting a sibling ID — one convention, one rule.
- 2026-07-30 — iss-156: the PII rules domain gains the network/infra recall vocabulary the leak incident needed (`ip`, `ips`, `ipv4`, `ipv6`, `vpn`, `tailscale`, `tailnet`, `wireguard`, `firewall`, `network`, `reachability`, `reachable`, `dns`, `ssh`, `subnet`, plus the `mac address` alias) and one new rule line forbidding committed hostnames, IP/MAC addresses, and other live network identifiers: redact or omit, and use a reserved documentation value (RFC 5737/3849/2606/7042, the same citation set as the scanner and the audit privacy-hygiene rule) only where an illustrative example is needed — the rule does not tell an agent to substitute a plausible fake identifier into a factual write-up. Data-only: recall matching is already word-bounded (the prompt is normalised to space-separated tokens and a single-token term must match a whole token), so the bare `ip` keyword is safe and no matcher change was needed — a substring matcher would have forced a phrase form like `ip address`. Placement is dictated by the matcher, not by word count (`aliases` also holds single words such as `pr` and `diataxis`): a term goes in `recall` when it should hit as a standalone token, and in `aliases` when only the multi-word phrase is safe — `mac` alone would recall on an Apple Mac, so `mac address` is a phrase alias and matches via the stemmed-phrase path. The stemmer's own limits force the explicit variants: a three-character floor keeps `ips` from stemming to `ip`, and `-ability` is not bridged to `-able`, so `reachable` is its own entry. Accepted trade: broad tokens like `network` over-fire on prompts with no privacy stake, which is benign here — the injection is four short rule lines, deduped once per session. The never-commit-identifiers rule previously existed only in a parent CLAUDE.md privacy section, so the loader could never inject it; the 2026-07-29 reserved-identifier principle now has a rule-text home the hook actually emits.
- 2026-07-30 — iss-169: the visibility-driven `.gitignore` block now carries the brief's §1 table verbatim — private ignores `.abcd/.work.local/` only, public ignores the anchored `/.abcd/` plus the legacy root-level `/memory/` (the leading slash pins each public entry to the repo root; an unanchored pattern matches at any depth and would also ignore nested paths such as an `internal/memory/` source package) — replacing a phantom root-level `.work/` that appeared under both visibilities. The brief was checked first and is current: the tier table directly above §1 commits `.abcd/development/` and `.abcd/work/` and gitignores the local tier, so the code alone had drifted, and the issue's hedge that the public set "needs rethinking" resolves to a path fix, not a policy question. Under public no separate `.abcd/.work.local/` entry is needed because `.abcd/` subsumes it — visibility stays one switch with no per-subdirectory exceptions. Upgrade needs no migration verb: `gitignoreBlockDrifts` compares set-wise so an old block reads as drift, and `applyVisibilityBlock` already strips every block before writing the canonical one; the test pins that shape for both visibilities. This closes an installer-versus-auditor contradiction — `three-tier-layout` asserts the local tier is gitignored, which the installer's own output guaranteed it was not.
- 2026-07-30 — itd-74 (round 6), increment 1: the private guard layer end to end plus the maintenance verbs on both layers (AC1–AC4, AC6). Private entry format is `KEY<whitespace>PATTERN` with the key charset `[A-Za-z0-9][A-Za-z0-9._/-]*` — deliberately free of every regex metacharacter, which is what makes legacy compatibility safe rather than best-effort: a line whose first field is really the head of a regex cannot pass for a key, so it falls back to the whole-line reading under the synthetic key `entry-<line-number>`, and even where a split does apply to an old two-word pattern the resulting match is a superset of the old one (over-blocking, never under-blocking). Refusal names the key alone; the matched text and the pattern never reach any output, and a malformed line is a refusal naming its line number, because a banlist that cannot be read must not look like a banlist that found nothing. Two hook-internal fixes fell out of writing the proof: the candidate text moves from a pipe into `grep` to a temp file (under `pipefail`, a matching `grep -q` exits early and the writer left holding a closed pipe reports 141, which the pipeline surfaces instead of grep's verdict — a staged diff larger than the pipe buffer could silently defeat the guard), and grep's own stderr is discarded so an engine error message can never echo a pattern. Public layer: entries are managed IN the existing `banned_tokens` family of `.abcd/docs-lint.json` under a `names/` id prefix, and the prefix is the ownership boundary — `list` renders the whole family (hand-curated harness/present_tense entries included, marked as such), `remove` refuses anything outside the namespace. Config edits are byte surgery on the array located through the standard decoder's input offsets, never a re-marshal: an add is one inserted line plus a separating comma, a remove is one deleted line, and add-then-remove is byte-identical to the original — asserted against this repo's own docs-lint.json rather than a synthetic fixture, because a surgical editor proven only on a two-entry toy is not proven. Redaction is structural, not conventional: the exported private entry type has no pattern field at all, so no future rendering can leak the value, and pattern-validation errors discard the engine's message because Go's regexp errors quote the expression. One format, two readers, one fixture: `testdata/parse-corpus.txt` and one shared probe table drive both the Go parser and the committed shell hook, so their agreement is checked rather than assumed. Residual: the two engines are RE2 and the platform's POSIX ERE, so a pattern valid in one and not the other is possible — the shared corpus is restricted to constructs both accept, and the malformed-line path fails safe on either side. Deferred to later increments: `ahoy` scaffolding of the guard artefacts and the seeded stub (AC5), the honest-reach line on the status/report surfaces (AC7; `abcd banlist` itself states it), and the intent/spec lifecycle moves.
- 2026-07-30 — itd-74 (round 6), fix pass after two adversarial reviews (correctness + security) both returned BLOCK. Five decisions, each replacing a mechanism rather than patching an instance. (1) THE STORE DECLARES ITS FORMAT. The private banlist's first line — `# abcd-banlist: keyed` — decides the whole file: keyed means every line must parse as `KEY<space-or-tab>PATTERN`, no declaration means every line is one whole-line pattern under `entry-<line-number>`, and no line is ever split. The previous per-line heuristic ("is the first field key-shaped?") was wrong in two independent ways at once: it printed part of a legacy line as a key, and on this layer a pattern IS the secret, so the guard leaked what it existed to withhold; and it narrowed an old whole-line pattern to the remainder after its first field, so protection did NOT "never weaken because the format grew a column" — the earlier record line and the brief both claimed otherwise and were wrong. A declaration costs the user one line and makes both classes unrepresentable; `add`/`remove` refuse a non-empty legacy store rather than migrate it, because writing a keyed line in would silently reinterpret every other line. Both readers now strip ASCII space and tab only: `strings.TrimSpace` and bash `[[:space:]]` are different sets, so a U+00A0-indented line was keyed by one reader and dead to the other, and a U+000B-separated line the reverse. (2) VALIDATE AGAINST THE ENGINE THAT ENFORCES, not a convenient third one. A private pattern is screened for the constructs POSIX ERE does not implement (a backslash before an alphanumeric, `(?`) and then handed to grep ITSELF on stdin to accept or refuse; RE2 is the fallback only when grep cannot be run, and the refusal says so. Checking under RE2 accepted `\d` and `(?i)` as healthy (grep reads them as something else — inert protection reported live) and accepted `[a-z-.]`, which grep refuses (its fail-safe branch then blocks every commit). The public layer's engine is Go's regexp because `abcd docs lint` enforces it, so a public add compiles the EXACT string it stores through the linter's own compile path, and stores it with the `(?i)` prefix all sixteen hand-curated entries carry — without it a verb-written entry was case-sensitive while the docs promised otherwise. `list --private` reports unusable and inert lines APART: the first stops every commit, the second stops nothing, and one message for both misdirects an incident. (3) THE GUARD READS STAGED BLOBS, not diff text, and fails closed. Four shapes staged a banned name that no diff-text reading could see: a content line beginning `++` becomes `+++` and is dropped with the headers, a NUL-bearing blob has no textual diff at all, a committed `.gitattributes` with `-diff` disables the reading repo-wide in one line, and a rename is status R which the `ACM` filter excluded (as it excluded T). `git show :<path>` over `--name-only -z --diff-filter=ACMRT` asks the question the guard is actually asking and has no shape to route around. Every git step is checked, replacing a `|| true` that turned any failure into a clean pass: a check that could not run must never be indistinguishable from one that passed. (4) CONTAINMENT IS THE WRITE PATH'S JOB. Reads and writes resolve through `os.Root`, so a symlinked `.abcd/.work.local` cannot land the private patterns outside the repo while the verb reports the in-repo path; the verbs resolve the repo root as the rules loader does rather than trusting cwd, which had created a second nested store that the root-anchored gitignore does not match and the guard does not read; `add --private` refuses when git does not ignore the store path, because the layer's whole safety is that the file is untracked and the guard cannot catch its own source; and both stores hold the shared flock across load-modify-write. (5) A PATTERN NEVER TRAVELS IN ARGV. The hook passes it to grep via `-f -` on stdin, the verb accepts `-` to read one line from stdin (the documented form), and a flag-parse failure withholds the offending token instead of quoting it. Accepted trade on that last point: `SetInterspersed(false)` was rejected because the documented `add --private KEY "PATTERN" --json` puts a persistent flag after the positionals; a flag-error surface closes the leak without breaking it, and a test pins the trailing flag. Residual, stated rather than fixed: the guard still trusts an out-of-repo `sync-banlist` executable it neither verifies nor sandboxes, and a garbling refresh is only partly caught (the zero-entry warning); and the empty-`$toplevel` branch is guarded but not test-covered, because git resolves the repo before invoking a hook.
- 2026-07-30 — itd-74 (round 6), SECOND fix pass after two fresh adversarial reviews returned BLOCK on the first fix pass's OWN code. Detector-first throughout, each fix reverted in place and watched to fail before it passed. The guard's staged-content read was the cluster: it now reads each staged blob stage-explicitly (`git show ":0:$path"`, closing the `0:README.md` rev-magic bypass), derives staged modes from `git diff --cached --raw -z` so a gitlink (mode 160000, no blob here) is SKIPPED rather than fail-closing every commit forever, scans the staged PATH strings alongside content so a banned name in a filename is refused by key, refuses any staged path under the local tier (the private store must never be committed, and the guard cannot catch its own source), and announces the format and entry count it actually read before the scan so a stripped `# abcd-banlist: keyed` line cannot silently downgrade every keyed entry to a non-matching whole-line pattern. The verbs: `add` now proves the composed `KEY<space>PATTERN` line round-trips (a whitespace-only pattern wedged the store as `key  `; leading/trailing whitespace was silently trimmed so the enforced pattern differed from the validated one) and rejects a NUL the two readers disagree on; the stdin path reads all of stdin and refuses trailing data rather than storing the first of a multi-line pattern silently; the verb resolves the git working-tree toplevel — the exact root the guard enforces at — so a repo nested under a parent holding a .abcd/ no longer writes its store into the parent where the guard never reads it; no cobra path echoes an unknown token (a would-be private value); the inert verdict is driven by the grep probe, not a static screen that falsely called `\b`/`\w`/`\s` "matches nothing" (GNU grep implements them); `remove --private` deletes ALL lines for a key (a duplicate no longer survives under a key the report calls gone) and `remove --public` no longer lets a bare hand-curated key shadow the managed `names/<key>` target it owns; the read path reports a store git does not ignore; and a mutation's entry count excludes unparseable lines so it agrees with `list`. Records corrected: the brief and this ledger had the RE2-vs-ERE divergence backwards — Go ACCEPTS `[a-z-.]` while grep -E refuses it (exit 2), the opposite of the earlier wording; engine.go was already right. Residual, recorded not fixed: the fsutil lock files are 0644 and never unlinked (a shared primitive; the banlist lock lives in the 0700 gitignored tier and the file is empty), a public-only add still creates the gitignored local-ephemeral tier to place its lock (nothing leaks; relocating hits the atomic-rename-inode problem), the `.abcd`-symlink asymmetry and a hand-written store's 0644 mode; and the out-of-repo `sync-banlist` trust and the empty-`$toplevel` branch carry over from the first pass. `RemovePrivate` surfaces the not-ignored condition only via the shared read path (list/bare), since `PrivateResult` carries no health field.
- 2026-07-30 — itd-74 (round 7), increment 2: the scaffolding and honest-reach halves (AC5, AC7), closing spc-20's mapping. `ahoy install` writes three artefacts, all create-if-absent: the committed guard hook at `.githooks/pre-commit`, a `.abcd/docs-lint.json` carrying an EMPTY public banned-names family, and the documented private stub in the gitignored local tier. Three decisions worth the record. (1) THE SCAFFOLDED HOOK IS A GENERALISATION, NOT A COPY. It drops this repo's iss-62 identity gate and the itd-76 dogfood refresh and keeps the name guard alone, so a byte-equality drift test between the two files would be wrong; the template is proved BEHAVIOURALLY instead — the scaffold test installs it into a real temp repo and drives three commits through it (loud warning and exit 0 with no entries, refusal naming the key alone with the pattern absent from all output, refusal to stage the store itself), which is the only evidence that distinguishes a guard that works from one that merely parses. Rejected: a symlink from `.githooks/pre-commit` to the embedded default (zero duplication, but breaks a Windows checkout and would have rewired increment 1's own test harness), and copying this repo's hook verbatim (it would scaffold repo-specific gates into every managed repo). (2) THE PUBLIC FAMILY IS SEEDED EMPTY. abcd cannot know which names a repo may not publish, and a ban nobody declared would fail a build over a word the maintainer never chose; the array's PRESENCE is what makes `banlist add --public` usable, which is all AC5 asks for. A docs-lint config that exists but carries no usable array is a NON-resolvable diagnostic gap: the config gates CI and a contributor owns it, so abcd reports the fault and never rewrites a file it cannot read. (3) THE STUB'S GAP IS RESOLVABLE ONLY ONCE THE FENCE COVERS IT. Writing the store into a repo git would track is the exact hazard the layer exists to prevent, so the stub write runs after the visibility step and is gated on the canonical `.gitignore` block being on disk; advertising it as resolvable regardless would leave a repo permanently "partial" on a fix apply refuses to make (the markerSymlink precedent). No new gitignore entry was needed — iss-169's fence already covers `.abcd/.work.local/` under private and `/.abcd/` under public — so AC5's gitignore clause is pinned by a new test against real `git check-ignore`, keyed to `banlist.PrivateRelPath` rather than to a spelling of the tier, and it passed on first run. AC7: the reach sentence lives once as `banlist.PrivateReachNote` and travels INSIDE the reported state (`BanlistHealth.Reach`), not as renderer prose — a machine consumer reading "hook installed" beside a present store would otherwise draw exactly the wrong conclusion, and the JSON envelope is a report surface too. The health pass reads no entry and spawns no subprocess: the store's content is the secret, and a status board is the surface that must not hold it. Stub examples are all commented out (a fresh scaffold parses to zero entries, so the guard warns loudly rather than looking like protection) and every illustrative value is a reserved documentation value or a persona-derived fixture host, judged by the repo's own network-identifier detector — with a control value asserted flagged first, so the assertion cannot pass vacuously. Residual, recorded not fixed: abcd does not set `core.hooksPath`, so a clone arms the hook by hand (the record scopes AC5 to the hook being committed, and silently repointing a repo's hooks path is a git-config change no acceptance criterion asked for); and the scaffolded template and this repo's prototype now share ~250 lines that no mechanism holds in sync. FIX PASS after two fresh adversarial reviews returned BLOCK on this increment's own code; detector-first throughout, each behaviour change watched failing first. (A) THE FENCE IS GIT'S DECISION, NOT A TEXT COMPARISON. Both the stub write and the gap's resolvability were derived from `gitignoreBlockDrifts` — a set comparison of the abcd-managed .gitignore block — which answers a different question from the one that matters. A repo can carry a byte-perfect block and still track the store (a negation after it, a tracked tier), a fully-configured repo never reaches the visibility step at all, and `stepConfigValues` returns nil whenever ANY unrelated config value is missing under a declined ConfigChange, which silently disabled the write while `localTierIgnored` reported the gap as resolvable — a permanent partial. Both now call `gitutil.IsIgnored`, the same primitive increment 1's `requireIgnoredStore` uses, evaluated on disk after the visibility step, and skipped outside a git repo for the same stated reason (a check cannot demand proof no one can supply). Health carries the verdict and a present store git would track is called out on every surface. (B) CONTAINMENT. The three writes used `os.MkdirAll` plus an atomic rename, and a repo that commits a symlink at `.githooks` — which a checkout materialises before abcd runs — took the 0755 hooks OUTSIDE the repo while every surface reported the in-repo path; watched, and confirmed by the reverted code writing `pre-commit` into the link target. They now go through `fsutil.CreateExclusiveIn` under an `os.Root` opened at the repo, which is both guarantees at once (no clobber, no check-then-write window) and refuses symlink traversal at every level. The local-tier case was already contained by increment 1's `os.OpenRoot` read path, which reports such a store as unreadable rather than absent. (C) BOM FAIL-OPEN, IN BOTH READERS. A leading UTF-8 BOM made the first line differ from the format declaration, so a KEYED store read as LEGACY: every keyed entry became a whole-line pattern matching nothing a commit contains, while `entries` stayed >=1 so the zero-entry warning never fired — a store that looked healthy and checked nothing, reproduced as a real commit going through. Fixed in the Go parser, the scaffolded template, and this repo's own prototype; and a first line that is not the declaration but WOULD be after stripping leading blanks is now a DAMAGED declaration that fails closed, because reading it as legacy is the same downgrade by another route. (D) MERGE-COMMIT BYPASS. git runs no pre-commit hook for a merge, so a banned name entered history the moment a branch carrying it was merged; a `pre-merge-commit` half now runs the same guard through one implementation (a shim, not a copy) in both the template and this repo. The reach sentence widened with it: `PrivateReachNote` now names what a hook cannot see — a rebase, `git am`, a cherry-pick, `--no-verify` — because "machines that have opted in" is necessary and not sufficient, and the shorter sentence let a reader believe an opted-in machine was covered. (E) TWO CLAIMS WITHDRAWN. A foreign pre-commit hook was reported as the abcd guard: presence is not identity, so each template carries `# abcd-name-guard: v1`, a hook without it is a non-resolvable diagnostic abcd never replaces, and "installed" became "committed" with the arming instruction beside it (git runs the hook the clone's hooks path selects, which abcd neither sets nor fully observes). And under `visibility: public` the fence ignores the anchored `/.abcd/`, so the "committed, CI-enforced" public family is untracked exactly where public exposure is the risk — detection now reports `banlist.public_family_ignored` and the board reads NOT ENFORCEABLE. The placement question is NOT resolved here (moving the file amends the iss-169 record; carving an exception gives up its one-switch property) and is captured as iss-176 for the maintainer, with three candidate reconciliations. Minors in the same pass: the private layer's shape comes from a new `banlist.SummarisePrivate` — the shared parser without the per-entry grep, so a status pass costs no subprocess per line — the public family's four faults report apart (an unreadable file is not an absent array), the local tier's 0700 is asserted rather than assumed from the creating call, the staged-path refusal is case-folded, the format is announced before the parse loop, and stale scratch files are swept. Residuals the reviews verified as accepted: the scaffolded template and this repo's prototype share ~250 lines that no mechanism holds in sync (byte-equality would be wrong — the template drops the identity gate and the dogfood refresh — and the template is proved behaviourally instead); `--no-verify`, rebase and server-side pushes remain uncoverable by construction; GNU/BSD grep divergence stands as increment 1 recorded it; commit messages, tag names and branch names are outside the guard's scope (it reads staged blobs and paths); a scaffolded repo gets no CI wiring for the public family; abcd still does not set `core.hooksPath`; and `a.note()` renders absolute paths across every apply step, pre-existing and repo-wide, captured as iss-177 rather than half-fixed here. FIX PASS 2 after a second pair of independent BLOCK reviews, both on the fix pass's own code. (F) THE MERGE SHIM NEVER STANDS IN. It was written whenever its own gap was present — including beside a FOREIGN pre-commit hook — so abcd's marker landed in the shim, the board read "pre-merge-commit hook committed", and merges stayed unchecked; worse, the maintainer's hook silently began running on merge commits, which git never did, so apply was taking over exactly the wiring the foreign-hook gap disclaims owning. The shim is now written only beside abcd's own guard (including one the same step just wrote — keyed on the artefact's STATE, not on a gap a fresh repo deliberately does not raise), and `banlist.merge_hook_inert` reports the alternative. The wrong assertion in the increment's own test — which asserted the buggy behaviour — was flipped and watched fail. (G) A COPY OF THE STORE IS A MISTAKE-NET, NOT AN ADVERSARY-NET. The staged-path refusal matched only the local tier, so `cp` of the banlist to notes.txt, or a `git mv` out of the tier, committed every pattern in clear while the guard announced a clean check — the entries cannot catch their own text, because they are escaped regular expressions and `carol-server\.example\.net` does not match itself. Three tests now: the tier path (not escapable, and applied to a rename's SOURCE path too — the loop had been overwriting it with the destination and discarding the one field that identified the file), a staged blob whose FIRST LINE is the format declaration, and a staged path whose basename is the store's filename. Their reach is stated rather than implied, here and in the brief and the spec: a copy with the declaration stripped, altered by a byte or displaced below a preamble passes the first-line test; a LEGACY store declares no format at all and passes it always, and passes the basename test the moment it is renamed; and the rename-source test only fires for a store git already tracks. What they catch is the accident that happens — a `cp`, a `.bak`, a stray duplicate — and they are NOT widened further on purpose: matching a store's key list or scanning deeper into every staged blob would route the secret through more code paths to catch fewer accidents. Round 3 also found the other direction: the first-line test blocked this repo's OWN committed corpus fixtures, and `--no-verify` is an off switch for the whole guard rather than a per-file escape, so the refusals now honour a published second-line `# abcd-banlist-example` marker — exempting a blob from the COPY tests and from nothing else, its content still scanned against every entry — and the two keyed corpora carry it. The tests also moved OUT from behind the absent-store early exit (whether this machine has a store says nothing about whether a commit is carrying one, and that exit was itself how the rename escaped), and the guard no longer creates its local tier in a repo that never opted in: with no store it scratches in a 0700 temp directory instead of leaving an unfenced abcd directory behind on every commit. (H) THE DECLARATION IS LINE 1 OR NOWHERE, in all three readers and for BOTH formats. A blank line or a comment above it, a duplicate below it, or any prefix bytes before it (a UTF-16 byte-order mark, an editor artefact — caught by a suffix test rather than a byte-class check no portable shell has), is a damaged declaration that fails closed. Round 3 found the header scan gated on the store not already being keyed, so a keyed store with the declaration repeated on line 2 read as healthy to Go — "present, keyed, 1 entry" on the status board — while the shell blocked every commit; a store one reader calls healthy and the other refuses is worse than either verdict alone, and a fourth shared corpus now drives both readers over exactly that file. (I) THE ENVIRONMENT IS PINNED FIRST, in all four hooks: xtrace off, IFS, LC_ALL and PATH as the FIRST statements, before anything is read or run — the pin had been landing after `rev-parse`/`cd`, and the merge shim had none at all while resolving its delegate through an external `dirname` (now `${0%/*}`, pure parameter expansion). The pin NARROWS the substitution class and does not close it: two of the pinned directories are user-writable on a typical developer machine, and `#!/usr/bin/env bash` resolves the interpreter through the inherited PATH before any of it runs — the earlier "root-owned" wording in this line was wrong and is corrected here. Every external tool the hooks call is probed after the pin so a missing one blocks loudly instead of surfacing as a mute exit 127, and a nonstandard prefix extends the pin through `git config --local abcd.guardPath` — read through the already-pinned git, and deliberately NOT an environment variable, since a repo-scoped direnv sets the environment and would reopen the exact hole. Minors in the same pass: scaffolded modes are chmod'd explicitly because `CreateExclusiveIn` and `MkdirAll` are umask-masked and a stripped exec bit turns the shim's fail-closed branch into a permanent merge block; the marker is matched by PREFIX so a v2 template cannot reclassify every v1 hook as foreign; `storePathIsSafe` distinguishes "not a repository" (skip, as before) from "repo-shaped and git will not answer" (fail closed) — `InRepo` is false for git-absent-from-PATH too, which the old comment's justification did not cover; `hooksPathArmed` resolves both sides before comparing, so an absolute `core.hooksPath` is no longer read as unarmed and told to downgrade itself; the absent-public-family gap is non-resolvable where git would ignore the path and abcd declines to write a config it would immediately call unenforceable; the detection envelope omits the banlist object for an unmanaged folder rather than serialising undeclared zero states; detection reads now resolve through the same containment root the writes use; the reach note is explicitly non-exhaustive and names `git revert` and a reapplied stash; and `abcd ahoy` scaffolds the `.githooks/* text eol=lf` attribute into managed repos (append-if-missing, never a rewrite) rather than leaving them the CRLF-shebang failure this branch fixed for abcd itself. CORRECTION to this line's earlier claim that the health pass "spawns no subprocess": it always did — it now spawns four (one `rev-parse`, one batched `check-ignore` for both paths, one `config --local`, and one `check-attr` for the hooks' EOL), down from six or seven, and `ahoy.Detect` is on the session-hook path, so that count is a cost paid per prompt. One earlier test was vacuous and is fixed rather than deleted: the local-tier half of the symlink containment test passed with containment reverted, because the escaping store is classified unreadable before any write is attempted, so it now asserts that protecting state by name. FIX PASS 3 (converge). MARKER IDENTITY: `classifyGuardHook` matched the marker as a substring of the whole blob, so a foreign hook that merely MENTIONED it — a comment, a grep for it — classified as abcd's own, the board claimed coverage, and the merge shim was written beside it; identity is now a whole LINE matching `^# abcd-name-guard: v<digits>$`, with the version a group so a v2 template cannot reclassify v1 hooks. The same line-wise fix went to the `.gitattributes` pin, where a commented-out attribute had read as pinned. REACH ACCURACY: a fast-forward `git pull` creates no commit and so runs no hook — the commonest uncovered path of all, now named — and the stash claim was probed FALSE (`git stash pop` then commit runs pre-commit normally) and removed: a bypass list naming a covered path teaches a reader to distrust the rest of it. Minors: the merge-inert gap no longer asserts a delegating shim that may not exist; `createContained` pinned modes on directories it did NOT create, widening a maintainer's deliberate 0700 to 0755, and now pins only what it made; `repoShaped` walked no ancestors, so in a repo SUBDIRECTORY with git unavailable the stub would have been written into a tracked tree; the stale-scratch sweep and the tier creation are conditional on the tier being the scratch home; and the foreign-hook gap's `Required: false` is documented as deliberate (a required gap nothing can resolve is a repo permanently reported incomplete for a state its maintainer chose). Also re-classified: `stepBanlist` now asks disk what occupies the hook paths rather than trusting the detection snapshot taken before the earlier steps ran. FIX PASS 4 (prose currency plus one-liners). The two CHANGELOG bullets, internal/README.md, and both brief surfaces described THREE scaffolded artefacts and a stale reach; there are FIVE (both guard hooks, the .gitattributes LF pin, the public family, the stub), and the reach now names a fast-forward `git pull` — the commonest uncovered path of all — and no longer names a reapplied stash, which probing showed IS covered and which reach_test.go now pins out. Security minors: an inherited shell FUNCTION shadowing `grep` defeated the PATH pin entirely (probed — the guard reported a clean check under `BASH_FUNC_grep%%=() { return 1; }`), so all four hooks now `unset -f` the commands they run as their first statement after the xtrace guard; `abcd.guardPath` refuses a value with an empty PATH element, which every consumer reads as the current directory; the scratch-directory trap is installed before the three inner mktemps that could exit past it; and `mktemp -d` takes an explicit portable template. The `.gitattributes` classification stops pattern-matching abcd's own line and asks `git check-attr eol` instead — git is already this package's authority for the ignore question, and a later `* text eol=crlf` overrides a line a regex still calls pinned; the line-anchored regex survives only to keep an append from duplicating. The marker regex tolerates a trailing CR, so a CRLF-mangled abcd hook classifies as OURS-but-unpinned and install can heal it rather than reporting a foreign hook for ever. Three residuals are widened rather than fixed, in the brief and the spec: the environment pin narrows common accidental and repo-scoped routes and closes nothing (anyone controlling the committer's environment wins); the example escape keeps content scanning so it cannot smuggle a plaintext banned name, but it protects no copy of a STORE — escaped patterns never match their own text — and a live store carrying the marker exempts every copy of itself; and the marker asserts identity, never integrity, so any file quoting it is treated as abcd's guard. Lifecycle: itd-74 sits in shipped/ with its fidelity review OWED (receipt rcp-3ceed52bdb99). That is the designed follow-up, not an omission — `abcd spec close` stamps the receipt and the review is a separate verb (`abcd intent review`), which record-lint and audit accept; no record file was hand-edited to produce or to clear it.
- 2026-08-01 — iss-96 (item A6), a VERIFICATION milestone rather than an implementation: the transcript scanner's tracked coverage gaps re-checked against the network-identifier set A2 folded into `DefaultPatterns`, and the residue re-scoped in place. Two findings, both empirical rather than inspected. (1) A2 DID reach this path — every scanner consumer inherits the network set, so a non-reserved address or a LAN/device hostname in a captured transcript is now a finding (all five kinds — `net:ipv4`, `net:ipv6`, `net:mac`, `net:lan_hostname`, `net:device_hostname` — exercised on the transcript path itself in the same corpus, not inferred from their presence in the pattern set, and a flagged private address asserted REDACTED in the stored record at the store boundary); that class is therefore newly covered on this path since iss-96 was captured — it was never part of the entry's residue, which names unanchored secrets and home paths only, so this is coverage the re-check ESTABLISHES rather than a gap it closes. (2) The classes iss-96 actually names — a 40-character secret-key value with no prefix, a bare password, a prefix-less API token, and now a genuinely high-entropy 40-character value — still produce ZERO findings at the transcript path's own entry point, with or without a `password:`, `api_key =` or `Authorization: Bearer` key name beside them, because the TOKEN patterns are prefix-anchored APART FROM `rp_session_key`, which keys on the literal JSON field name `"sessionKey"` rather than on a generic key-name class, and no pattern measures entropy. The set as a WHOLE is not uniformly prefix-anchored and must not be described as such: the network kinds are an allowlist inversion, the identity kinds key on the probed identity, `token:pem_private_key` is a literal header, and `Pattern.SkipAt` already lets a pattern accept or reject a match by what SURROUNDS it — none of which reaches an unlabelled value, which is why the residue stands, and two of which (`rp_session_key`, `SkipAt`) are the shipped precedent remediation option (b) would generalise rather than introduce. Disposition: iss-96 stays OPEN with a dated verification section appended in place (the iss-24 precedent; a `capture resolve` would have had to state an action nobody took, and the ledger's resolution field is exactly that claim), re-scoped from "coverage may have moved" to a single unresolved DECISION between an entropy/charset detector with a length floor, key-name context matching, and the opt-in external-scanner adapter over this path only — three options that differ in REACH, not merely in false-positive cost, and saying otherwise would misdescribe them: (a) reads the value, so it is the only one that reaches an UNLABELLED value; (b) covers labelled values only and is structurally incapable of this entry's own first-named case, a bare secret-key value with no key name, because there is no key to match; (c) has whatever reach its ruleset delivers, and the measurement recorded in the entry corrects an earlier one that was wrong in both fixture and conclusion — re-run 2026-08-01 with pinned, sha256-verified gitleaks 8.24.3 default rules (`gitleaks dir`, no repo config) over a NAMED fixture, one specimen per line, comprising every case of `TestTranscriptPathMissesUnanchoredEntropy`'s table, every line of the transcript `TestCaptureStoresUnanchoredEntropyVerbatim` captures, and delimiter probes on the same values: 21 lines, 6 findings, all `generic-api-key`, whose condition is keyword + `=`/`:` DELIMITER + entropy at or above 3.5 bits per character, so `password: <passphrase>`, `--password=<passphrase>`, `api_key = <high-entropy>` (the branch's own corpus case), `session_token = <high-entropy>` and `session_token: <high-entropy>` are all CAUGHT, while bare values with no key name, key names separated from their value by prose rather than a delimiter (the store fixture's hand-built `session_token <high-entropy>` line, which the earlier measurement mis-read as a key-name failure when the cause is the absent delimiter), and every sub-floor repetitive specimen whatever key name precedes it, all pass — so (c) as shipped reaches LABELLED high-entropy values, subsuming much of (b) and adding an entropy floor on top of it rather than being the narrowest option, and (a) remains the only one that reaches an UNLABELLED value; the anchored `ghp_` control passing is a fixture artefact (36 repeated characters fall under gitleaks' own entropy filter — a realistic-entropy `ghp_` token in the same run IS caught by `github-pat`), not a coverage claim. False-positive cost is the SECOND axis and lands hardest on (a) — a redaction false positive corrupts the record redaction exists to preserve — so reach and cost together are the open question and the bar is the maintainer's to set: grill-then-implement, not autonomous. The verification is PINNED in the tree rather than asserted in prose: `TestTranscriptPathMissesUnanchoredEntropy` at the pattern-set boundary and `TestCaptureStoresUnanchoredEntropyVerbatim` at the store boundary (each specimen present verbatim in the written record, the anchored token absent AND its `fingerprintSpan` fingerprint present — the byte-level mirror of `maskSecret` that `sealLine` actually applies on the Capture path, not `maskSecret` itself — since absence alone would also be satisfied by a store that dropped the line). THE PINS' REACH IS THE PATTERN SET, and the earlier claim that they "fail by design, never a silent pass" was too broad on two counts, both now fixed: the original specimens all REPEAT and measure 3.12, 2.16 and 2.75 bits per character, below the ~3.5-bit floor an entropy detector conventionally uses, so an entropy detector could have landed with every pin green — a 5.32-bit specimen assembled at run time from a fixed-seed shuffle, with its entropy asserted in-test at or above 4.5 bits per character AND its exact value asserted against a golden constant on both sides (entropy alone cannot see the two duplicated generators diverging, since a changed stream would still measure ~5.32 bits; the golden also pins determinism, and is gitleaks-clean as a literal because a bare 40-character alnum run carries no keyword and no delimiter), now closes that hole; and option (c) never consults `DefaultPatterns` at all, so NO ScanText-level pin can alarm for it by construction, which is stated in the entry so a stale close cannot lean on the pins alone. Within the pattern set they do fail by design — a charset/length floor, key-name matching or any new `DefaultPatterns` entry THAT REACHES THESE SHAPES trips them (the qualification is the test file's own, and it matters: an entry reaching nothing in the corpus rightly leaves both pins green) — and that failure is the signal to re-point them and close the entry. Never close on assumption: the anchored controls in the same corpus are caught, and one of them sits INSIDE the negative test, which is what proves the specimens reached a working detector rather than a mis-wired or emptied one; the negative assertion is scoped to the token/secret kind family, so an unrelated widened pattern fails it under a DIFFERENT message rather than being credited as entropy coverage — and that message now names BOTH readings, because a secret or entropy detector shipping under a namespace the family list does not know (`secret:entropy`, say) arrives down the same branch, where telling the reader to adjust the specimen would hide the very coverage growth the pin exists to notice. Every credential-shaped specimen is assembled at run time — the discipline network_test.go already states for flagged network identifiers — so, beyond the one golden constant this entry discloses as measured clean, no literal of that shape enters this repo's history for a full-history secret scan to fire on.
- 2026-08-01 — guard round (iss-148, iss-159, iss-144): the matcher's wrapper walk now steps over a wrapper's OWN arguments rather than its name alone, off two explicit tables — the flags each wrapper documents as taking a value, and the mandatory operand no flag stepping can reach (`timeout [OPTIONS] DURATION COMMAND...`, where `timeout 30 rm -rf /` read as a command called `30`). That was the sharpest of the three items and the reason to take it first: `sudo -u bob <hazard>` read `-u` as the command name, so one extra token turned an entry the registry DOES describe into an allow, and a guard that has stopped refusing looks identical to one with nothing to refuse. The complementary miss is deliberate and now stated rather than implied — a value flag the table does not name (a bundled `sudo -Hu bob`) is not stepped over, and a miss is a non-match, never a false block. `xargs`, `timeout` and `exec` joined the wrapper set. Backtick substitution reaches command position through the same segment flush the grouping parens already gave `$( … )` by accident, scoped to parity with today's behaviour (unquoted only) rather than POSIX completeness, so the two spellings of one substitution stop disagreeing. Registry content took four additive Pattern fields, each optional and per-repo overridable like the rest: `subcommand2` for a two-level grammar (`gh repo list` and `gh repo delete` share their first level and only one is a hazard), `flag_values` for a flag's SETTING rather than its presence (presence alone would force a choice between missing `-X DELETE` and refusing `-X GET`; the value is compared without regard to case, since a constraint a change of case walks past is not one), `arg_paths` for a root segment AND an exact depth (the depth is the entry's scope: `repos/{owner}/{repo}` IS the repository, while DELETE on a branch ref under it stays ordinary work), and `arg_prefixes` for a hazard carried by an operand with no flag to look at (`git push origin +main:main`). The empty form of each is a load-time rejection, because an empty flag group defangs and an empty prefix over-blocks, and both are invisible in the file. Two behaviours are pinned as tests rather than left to be rediscovered: an api path written as a fully-qualified URL is NOT matched (a stated limit, the family of the payload gap), and `--help` is not an exemption anywhere in this registry — exempting it would mean teaching every entry which of its flags mean 'do nothing', and a guard that reasons about intent can be argued out of refusing. iss-144 needed no code at all: its incident shape was already known-bad fixture #1 and the incident capture already known-good #1, so the only change was adding the three-step form it actually occurred in (mkdir → cd → `rm -rf *`) to the corpus, making the resolve verifiable rather than asserted. Residual and deliberately not acted on: a `cd` in one tool call and an `rm -rf *` in the next is still an allow, since the tokenizer never sees two invocations as one candidate — a harness-level concern, not a registry one.
- 2026-08-01 — guard batch (iss-159/144/148), FIX pass after two fresh adversarial reviews returned BLOCK (security) and FIX FIRST (correctness). Detector-first throughout: each fix has a test watched failing before it and passing after. Two claims in the entry above are superseded by it. First, "backtick substitution reaches command position through the same segment flush the grouping parens already gave `$( … )` by accident" — the flush was the bug, not the mechanism. `flushSegment` permanently ENDS the enclosing command's segment, so every flag and operand written after a substitution closed started a new, unrelated command: `cd scratch && rm ` + a backticked substitution + ` -rf *` went from block on main to allow on this branch, a regression this branch introduced, and the identical hole already existed for `$( … )` on main (`rm $(true) -rf *`). Both are fixed together rather than one fixed and one documented, because the mechanism to fix them is the same: a substitution pushes a FRAME that parks the enclosing tokens, accumulates its own content, flushes that as its own segments (a hazard inside a substitution is still matched — the point of the original fix), and restores the enclosing tokens so the segment resumes. Segments come back in SOURCE order rather than at the close boundary, which is what keeps `precededByCD` reading a cd written before a substitution as preceding it. A bare `(` deliberately keeps the old flush: it is a subshell group, not a substitution inside a command, and it genuinely ends the token run before it — `f() { rm -rf *; }` would otherwise glue the body onto the function's name and lose the `rm` from command position. Imbalance degrades rather than errors (a close with no open falls back to the plain flush; an unterminated span is unwound at end of input rather than dropped, since a hazard inside it still executes), consistent with the file's existing philosophy that only an unterminated quote or heredoc — which changes what the REST of the input means — is `ErrUnparsableCommand`; a real shell rejects unbalanced parens itself, and this guard's job is to refuse to misparse into an allow, not to validate shell grammar. Second, "an api path written as a fully-qualified URL is NOT matched (a stated limit)" — it is matched now. `gh` passes an absolute URL through to the API unchanged, so the URL form was a real bypass of a real repository deletion, and documenting it was the wrong end: an operand is normalised to its path (scheme/host, query, fragment dropped) before the depth check, which only ever shortens, so the depth limit that keeps deeper work allowed is untouched. The host is deliberately NOT inspected — deciding which hostnames are the real API is a lookalike-domain problem this guard cannot settle, and normalising every authority away can only make the check see a path it would otherwise have missed. What is left is narrower and now disclosed in all four surfaces rather than in a Go comment and a test: a host that MOUNTS the API under a prefix (GitHub Enterprise Server's `/api/v3/`) is not seen, because matching a root segment wherever it appeared would falsely refuse `DELETE /teams/{id}/repos/{owner}/{repo}`, which removes a repository from a team and destroys nothing. Third item, no supersession: `Validate` rejected a dashed `subcommand`/`subcommand2` but not the two operand fields this branch added, so `"arg_prefixes": ["-f"]` and an `arg_paths` root carrying a slash both loaded clean and could never fire — the "looks armed, never fires" class the validator exists to catch. Both are load-time rejections now, with the well-formed shapes pinned as accepted so the checks cannot quietly take the fields out of use.
- 2026-08-01 — guard batch (iss-159/144/148), REVERT of the backtick sub-part after a THIRD adversarial round; the other three sub-parts stand. Following backtick command substitution went in, then grew a frame/stack mechanism intended to hold two properties at once — a substitution's content checked as its OWN segment (so a hazard hidden inside one is matched) while the ENCLOSING command's tokens, chain index and command position survived the boundary. Three rounds, three fresh independent reviewers, three BLOCKER findings, all in that mechanism rather than in one instance of it: (1) a substitution before trailing flags truncated the enclosing segment, so `rm` + substitution + `-rf *` went block to allow; (2) the frame fix for (1) renumbered the enclosing command's chain when a newline fell inside a substitution, defeating every `after_cd` entry (`cd scratch && rm -rf $(\nfind . -name x\n)`); and (3), the one that decided it, a substitution in LEADING command position left a bare `$` as the enclosing command's argv[0], so `commandOf` returned `"$"` and NO entry could match — that is not a hole in `rm-rf-after-cd-chain`, it is a hole in the registry, reachable by prefixing any hazard with `$(true)`. Two more in the same round: a bare `(` nested inside `$( … )` mis-popped the frame and re-opened (1) in a narrower shape, and an unterminated backtick failed OPEN. DECISION: revert rather than patch a fourth time inside one round. A mechanism that has produced a security-relevant regression on every attempt is not converging, and each fix has been strictly worse than the last in blast radius — the third defeats the entire registry. `internal/core/guard/tokenize.go` is restored byte-for-byte to `main`: no backtick handling at all (literal text), `$( … )` exactly as it was, and the four guard.json fixtures the frame mechanism added removed with it. The two claims of the entry above are therefore WITHDRAWN, not superseded: backtick substitution does NOT reach command position, and the enclosing command does NOT survive a substitution. The literal ledger ask ("a backtick command substitution is not followed, while the dollar-paren form is") asked for PARITY with `$( … )` as it behaves today, and parity is what leaving both forms untouched delivers — the gap is disclosed again as a stated v1 limit on all four coverage surfaces (brief, plugin command, CLI long help + generated reference, package map), which is the honest form of it. Recorded rather than quietly carried: `$( … )` on `main` already loses the flags written AFTER a substitution — `rm $(true) -rf *` does not read as a recursive force delete — a pre-existing gap this round's review process surfaced and did not introduce, undisclosed before it and undisclosed still, because closing it is the same design problem as following a backtick and the two should land together or not at all. iss-148 is re-scoped IN PLACE back to open on that one sub-part (the iss-96 precedent: a `capture resolve` would have to state a resolution for scope that is not settled), with the three shipped sub-parts and their commits named in the entry; iss-144 and iss-159 are untouched and stay resolved, neither being implicated in any review finding. Deferred as its own future work, and explicitly NOT a patch to the reverted code: a substitution model that follows the payload AND preserves the enclosing command across the boundary, which needs the tokenizer to carry a command-position stack rather than a segment flush — the reason three attempts to bolt it onto the flush all leaked.
- 2026-08-01 — itd-105 / spc-21 implementation: the plugin provisions its own binary from a committed POSIX-sh `hooks/bootstrap.sh` wired as the FIRST `SessionStart` hook, per spc-21's Decisions (fast path on an executable binary, darwin/linux × amd64/arm64 gate, atomic `mkdir` lock with a 10-minute stale break, same-origin SHA-256 against the release `checksums.txt`, atomic rename install, `.binary-meta` provenance). Three judgment calls a reviewer should see. (1) The script's TEST-ONLY overrides are TWO env vars, not one: `ABCD_BOOTSTRAP_BASE_URL` (download base, as spc-21 names) plus `ABCD_BOOTSTRAP_API_URL` (release-commit lookup base) — the release-sha resolution hits a different URL shape than the asset download, and folding both behind one variable would have made the fixture server lie about which endpoint was called. Both mirror `ABCD_BIN_TARGET`'s pattern. (2) `release_sha` is read from the commits API (`/commits/<tag>`), NOT the releases API's `target_commitish`: this repo tags annotated (`git tag -a`) and creates the release from an existing tag, so `target_commitish` is the default branch name rather than a commit — a value that would have made the skew notice compare a branch name against a SHA. A value that is not 40 hex is recorded as `unknown` and renders nothing. (3) spc-21's CI-smoke line ("the existing smoke job additionally runs `sh hooks/bootstrap.sh` against a fixture server on both OS runners") is satisfied WITHOUT touching workflow YAML: the script-driving test lives in `internal/surface/cli/bootstrap_test.go`, which `go test ./...` already runs on both legs of ci.yml's ubuntu+macos `check` matrix, and it shells out to the committed script against an `httptest` fixture exactly as the AC describes. A bespoke workflow step would have duplicated that coverage while adding background-process management to CI. Rejected: a PATH fallback in the hook commands (spc-21 forbids it); signature verification (out of scope, same-origin checksums are the accepted bar); guessing `plugin_sha` from `git rev-parse` when the plugin root is not a commit-stamped cache directory (the meta file never guesses).
- 2026-08-01 — itd-105 / spc-21, FIX pass after two fresh adversarial pre-PR reviews (one security, one correctness) that converged independently on the same top items. Detector-first throughout; each behavioural fix has a test watched failing before it and passing after. Four claims of the entry above are superseded. (1) `ABCD_BOOTSTRAP_BASE_URL` / `ABCD_BOOTSTRAP_API_URL` were bare `${VAR:-default}` reads, which made them a code-execution primitive rather than a test seam: the binary AND the `checksums.txt` that verifies it are fetched from the same base, so whoever set the variable supplied both the payload and the manifest that "verified" it — verification became vacuous — and the installed binary is then run unattended as the Bash shell guard on every tool call. Both are now honoured for `http://127.0.0.1:*` / `http://localhost:*` only and REFUSED otherwise (refusing beats ignoring: a silently dropped override reads as a passing test), and the production calls pin `--proto =https --proto-redir =https` so no redirect can downgrade the transport or point at `file://`. (2) `release_tag` was scraped from `%{url_effective}` after `-L` on the asset download, which was verified against the real repository to land on `release-assets.githubusercontent.com/<numbers>?<signed query>` — no `/releases/download/<tag>/` segment exists there, so in production the tag was ALWAYS `unknown`, `release_sha` was always `unknown` behind it, and the entire version-skew notice (an intent AC) could never fire. The tag is now resolved BEFORE any download, from `%{redirect_url}` on an unfollowed request to `/releases/latest` (302 -> `/releases/tag/<tag>`), and both the asset and `checksums.txt` are then fetched from the TAG-PINNED path — which also closes the narrower hole that a release cut between the two downloads would have checked a new manifest against an old binary. The fixture was the reason this passed review-blind: it redirected to a conveniently tag-shaped URL, so it now mimics the real CDN shape and serves `checksums.txt` only from the pinned path. Consequence accepted: an unresolvable tag is now a refusal rather than a silent `unknown`, because the alternative is an unpinned pair the checksum step cannot honestly verify. (3) The bootstrap's two messages for a person — the unsupported-platform statement and the one-time `abcd ahoy install` suggestion — went to stdout with exit 0, which on `SessionStart` means model context and nothing the human ever sees; both now go to stderr with exit 2, matching `cli.go`'s existing "non-zero so SessionStart shows it; SessionStart never blocks" convention. This deviates from spc-21's Approach step 2, which specified exit 0 for the unsupported platform: the spec's reason (a reported condition, not a hook fault to retry) is preserved — nothing is changed, nothing is retried — but exit 0 also meant the statement was unreadable by its only audience. (4) `trap cleanup EXIT HUP INT TERM` where `cleanup` returns does not terminate on a signal; POSIX RESUMES the script, verified empirically here, so a SIGTERM mid-run deleted the temp dir and then carried on to report a checksum mismatch that never happened. Now `trap cleanup EXIT` plus `trap 'cleanup; exit 1' HUP INT TERM`, cleanup being idempotent. Smaller, same pass: `.binary-meta` is written through the temp dir and renamed in (a crash mid-write otherwise left a truncated `release_sha` that still parsed) and carries the verified `binary_sha256` so a later check can tell the binary that was verified from one that replaced it; `resolvedSHA` now requires forty lowercase hex rather than merely "not the literal unknown"; the skew notice is non-directional, since comparing two commits establishes that they differ and never which is ahead; the fast path requires a regular file, `[ -x ]` alone being true of a directory too; a run holding the lock sweeps `.bootstrap.tmp.*` orphaned by a SIGKILL (which runs no trap); `hooks/*.sh` is pinned `eol=lf`, a CRLF checkout of this one breaking the script that repairs the hooks rather than merely a hook.
- 2026-08-01 — itd-105 / spc-21, hook-ordering assumption DISCLOSED and defensively mitigated rather than verified. spc-21's Approach asserts "ordering within one event's hook list is preserved by the harness, so the binary-backed session hooks run after the bootstrap in the same event". That is an assumption about the harness, and nothing in this repository verifies it; if it is false, a fresh install's first session runs `abcd hook prompt-router-reset` and `abcd hook session-start` before any binary exists, and the intent's AC ("never a raw shell 'No such file or directory'") is missed by the two commands the bootstrap is supposed to protect. A PATH fallback is forbidden by the intent's Decisions and is NOT added. Instead both commands gain an inline existence check in `hooks/hooks.json` — the same shape as the `PreToolUse` guard entry's existing inline wrapper, which turns a bad exit code into a clean message — so if the assumption turns out to be false the failure is at least a plain-language message and a non-blocking exit 2 rather than a raw shell error, with no fallback binary resolution anywhere. This mitigates the SYMPTOM; the assumption itself still wants verifying against the real harness, and the maintainer should treat it as open. In the same file the bootstrap entry gains `"timeout": 240` (seconds): the three curl budgets plus the tag resolution total 180s worst case, which the harness's 60s default would have killed mid-run — which is precisely how the trap bug above and an orphaned temp dir would both have been reached in production. Deferred, deliberately, and bounded: two racers can both proceed when one breaks the other's ten-minute-stale lock, which last-write-wins resolves into a verified binary either way; and the trap is installed just after the lock is taken, leaving a signal window bounded by that same stale-lock recovery. Both are cheaper to state than to fix, and neither can install an unverified binary.
- 2026-08-01 — itd-105 / spc-21, THIRD-round fix pass: the environment-driven URL seam in `hooks/bootstrap.sh` is REMOVED rather than patched a third time. Two independent adversarial reviews had already returned BLOCK on two successive allowlist attempts at the same defect — first a bare `${VAR:-default}` read, then a `case http://127.0.0.1:*|http://localhost:*` glob — and both were defeated empirically by fresh reviewers: URL userinfo (`http://127.0.0.1:1@attacker.example`, where everything before the `@` is a username and the real host is the attacker's) walked straight through the glob, and the override path also cleared the `--proto '=https' --proto-redir '=https'` pin, so a redirect could downgrade the transport on top. The pattern was the finding, not the instances: an allowlist over URL syntax is a parser-differential surface, and the thing it guards is a code-execution primitive — the binary AND the `checksums.txt` that verifies it come from the same origin, so whoever names the origin supplies both the payload and the manifest that "verifies" it, and what is installed then runs unattended as the Bash shell guard on every tool call. So `ABCD_BOOTSTRAP_BASE_URL` and `ABCD_BOOTSTRAP_API_URL` are gone; `grep ABCD_BOOTSTRAP hooks/bootstrap.sh` now returns nothing, the script's only environment read at all is `${CLAUDE_PLUGIN_ROOT:-}` (a filesystem destination, never a fetch origin), and the two transport flags are literal on all four curl calls with no variable to clear. This supersedes claim (1) of the fix-pass entry above and the `ABCD_BOOTSTRAP_BASE_URL` mention in spc-21's Testing section, which named a seam that no longer exists. THE TEST SEAM MOVED TO THE TEST SIDE. `internal/surface/cli/bootstrap_test.go` reads the committed script's bytes, replaces the two quoted origin literals with an `httptest` base, writes the result to a temp file, `chmod +x`es it and execs THAT — the shipped bytes are what runs on a user's machine, with no test-only branch anywhere in them. Three properties make the copy trustworthy rather than a fixture that hides the bug (a failure this item already shipped once, when a friendlier fixture concealed that the release tag was always `unknown` in production): the match is the QUOTED, complete assignment value, so a longer origin cannot be prefix-rewritten into a mangled URL; each literal must appear exactly once before and zero times after, or the helper `t.Fatal`s naming the drift; and the happy path additionally asserts the fixture served at least one request, so a substitution that silently missed cannot pass by testing the real GitHub. The fixture speaks TLS (`httptest.NewTLSServer`) because the pin is unconditional, and trust is supplied through `CURL_CA_BUNDLE`/`SSL_CERT_FILE` — curl's own variables, which only ever ADD an issuer curl will accept and can never name a different server, so they are not the class of seam that was removed. Reviewer-facing residue, stated rather than implied: an actor who already controls the hook process's environment can still influence curl through curl's OWN variables (a proxy setting, a CA bundle), which is true of every curl invocation on the machine and is not something a shell script can close — such an actor can also replace `curl` on `PATH`; what is closed is the seam this script itself honoured. Detector-first throughout, each fix reverted in place and watched to fail: the reverted script reproduces the round-2 bypass verbatim (recorded fetch to `http://127.0.0.1:1@abcd-bootstrap.invalid/releases/latest`, no HTTPS pin), and against a live attacker-controlled release server on loopback the pre-fix script took four requests from it and installed its payload while the fixed script took zero and installed the genuine ELF release. Four smaller items in the same pass. (1) `plugin_sha`'s forty-hex gate rests on itd-105's unverified warrant that the harness names each plugin cache directory for its source commit; if that stops holding, every install records `unknown`, the skew notice goes silent permanently, and nothing anywhere says why — so the RAW basename is now recorded as `plugin_root_basename` beside the gated field, never compared and never rendered, purely so the failure is diagnosable from the file rather than only from a comment. Control characters are stripped from it and it is capped at 120 characters, since a directory name may contain a newline (forging a `key=value` line) or enough bytes to push `.binary-meta` past the guarded read and silence the notice by a second route. (2) `mkdir "$lock"` failing was indistinguishable from losing the race and took the same silent `exit 0`; it now refuses loudly when no lock DIRECTORY exists afterwards, because that case is permanent and repeats every session while the race case is transient and correctly quiet. (3) Both message sinks strip control characters from the values they echo, and the skew notice sanitises `release_tag` through `termsafe` like every other untrusted rendered string in this repo — the tag is read out of an HTTP redirect and is the one value in that line with no shape check. (4) `binary_sha256` is recorded and re-checked by nothing; the comment now says so plainly instead of implying an automatic check, because the fast path's whole contract is one file test and adding a SHA-256 recomputation to every session start would trade the AC away for a check nothing yet consumes. Two known asymmetries re-confirmed as disclosed rather than accidental: only the two binary-backed `SessionStart` entries carry the inline existence wrapper — `UserPromptSubmit`, `PreCompact` and `SessionEnd` still invoke the binary bare, which is itd-105's "every hook reports, for now" testing posture and not an oversight; and the unsupported-platform notice's move to exit 2 means it now repeats EVERY session for as long as the platform is unsupported, which is the accepted cost of that same posture (the one-notice-per-session aspiration is iss-168's). Not fixed, noted: the orphan sweep could still delete a live racer's temp directory in the already-accepted >10-minute stale-lock race, and the transcript scanner's `local_username` hard-fails on this script are the same pre-existing false-positive class already firing on `main` for `README.md`.
- 2026-08-01 — itd-105 / spc-21, FOURTH round: two fresh independent reviews of the seam-removal fix pass above (one security, one correctness) both returned MERGE, each naming a short list of cheap items rather than a blocker — three rounds of BLOCK was not treated as licence to manufacture a fourth by inflating what these were. All were landed on this branch directly rather than deferred, since none needed a fresh review cycle to verify. (1) The structural regression test guarding the seam removal, `TestBootstrapFetchOriginsAreConstants`, was a DENYLIST — it scanned for `${`, `$ABCD`, `$HTTP`, `$CURL` — so a bypass renamed to any other variable (`$GH_MIRROR` was the reviewer's example) would satisfy it while still redirecting a fetch; reproduced by planting exactly that shape against the committed script and watching the old test pass regardless. Replaced with an ALLOWLIST: every `$NAME`/`${NAME}` reference in the script, after subtracting names the script assigns to itself, must be `CLAUDE_PLUGIN_ROOT` and nothing else — verified this catches the planted `$GH_MIRROR` bypass and passes on the real script. (2) `safe()`'s `tr -d` range stripped ESC and other C0 controls but not CR (`\015`) or DEL (`\177`); a bare CR overprints from column zero, the exact attack class `internal/termsafe`'s own doc comment names, and DEL is the same family. Range widened to `\000-\010\013-\037\177` (keeps only tab and newline, as before); the existing control-character test now plants CR and DEL alongside ESC in the same specimen and watches both survive the pre-fix range before the widened one strips them. (3) `mv -f` onto an existing DIRECTORY at `$binary` moves the downloaded file INTO the directory rather than replacing it — reproduced: the script reported success, exited 2, wrote a truthful-looking `.binary-meta`, and the binary landed at `$binary/abcd-<os>-<arch>` while the hooks stayed broken forever, since the fast path's `[ -f ]` check never becomes true. A guard now refuses before any download when `$binary` exists and is not a regular file; the existing directory-fast-path test was tightened to assert the actual refusal (it had previously accepted a substring of the SUCCESS message as if it proved a refusal happened, which is why the bug had a green test). (4) `plugin_root_basename`'s control-character stripping and 120-byte cap (added in the prior round) had no detector: deleting the sanitisation left the full suite green, and a directory literally named `aa\nrelease_sha=<40 hex>` forged a second `release_sha` line that `skew.go`'s last-wins map parsing would have preferred over the genuine one. Two new cases prove the fix is load-bearing: an embedded-newline specimen asserting `.binary-meta` holds exactly its six declared lines and the genuine value survives, and an oversized-basename specimen asserting the recorded value is exactly 120 bytes. Noted, not landed: the structural allowlist test is still a snapshot of the *shape* the script must hold, not a guarantee against every conceivable future evasion of that shape — it is the cheapest check that catches the two concrete bypasses two independent reviewers actually constructed, not a formal proof.
- 2026-08-01 — itd-105 / spc-21, MERGE-GATE fix (concurrent session A): a fresh merge-gate security reviewer found a real gap in that same allowlist test, not in the shipped script. `TestBootstrapFetchOriginsAreConstants` subtracted any name the script "assigns to itself" before checking it against the allowlist, so a SELF-referential seam (`mirror="${mirror:-}"`, then `[ -n "$mirror" ] && repo_url="$mirror"`) read as script-local and the test passed green while a curl shim recorded a fetch to an attacker-named host. Reproduced by planting exactly that shape against the committed script before fixing. The allowlist now taints any name whose own assignment line references itself, on every line where it is assigned, so neither the self-reference nor any later use of the name reads as safe — verified this catches the planted bypass and still passes on the real script. One more merge-gate MINOR landed in the same pass: `mkdir -p "$tmp"` (the temp-download directory) succeeds on a directory that reappears, symlinked, in the window since the preceding `rm -rf`; plain `mkdir` (no `-p`) turns that same-name race into a refusal instead of a curl write through a planted symlink. Both merge-gate reviews (fresh, independent, full-PR-diff) had already returned MERGE before these were found — landed anyway per the merge gate's "fix everything raised" step rather than waiting for a fifth review round, since both are narrow, each has a watched-fail-then-pass test, and neither changes the shipped script's externally observable behaviour except the second (a same-name TOCTOU race becoming a loud refusal, strictly safer).
- 2026-08-01 — itd-105 / spc-21, FIFTH round (concurrent session B, landed alongside the entry above): a fresh security review returned BLOCK on a bypass it reproduced, and a fresh correctness review returned MERGE with minor findings; both are closed here on the same branch. (1) BLOCKER — none of `hooks/bootstrap.sh`'s four `curl` calls passed `-q`, so curl loaded its own configuration surface: `$CURL_HOME/.curlrc` (curl >= 7.73, falling back to `$HOME/.curlrc`), where a single `connect-to` or `resolve` line re-points the connection while the URL on the command line still literally reads `https://github.com/…`. The transport pin `--proto '=https' --proto-redir '=https'` therefore still HELD and proved nothing, and because the binary and the `checksums.txt` that verifies it travel the same route, the SHA-256 check was vacuous — the poisoned config supplied both. Reproduced end to end before the fix: a `.curlrc` carrying `connect-to` plus its own `cacert` diverted all five requests to a second local server and installed the forged binary at the guard path, where it would run unattended as the Bash shell guard on every tool call. Fixed by passing `-q` as the FIRST argument of every call (curl honours it in no other position) and by removing, before any fetch, the other names curl reads without being told to: `HTTPS_PROXY https_proxy HTTP_PROXY http_proxy ALL_PROXY all_proxy CURL_HOME` and `CURL_CA_BUNDLE SSL_CERT_FILE SSL_CERT_DIR`. ACCEPTED COST, deliberately: a machine that can only reach the network through a proxy no longer bootstraps automatically — `refuse()` already names the manual-install and build-from-source ways out, and a proxy plus an attacker-chosen CA is precisely the pairing that makes the redirection succeed on TLS, so the seam is not worth keeping for the convenience. (2) MAJOR — the suite could not have caught (1): it only exercised the removed `ABCD_BOOTSTRAP_*` script variables, never curl's own config surface. `TestBootstrapIgnoresCurlsOwnConfigSurface` now runs the fixture-pointed copy with a poisoned `.curlrc` under both `CURL_HOME` and `HOME`, and asserts the published bytes are installed and the forger is contacted zero times; it was watched failing against the pre-fix script in exactly the shape described above. `TestBootstrapFetchOriginsAreConstants` gained the structural half — every `curl -` line must carry `curl -q ` as well as the literal transport pin, and both scrub lines must be present exactly once — verified by deleting a `-q` and shortening the scrub list and watching it fail. The one test-side concession is that `bootstrapFixtureScript` neutralises the CA-scrub line in its throwaway COPY, because `CURL_CA_BUNDLE` is the only way a real curl trusts the self-signed local fixture; that widens what curl will ACCEPT and can never name a different server, the `CURL_HOME`/proxy scrub stays intact in the copy (which is what makes the poisoned-config case meaningful), and the shipped bytes are held by the structural test. (3) MINOR — the `.binary-meta` install had no regular-file guard, so a directory at that path swallowed the record as `.binary-meta/binary-meta` (the same `mv -f` hazard `$binary` already refuses) while the success notice claimed provenance had been recorded. The install itself is genuine there, so this REPORTS rather than refuses: the notice now says the path exists and is not a regular file, and `TestBootstrapReportsAnObstructedBinaryMeta` proves the record is never moved into the obstruction. NOT FIXED, deliberately: the stale-lock double-break race (already disclosed and accepted above), the "120 bytes" comment wording and the `checksums.txt` text-vs-binary-mode parsing convention (cosmetic, not worth the churn at a merge gate), and the hook-ordering assumption (already disclosed, no code change).
- 2026-08-01 — itd-105 / spc-21, coordination note: two independently-running instances of the v0.5.0 loop each reviewed PR #184 at its merge gate concurrently, found different real findings (a self-referential allowlist bypass; a curl config-surface bypass), and pushed both fix commits to `v050/itd-105` minutes apart — rebased together here rather than either overwriting the other. Same collision class the state issue's round-9 comment already disclosed for `v050/iss-159`; recorded again per that precedent, no new action taken beyond the rebase.
- 2026-08-01 — itd-105 / spc-21, MERGE-GATE fix (sixth round, test-hardening only): a fresh merge-gate security reviewer reproduced a TWO-HOP bypass of the same fetch-origin allowlist test, and this entry also corrects the claim two entries above. That entry said the allowlist "taints any name whose own assignment line references itself", which is exactly what it did — and exactly why it closed only the ONE-hop case, not "the self-referential bypass class" the fix commit's message implied. Two names that reference each OTHER (`repo_url="${GH_MIRROR:-$gh_default}"` then `GH_MIRROR="$repo_url"`) leave neither line self-referential, so the per-line check passed the planted script green while `GH_MIRROR=https://attacker.invalid/x` genuinely redirected the fetch — the reviewer verified that against the real script logic with a curl-argv-recording shim. `TestBootstrapFetchOriginsAreConstants`'s allowlist is now computed by `bootstrapSafeNames`, which propagates taint to a FIXPOINT over the assignment graph instead of scanning each line once: a name is script-local only if every assignment of it draws on constants, on `CLAUDE_PLUGIN_ROOT`, or on names already safe AT THAT POINT in the file, and the pass repeats until no name is newly tainted. What that closes is chains of self- and mutual reference of ANY length — one hop, the reviewer's two, and the three-hop laundering variant are all detector cases now, alongside the reproduction spliced into the committed script's own bytes. What it does NOT claim is completeness: it is a fixpoint over `$NAME`/`${NAME}` references in `name=` assignments, not a shell parser, so a value laundered through a construct that scan does not model (an `eval`, an eval-time expansion, a name assembled at runtime) is outside what it proves. The reference scan deliberately stays order-INSENSITIVE on the usage side, because a shell function body runs long after the line defining it and "used before assigned" is not a defect there. Second finding in the same pass: the curl scan filtered lines on the substring `curl -`, so any invocation whose first argument is not a flag (`curl "$url" -o /dev/null`) was skipped ENTIRELY — it could be missing `-q` or the transport pin and never be looked at, while the call count stayed consistent with itself because the scan was its own counter. curl is matched as a command WORD now (line head, after a shell operator, command substitution, path prefix, or keyword), with mentions — `command -v curl`, the refusal message's prose, `.curlrc` — held out as non-invocations by their own detector cases. Detector-first on both, each pre-fix behaviour restored in place and watched to fail: the two-hop, three-hop and planted-script cases reported nothing under the old taint rule, and five curl shapes (flags after the URL, no flags at all, an absolute path, after `if`, downstream of a pipe) were invisible to the old substring filter. `hooks/bootstrap.sh` is UNCHANGED — the reviewer's finding is that the regression guard was weaker than its own comment claimed, not that the shipped bytes have a live seam. Left alone deliberately, out of scope for this pass and unfixed: `SSLKEYLOGFILE`/`QLOGDIR` are not scrubbed, the lock and temp `mkdir`s take the ambient umask, and every tool the script runs is PATH-resolved (an actor who can write the PATH can replace `curl` outright, which no shell script closes).
- 2026-08-01 — iss-30 (v0.5.0 item C7), FINAL increment, test-only: the memory-ingest boundary's last four acceptance instances were absences, not defects, and are now drained behind detectors in `internal/core/memory/ingest_boundary_test.go` and `internal/core/memory/yaml_boundary_test.go`. The no-new-dep STOP the entry flagged for PDF extraction never fired, and the reason is the design rather than luck: `PDFExtractor` is a plain `func([]byte) (string, error)` the caller supplies, so a fake extractor covers the nil / error / empty-text / success matrix on the fetched path and both local sniffs (`.pdf` extension, `%PDF-` magic) without a parser ever entering `go.mod` — the nil case (no parser installed) is itself one of the behaviours under test. Likewise `Fetcher` carries the whole URL-ingest success path with no network: the fetcher is asked for the REQUESTED url, the FINAL url after redirects is what reaches the citation and the registry origin, the response headers reach licence detection (an HTTP `License:` header has no local-path equivalent), and `--keep-original` takes its extension from the final url's PATH, so a query string or fragment cannot leak into it. Two judgement calls worth recording. (1) On the parser's block-scalar limit: only the literal `|` opens a block — `>`, `|-` and `|+` are read as ordinary scalars — and rather than treat that as a defect to fix, the increment pins the property that makes it safe, which is that an unsupported indicator fails LOUDLY (the indented lines beneath it are refused as top-level content) rather than silently swallowing or dropping them. Widening the parser was rejected as scope the memory store has no shape asking for; the dumper never emits a block scalar at all. (2) No CHANGELOG entry: nothing user-facing changed, and the repo's own definition of done ties an entry to a user-facing change — the precedent is the behaviour-preserving `internal/urlguard` extraction, which carried none either. Every case was mutation-checked (sixteen targeted reversions of shipped behaviour, each turning at least one new case red), and no production behaviour was found to diverge from its own documentation, which is why this increment adds no fix.
- 2026-08-01 — iss-30 (v0.5.0 item C7), CORRECTION to the entry immediately above, and two production fixes: that entry closed the item as "test-only" and stated that "no production behaviour was found to diverge from its own documentation, which is why this increment adds no fix". That claim was false and is retracted here. Two independent fresh pre-PR reviews (correctness and security) each reproduced a real defect in `internal/core/memory/yaml.go`, both now fixed detector-first on the same branch. (1) `collectBlockScalar` set the block's indent from the first non-blank line WITHOUT requiring it to exceed the indent of the `key: |` line that opened the block. Since the function is only ever reached for a top-level key (`parseYAMLLines` refuses any other indent), a `|` with no indented body took the next COLUMN-0 line as content and went on consuming every line after it: the remaining keys — `domain`, `slug`, `source` — were absorbed into the value and no error was raised. An unindented first line now ends an EMPTY block and is left unconsumed for the top-level parse. (2) `dumpString` triggered its quoting on `\n` and `\t` but not `\r`, so a bare carriage return was emitted RAW into the YAML region, while `parseFrontmatter` normalises `\r` to `\n` before splitting lines — the writer and the reader disagreed about what a line is. A value carrying `\r---` therefore re-read as an early frontmatter terminator and every key below it fell into the page body; a distiller-supplied `recall` string was enough to make `Ingest` report success while writing a page that reads back with an EMPTY source block and no source hashes, which is exactly the state the reconcile/repair path treats as an orphan. `\r` now joins the quote trigger; `doubleQuote` already escaped it correctly, so only the trigger was missing. The lesson worth keeping is about detector siting rather than either bug: the earlier increment's `\r` cases DID pass, because they round-tripped through `parseScalar`/`parseFlowMap` directly and never crossed the `dumpFrontmatter` → `joinFileFrontmatter` → `parseFrontmatter` boundary where the invariant is actually stated (yaml.go:11-16) and where the defect lived. A round-trip test that does not use the real entry points tests the parser's internals, not the invariant; the new cases go through the file boundary and, for the CR case, through `Ingest` itself. Both detectors were watched fail before the fix and pass after. Consequently the increment is no longer test-only and DOES carry a CHANGELOG entry under Fixed — the second judgement call in the entry above ("no CHANGELOG entry: nothing user-facing changed") lapses with the claim it rested on.
- 2026-08-02 — iss-34 (v0.5.0 item C8): the five untested refusal guards now carry detectors, and the audit that opened the round is worth keeping because the ledger's "zero coverage" claim was true for only three of the five. ML001's single-source branch was ALREADY covered by `TestLintRejectsMalformed/external_missing_licence` — forbidden page, blocker code, exit contract — so nothing was added there; padding a satisfied guard with a second test buys no detection and costs a reader's trust in the corpus. The symlink-dereference guard was covered on its REJECT half only (escape, denied target), which is the interesting general point: a guard whose job is to refuse SOME inputs and safely dereference the rest is only half-pinned by refusal tests, because dropping the dereference silently ships nothing rather than shipping something forbidden — a regression that reads as "stricter" and never trips a deny assertion. Both halves are now pinned, and the cycle guard turns out not to mislabel a rejection when removed but to hang the resolve outright. The three genuinely uncovered guards — the scripts runtime closure, the MQ001 quotation budget, and the `ask --file-back` write path — each gained the full shape the convention asks for: forbidden input presented literally, the specific reason code or error type asserted (not merely "an error"), and absence of side effects proved, which for file-back means a byte-level snapshot of the whole store taken before and after each refusal rather than a spot check on one filename. Two design choices are reusable: the scripts test drives the shipped `ClosureFn` seam and adds a no-closure polarity control, so the exclusions are demonstrably the gate's doing rather than an unrelated default-deny; and each case was verified by disabling its guard and watching the test fail, which for already-shipped guards is the only available form of "watched fail before". No guard was found broken, so the increment is test-only, `go.mod` is untouched, and there is no CHANGELOG entry — the repo's definition of done ties an entry to a user-facing change, and coverage alone is not one. The entry's promotion path — a pairing lint between declared invariants and named tests — is deliberately NOT built here; it is a convention increment, not part of the coverage corpus this item scoped.
- 2026-08-02 — iss-170: `resolvePluginRoot` now canonicalises the executable path with `resolvePath` (EvalSymlinks, falling back to `filepath.Abs`, then the original path, on error) before the ancestor walk, because `os.Executable` reports the path abcd was INVOKED through and `ahoy install` itself pins a PATH symlink at `/usr/local/bin/abcd` -> `<plugin-root>/abcd` — so the walk climbed the symlink's ancestors (`/usr/local/bin`, `/usr/local`, `/usr` — the loop's own guard excludes `/`, so it was never actually a candidate) and never saw `hooks/`, a permanent "plugin root not resolvable" gap on exactly the layout the installer writes, in a normal shell where neither `ABCD_PLUGIN_ROOT` nor `CLAUDE_PLUGIN_ROOT` is set (`CLAUDE_PLUGIN_ROOT` is set by the harness inside hook invocations, which is why every env-driven test — each of which sets it directly via `t.Setenv` — passed over the defect; this codebase has no independent evidence for whether any other context also sets it). The gap is platform-specific: `os.Executable` resolves through `/proc/self/exe` on Linux, already symlink-free, so the defect and this fix both bite only where the OS returns the invoked (unresolved) path — macOS's `execve`-argument implementation, per Go's `os/executable_darwin.go`. Reusing the existing `resolvePath` rather than calling `filepath.EvalSymlinks` inline keeps the fallback chain shared with the symlink-classification path; the `Abs` middle step is CWD-dependent in principle, but `os.Executable` returns an absolute path whenever it reports no error on both shipped platforms, so that step never fires here and no case that worked before can regress. Testing it required a seam: `os.Executable` is not overridable, so a package-level `var osExecutable = os.Executable` was added and the detector substitutes a fake returning the symlink path, with both env candidates blanked via `t.Setenv` so ONLY the fallback runs — watched fail (no plugin root) with the seam in place but the walk unresolved, and pass after.
- 2026-08-02 — iss-38 (v0.5.0 item D): hand-maintained indexes are either DELETED or GATED, never corrected and left hand-kept, because a corrected list drifts again on the next commit and nobody notices the second time either (adr-5, derive don't store). The ledger named four stale indexes; three were real and one does not hold. (1) `intents/README.md` transcribed four lifecycle directories as ASCII trees and had fallen sixteen drafts, one promotion (itd-73, listed as a draft while living in `planned/`), two disciplines and two superseded intents behind, with a `shipped/` section still calling the directory empty over thirteen shipped intents — deleted, because the enumeration carried nothing the directory does not carry better, and what a filesystem cannot state (the kind taxonomy, the lifecycle contract, the bundle history) stays. (2) `commands/README.md` (fourteen verbs named, eighteen files present) and (3) `internal/README.md` (core described as "identity/version and the read-only status snapshot" while its own later bullets described twenty packages, and `adapter/scanner` filed under planned seams although it ships with its own pattern set, redactor and tests) keep short, high-value enumerations — so those ship WITH the detector rather than as a promise to remember. The new `index_drift` rule holds a marked region (`<!-- index: <id> -->`) to the directory it enumerates in one of two modes: `exact`, where listing and directory must agree both ways, and `absent`, where every listed path must still be missing — the planned-seams shape, whose drift is a seam that shipped. Scoping is an explicit marker plus a configured entry pattern rather than a markdown-list parser: one generic rule, no bespoke parser per README, and unrelated prose beside a list cannot fire it. It fails closed three ways (region gone while the config entry remains; region parsing to zero entries; malformed spec), so half-deleting an index is a finding rather than a silent disarm. (4) The fourth instance — "the repo README Layout omits `skills/` from the plugin surface" — was investigated and does NOT hold: there is no `skills/` directory, abcd ships zero skills deliberately (the three workflows once shaped as skills became commands, closing iss-61; `brief/05-internals/08-skills.md` states it), so the Layout section is correct by omission and was left untouched. Inventing a `skills/` reference to satisfy a finding would have written a falsehood into the record to close a ticket — the same disposition iss-37 reached on its one non-holding instance. CHANGELOG: an entry, unlike iss-34's test-only pass, and impact `additive` rather than `internal` — the rule lives in the shipped lint engine that `abcd docs lint` runs, so any abcd-managed repo gains it by declaring its own pairs in `.abcd/docs-lint.json`.
- 2026-08-02 — iss-39: record-lint gains `record_schema`, the first rule that reasons ACROSS the record's stores (ADRs, intents, specs, the issue ledger) rather than inside one, and the `superseded/` exemption narrows for the intent tree: both rules that read it (`intent_lifecycle`, `intent_impact_valid`) now run there too, while the spec-store rules (`spec_lifecycle`, `spec_id_unique`) still honour it as before and the cross-store `record_schema` never consulted it at all. Four invariants, one scan: a cross-reference field names a record the corpus has; a supersession is declared from BOTH sides; a filename and the id inside it agree; and every lifecycle directory is enumerated, so an undeclared bucket is a finding rather than a state no rule reads. Scope is deliberate on two axes. (1) Cross-references are checked in the machine-readable frontmatter fields (`supersedes`, `related_adrs`, `related_intents`, `builds_on`, `blocked_by`), not in bare prose: a frontmatter handle is a claim that the record exists and is a live input, whereas prose legitimately narrates ids that do not resolve ("itd-38's id was released, not reserved"; a test fixture's `itd-9999`; adr-29's account of what ADR-6 settled) — arming prose would have cost ten false positives against one true one, the same false-positive budget that scopes GL002. (2) A handle resolves either to a file or to a `supersedes:` declaration elsewhere, because the ADR lifecycle PRUNES a superseded record and keeps the trace in its successor — that is what distinguishes adr-4/adr-8/adr-14..18 (pruned, accounted for) from adr-6 (cited, never accounted for). The exemption narrowing is the root cause of the corpus defect it exposes: `intent_lifecycle` is a schema rule, but the `superseded/` bucket exempted its own records from the one rule that checks a supersession is recorded, so itd-47 and itd-49 sat there for months with a prose blockquote and no `superseded_by` field. Being historical excuses a record from how it is WRITTEN, never from being well-formed. `superseded_by` also widens from `^itd-\d+` to `^(itd|adr)-\d+`: an ADR that redecides the question an intent rested on retires that intent as surely as a successor intent does, and refusing to spell that forced those two records to say nothing at all. Corpus repairs in the same change: adr-6 is dropped from the two `related_adrs` lists and the two brief pages that cited it re-point at adr-29 (which records the two-stage redaction model and states in its own body that adr-6's decision stands — adr-29 does not supersede it, so claiming it does would have written a falsehood to clear a finding); adr-12 and adr-32 gain the bidirectional pair the ledger-location redecision always implied, adr-12's status moving to `superseded` and the index row with it; itd-47/itd-49 gain `superseded_by: adr-22`/`adr-26` with `kind_at_supersession`, and those ADRs name them back; and the pre-existing one-way links itd-32→itd-31 and itd-31→itd-48 gain their reverse halves. CHANGELOG impact `additive`, matching iss-38's disposition on the same axis: the rule ships in the lint engine `abcd docs lint` runs, and any abcd-managed repo gains it by declaring its own `record_stores`.
- 2026-08-02 — iss-39 (review pass): four correctness repairs to `record_schema`, each found by mutating the rule rather than reading it. (1) The "no lifecycle state escapes" invariant was implemented for bucketed stores only — a FLAT store (the ADRs) declares no buckets, so the walk read its files and never looked at its directories, and a whole `adrs/archive/` of records with mismatched ids and phantom successors was invisible to every check in the rule. A flat store declares zero buckets, so any subdirectory of it is undeclared by definition, and now says so. (2) The retirement escape hatch was self-attesting: `retired` was built from every record's `supersedes` before the resolution loop ran, so one record naming `adr-9999` there made that id resolve in every other record's `related_adrs` — reopening the exact phantom-reference class the rule exists to close. It is now bounded by the store's allocation high-water mark (an id above it was never issued, so nothing can have pruned it) and the declaration itself is reported; `supersedes` also leaves `recordRefFields`, where it could never fail its own check and read as coverage it did not provide. (3) The shared frontmatter scanner is a same-line scanner, so a block sequence (`supersedes:` then indented `- adr-12`) read as empty — and an empty read here did not merely under-report, it made the bidirectional check assert that ANOTHER file omitted a link that file plainly carried. Other rules degrade to "field missing on this file"; this one degraded to a confident false claim about a different file, which is worse, so the block spelling is folded in at the one place the rule parses handles. Local to `record_schema` rather than in the shared scanner: the canonical scanner has many consumers whose keys are scalars, and widening it for one rule's list fields would change what every other rule reads. (4) A dot-directory (`.obsidian`, `.vscode`) tripped a blocker as an undeclared bucket; tooling state is not a lifecycle the record authored. Also disclosed rather than fixed: widening `superseded_by` to `^(itd|adr)-` makes `intent_lifecycle` looser ALONE — it reads only the intent tree and cannot resolve an `adr-` target — so the CHANGELOG and the pattern's own comment now say the two rules are armed together and `record_schema` is what resolves a cross-store target. The surviving adr-6 prose citations are captured as iss-179 rather than papered over: adr-29 states in its own body that adr-6's decision stands and that adr-29 does not supersede it, so the supersession vocabulary cannot record it.
- 2026-08-02 — iss-39 (merge-gate pass): four more `record_schema` corrections, all of the same family as the previous round — a check that was right for one shape and absent for its sibling. (1) The undeclared-directory finding was implemented for store ROOTS only; a directory inside a declared bucket (`intents/shipped/archive/`) was skipped with no finding at all, so the previous round's flat-store fix closed the escape for one store of four. Both cases are now reported from the one place that reads a directory of records, because a flat store and a declared bucket hold records the same way and a directory in either is a lifecycle nobody declared. (2) `spc` was missing from the handle pattern although the spec store is indexed and armed: a bidirectional spec supersession tripped a false blocker whose message named the very prefixes it omitted, while the reverse direction parsed to zero handles and was never checked. The alternation now covers every store the rule indexes, and the message is composed from one constant so it cannot drift from the pattern again. (3) `superseded_by: []` was a false blocker: the empty flow sequence is this record's house spelling for an empty list (`related_rfcs: []` across six ADRs), so it means absence. Fixed with a local `isAbsentValue` rather than by widening the shared `isNull`, which also judges scalar fields (kind, impact, slug) where a list literal is a wrong value rather than an unset one. (4) The filename/id check compared strings, so `id: adr-0012` in `0012-*.md` was reported as a mismatch — contradicting the rule's own stated design that the padded and bare spellings are one handle; it now parses both sides and compares numerically, as every other comparison in the rule already did. Record-side: iss-179 now cites the recoverable evidence for what adr-6 was (`.abcd/work/reviews/2026-07-06-plan-consistency/05-link-structure.md` lines 47-49 name the original file, `03-cross-corpus-consistency.md` line 42 records it as voided by DECISIONS.md lines 8-9 together with adr-8 and adr-17 — the only other two ADRs that decision voids, both since pruned by a successor that names them (adr-25, adr-22), which is what leaves adr-6 alone — and observes that the unused `deprecated` status is the state it fits) so the disposition is a decision with evidence rather than an open question; the two brief re-pointings drop a reflexive attribution that had adr-29 reusing a model from itself; and adr-22, adr-26, and adr-32 gain one body clause each naming what their newly-declared `supersedes` handles retire, so the frontmatter and the prose say the same thing.
- 2026-08-02 — iss-39 (merge-gate round 3): the exemption-scope enumeration is made exhaustive, and the block-sequence parse stops truncating. `scanIntentTree` lost its `contentExempt` filter, and that one scan feeds TWO rules — `intent_lifecycle` and `intent_impact_valid` — but the CHANGELOG, the `ExemptPaths` doc, the per-file walk comment, and `contentExempt`'s own doc each named only the first while reading as an exhaustive partition. A reader arming `intent_impact_valid` with `exempt_paths` would therefore conclude it was unchanged and be surprised on upgrade by a new blocker over an exempt record's illegal `impact:` value, so the changelog now states that consequence outright. `record_schema` is dropped from that partition and stated separately: it is cross-store and never consulted the exemption, so describing it as having "stopped" honouring one was wrong in a way that would mislead anyone trying to reason about which rules the exemption still governs. Separately, `blockSequenceAt` broke on a blank or comment line inside a sequence rather than skipping past it. YAML reads `- a`, blank, `# note`, `- b` as one two-item list; stopping at the interruption dropped the tail, and a dropped `supersedes` handle is not a quiet under-read — it made the bidirectional check assert that another file omits a link that file plainly carries, which is the same false-claim failure the block parse was added to prevent. The corpus has no interrupted sequence today (every `.md` scanned), so this closed a latent hole rather than a live one. The closing `---` is neither blank nor a comment, so the scan still ends at the frontmatter boundary — covered by its own test, because a skip that ran past the delimiter would read a body list as the key's value.
- 2026-08-02 — iss-40: one canonical glossary, chosen as a ROUTING rule rather than a migration. `.abcd/development/brief/glossary/` keeps the name "glossary" and 04-naming.md stops claiming it; what stays behind in 04-naming.md is its ~60-row Reserved-vocabulary table, unchanged, because those rows are closed enums and spec-pinned reserved names (`dormant`, `task_classes` members, `index.json` field names) and the glossary schema's `aliases`/`forbidden_synonyms` fields mean nothing for an enum value. Folding them into term files would have invented scope and misused the schema; the defect was the false self-identification, and only that is retired. `VR001` keeps its reserved status and its two registration targets — the word "glossary" is removed from its descriptions across the brief and itd-37 so the term resolves to one place, but nothing about what the reserved lint would check is changed, since no Go code implements it. The index is made derived on the brief↔lifeboat-mapping pattern (`internal/core/glossary` renders, a test in the same package gates the committed README against it) rather than by a new schema-validation engine, which the README had been promising through a JSON-schema file and a `abcd lint terminology` verb that do not exist. Three limits are recorded rather than papered over: the drift gate checks the index against the term files, NOT each term file against the field tables, so the frontmatter shape stays a review responsibility; the banlist entry `names/record/glossary-self-identification` reaches only the docs-lint roots (`docs/`, root `README.md`), so it guards the published surface, not the brief where the retired phrase lived — adding `.abcd/development/brief` as a lint root surfaces 75 pre-existing blockers and is its own piece of work; and `05-internals/06-lint.md`'s `TM003`–`TM011`/H1 rows still claim `terminology.schema.json`-backed schema validation "Delivered" that no Go code implements — the same phantom-enforcement-claim class this change retires for the glossary README's validation section, but for TM003+ it sits outside iss-40's self-identification scope and inside iss-37 (phantom-enforcement-claims), stopped at round 13 pending maintainer disposition; left untouched here rather than pre-empting that disposition.
- 2026-08-02 — iss-41 (v0.5.0 item D): the interim delivery-state rule is stated as the one the lifecycle already implies — **delivered capability is represented by the intent leaving `drafts/`, and nothing else says it** — because the alternative on offer (a new tier, or a per-intent "partly shipped" field) invents lifecycle vocabulary to describe a citation problem. The ledger's premise that `shipped/` is empty has aged out: thirteen intents live there now, so the mechanism exists and what was missing was the gate holding the changelog to it. `delivery_state` reads each version entry's delivery sections (`Added`, `Changed`; a repo's configured list is unioned with those, never substituted for them) and blocks an `itd-N` citation whose intent is still in `drafts/`. Scoping by section rather than by whole entry is what keeps it precise: the ids under `Fixed` in this corpus are provenance for a defect — which draft two branches minted at once — and blocking them would force either a falsehood or a deletion of real information. It fails closed three ways: no changelog, no intents store, and a store holding no lifecycle bucket — the last being a root pointed one level too deep, which resolves every citation to nothing and reads exactly like a clean corpus. The corpus had TWO instances, not the one filed: itd-60, whose deterministic layer shipped as `abcd docs lint` while its semantic layer is still five open questions, and itd-85, which shipped WHOLE as `abcd audit` and was simply never promoted. They get different treatments because they are different facts. itd-60 stays in `drafts/` and loses the citation — an intent is delivered whole or not at all (split-the-intent doctrine), so citing a partial delivery reads as a promotion that never happened, and splitting itd-60 to make the citation true is editorial work this issue does not carry. itd-85 loses the citation too, but under protest: the honest fix is promotion, and promotion is blocked on the `shipped/` schema wanting a non-null `spec_id` the audit verb has never had. Filed as iss-180 rather than papered over or forced — writing a retroactive spec to satisfy a lint is the failure mode this repo keeps naming. The out-of-scope half is a REUSE, not a new rule: `index_drift` gained one field, `dir_entry`, the mirror of `entry` on the directory side, which reduces a filename stem to the id a document actually enumerates. Without it a hand-written listing can only agree with a directory by transcribing whole slugs, which is a listing nobody writes, and the alternative was a second rule shaped exactly like the first. The list itself had drifted nineteen entries while claiming in its own prose to be derived and "not hand-counted": itd-47, itd-73 and itd-74 had left `drafts/` for three different buckets, and sixteen captures were missing. Regenerated, wrapped in the marked region, and the sentence promising hand-kept lockstep is replaced by the gate that enforces it. The four later-phase items that never had an intent id move OUT of the derived list — a derivation cannot produce them, so their presence was what made the claim false in the first place.
- 2026-08-03 — iss-42 (v0.5.0 item D): CONTEXT.md's sharp-edges list gains a detector rather than only a correction, because the correction alone would have been the third one. The stale trust caveat (still calling iss-35's brief-vs-surface reconciliation "the open cross-check" months after `surface_coverage` shipped and iss-122 pinned the Direction-A crosscheck) is an instance of a class: the orientation doc grounds a live claim in a record, the record reaches a terminal state, and nothing in the workflow asks anyone to go back. `context_citation_currency` is that missing requirement made mechanical — it resolves every `iss-N`/`itd-N`/`spc-N`/`adr-N` handle in the "Live constraints / sharp edges" section against its store and blocks a citation to a terminal bucket (or, for the flat ADR store, a declared supersession or retired status). Deliberately narrow on two axes: ONE section of ONE document, because a terminal-state citation is legitimate in an evidence trail, a supersession chain, and a dated plan — history is supposed to name closed records; and terminal-state only, because a dangling handle is `record_schema`'s question and a second weaker implementation of an answered check is worse than none. Rejected: a git-diff-shaped handoff rule ("a change to a cited record must touch CONTEXT.md in the same commit"), which the ledger entry's wording suggests but which gates the wrong thing — it fires on commits that need no orientation change and stays silent on the case that actually bit us, a record moved in one branch and the caveat left standing in another. The state-based check has no such blind spot: whenever the citation is stale, the gate is red, regardless of which commit made it so. Alongside: `research/` stops contradicting its own README (five dated notes move into `notes/`, where ~25 peers already sit, with every citation followed — the itd-88 fidelity evidence trail, two DECISIONS.md evidence paths, `.abcd/rules.json`, and a plan link), the development record map's routing row names `research/`'s real children (`notes/` + `prompting/`, not the phantom `spikes/`), and two banned_tokens entries make both stalenesses non-silent on reintroduction. The dated plan and the ratified ADR that also mention `spikes/` are left untouched: they are chronological snapshots and a ratified decision, not living indexes, and rewriting them to match today would falsify the record — `development/README.md` alone is corrected because it alone claims to be current.
- 2026-08-03 — iss-42, CORRECTION to the entry immediately above and three fixes from a fresh adversarial correctness review. (1) That entry claimed two banned_tokens entries made both stalenesses "non-silent on reintroduction". The `phantom-research-spikes` half did not: its pattern was the literal string `research/spikes`, which never occurs in the prose it was meant to catch — the record map spells the row as two separate backtick spans (`notes/` ... `spikes/`), so restoring the exact original stale line left record-lint at zero blockers. The claim was written from reading the pattern instead of reproducing the defect, which is the failure the detector-first rule exists to prevent, and the lesson is that a ban's acceptance test is the original line, not the pattern's plausibility. The fix is not a wider regex: ADR-30 and the dated go-rebuild plan carry the same prose shape legitimately, and no pattern separates the living index from the two historical records without an escape they must not be edited to carry. So the row is GATED instead, per iss-38's "delete or gate, never correct and leave hand-kept" — an `index_drift` region over the row holds it to `research/`'s actual subdirectories, which catches every spelling of a phantom child AND a real child the row omits, and fails closed if the region is deleted. (2) `abcd ideate record` wrote its dated verdict record to `research/` root, making the shipped binary the standing producer of the very convention violation this item cleared out of that directory by hand; `ResearchRelDir` now points at `research/notes/`. spc-18's rationale ("killed ideas are research outcomes, and the research directory is where a future session looks") argues for the research area, not its root, and a verdict record is by spc-18's own words a dated research note — so `notes/` is where the record's convention already put it. The CLI surface test asserted the path through the constant, so it would have followed the constant anywhere; it now asserts the literal directory. (3) The rewritten trust caveat over-claimed `surface_coverage`: the rule reads `commands_dir` and `skills_dir` only, while the crosscheck manifest's Direction B enumerates five surface kinds, so the agent, hook, and CLI-verb surfaces are periodic-pass-only. A caveat whose entire job is saying how much is machine-enforced must not overstate it — the same phantom-enforcement class iss-40 flagged — and the wording now names the two surfaces the structural check actually covers.
- 2026-08-05 — iss-37 (v0.5.0 item D): re-scoped in place per the 2026-08-03 maintainer disposition, from five instances to the three doc claims that describe the gate suite wrongly, and the fix direction is doc-side throughout because the issue's class is claims-must-match-reality and the disposition frames all three as doc claims — the Makefile and every gate's behaviour are untouched. `AGENTS.md` names the three lint gates `make preflight` runs (`lint-reviews`, `record-lint`, `docs-lint`) ahead of build/vet/test/race, and its definition of done keeps `gofmt -l .` as a requirement while attributing it to CI's own format step rather than to preflight, which does not run gofmt. The `ci.yml` record-lint comment claimed the step was "visible but non-blocking for now"; the step has no `continue-on-error` and `record-lint` exits non-zero on a blocker, so the comment described an intention the workflow had already outgrown — it now states the gate blocks and that warns surface in the log only (comment-only; no step structure touched). `README.md` and `CONTRIBUTING.md` describe the real suite, and CONTRIBUTING stops claiming preflight and CI run "the same checks" where they do not. Instance 1 of the original five (the `docs/reference/cli/README.md` freshness claim) was verified accurate in round 13 and carries no work. The brief's 06-lint.md section-1 catalogue — a predecessor project's Python-era `IL001`–`RC007` families and class names — plus the gate cross-check detector are re-filed as iss-181 rather than reworked here, because the disposition on them is substantive (remove the catalogue, never archive it; do NOT mint a Go-equivalent numbered scheme, which is taxonomy design and would ship already-stale against this cycle's own rules; rework section 1 generically over the live armed rules and `internal/core/lint`, with any literal enumeration generated and gated) and the detector's scoping — scan root, and what "a live definition" means for a prose-named gate — is unsettled and recorded as two open questions in that issue's body. iss-37 is resolved against the re-scoped body.
- 2026-08-05 — iss-44 + iss-160 + iss-161 + iss-162 (v0.5.0 item D, one work item): the plugin surface is reconciled with every document describing it, and the reconciliation is a MOVE rather than a rename, because the harness's namespace rule leaves no other answer. Verb files under `commands/abcd/` registered as `/abcd:abcd:<verb>` — a `commands/` subdirectory IS a namespace segment, and this one was named after the plugin, so the prefix appeared twice and every documented `/abcd:<verb>` was an unknown command (iss-161). Flattening into `commands/` is what makes the printed guidance runnable, which is why iss-162 needed no rewrite at all: all eighteen verb files already spelled `/abcd:<verb>` in their self- and cross-references — verified by grep across every one rather than assumed — so the guidance was correct prose about a name nothing registered. On iss-160 the fix direction was settled by evidence, not preference: the loader registers every markdown file under `commands/`, requiring no frontmatter and exempting no name (this repo's own iss-110 records `agents/README.md` registering as an agent; the published plugin reference describes commands purely as flat `.md` files with no ignore list), so NO in-place form keeps a readme unregistered and the only reliable home is outside the auto-discovery root. It goes to the brief's surface registry, which already enumerates the same surface and is already machine-checked against it — folding the doc into the gate that guards it rather than leaving a second enumeration to drift. The cost is one repointed link in a 2026-07-12 plan, paid because `links_resolve` is a blocker and a dangling pointer is not history. iss-44's three instances close with them: `ahoy` gains a section per sub-verb (with `identity-check` carrying an explicit CLI-only note, because its exit code is the point of wiring it into a hook — an absence recorded as a decision is not a gap), every `make build` remedy is replaced by the `go run ./cmd/abcd …` fallback the same paragraphs already carried, and `memory`'s renders stop naming a front door from `internal/core`, which is transport-agnostic by boundary rule: the heading a plain-CLI reader sees must not claim they invoked a plugin command they may not have installed. Detectors first, both watched failing: a surface-parity test over the real command tree and the real directory (flat-and-commands-only, every sub-verb reachable or scoped, no unrunnable remedy), and a string-literal check over `internal/core/memory` — literals, not comments, because prose explaining a surface may name it. `surface_coverage` gains a `bare_command` field so the flat directory's top-level board file is excluded the way README already is, rather than demanding a registry row that cannot exist; hardcoding the name would have been wrong in a lint engine that ships to repos whose plugin is not called abcd. Left alone deliberately: the historical citations of `commands/abcd/` across decisions, shipped intents, closed specs, research and resolved issues — they describe the tree as it was, and rewriting them would trade a true record for a tidy one. Not fixed here, and reported instead: the release-gate manifest's `promptHash` pins prompt text this change edits, by an algorithm nothing in the tree computes or verifies, so it is left untouched rather than guessed at.
- 2026-08-05 — iss-80 (v0.5.0): resolved as already-fixed, with the residual verification gap closed rather than the defect re-fixed. The branch-local id-collision class this item tracked — `itd-N`/`spc-N`/`iss-N` allocators scanning only the working tree, so two agents on branches cut from one base each mint the same id and collide at merge — was already closed by iss-115 and iss-120, which introduced `recordid.MaxAcrossRefs` (scan every git ref for the family's highest committed id) and folded it into all three minting paths: `internal/core/spec/store.go`, `internal/core/intent/create.go`, and `internal/core/capture/workflow.go`. What survived was uneven evidence, not an uneven fix: only the `spc-N` family had an end-to-end regression test standing up a real two-branch history (`internal/core/spec/refunion_test.go`), while none of capture's or intent's existing tests stood up a real two-branch history through `Capture`/`CreateFromText` and so never drove the ref scan through their public entry points at all — `MaxAcrossRefs` was exercised only as a primitive in its own package test, never as wiring. That is the shape in which a fix silently un-wires: a refactor that dropped the `MaxAcrossRefs` call from `Capture` or `CreateFromText` would have left every existing test in both packages green. So this round adds `internal/core/capture/refunion_test.go` (`TestCaptureMintsPastACommittedBranch`) and `internal/core/intent/refunion_test.go` (`TestCreateFromTextMintsPastACommittedBranch`), each mirroring the spec test's exact fixture via `internal/gittest.NewRepo` — branch A mints and commits record 1, branch B is cut from before that commit so its working tree carries no record, and B must mint record 2. Both were mutation-checked before being trusted: with `MaxAcrossRefs` stubbed to return an empty `RefScan`, each fails with precisely the collision it names (`iss-1`/`itd-1` re-minted on branch B), and all three families fail together — so the tests are pinned to the ref-union behaviour rather than passing incidentally. No production code was touched, which is the point: the minting paths were verified correct by reading before the coverage was written, and the change is test-and-record only. Impact recorded as `internal` and no CHANGELOG entry, matching the iss-34 precedent for a test-only closure with no user-facing behaviour change.
- 2026-08-05 — CORRECTION to the entry immediately above, from a merge-gate record-accuracy review on PR #195. That entry, and iss-80's own `resolution:` field, said the branch-local id-collision class "was closed" by iss-115/iss-120. It was not: `internal/core/recordid/recordid.go`'s own package doc documents an ACCEPTED, undocumented-in-that-entry residual window — two branches that BOTH mint before either commits still collide — left to the already-armed record-lint uniqueness detectors (`issue_id_unique`, `intent_lifecycle`, `spec_id_unique`) as backstop, the same trade-off iss-120's own resolution names verbatim ("the armed record-lint detectors as the residual-window backstop"). iss-80's body asked for a fully collision-free minting scheme (forge-minted / random-suffix / timestamp / reserve-registry); that was never built, that trade-off was made when iss-115/iss-120 landed rather than here, and no open item currently tracks the residual window separately. Both iss-80's `resolution:` field and this log now say "already addressed, not closed to zero" and name the trade-off explicitly, rather than implying the class was eliminated. The test-coverage work in the entry above (the two new refunion tests) is unaffected and stands as described.
- 2026-08-05 — iss-43 (v0.5.0 item D): the three-claim corpus is closed as OVERTAKEN rather than fixed — the Status section and the surface list were removed by `73428b6` (2026-07-17, "manual: Revise README with new badges and project details"), a manual revision predating the v0.5.0 plan, so the Phase 0 claim and the native review oracle / spec-task engine / autonomous run claims no longer exist to correct, and re-deriving a corpus against a rewritten document would be inventing findings to justify a ticket. That attribution is itself a correction worth recording: the round first credited the iss-143 tagline commit `48a3524`, which touches ONE README line (the strapline), because it read a SHALLOW clone whose truncation window opened well after 2026-07-17 and therefore could not see the real commit — a truncated history answers a `git log` question confidently and wrongly, and the cheap guard is `git log -S` over the file for the exact removed string, unshallowed first. Two corollaries earned the same way: the corpus's third claim was never `73428b6`'s at all (`git log -S "never shipped"` puts that line in the scaffold commit and in this branch alone), so "all three overtaken" was a second wrong attribution riding on the first; and a whole-phrase probe can miss a claim that was line-wrapped in the original, which is why `"spec/task"` reproduces where `"spec/task engine"` returns nothing. One instance survives, and it is half true rather than false, which is why it needed splitting instead of deleting: the Layout line called `.abcd/` "never shipped", which holds for the released binaries alone (release.yml uploads the four of them and `checksums.txt`, nothing else) and fails everywhere else — `.claude-plugin/marketplace.json` declares `source: "./"`, so a marketplace install takes the whole repository, and GitHub attaches auto-generated source archives to each release, which carry the directory too because `.gitattributes` declares no `export-ignore`. The line now divides on the boundary that is real: every repository checkout against the released binaries, rather than a per-channel split that would have been wrong about the release page. Doc-side deliberately: the marketplace manifest schema has no per-path exclusion, so making "never shipped" true everywhere means building a curated publish path — feature work with its own design questions, not doc repair, and writing the doc to describe a packaging mechanism that does not exist is the phantom-claim failure this item belongs to. The proposed detector (README capability and status claims resolve to wired verbs and to the roadmap) is DROPPED here and re-pointed at iss-181 as a candidate extension of the gate cross-check detector's scope, recorded as a candidate rather than a commitment because it shares that detector's two unsettled scoping questions and a second half-specified scanner would answer neither. The same claim survives across the record — CONTEXT.md's sharp-edges list, AGENTS.md, both `.abcd/` READMEs, `02-constraints/01-platform.md`, `05-internals/03-configuration.md` and `phase-1-ahoy.md`, with adr-0028, the dated plans and the planned/shipped intent bodies exempt as decision and historical records — and it is recorded as iss-183 rather than fixed here, since the disposition scoped this item to one README line. That issue also corrects a claim this round made twice: the exclusion is NOT unimplemented. `internal/core/launch/bundle.go` denies the `.abcd` namespace structurally, with tests, reachable from the wired `abcd launch ship`; it is unwired, because `Ship` stops at `WouldPublish` with no network call and `release.yml` uploads the binaries without ever invoking the verb. "No mechanism exists" and "the mechanism never runs" ask for different fixes, and only the second is true.
- 2026-08-05 — bug-hunt loop round 1: iss-184 (critical, guard tokenizer heredoc misparse) fixed at root cause, in two passes after a pre-PR adversarial security review caught the first pass narrowing the hole rather than closing it. `internal/core/guard/tokenize.go`'s `<<` handling already special-cased the literal-digit arithmetic-shift form (`$((1<<20))` is not a heredoc), but `isDelimStart` accepts any identifier-shaped word, so `$((1<<shift))` — an identifier operand — still read as a heredoc delimiter, and `skipHeredocBodies` then silently consumed every remaining line of the command looking for a "shift" terminator line that never comes, dropping later commands (a force-push, an `rm -rf`, anything) from ever reaching command position — a silent allow, not the loud fail-open the hook's own doc comment promises for anything the tokenizer cannot answer confidently. The first pass made `skipHeredocBodies` report whether it actually found each pending heredoc's terminator, turning a miss into `ErrUnparsableCommand`. The security review then constructed the exploit this missed: an attacker supplying a later line that happens to equal the misread "delimiter" (e.g. appending a bare `shift`) still finds a match, so the swallow still succeeds silently — the fix only closed the *unterminated* half of the class, not the *coincidentally-terminated* half, which is the one an attacker actually controls. The real fix is classification, not detection-of-failure: a heredoc delimiter word immediately followed by a bare `(` or `)` with no separator is never a real heredoc — its body and terminator line have to come first, so nothing legitimate places a paren directly against the delimiter word, and `$((expr<<ident))` produces exactly that shape. `<<` in that position is now read as the arithmetic operator at tokenize time, same as the literal-digit case, so the guarded line is reached and matched normally regardless of what any later line contains. The unterminated-body-to-error fix from the first pass is kept as defense in depth for genuinely malformed heredocs (`cat <<EOF` with no closing `EOF` line), a distinct, narrower gap the same swallow also covered — an existing test that pinned that silent-swallow as intended behaviour is updated to assert the corrected `ErrUnparsableCommand` contract. Detectors: `TestArithmeticShiftByIdentifierIsNotAHeredoc`, `TestArithmeticShiftCoincidentalDelimiterStillBlocks` (the adversarial payload the security review demonstrated), and `TestTokenizeRejectsUnterminatedHeredoc` — each watched failing on pre-fix code for the claimed reason and passing after. Three more bugs surfaced by the same round's multi-angle sweep and independently verified with failing tests are captured but not fixed here — iss-185 (critical, scanner leaves a second back-to-back secret token undetected and unredacted, defeating history's fail-closed residual re-scan), iss-186 (minor, a failed source-unlink in a capture ledger transition can permanently strand an issue id across two status directories), iss-187 (minor, `rules.Merge` panics on a nil-Domains base — currently unreachable through any live caller, so a latent exported-API contract defect) — left open for a future round, highest severity first.
- 2026-08-05 — iss-43, CORRECTION to the entry immediately above, from a merge-gate record-accuracy review on PR #196. That entry stands unedited; four of its claims are corrected here rather than in place, because it is merged history. (1) It opens "the three-claim corpus is closed as OVERTAKEN rather than fixed", and the corpus does not close as a unit: claims 1 and 2 — the Phase 0 status claim and the native-capability surface list — close as overtaken via `73428b6`, while claim 3's surviving half, the Layout line, is FIXED by that same PR. The entry knows this later in its own text and in the issue's `resolution:` field, so the opening verdict was a second over-claim of the same shape as the attribution it goes on to retract: a single verdict asserted over a corpus whose members had different fates. (2) "a SHALLOW clone whose truncation window opened well after 2026-07-17" over-claims. The evidence bounds the boundary to the interval after 2026-07-17 and no later than the oldest visible substantive README rewrite; "well after" asserts a distance nothing measured. Read it as "after 2026-07-17". (3) "writing the doc to describe a packaging mechanism that does not exist" is refuted by the entry's own closing paragraph. The clause scopes to the all-channel curated publish path, which genuinely does not exist and which no release has ever taken; the namespace deny itself exists and is tested. (4) The call edge is wrong in the closing paragraph: "reachable from the wired `abcd launch ship`; it is unwired, because `Ship` stops at `WouldPublish`" reads as one chain, implying the verb runs through `launch.Ship`. It does not. `launch.Ship` has NO production caller — the only calls are three in `dryrun_test.go` — while the wired verb exercises the deny through `launch.PrecheckPayload` and `launch.RenderPayload`, which resolve the bundle at `internal/core/launch/render.go:192`, reached from `internal/surface/cli/ship.go`. `release.yml` invokes neither. Both facts stand; the edge joining them did not. The lesson generalises past this entry: a summary sentence written before the detail paragraphs are finished tends to keep the verdict the round started with, and a call chain assembled from two true sentences about neighbouring functions is not evidence that either calls the other — `grep` for the callers.
- 2026-08-05 — iss-118: DECISIONS.md and ACKNOWLEDGEMENTS.md gain `merge=union` in `.gitattributes`, the same remedy already in place for CHANGELOG.md. Both are the identical shape — append-only ledgers of anonymous, dated, order-independent entries that never need identity — so a concurrent-append conflict has nothing to actually disagree on; the union driver keeps both sides instead of stopping the merge. Prompted directly by the bug-hunt loop's `bugfix/iss-184-…` branch hitting exactly this conflict against `.abcd/work/DECISIONS.md` on PR #199, which iss-118 had already diagnosed (filed after an earlier merge hit the same conflict) but left unresolved pending this design call. Full atomicisation (per-decision records, a new id family, an armed uniqueness detector) was the other option iss-118 named; not pursued here as disproportionate to a minor, low-traffic hotspot. Impact recorded as `internal`, no CHANGELOG entry — matching the iss-80 precedent for a process/tooling change with no user-facing behaviour.
- 2026-08-05 — iss-185 (bug-hunt loop, round 3): `ScanText`'s leading-`\b`-anchored secret patterns could never detect a second fixed-length token immediately abutting a first with no separator — the byte before the second token is itself a word character (the first token's own last byte), a word/word transition where `\b` never holds, so `FindAllStringIndex` silently returns only the first match. `Redact` then left the second token's whole body raw, and `history`'s stage-two fail-closed residual re-scan reported the redacted text clean anyway, so a concatenated pair of secrets reached disk with one still live. Landed after three rounds of pre-PR/pre-merge adversarial review, each catching a real problem in the pass before it: (1) the first pass's own-pattern-only probe missed a mixed pair of different fixed-length patterns, and double-counted a `google_api_key` match whose dash-terminated 35th char already satisfied `\b` on its own; (2) the fix for that, an unanchored `FindStringIndex` probe, cost O(matches × patterns × remaining line length) — measured at 14–49s on large single-line input; (3) anchoring the probe with `\A` fixed the *restart-at-every-offset* cost but not the cost of one attempt against two patterns (`net_lan_hostname`, `net_device_hostname`) that carry their own unbounded internal quantifier, still quadratic on many back-to-back fixed-length tokens. The landed shape: `adjacencyProbe` compiles an `\A`-anchored, `\b`-stripped (past any inline flag group) variant of each pattern; `scanAllPatterns` probes every pattern's variant, within a small fixed window (`maxAdjacencyProbeWindow`, 512 bytes) so one attempt can never cost more than a constant regardless of what follows, at the byte offset immediately after each already-found match, deduplicated against matches already found. A pattern with an open-ended quantifier can still greedily consume into a following token before this recovery ever runs — a separate, broader gap, captured as iss-188 rather than folded in here. Repro: `internal/adapter/scanner/adjacency_test.go`, `TestConcatenatedSecretsBothDetected` (the original bug), `TestConcatenatedDifferentFixedLengthSecretsBothDetected` and `TestGoogleAPIKeyDashJunctionNotDoubleCounted` (review round 1), `TestAdjacencyProbeStaysLinearOnLongLines` and `TestAdjacencyProbeWindowIsBounded` (review rounds 2 and 3).
- 2026-08-05 — iss-188 (bug-hunt loop, follow-on to iss-185): a secret pattern whose quantifier is open-ended (`\bghp_[A-Za-z0-9]{36,}` and the seven other `{n,}` families) greedily consumes a following token's own leading bytes when those fall inside its trailing character class, so the true junction sits BEFORE the match's reported end and iss-185's forward adjacency probe — which only ever looks at that end — never runs where the second token actually starts. `"ghp_"+36×a+"ghp_"+36×b` reported one finding covering `ghp_aaa…aaaghp` and left `_bbb…bbb` raw through `Redact`, with `history`'s fail-closed residual re-scan reporting the output clean: the same live-secret-reaches-disk violation iss-185 closed for the fixed-length family. Fixed in `internal/adapter/scanner/scanner.go:465` (`stolenJunctions`): before a match's reported end is accepted as final, a bounded backward search walks candidate cuts and reports each one where the shortened prefix is STILL a whole match for its own pattern AND a token can begin, and `scanAllPatterns` probes those cuts exactly as it probes the forward end. Open-ended is detected from the compiled regexp — a match that cannot still match one byte shorter has a rigid length and is skipped after one test — not from hand-annotated metadata, so a pattern added later inherits the classification. Three things keep the cost per match rather than per match LENGTH, which the ledger entry named as the trap (an unbounded backward scan would trade the leak for a resource-exhaustion cliff on exactly the huge single-line input a scanner must handle): that one-test early-out, `maxAdjacencyBacktrack` (512, mirroring `maxAdjacencyProbeWindow`), and a single combined `junctionProbe` alternation over the whole pattern set that generates candidate cuts in one linear pass instead of one pass per pattern — measured 40s unbounded against 0.3s bounded on a 200KB line. The over-long match is deliberately left untrimmed: it only over-reports the first token's span, and `sealLine` already forces every overlap byte to `*`. Repro: `internal/adapter/scanner/adjacency_test.go`, `TestConcatenatedOpenEndedSecretsBothDetected` (the ledger entry's literal reproducer) and `TestOpenEndedSecretSwallowingDifferentFamilyBothDetected` (a whole `AKIA` key swallowed by an alnum class run), both watched failing on pre-fix code for the claimed reason; cost guard `TestJunctionBacktrackIsBounded`.
- 2026-08-05 — iss-188 follow-up (two independent adversarial reviews of the fix above, both landing on the same defect): the bounded backward search in `internal/adapter/scanner/scanner.go` (`stolenJunctions`) could step OVER the junction it was looking for. `junctionProbe` is unanchored, so one of its hits can SPAN a real junction — begin before it and end after it — without beginning at it; the loop nevertheless resumed at that hit's END (`off += loc[1]`) even when the hit's own offset had just been REJECTED by `wholeMatch`, skipping every byte in between, the true junction among them. Nothing revisits a skipped range, so the search returned fewer cuts than it should and iss-188's own failure mode reopened through a narrower trigger: `"ghp_"+32×a+"AIza"+4×b+"ghp_"+36×c` reported one finding and left the second token's tail raw, and `"ghp_"+10×a+"AIza"+22×z+"sk-proj-"+40×i` — where the skipped-over decoy is itself a syntactically valid secret — left 48 raw bytes of an OpenAI project key through `Redact` with the fail-closed residual re-scan reporting the output clean. Fixed by always resuming at `cut + 1`, dropping the jump-to-hit-end fast path entirely: a rejected candidate can hide a real junction one byte later, and an accepted one is not worth a special case. This stays bounded for the reason the whole feature exists: the loop is capped at `maxAdjacencyBacktrack` (512) iterations per match regardless of how long the match is, and each iteration's search is capped at `maxAdjacencyBacktrack + maxAdjacencyProbeWindow` bytes by RE2's linear guarantee — so the per-match cost is a constant factor independent of the REST of the line, and the whole scan stays linear in line length rather than quadratic. Measured: doubling and quadrupling an adversarial 247KB line doubled and quadrupled the time (2.3s / 4.6s / 9.2s), which is the property that matters. Repro: `internal/adapter/scanner/adjacency_test.go`, `TestStolenJunctionSearchDoesNotSkipPastRejectedCandidate` and `TestStolenJunctionSearchSkipsPastValidDecoySecret`, both watched failing on pre-fix code for the claimed reason; cost guard `TestJunctionBacktrackIsBounded`, extended with `dense_rejected_candidates_in_backtrack_window` — every match's window packed with candidates that all fail validation, the worst case for advancing one byte at a time. Two further gaps the same reviews found were CAPTURED, not folded in: iss-190 (a recovered match longer than `maxAdjacencyProbeWindow` is truncated, and the misaligned artificial end breaks the chain that would recover a THIRD abutting token — the pre-existing iss-185 window trade-off reached through a new path, whose real fix is an adaptive window that risks reintroducing the unbounded per-match cost) and iss-191 (`junctionProbe`'s `(?s).` compile fallback would validate once per byte instead of once per candidate, unreachable with the bundled pattern set). Two comment claims were also corrected as inaccurate: cost is bounded independently of the rest of the line but IS proportional to the match's own length, since `wholeMatch` re-runs the probe over the prefix; and a window-exceeding recovery can miss a token ENTIRELY, not merely truncate it, when the pattern's required structural markers (`jwt_shaped`'s two `.` separators) both fall outside the window.
- 2026-08-06 — iss-186 (bug-hunt loop, round 4): `commitTransition` (`internal/core/capture/workflow.go`) wrote the destination file before removing the source, so a non-ENOENT `os.Remove(src)` failure (EPERM/EROFS/EIO — e.g. an immutable source status dir, or a read-only remount) returned an error after `dst` had already landed, with no rollback. The issue id was then present in both status dirs at once, and `findIssue` rejects any id present in more than one file as `ErrDuplicateIssueID` — so the stranded copy could never again be resolved or wontfixed without a human manually deleting one file; `List`/`Status` also double-counted it, since `scanLedger` does not dedupe. Fixed by rolling the destination back (best-effort `os.Remove(dst)`) whenever the source remove fails for a reason other than "already gone", restoring the pre-call state so a retry is all that is needed once the underlying failure clears; a failed rollback itself is surfaced in the returned error rather than swallowed. A non-ENOENT remove failure has no portable, cross-platform way to trigger deterministically in a test (immutable attributes and read-only remounts are Linux/ext4- and permission-model-specific, and this repo's CI runs macOS and Linux), so a test-only seam (`removeSourceHook`, nil in production) was added, mirroring the existing `beforeOrphanRemoveHook` pattern in `alloc.go`. Repro: `internal/core/capture/workflow_test.go`, `TestTransitionRemoveFailureDoesNotStrandIssueInTwoDirs`, watched failing on pre-fix code for the claimed reason (destination not rolled back) and passing after.
- 2026-08-06 — v0.4.2 release cut: content commit `3377980` (changelog roll; eight
  [0.4.1]-duplicate Unreleased entries removed first), semantic gates both PROMOTE
  at full tier. Docs-currency: zero findings. Brief↔surface crosscheck: 26
  discrepancies, all design-record drift, dispositioned per the iss-152 precedent —
  captured as iss-192, release not blocked. Version derived v0.4.2 (additive,
  pre-1.0 → patch bump per plan §3/adr-37; the run plan's "v0.5.0" was a
  prediction the derivation corrects). Crosscheck ran via direct harness subagents
  after the workflow runner failed on a session permission-handler fault (22/22
  vacuous errors — discarded, never treated as a pass); pinned prompts unchanged.
- 2026-08-06 — iss-187 (bug-hunt loop, round 1): `rules.Merge`
  (`internal/core/rules/rules.go`) assigned override domain keys straight into
  `out.Domains` without ever allocating that map, because its `cloneRuleSet`
  helper deliberately preserves a nil `Domains` (it only allocates when the
  source map is non-nil). A base such as `RuleSet{SchemaVersion: 1}` — which
  `Validate` accepts — therefore panicked with "assignment to entry in nil map"
  the moment the overlay carried a domain, contradicting `Merge`'s own doc
  comment ("New domain keys are added"), which states no such precondition. Not
  reachable in production today: the sole call site, `RuleSet.Load`, always
  passes `Merge(Defaults(), over)` and `Defaults()` always has a populated map —
  so this was a latent defect in an exported API contract that would go live for
  the first caller merging onto a non-`Defaults()` base (a future multi-tier
  overlay starting from an empty set). Fixed at the `Merge` call site by
  allocating `out.Domains` when it is nil and the overlay has at least one key,
  matching the idiom the sibling loader `guard.Merge`
  (`internal/core/guard/config.go`) already uses for `out.Entries`; `cloneRuleSet`
  keeps its nil-preserving semantics, so no other caller's behaviour moves.
  Repro: `internal/core/rules/rules_test.go`,
  `TestMergeNilBaseDomainsAddsNewKeys`, watched failing on pre-fix code for the
  claimed reason (panic: assignment to entry in nil map) and passing after.
- 2026-08-06 — iss-183: of the two fix directions the issue offered, REWORD is
  taken and wiring is not. The descriptive instances now carry the
  channel-truthful phrasing iss-43 established in the README — `.abcd/**` is
  present in every repository checkout, marketplace installs and release source
  archives included, and never in the released binaries — with the launch
  bundler named as the implemented structural namespace deny that no cut release
  has yet run, rather than as an operating packaging filter. Wiring the launch
  publish path so the deny actually runs on a real release stays open feature
  work with its own design questions (the marketplace manifest schema has no
  per-path exclusion, and `release.yml` uploads the binaries without invoking the
  verb), and is deliberately not assumed by this wording. Two candidate sites are
  classified exempt on genre rather than reworded: `01-press-release.md`, which
  states the intended product in press-release voice throughout, and
  `04-surfaces/04-launch.md`, whose mention sits inside its explicit "full-cut
  design" framing and is already truthful. One instance the issue body's list
  missed surfaced only under a case-insensitive re-enumeration — AGENTS.md's
  working-tree-layout line, capitalised "Excluded from the release artifact" —
  which is why the closing sweep runs the three phrases with `-i`.
- 2026-08-06 — attribution epoch: the full git history (main, all release
  tags, and the open work branch) is rewritten to normalise commit
  attribution — AI-tool author/committer identities and two misconfigured
  machine-local identities all map to the maintainer's account, AI
  co-author trailers are removed, and every commit that had AI involvement
  carries an `Assisted-by` trailer (the disclosure convention; added where
  missing). Every rewritten commit's tree is byte-identical to its
  predecessor — content is untouched; only hashes and attribution change.
  Consequence for the record: any commit SHA cited in records, receipts,
  or issue threads dated before 2026-08-06 refers to the pre-rewrite
  history and no longer resolves from a current checkout; treat those
  citations as historical, keyed to the old epoch.
- 2026-08-07 — release retention: the newest-per-line prune (brief § 3, itd-70)
  is computed and previewable but has never run, so v0.4.0 and v0.4.1 survive
  against the policy. They stay: pruning them by hand would spend the only live
  fixture itd-70 has to prove itself against, and the first real prune should be
  the shipped mechanism's. The doc-currency half — prose that describes the
  policy as operating — is captured as iss-194.
- 2026-08-07 — iss-191 re-scoped by maintainer ruling: the round-3 merge-gate
  review's cost finding is a different mechanism from the compile-fallback
  iss-191 actually describes, so the two are split. iss-191 keeps its narrow,
  still-accurate scope (junctionProbe's `(?s).` fallback; verified not to
  trigger on the bundled set, whose combined alternation compiles cleanly).
  The reachable half is captured as iss-195: the rigid/open-ended heuristic
  classifies net_ipv4/net_ipv6 as open-ended, so every address match in
  ordinary content enters stolenJunctions' backward search — benchmarked at
  1.9x–27x versus the pre-iss-188 tree on identical reserved-documentation
  content (full table in iss-195's ledger entry; scratch benchmark artefacts
  are local-tier and ephemeral, so the entry carries the numbers). Capture is
  record-only; no behaviour changes.
- 2026-08-08 — iss-189/iss-190 shared root cause identified, fix designed but not
  implemented: bug-hunt loop rounds 6-8 (state issue #197) each attempted a local
  patch to scanAllPatterns/probeAt's 512-byte adjacency window
  (internal/adapter/scanner/scanner.go) and each was BLOCKed on review for the
  same underlying reason — a truncated, possibly-ambiguous window-edge view was
  driving a discard/skip decision that reached further than the ambiguity itself
  (round 6: discarding a window-edge match lost the forward-chaining offset;
  round 7: discarding lost a genuine shorter match, and its boundary classifier
  mis-fired on multi-alternation patterns; round 8: an uncertain match end wrongly
  suppressed stolenJunctions, a backward search that never depended on that offset
  being exact). Recommended structural fix: replace the fixed-window single-shot
  probe with a galloping/exponential-doubling probe (`trueMatchEnd`) that grows
  the window only while a match keeps running into its edge, and stops the moment
  the match ends short of the edge or the window reaches the real end of line —
  so `m.end` is always the true end, never an artifact of truncation. This removes
  iss-189's boundary-ambiguity question outright (no more guessing whether `\b`
  is real, so no classifier to get wrong) and iss-190's "clipped, so skip
  chaining" special case (nothing is ever clipped), without reintroducing round
  6's cost regression — it only grows for matches that are genuinely still
  growing, at the same amortized cost class the top-level unbounded match in the
  same function already pays. Not yet implemented or reviewed; captured here so
  the next attempt (an autonomous ROOT-CAUSE ESCALATION round, or a human) starts
  from this direction instead of re-deriving it from the same three round
  post-mortems.
- 2026-08-08 — bug-hunt loop round 9 (state issue #197): a multi-angle sweep (5
  blind hunt angles plus independent adversarial verification) surfaced a
  complete, silent bypass of every guard blocker via `env -S`/`--split-string`
  (GNU env re-splits and runs that value as the command; the guard's wrapper
  table treated it as the wrapper's own value and stepped past it). A fix
  attempt (branch `bugfix/env-split-string-guard-bypass`, closing the
  separate-token spelling `env -S git push ...`) was BLOCKed at pre-PR review by
  two independent reviewers (correctness and security), who each independently
  found the fix incomplete: the glued spellings `env -Sgit ...` and
  `env --split-string=git ...` execute identically to the closed form and
  remained a live bypass. Per protocol this is a first-attempt BLOCK, not a
  repeat of the unrelated iss-189/190 scanner root cause, so it stops here
  rather than escalating — nothing from that branch was pushed. The complete bug
  (all three spellings, plus the reviewers' converging suggested repair —
  handle env's `-S<value>`/`--split-string=<value>` forms alongside the
  separate-token form in `commandOf`/`skipWrapperArgs`, no nested re-tokenizing
  needed) is captured fresh as iss-200, since iss-196 (the narrower, incomplete
  framing) only ever existed on the abandoned, unpushed branch and was never
  merged; treat iss-196 as superseded by iss-200 the way iss-191 was re-scoped
  into iss-195.

  The same sweep also confirmed three further bugs, captured here record-only
  (no behaviour changes in this commit): iss-201 (`abcd guard hook`'s stdin read
  has no overflow check, so a payload padded past the 1 MiB cap silently
  truncates, fails JSON parsing, and fail-opens with a diagnostic that
  misleadingly blames the host instead of reporting the cap overflow — major,
  not critical, since the practical exploitation cost is a single tool_input
  exceeding roughly 1 MiB of model-generated content, and the fail-open is loud,
  never silent); iss-202 (`scanner.New` reads `.abcd/config/pii.json` with a
  bare, unguarded `os.ReadFile` — no symlink guard, no size cap — unlike every
  sibling trust-boundary config reader in this codebase; a FIFO hangs it forever
  and a git-committable symlink to a device file grows it toward OOM, both
  reachable automatically via the SessionEnd hook, and the hang also wedges
  `history.repoLock`'s unbounded flock, permanently disabling transcript
  capture for that repo — the same bug class already fixed once under iss-97,
  which this instance escaped); and iss-203 (`abcd audit`'s privacy rule guards
  on `scanner.New`'s error return expecting a fallback to the built-in pattern
  set, but `scanner.New` never returns a non-nil error, so the guard is always
  true, the documented fallback is dead code, and a broken `pii.json` override
  silently drops a repo's raised severities and downgrades the audit exit code
  from 2 to 1 with no diagnostic).

  Also reconfirmed this round (captured round 7 as iss-195, previously flagged
  as needing adversarial verification before fix-eligibility): the scanner's
  rigid/open-ended heuristic false-positive `hard_fail`-flags RFC 3849 reserved
  documentation addresses and silently corrupts them via `Redact`, on top of
  the previously-known cost regression (up to ~54x on colon-hex content, worse
  than the ledger's original 1.9x-27x table) — worse than its `minor` label
  suggests. Its own fix is out of scope for this round, but iss-195 is a
  distinct bug in the scanner's rigid/open-ended heuristic, not part of the
  shelved scanner adjacency-recovery mechanism (iss-189/190/191, shelved per
  the maintainer's 2026-08-07 ruling); iss-195 itself remains fix-eligible, not
  shelved. No ledger or code change made for iss-195 this round beyond this
  reconfirmation note.

- 2026-08-11 — Pull-request lifecycle automation for autonomous rounds settled
  five points, recorded against the existing records rather than a new intent
  (one-canonical-primitive: iss-172, iss-178 and itd-107 already hold this
  ground; `capture promote` would have minted a sixth). (1) AUTO-MERGE
  ELIGIBILITY splits on additive-versus-editing, not documentation-versus-code:
  a new record file plus an append to this log, insertions only, record gate
  green, single commit. The evidence is this repo's own correction history —
  the post-merge correction commits in the recent window each fix a FALSE CLAIM
  about the system ("describe the real fix", "state what is true", the
  packaging and exclusion claims), which no deterministic gate catches, and all
  of them sit in changes that REWRITE existing prose; the purely additive
  captures needed none. (2) The eligible set is an ALLOWLIST of inert paths,
  never a denylist of source extensions: the plugin command pages, the rules
  file, the lint configuration and the agent-instruction router are markdown or
  JSON that change behaviour, so a "no source touched" test fails open on all
  of them. (3) Intent DRAFTS count as additive — the lifecycle already gates at
  promotion (adoption is a maintainer decision; implementing an unready intent
  is a STOP), so a second gate at merge is redundant. (4) The strict
  status-check policy is NOT relaxed to smooth the stalled-behind-base
  condition: it is the only thing that gates a concurrently-minted duplicate
  record id, because the mint lock is a filesystem lock that does not span
  checkouts and each branch passes in isolation; the stall is cleared by
  updating the branch instead. (5) Delivery is rung 1 first — protocol in the
  rendered routine, host credentials — per script-first-mvp, with the binary
  acquiring no platform CLI dependency. Deferred to the itd-107 grill: the
  rung-2 credential question, a declared merge style, peer-session concurrency
  policy, and attribution-footer defence depth.
- 2026-07-29 — server-side merge backstop for the autonomous loops: "main protection" rulesets in the predecessor CLI's established shape (PR required with 0 approvals, required status checks, strict up-to-date, force-push/deletion blocked, no bypass actors) added to abcd-cli, Bauhaus, Testimony; itemdeck.app got the PR/force-push/deletion rules only because it has no CI yet (adding CI, then the checks rule, is the follow-up). The three private repos cannot carry rulesets on the current GitHub plan, so their loops rely on the prompt-side merge gate alone. Consequence accepted: direct commits to main are now rejected on the gated repos — trivial changes ride a docs:/chore: PR with auto-merge armed. (Recorded 2026-08-15; the line was parked uncommitted across the attribution epoch.)
- 2026-07-30 — a cross-agent transcript-capture source is the current default SOTA upgrade for the history corpus, triggered when compatibility beyond Claude Code becomes a concern; adopted import-only: its output enters through the pre-declared `specstory-import` seam over the native store, so the fail-closed redaction applies to imported material, and the native floor stays the default with capture degrading to it when the tool is absent. The candidate tool, target harness, and provider-coverage caveats live in research/notes/2026-07-30-session-recording-sota.md (§6 addendum) — commitment records name no external tool before adoption is decided. The SOTA verdict is re-verified at the adopting intent; a move from optional to required tool is a sota-per-intent path-1 hard stop (new-dependency approval). Rejected: adopting the tool's in-repo history directory as a store (adr-29 already rejected external stores; transcripts stay out of the repo) and bespoke per-host parsers (a permanent maintenance burden the external tool already carries). See iss-217. (Recorded 2026-08-15; decided 2026-07-30, parked across the attribution epoch.)
- 2026-08-15 — three forward plans adopted at a maintainer grill, absorbing the existing queue rather than adding to it: `2026-08-15-plugin-user-safety.md` (execution priority 1; owns install-experience Cut B pick-up and the v0.5.0 security remainder; fix queue opens iss-200 then iss-195), `2026-08-15-predictable-development.md` (priority 2; itd-109 promotion is the headline; absorbs the v0.5.0 consistency remainder), `2026-08-15-facilitator-experience.md` (continuous, rides along; supersedes the explainability plan's pick-up role). The plan 1/3 boundary is touch-vs-machinery: what a facilitator directly invokes or reads versus automation that guarantees behaviour without human action. Each plan admits items by a stated admission test with the ledger remaining the backlog of record. Same grill: iss-189/190 retired by SUPERSESSION, not wontfix — the defects are real, the local-patch avenue is exhausted (three BLOCKed rounds), so both resolve into the new iss-229 (the recorded 2026-08-08 galloping-probe/trueMatchEnd structural design) with their repros as its acceptance corpus; and itd-111's one-tap micro-prompt open question graduated to its own capture iss-230, homed in the facilitator plan. Rejected: wontfixing 189/190 (asserts the defects don't matter), a full 104-issue triage into the plans (stale mirror of the ledger), and executing the plans in the listed 1-2-3 order (the safety floor and the compounding accelerators come first).
- 2026-08-15 — itd-84 (intent decomposition) ADOPTED at the MVP rung, per its own staged promotion ladder: the `decompose-before-filing` principle is filed, the hand-run four-piece protocol lives in the /abcd:intent surface page (capture and the planning interview both run it, loud-staged as not-yet-automated), a DECOMPOSITION rules domain injects the rule at recall time, and every hand-run is graded into `.abcd/development/research/notes/2026-08-15-decomposition-calibration.md` (the 2026-07-13 auto-merge SPLIT case seeds the corpus retrospectively). The deterministic Go pre-pass is admitted to the predictable-development plan's queue (front-door design open: `decompose` subverb vs a check in `intent ready`; collision with iss-210 noted). The capture-time agent rung stays DEFERRED until ~50 graded captures exist (itd-81 calibration); building or running it earlier is a recorded STOP. Rejected: overriding the bundled INTENTS rules domain to carry the new rule (a per-field override replaces the whole array and would withhold future bundled upgrades — iss-174's exact failure mode), hence the additive custom domain instead.
- 2026-08-15 — itd-111 planned at the live interview (spc-22 minted, `intent ready` exit 0). The itd-84 decomposition's first live hand-run rendered SPLIT, confirmed: the network trichotomy extracted to adr-38 ("implicit checks are disk-only — the network answers only an explicit ask") + brief invariant 7, which the intent now cites rather than declares; the SessionStart staleness notice vs the iss-206 skew-notice retirement was ruled a SCOPED REPLACEMENT (`refines`, not `reverses`) — steady-state skew machinery stays retired, itd-111 covers dev checkouts and failed provisioning. Interview rulings: explicit check at `abcd version --check` (ahoy stays disk-only); refusal-on-stale deliberately narrow (ahoy install only); unknown vintage (unstamped/dirty builds) is a first-class comparator outcome that fails closed at the refusal gate; harness portability of the session-start channel explicitly deferred to the itd-22 lineage. Independent SOTA fit-challenge UPHELD path 2 with three recorded caveats (seam = version-source provider interface, not a fixed-arity function; git's behind-upstream notice joins the anchors; BuildInfo stamping holes motivate the unknown outcome). AC 6 narrowed to the transition report (the fetch is itd-105/108's); one AC added (unknown vintage). Hand-run graded into the decomposition-calibration note (entry 2).
- 2026-08-15 — iss-200 (env -S guard bypass) fix attempt 2 BLOCKED by two independent reviewers (correctness + security), round stopped per the plugin-user-safety plan's first-attempt-BLOCK STOP. The splice-and-restart approach (`commandOf` re-tokenizes the -S value and re-walks) is REJECTED, not to be re-patched: it introduced a quadratic cost regression (O(n^2), ~5 min at the 1 MiB stdin cap — a DoS on the very hook meant to gate the command) AND still let all three in-scope spellings through (env quote-removal, the `\_` separator, and a leading env option in the value all desync a strings.Fields split and a command-position restart). Branch bugfix/iss-200-env-split-string-guard-bypass-v2 left unpushed. iss-200 enriched with the complete env -S spelling set the next attempt must cover and the design question it now turns on: faithfully modelling env's parser (bounded/quote-aware/option-aware) is more than the "no nested re-tokenizing" the remediation assumed, while the tokenize.go "sh -c unparseable => documented non-match" precedent is a FAIL-OPEN for a denylist guard, so it does not transfer. This is the iss-189/190 pattern — repeated local-patch BLOCKs on one root cause mean the next move is an agreed design, not another patch. Rejected here: hot-patching the reviewers' suggested fixes and re-reviewing in-session (pushing through the STOP on a security-critical guard).
- 2026-08-15 — Multi-harness support is IN SCOPE, reversing (`reverses`) the "obsolete under no-hard-deps" annotation the 06-delivery out-of-scope index carried on itd-22; the intent stays in drafts/ as the lineage carrier pending rework. The commitment: every supported harness drives the same abcd CLI functionality — behaviour lives once in the transport-agnostic core (adr-23) and is never double-written for one host. Per-harness delivery follows an adaptor ladder, most-native first: (1) the host's own plugin/packaging format, (2) any other native seam (lifecycle/hook wiring, config), (3) an MCP server as the universal fallback floor every MCP-capable host gets when rungs 1–2 are unavailable. Where a host exposes a native work-plane (goal/work-item tracking, a canonical wiki or knowledge base, trajectory judging), abcd's concepts (intents, brief, reviews, capture, lifeboat) map onto those native concepts rather than running as a parallel artefact set; hosts without such concepts get the adaptor rungs without the mapping. Candidate harnesses exist but commitment records name no external tool before per-host adoption is decided (2026-07-30 convention); each host adoption files as its own intent per the itd-22 lineage's own rule. Rejected: the reversed annotation's claim that a second harness is "just another host over the same core" needing no record — the core stays single, but hook/payload portability, packaging, and concept mapping are real per-host work the record must carry. The itd-84 decomposition hand-run routing this into intent/principle/brief rows precedes any filing.
- 2026-08-16 — the multi-harness routing ADOPTED at maintainer confirmation (itd-84 hand-run graded into the decomposition-calibration note, entry 3): adr-39 filed — the host-tier policy (MCP front door as the universal floor; the current plugin host's integration as the reference implementation, the assumed SOTA surface for the time being; the shipping default ultimately an open-source harness, flipping only on the parity suite plus a deliberate maintainer act; adaptor ladder most-native-first; per-host adoption one intent each at an explicit decision, nameless until adopted). itd-22 renamed `harness-portability` (id unchanged) and reworked to carry the shared machinery only — host profile seam, ladder semantics, parity conformance suite — with a retire-the-name ban (`retired-itd-22-slug`) on the old slug. itd-113 minted: the MCP front door intent (the floor is the first build). The never-double-write stance routed into adr-39 rather than a new principle — it restates one-canonical-primitive at the host boundary.
- 2026-08-16 — CI inert-path skip gate (iss-234): the inert allowlist is `docs/**`, `.abcd/development/**`, `.abcd/work/**`, plus exactly README.md, CHANGELOG.md, CONTRIBUTING.md, SECURITY.md, ACKNOWLEDGEMENTS.md, LICENSE, LICENSE.md; everything else — notably the harness routers (AGENTS.md/CLAUDE.md/GEMINI.md), commands/, agents/, hooks/, .abcd/*.json, .github/**, evals/, scripts/, Makefile, Go source — is non-inert, and every unknown or degraded outcome (non-PR event, empty diff, unreadable base, aborted or failed classifier) runs the full matrix, fail closed and loudly. Deviation from the capture's skippable list, maintainer-confirmed 2026-08-16: the ubuntu unit lane ALWAYS runs — lifeboat brief-coverage, glossary-index and README-grounding tests assert against the live tree inside the allowlist (reproduced: a prose-only brief edit turns TestEveryBriefSectionHasARow red while record-lint and docs-lint stay green), so skipping `go test ./...` on an inert change is fail-open; what stands down is the macOS leg's heavy steps, the race lane, smoke, and zizmor. The skipped-conclusion-satisfies-ruleset behaviour is unverified until the rehearsal PR passes; nothing arms auto-merge on it before that. Rejected: a fromJSON dynamic matrix (a never-reporting `check (macos-latest)` wedges every PR) and a marketplace paths-filter action (plain git suffices, no new pinned dependency).
- 2026-08-16 — the assessment vocabulary settled at a maintainer grill and filed as adr-40 (proposed): **four buckets — lint / review / audit / gate — separated by what each compares**, not by determinism, not by trigger. The rule was already half-recorded (05-internals/01-agents.md rules the two verdict families disjoint: "reviews assess a change, criterion verdicts assess a promise vs reality") and three shipped surfaces contradict it — `intent review` emits family 2 so it is an audit, `abcd audit` is deterministic rule-checking so it is a lint (and it consumed the `/abcd:audit` name 02-constraints/04-naming.md reserves for itd-16), and `disembark oracle` emits family 1 while the brief calls its output an audit. Grill rulings: **one surface performs exactly one act** (multi-act surfaces split at design time — cheap, because every multi-act surface abcd has is unbuilt, so they are specified rather than refactored); the buckets are a **closed list extended by PR, not a criterion** (an autonomous run given a criterion interprets it; a table lookup matches or fails); `intent ready` survives as the fourth bucket **gate** because it names a decision rather than a comparison, preserving itd-94's exit-code contract; determinism is orthogonal (itd-16's hash-chain check is a deterministic *audit*; itd-85's conformance check is a *lint* however built); automatic-vs-manual never enters a verb name (the trigger is the facilitator's, per itd-97); the oracle is the model-access seam beneath review and audit, never a bucket. **Clean break, no aliases** — pre-1.0.0, `--impact breaking` drives version derivation (iss-171 precedent), users re-download; an alias would additionally collide with the documentation discipline's ban on change-narration, leaving an undocumented silent forward. The role mapping is the operational reason the split earns its cost: product thinker reads `audit` and essentially nothing else, which makes itd-99's "never hand a product thinker a technical decision" mechanically checkable. Same grill corrected a wrong finding: iss-246 was filed claiming abcd's record documents a surface far larger than the binary with no gate catching it; **false** — the `Status` column in 04-surfaces/README.md is machine-checked by `surface_coverage` (iss-35), the brief carries ~100 inline `design target` markers, and adr-5 makes the brief aspirational by design. The real defect is narrower: `surface_coverage` is blind *inside* a row, so six of twenty rows read `shipped` then qualify themselves in prose, and documents outside the registry cite those sub-verbs as live (all three citing **predecessor-store** spec ids spc-29/spc-30, whose prose survived the migration to the native store). Fix per adr-40 §6: sub-verb tables carrying two facts per verb — bucket and existence — with `surface_coverage` extended to registered cobra sub-commands, one table serving both disciplines. Also established: the twelve-step record walk completes at ten steps; step 2 (`capture promote`) has no verb and step 12 (`capture resolve`) cannot write `resolved_by`, so **walkability is sequenced before coherence** — renaming ~90 files across a process nobody can walk is polish before function. Plan at plans/2026-08-16-process-coherence-and-walkability.md; six intents to plan, none plannable unattended. Rejected: one polymorphic `review <id>` verb (merges the disjoint families, erases the role signal), classifying acts rather than surfaces (makes the bucket a statement of intent rather than a guarantee, and hands product thinkers lint findings), three buckets without `gate` (forces `intent ready` to become `intent lint`), and adding `verify`/`validate` (itd-109's verification suite is a packaging of lints and audits, not a fifth kind).
- 2026-08-16 — the v0.5.0 release gate ruled PROMOTE ×2 (the first cut the armed
  public-repo receipt gate will verify). The passes executed against content
  tree d1afd2c; the merge-currency ruleset forced a re-cut, so the receipts
  are keyed to its cherry-pick 2235cda, which differs only by five
  interleaved record-only files outside both detectors’ pinned scope.
  docs-currency: one finding — the terminology.md release-artefact
  parenthetical that escaped the iss-183 sweep — fixed in the receipts commit,
  the disposition the receipt records. Crosscheck (full tier per the pinned
  manifest, 22 checkers, 28 unique discrepancies): all design-record drift
  (the vintage/staleness surfaces, the citation-baseline gate, the banlist
  renders and the agents/ registry row landed ahead of their brief rows),
  captured whole as iss-250 for the next cycle — the iss-192 treatment at
  v0.4.2, now precedent. Rejected: holding the cut on brief drift (the brief
  is aspirational by design, adr-5, and no user-facing doc is affected), and
  fixing the 28 findings inside the release branch (the two-commit shape is
  the gate's own contract; drift repair is next-cycle work with its own
  detector already armed).
- 2026-08-16 — adr-40 §5 REVERSED at the process-planning interview (amendment recorded in-place in adr-40): `disembark oracle` renames to `disembark review` (itd-125, breaking) instead of keeping the verb and fixing prose. The investigation finding that turned it: the binary verb never invokes the oracle seam — it is a compute-or-ingest verdict endpoint (deterministic manifest+coverage mapping, or validate-and-ingest of a host-produced verdict), so the seam name claims transport the verb does not touch and collides with the `/abcd:oracle ask` design target, the seam's real surface. The sweep carries the artefact (`audit/oracle-<manifest12>.*` → `review/review-<manifest12>.*`, clean-replacement across the rename so one manifest never holds two verdicts) and the agent (`lifeboat-oracle` → `lifeboat-reviewer`). Naming convention ruled in the same session: assessment agents are target-grain + role (`intent-auditor`, `lifeboat-reviewer`). adr-25 and the seam name `oracle` stand unchanged. Rejected: prose-only fix (leaves the artefact contradicting the vocabulary), renaming the seam itself (adr-25 is correct), a superseding mini-ADR (same-day clause amendment suffices).
- 2026-08-16 — academic references baseline adopted: the References & sources section of ACKNOWLEDGEMENTS.md carries a curated set of primary sources (end-user programming, AI-assisted development in practice, security of AI-generated code, human judgement and method, records and specification) as one ACM-style numbered list ordered alphabetically by first author, with canonical metadata in `.abcd/development/research/references.csl.json`. CSL-JSON extends the 2026-07-08 bibliography ruling to the repo level — one canonical format, `.bib` derived on demand (pandoc one-liner in `_references.md`) and gitignored. Admission criteria (primary sources only, verified citations only, curated not exhaustive) and the per-group rationale live in research/notes/2026-08-16-academic-references-baseline.md; store↔markdown sync is the documented protocol in `_references.md` per script-first-mvp, with a `references_sync` lint captured as iss-248, the possible next rung. A root CITATION.cff (CFF 1.2.0) answers the reverse direction — citing abcd itself. Rejected: `.bib` as the source of truth (a second canonical bibliography format beside the recorded CSL-JSON ruling), and generating the markdown section from the store (inverts markdown-as-source-of-truth). Consequence of the numbered list, from the independent adversarial review of the sources plan: ACKNOWLEDGEMENTS.md leaves the iss-118 merge=union set — an alphabetical insertion renumbers subsequent entries, so a union merge of two concurrent additions would silently interleave both renumberings; a loud conflict is correct there.
- 2026-08-16 — itd-76 planned at the live interview (spc-31 minted, `intent ready` exit 0). The itd-84 hand-run rendered SPLIT, confirmed: itd-76 narrows to the personal core (source add / ledger / sync-banlist / cite-check + guards); team share/ingest files as itd-126 and paper reconstruction as itd-127 (both `refines` itd-76, seeded criteria marked unconfirmed); the trust rule extracts to adr-41 (proposed) + brief invariant 9 — documents and ledgers never leave the user tier, a public citation requires both gates — which the intents cite rather than declare; the stance files as the consult-freely-cite-deliberately principle. Interview rulings: the share surface is the existing `.abcd/development/research/references.csl.json` research store, not a second `.abcd/work/references.json` exchange file (one committed bibliography — resolves the store clash the sources-plan adversarial review flagged); the draft's "Dogfood (already running)" claim was stale and is rewritten as target (no corpus on the development machine); multi-machine ledger ownership explicitly deferred until a second machine exists; persona Maya corrected to Alice. Seven Given-When-Then criteria authored and accepted in full. Rejected: planning the monolith (both remaining open questions gated only the share/ingest part), and holding for itd-77 (the corpus default path works without relocation). Hand-run graded into the decomposition-calibration note.
- 2026-08-16 — the v0.5.1 release gate ruled PROMOTE ×2 at content commit
  264d9b1 — the first fully derived cut: changelog composed from the cut's
  resolved records and ingested behind the completeness bijection (three
  entries citing iss-254, iss-255, iss-257). Full tier (the cut is additive by
  iss-257). docs-currency: two findings — the README ladder sentence
  over-quantified to every hook (fixed in the receipts commit: "every other
  hook") and the removed contributor-guidance pointer (maintainer-directed,
  no change). Crosscheck: 25 discrepancies, substantially the iss-250 corpus
  re-confirmed between two same-day cuts, dispositioned against that open
  issue rather than re-filed. Rejected: re-filing the drift as a new issue
  (iss-250 already carries the corpus and stays the single tracking home),
  and holding an additive cut on record drift the brief's own aspirational
  posture (adr-5) anticipates.

- 2026-08-17 — spc-27 population calls, made unattended within adr-40's test
  ("what is compared to what") and recorded for re-litigation avoidance:
  (1) every `ahoy` sub-verb bucketed `—` — doctor/identity-check are install
  diagnostics of the ops environment, not assessments of the record/repo/change
  the adr-40 roles reason about; (2) `launch ship` carries bucket `gate`,
  representing the binding "launch changelog guardrail = gate" pre-ruling on
  the verb that runs the guardrail (the guardrail is not separately
  registered); (3) intent's unbuilt design-target sub-verbs (refine / grill /
  ship / consistency / shape / reclassify) get NO staged rows yet — `intent
  consistency` is the adr-40 multi-act case whose split is design-time work,
  so bucketing it now would guess into a closed list; the registry prose
  still names them design targets and the armed check stops any live claim.
  Rejected: a `partial` surface status (ruled out at planning), and staged
  rows with guessed buckets.

- 2026-08-17 — README bucket-vocabulary alignment (#289), made unattended
  against adr-40's own table rather than by taste: grading delivered work
  against acceptance criteria is an *audit* (reality vs a recorded
  commitment), so four "review" call sites move to audit wording, agreeing
  with the one call site that already said "intent auditor". Two further
  calls inside the same passage: (1) "a privacy review" becomes "a
  privacy-hygiene lint" — a cross-cutting rule every feature must satisfy is
  adr-40's definition of a lint, and abcd's own `privacy-hygiene` emits a
  rule id and an exit code, not SHIP/NEEDS_WORK, so the example contradicted
  the sentence defining it; (2) "owned by the AI-engineering team" becomes
  "run by the AI-engineering team, read by you" — the role table gives the
  product thinker the audit bucket, so *owned by* pointed the reader at the
  wrong end of the handover and made the one bucket they do own read as
  someone else's property. Also corrected a factual gap: cross-cutting rules
  do not merely "go straight into the brief", they have a record home in
  `intents/disciplines/` (`kind: discipline`; itd-1/5/37/79/81/84 live there
  today). Rejected: leaving the prose loose on the grounds that user-facing
  writing need not track the code's vocabulary — adr-40 closed the bucket
  list precisely because an autonomous run given a criterion will interpret
  it, and the README is the first thing both a contributor and an agent read.

- 2026-08-17 — Command parents owe TWO guarantees, not one (iss-266, #293).
  A parent refuses a mistyped sub-verb only if it is Runnable AND declares an
  Args validator; missing either exits 0 silently (no RunE ⇒ cobra returns
  flag.ErrHelp before ValidateArgs; Args nil ⇒ legacyArgs falls through to
  ArbitraryArgs). The house convention is therefore `RunE: helpRunE` plus an
  explicit validator on every parent, asserted structurally over the live
  command tree rather than a hand-kept list. `capture` and `intent` are a
  named exemption — their positional is free text, guarded by their own
  suspected-typo check — and the exemption list is itself tested against
  reality so a stale name cannot quietly drop a parent from the sweep.
  Consequence accepted and filed as iss-267: exit 2 is the *blocking* status
  on the hook plane, so hook sub-verb names are now a compatibility contract
  and a future rename must carry an alias rather than rely on the old
  silent-exit-0 cushion. Rejected: fixing only the parents named in the issue
  (the same hole would return with the next parent added), and asserting the
  behaviour with a hand-maintained list of parents (it had already drifted —
  `identity` and `intent audit` were absent yet not exempt).

- 2026-08-17 — The attribution gate's list-container fence limit is ACCEPTED as a
  standing residual (iss-270, wontfix), not deferred. A fence opened inside a
  first-level list item sits at column 2 (ordered: column 3), under the matcher's
  absolute three-space limit, so it is read as document-level and a footer inside
  that span is stripped although the forge renders it as an ordinary paragraph.
  All three remedies are worse than the hole: tracking container context means a
  markdown block parser inside a bash gate; refusing to strip from any document
  containing a list marker is safe by construction but would disable the fence
  concession in almost every real body here, reverting iss-268 in practice while
  appearing to keep it; and forcing openers to column 0 was tested by the security
  review and LEAKS, because under-recognising an opener is not conservative —
  markdown then closes its block at a line the matcher treats as the opener,
  shifting the whole span. Acceptance is defensible because the gate's defended
  property is untouched: it exists for a footer a tool appends BY DEFAULT (itd-91,
  78 pull requests), a tool appends last, and footer-last is caught whatever
  precedes it — fuzzed at 0 violations in 2400 differential bodies against a real
  CommonMark parser. This shape needs a body CONSTRUCTED AROUND the footer, which
  the design already declines to defend, and such an author would delete the
  footer instead. Pinned in the corpus for bulleted and ordered items so a change
  is visible. Rejected: all three remedies above; and leaving the record open,
  which would invite the indent-relaxing "fix" that widens the hole. Reopen only
  if the gate is asked to defend against a deliberate author — a different design.
- 2026-08-18 — The guard's parse layer is a MISTAKE FILTER, not a security boundary
  (adr-42, iss-272). Enumeration-completeness is abandoned as the matching strategy:
  Tier 1 (position-anchored, blocks) keeps the registry match; Tier 2 (position-agnostic,
  warns) speculatively re-matches from each later command-position token when NO ENTRY
  MATCHED, each suffix run through expandPayloads, converting the unbounded WRAPPER class
  from silent allow to loud warn without enumerating anything — 6 of the 10 verified
  bypasses, 7 once each speculative suffix runs through expandPayloads. It does NOT reach
  the exec-string class (su/runuser/script -c keep an opaque payload isShellFamily will not
  open), which stays silent until part C lands. Forced by three facts: the
  defect is the wrapper path's missing fail-safe (the interpreter path has shellUnresolved,
  the wrapper path has nothing), the wrapper set is unbounded in principle and its
  tractable part cannot be read off the documentation (nsenter's --help renders -S as
  optional-arg and unshare's renders the same letter, same package, same version, as
  required-arg, while BOTH binaries consume the next token — so only probing the installed
  binary is sound; gh-299 is the in-repo proof — its first list was taken from the bug
  report and was wrong three ways, omitting --shallow-file (a live force-push bypass in git
  since 1.9) and counting --exec-path/--super-prefix as value-taking when neither is, and
  it TOTALLED NINE EITHER WAY, so a size assertion certified the wrong list as complete), and every vendor shipping this control documents it as
  steering, not enforcement.
  Tier 2 is BOUNDED against a measured quadratic DoS — the drafted form ran 14.9s at 6000
  tokens, ~8h of CPU at the 1 MiB stdin cap, inside the PreToolUse hook — so O(N) starts, a
  ~64 cap, short-circuit, pinned by a benchmark. The gate is "no entry matched THIS SEGMENT" — never "argv[0]
  unknown", which would switch speculation off for git, and never per-Check, which would
  hand an author a one-token suppression (`git clean -fd ; nice git push --force` matches
  git-clean, so a whole-command gate disarms the fail-safe for the whole line).
  Per-segment is also what makes the false-positive figures a floor, but only with
  "unknown command" pinned to commandOf's OUTPUT (wrappers, assignments and reserved words
  stepped, basename taken) rather than the literal first token: matchSegment keys on exactly
  that value, so a segment commandOf resolves to something no entry names cannot match any
  entry, and the nesting is by construction. Under the literal reading it is false —
  `env git clean -fd gh repo delete .` resolves to git and matches git-clean, while a
  literal-token gate would see env and fire on git clean's own pathspecs.
  Also decided: a synthetic verdict raised from a speculative suffix (expandPayloads emits
  two blockers, envSpecialBlockSignal and depthBlockSignal) is DEMOTED to warn, never
  honoured at its own tier and never dropped — honouring blocks on two layers of
  uncertainty, dropping invents a second silent suppression path inside the fail-safe.
  The ~64-start cap is PER SEGMENT for the same warn-rate reason. Enumeration continues,
  demoted to an upgrade (warn → precise block) and safe to be incomplete. The exec-string
  family (su/runuser/script/flock -c) gets a small table, NOT a generalisation of
  shellCPayload, which would ship six new SILENT allows on the long spellings. Sequencing
  D → A → B → C. Rejected with evidence: add-the-missing-names as the primary fix (third
  instance of one defect); descend-into-any-spaced-token (breaks the 100% known-good floor —
  the spc-16 incident-capture fixture fires); flag-shaped-prefix-only (silences most true
  positives); allowlist (recorded for a future fail-closed mode, wrong for arbitrary developer
  work). Accepted cost: false positives on unquoted hazard-shaped text wherever no entry matched —
  0/1144 repo-mined lines, 21/79 adversarial, but BOTH corpora were run against an
  argv[0]-gated prototype, i.e. the gate this decision rejects, so the figures are a FLOOR
  (per the nesting above) with an unmeasured margin, and re-measuring under the adopted
  gate is a merge precondition. The fail-safe covers the WRAPPER enumeration only: a
  missing interpreter name (fish -c) and a missing git value flag are still silent allows
  under Tier 2, so this record covers the predicted fourth instance only if it lands in the
  wrapper set. The warn-storm STOP from the
  2026-08-15 note binds: measure on real agent commands before merge.
- 2026-08-18 — adr-42 parts D and A built. TWO amendments the build forced, both recorded
  in the ADR. (1) The ~64-start cap does NOT bound the cost on its own: every speculative
  start pays commandOf, which walks the whole leading wrapper chain, so 64 starts on a 1 MiB
  line of `env env env ...` measured 14.2s inside the PreToolUse hook — linear in line
  length and still an outage. A second bound (maxSpeculativeWindow = 512 tokens read per
  start) takes it to ~120ms; a truncated window warns through the same fail-loud path as an
  exhausted start cap, so the coverage trade is never silent. Pinned by a test at the real
  1 MiB stdin cap, watched failing at 14s with the window removed. (2) A speculative start
  must NOT skip known wrappers, though commandOf would step them: the wrapper is exactly
  where a payload lives, and `myrunner env -S '<hazard>'` needs `env` in command position for
  expandPayloads to read the -S value at all. Skipping wrappers is lossless for MATCHING and
  lossy for EXPANSION — caught by the demotion test, which failed as an allow.
  The warn-storm STOP is DISCHARGED with committed evidence: 961 mined command lines with
  zero Tier 2 fires, plus a labelled adversarial corpus (23 quiet / 18 caught / 5 Tier 1
  blocks / 2 accepted false positives), both under internal/core/guard/testdata/corpus/ with
  the ceiling enforced by a test. The shipped fail-safe is much quieter than the
  argv[0]-gated prototype predicted, because a hazard inside a quoted argument or a path
  never reaches command position. `git bisect run` and `git submodule foreach` now warn,
  exactly as the not-argv[0] gate predicted.
- 2026-08-18 — adr-42 parts B and C built; iss-272 closed. Part B named fourteen wrappers
  (nice setsid stdbuf ionice eatmydata proxychains chrt taskset unshare nsenter flock chroot
  runuser busybox), which upgrades each from part A's loud warn to a precise Tier 1 block.
  Every flag list is DERIVED BY PROBING the installed binary in a test, never from --help,
  and the probe caught two errors before they shipped: `nsenter -W/--wd` consumes NOTHING
  while `unshare -w/--wd` — same letter, same package, same version — is required-argument
  (listing nsenter's would have made the walk step over the COMMAND, a miss the table would
  have INVENTED); and `taskset -c` is a format switch for the MASK operand, not a value flag.
  A flag this environment cannot classify is listed as unprobed WITH ITS REASON (setsid --ctty
  needs a controlling terminal, chrt --sched-runtime needs SCHED_DEADLINE, nsenter/chroot/
  runuser need privileges) — silence is what gh-299 shipped. Part C added an exec-string TABLE
  for su/runuser/script/flock, not a shellCPayload generalisation, which would have resolved
  the short spellings free and silently allowed six long ones (su --command, --command=,
  --session-command, runuser --command, script --command, flock --command); each has a test.
  Classification runs on the raw token chain before commandOf because runuser and flock are
  also wrappers. Recorded limit kept honest: `su -c` runs the TARGET user's login shell,
  overridable with -s, so the payload is not guaranteed POSIX grammar — a false-negative cost
  only. All ten of iss-272's bypasses now produce a verdict; the acceptance test asserts that
  in the issue's own terms.
- 2026-08-18 — Two adversarial reviews of the adr-42 implementation; three findings that
  changed the code, all reproduced against the binary first. (1) DoS, 23.6s at the 1 MiB cap:
  maxSpeculativeWindow bounds the token COUNT and an execute-a-string payload is ONE token of
  unbounded length, so 64 starts each re-tokenized a 1 MiB payload. Two more bounds added — a
  payload-BYTES budget and a whole-line START budget, both per Check, after a per-segment byte
  budget still left 1,200 short segments at 8.4s. The lesson generalises: every per-item bound
  needs a whole-input bound behind it, and the ADR had said in advance that the benchmark must
  exercise the payload path — the first bound test pinned the cheaper half anyway. The bound
  test is now four adversarial shapes, each verified by mutation to fail on the bound it names.
  (2) FALSE BLOCK, and a suppression behind it: scanExecString ran to the end of the segment, so
  `flock /tmp/lock /bin/echo -c "<hazard>"` read a `-c` that real flock passes to /bin/echo as an
  argument (verified: it prints them). That both blocked a harmless command AND made the segment
  look understood, switching Tier 2 off for it — a `flock` prefix was a one-token off switch,
  the payload-level twin of the per-line suppression decision 4 rejects. Fixed with a per-verb
  count of the operands the verb owns before the launched command begins; only flock has one,
  because su genuinely permutes (`su root /bin/echo -c 'echo SEVEN'` prints SEVEN). This
  falsified the code's own claim that a mis-parse "never yields a false block".
  (3) The wrapper probe would have FAILED ON CI: `chrt -f 1` needs CAP_SYS_NICE and a GitHub
  runner is unprivileged, and the per-wrapper needsRoot blanket also hid the nsenter -W
  counterexample the file exists for. Replaced with a DERIVED control probe (if the wrapper
  cannot run its own baseline here, nothing it says is evidence) plus per-flag reasons. The
  candidate list — itself a hand-maintained enumeration, i.e. the CWE-184 shape inside the test
  written to defeat it — now comes from the binary's own --help, which is trusted to ENUMERATE
  and never to CLASSIFY. That immediately found three real misses: nice --adjustment, runuser -w
  and runuser -s, each degrading a precise block to a warn. 44 classifications became 121.
  Also: every synthetic id that contributed is now listed in Matches (a Tier 2 fire used to lose
  the slot to any earlier payload warn and vanish — which the warn-rate gate counts, so the blind
  spot hid itself); the cap warn no longer fires when only losslessly-skipped tokens remain; the
  warn ceiling is an absolute 2 rather than a 1% rate that permitted 9; matchesAny deleted as
  dead; and the three surfaces no longer list "launched through a wrapper outside the known set"
  as something an allow does not see, which is the case this change converts to a warn.
- 2026-08-19 — The `.abcd/work/issues/` ledger is the single canonical issue store; forge
  issues (GitHub) are a derived, opt-in mirror surface — consistent with the host-delegated
  boundary (forges are hosts). Per-field ownership: the ledger owns existence, content, and
  resolution; the forge owns nothing. Sync is one-way mirror-out, the forge id written back
  into the record as the echo suppressor; forge-side activity (triage labels, a close) is
  never auto-synced — it generates an import proposal a human accepts via an explicit import
  that writes the canonical file with provenance. A forge-side close never imported is
  reopened by the next mirror pass: the mirror is self-healing by construction. Autonomous
  hunts file findings as fingerprint-keyed iss-N records inside the gated PR they already
  open, validated by record-lint (the armed detector) — never as forge issues. Grounded in
  commissioned SOTA research (see
  `../development/research/notes/2026-08-19-issue-ledger-forge-sync-sota.md`): one-way
  canonical sync is the surviving pattern (coreos/issue-sync post-mortem); agent findings as
  records-via-reviewed-PR is 2025-26 practice (GitHub advisory-database, ClusterFuzz
  fingerprint dedup); ungated agent filing degrades the whole record (curl, 2025).
  Architecture-shaping: graduate to an ADR.
- 2026-08-19 — Review agents keep their Bash grant: running the project's tests is
  part of what makes their reviews evidence, and the exposure that motivated the
  question is closed by the intake rule that external-contribution review runs
  only in CI or a network-less container (.abcd/work/intake.md S4). iss-278
  narrows to the unbuilt PQ linter. Maintainer decision.
- 2026-08-19 — bughunt round 1 (branch bughunt-b/round-1): captured iss-291..iss-304 and fixed
  ten. Code: iss-291 the guard exec-string cluster reader ignored value-taking short flags, so
  `script -Tc out.txt -c '<blocker>'` was a silent allow that also disarmed the Tier-2 fail-safe
  (value flags now abort the cluster read); iss-293 ParseSemver swallowed strconv.Atoi's range
  error, clamping an oversized component to MaxInt64 (now propagated, matching spec.go); iss-292
  the wrapper-probe test wrote junk files (0,1,PATH,root) into the tracked tree because runsCommand
  had no cmd.Dir (now a temp dir; the four committed artefacts removed). Docs/record: iss-294
  version --check "only network" claim, iss-295 README unshipped brief-onboarding claim, iss-296
  ADR-index + ruleset-README count drift, iss-297 adr-40 verb-rename residue in roadmap+brief,
  iss-298 all 18 broken/stale brief heading anchors and section citations (anchorcheck now 0),
  iss-299 phantom "in-session dispatch" internals chapter, iss-300 persona glossary role-first
  contradiction. Recorded not fixed (stay open): iss-301/iss-302 external-review.yml approval-tally
  (paginated per-page --jq) and error-masking (|| echo none + under-permissioned collaborators
  endpoint) — required-check workflow, unverifiable without a real Actions run; iss-303 links_resolve
  never validates heading anchors (systemic fix behind the anchor cluster); iss-304 deferred nitpicks
  (a-4 memory ingest --source size TOCTOU, b-4 completion/help omitted from the CLI reference, d-12
  the CI job id `record-lint` runs the reviews-charter script). Refuted this round: the scripts/
  payload over-inclusion (knowingly-deferred open dependency, iss-34), the attribution/reviews gates
  running from the PR tree (CODEOWNERS contains it), AGENTS.md's CI-job list (ci.yml-scoped, iss-182),
  the release Go-version float (deliberate), and the 05-personas.md random-picker line (prior art iss-49).
- 2026-08-19 — The cold reading's `ACKNOWLEDGEMENTS.md` entry is not owed and no
  longer gates itd-86. The 2026-07-13 sensemaking-method note had deferred the
  entry pending the originator's crediting preference; the originator is a
  co-author and now co-maintainer of abcd — an author of record, not a third
  party the acknowledgements file exists to credit. The note's provenance
  paragraph is amended accordingly.
- 2026-08-19 — Design review of the candidate record extensions (scope
  conditions on intents, selection grounds at plan time and capture triage,
  per-record provenance of contributed material, a knowledge record, a framing
  passage in the brief) and of the cold-reading build path: one independent
  adversarial pass over the plan plus two SOTA surveys, written up as the two
  dated research notes 2026-08-19-*-sota.md. Six decisions. (1) No never-bluffs
  principle is minted — the brief's marking discipline is loud-staging applied
  to prose, and a framing passage cites loud-staging directly; the sensemaking
  entry's one-principle precedent (no near-copies) applies. (2) itd-60 is not a
  prerequisite for brief rewrites — the rewrite is existing human practice at
  ship time; itd-60 stays an unplanned ambition. (3) Nothing in itd-87 must be
  settled before itd-86 builds: capture carries no semantic dedup (id-collision
  guards only), so there is no store for a re-raising detector to fight;
  recurrence handling stays deferred to itd-87. (4) Promise-questioning —
  "delivered as promised, but the promise itself is in question" — if ever
  built, is its own surface with its own ADR extending adr-40's closed bucket
  list; it is never a slot inside the intent-fidelity verdict, whose one act is
  reality-vs-commitment. Nearest published vocabulary is Lean's
  hypothesis-invalidated / pivot; no surveyed practice has a machine-readable
  verdict separating built-wrong from wrongly-specified. (5) itd-86 is not
  plannable until its isolation mechanism is named in scope: no allowlist
  invocation surface exists, installed agents read the whole tree, and host
  subagents inherit project-context files by default — the blindness contract's
  mechanical half is a build item, and whatever remains disciplinary is
  reported as such per loud-staging. (6) The record extensions land as
  hand-run conventions before any automation, per script-first-mvp; the ideate
  rejected-alternatives shape and the wontfix_reason invariants are the two
  in-repo patterns they extend (one-canonical-primitive). Grounds and sources
  in the two SOTA notes.
- 2026-08-20 — bughunt round 2 (branch bughunt-a/round-2): captured iss-331..iss-343 and fixed (ids 339-343 re-minted from an initial 326-330 after main concurrently took those numbers in the v0.6.1 cut)
  ten (six fix-impact, four internal). Code/security: iss-339 the scanner's githubRemoteRe matched
  the github.com host case-sensitively, so a mixed-case remote (git@GitHub.com:…) left
  GitRemoteUsername empty and the github_username redaction kind never armed — the caller's handle
  survived history-capture redaction and the PII scan (added (?i)); iss-340 `abcd history show`/`list`
  printed the untrusted transcript body and metadata fields with no termsafe, replaying ESC/CSI/C1/bidi
  sequences raw (now SanitizeBlock on the body, Sanitize on the fields); iss-341 readLifeboatFile kept
  an Lstat→root.Open window over an untrusted lifeboat (FIFO hang / symlink-follow) — routed through
  fsutil.ReadGuardedInRoot, dropping the unused abs param; iss-342 SourceContext.ListDir opened
  directory entries with a blocking open, so a statically-planted FIFO named `docs` hung
  disembark probe/plan/pack over an untrusted target repo with no race (now O_RDONLY|nonBlock, matching
  ReadFile). Docs (fix): iss-343 prepare-this-repo.md named the pre-flattening commands/abcd/ path and
  "three levels up" (now flat commands/ and "two levels up", tied to CLAUDE_PLUGIN_ROOT); iss-331
  lint.md presented the abcd-lint:allow waiver as global when only privacy-hygiene honours it. Record
  (internal): iss-332 development/README.md denied the issue ledger adr-32 created; iss-333 the brief's
  05-internals/03-configuration.md development tree drew the pre-adr-30 layout (roadmap/intents, research/adr,
  research/phase, .abcd/specs) — redrawn flat by artefact type; iss-334 intents/README.md taught the
  v0.6.0-retired intent review verb and intent-fidelity-reviewer agent (swept to intent audit /
  intent-auditor); iss-335 the issues/ store contract omitted the mandatory impact field and the
  agent-observation source. Recorded not fixed (stay open): iss-336 gate_lockstep blank-path silent
  disarm (nitpick, no external arming path — defence-in-depth only); iss-337 probe.go inner-descent
  OpenRoot FIFO race (needs a raw openat restructure, os.Root.OpenRoot takes no flags); iss-338 seven
  registry personas given gendered pronouns against the itd-79 they/them discipline (nitpick cosmetics
  batch). Refuted this round: govulncheck non-gating (documented deliberate advisory — commit 61a15b4
  and ci.yml:362-363), and the CHANGELOG bijection gap (the bijection reads the composer's JSON payload,
  never CHANGELOG.md; the residual is the already-open iss-256). Model routing: orchestration on
  Claude Fable 5; hunters, per-finding refuters, and one of the two pre-merge reviewers on Claude
  Opus 5; the second pre-merge reviewer on Claude Fable 5 (dual-model merge gate).
- 2026-08-20 — bughunt round 2 (branch bughunt-b/round-2): captured the round-2 records (iss-346..iss-357, with the two colliding ids re-minted as iss-358/iss-359), fixed
  eleven records, closed iss-201, and wontfixed iss-358 on pre-merge review evidence.
  Code: iss-358 lanHostRe/deviceHostRe trailing-\b redaction miss (the iss-307 sweep's
  residual) — BUILT, then REVERTED: the pre-merge adversarial review proved the loosened
  pattern flags snake_case selectors (stream.local_addr, args.local_rank) and Stage-1
  redaction rewrites findings into the irreversible history store, so the warn-tier miss
  is the accepted cost, pinned by TestSnakeCaseSelectorsStayQuiet; iss-359 redirect-controlled FinalURL
  carried C1/bidi/zero-width runes raw into --json and the committed baseline (fetch
  boundary percent-encodes, baseline validates, termsafe/coverage doc claims corrected);
  iss-346 guard also-matched dropped a matched warn under a synthetic block; iss-347
  readTranscript symlink-follow + truncation (now fsutil.ReadGuarded); iss-201 hook
  stdin over-cap misdiagnosis (cap+1 probe at readHookInput and the guard hook);
  iss-356 batch (urlguard RFC 6890 ranges incl. CGNAT, repolint not-scanned warn,
  64-hex resolve sha, CLI help truths, release.yml wording + templates, dependabot pip).
  iss-357 records the memory-ingest FinalURL sibling site the cite fix did not reach.
  Records: iss-348/349/350 user-facing doc truths; iss-351 discipline roster -> pointer;
  iss-352 phantom research/adr repointed; iss-353 note naming yields to adr-30; iss-354
  go-version lockstep test (mutation-verified). Recorded not fixed: iss-355 — a batched
  merge-queue push tags the batch tip, so auto-release + the HEAD^2^ derivation wedge a
  release roll merged as a non-final queue entry (iss-326's outcome by a route itd-93
  cannot close); required-check workflow logic, left with two proposed fixes. Refuted
  this round (kept out of the ledger): the external-review pull_request_target check-sha
  wedge (check runs attach to the PR head, proven from run history), the docs-site
  ungated claim (docs-lint gates docs/ on every PR; iss-234 records the inert design),
  the empty Diataxis indexes (owned by iss-216 and the facilitator plan), identity email
  case-folding (byte-exact parity with the sh pre-commit gate is the contract), and the
  guard backtick parser gap as a fix target (iss-148 records the reverted attempt; only
  its missing help-text disclosure was new). Dedup boundary: iss-307/311/317/319/321/325
  and the v0.6.0 wedge (iss-326/327) were already held by hunt A or the maintainer and
  were not re-reported.
- 2026-08-20 — Bug-hunt round 1 (hunt A), landed late: the round ran 2026-08-19 on a branch that
  waited while v0.6.1 and bughunt round 2 merged; reconciled onto main today rather than replayed.
  Captured iss-305..iss-325 (21 findings, each adversarially refuted before capture). Ported with
  their tests: the privacy-rule leading-boundary gate and Windows-arm case fold (iss-305/308), the
  scanner ipv4/mac trailing-\b redaction miss (iss-307), the memory-ingest guarded read (iss-310),
  the record-lint/scaffold-sync isolated-env root discovery (iss-311), the hermetic attribution
  corpus (iss-313), the .PHONY completion (iss-314), the guard python/perl false "loud warn" claim
  across all four surfaces (iss-315), the AGENTS/ci.yml job list and dev-README record-map/personas
  corrections (iss-318/319), and they/them across seven intents (iss-321). Superseded, not
  re-applied: the SemVer over-int64 rejection (iss-309) — hunt B's iss-293 landed the same fix on
  main first with tests covering all three component positions, so main's implementation stands and
  the branch's code and test were dropped; likewise githubRemoteRe's (?i) (iss-306 = round 2's
  iss-339) and the wrapper-probe cwd isolation (iss-312, fixed on main with a sync.OnceValue
  probeDir). Re-derived rather than replayed: the doc fixes — main had already fixed iss-316/317/320
  its own way (round 2's anchor sweep is on main verbatim), so only the still-true corrections were
  applied. Open record-only: iss-322..iss-325.
- 2026-08-20 — itd-114 (collision-proof record ids) is pulled forward: two live
  parallel-mint collisions in one day (iss-330; the round-2 renumber), with a
  third structurally predicted whenever two minters start from the same max.
  The 2026-08-20 multi-minter morning is pre-registered as its field test
  (research/notes/2026-08-20-itd-114-collision-field-test.md); the planning
  interview opens with that note's graded results.
- 2026-08-20 — bughunt round 3 (branch bughunt-a/round-3): captured twelve findings (iss-360..iss-369, plus two renumbered to iss-373/iss-374 after round 2's own collision re-mint landed inside this round's block — the week's fourth collision,
  each adversarially refuted before capture; ids minted above the sibling bughunt-b/round-2's iss-357 via
  the cross-ref floor). Fixed eleven, one recorded-not-fixed. Code/security: iss-361 the lifeboat coverage
  renderer (per-repo and aggregate) printed untrusted cross-repo status/tier/tiers_present/section fields
  to the terminal with no termsafe — only the repo name was sanitised — so a crafted probe report injected
  raw ANSI/OSC-8/bidi (reproduced); both renders now route every repo-derived string through sanitize with
  widths from the sanitised strings; iss-366 termsafe.Sanitize masked the deprecated FEFF joiner but not
  its Unicode successor U+2060 WORD JOINER, the U+2061-2064 invisible operators, or U+00AD SOFT HYPHEN —
  all now masked; iss-364 the capture related_specs validator pinned the retired fn-N namespace (reFnID)
  instead of the live spc-N, making the field unusable — switched to reSpcID, dropped the orphaned reFnID,
  corrected issues/README.md. Surface (additive): iss-362 abcd banlist --json omitted the reach caveat its
  command file tells an agent to relay verbatim — added a reach field to PrivateReport (unconditional,
  mirroring the ahoy status board). Docs (fix): iss-363 commands/lint.md and prepare-this-repo.md
  enumerated five lint conventions omitting the shipped identity-positioning rule; iss-367 commands/launch.md
  documented a nonexistent top-level files count for launch --dry-run --json (the list is bundle.files).
  Infra (fix): iss-365 the govulncheck CI job pinned @v1.1.4 whose bundled x/tools caps its type-checker at
  go1.24 while go.mod declares go 1.25.6, so the scan failed at package loading and exited before analysing
  anything — masked as an advisory red since it landed 2026-08-19; bumped to @v1.7.0 (verified locally to
  load go1.25). Record (internal): iss-373 three ADRs (adr-11/13/20) ended with leaked tool-call scaffolding
  (</content></invoke>) — deleted; iss-374 the ADR index listed adr-44 proposed vs the record's accepted —
  corrected; iss-360 adr-7 carried no frontmatter against the ADR contract (sole outlier of 36) — added;
  iss-368 the core/disembark glossary term linked its counterpart to the interview-context embark (a
  different bounded context) rather than the /abcd:embark unpack surface — repointed. Ledger hygiene:
  resolved the stale iss-338 (round 2's persona-pronoun record already fixed by round 1's iss-321; all ten
  cited sites are they/them). Recorded not fixed (stays open): iss-369 readTranscript opens the
  hook-supplied transcript path without O_NOFOLLOW — the last bespoke external-input read not routed through
  fsutil.ReadGuarded; refuted as no reachable exploit (harness-supplied path, no path policy, so a direct
  path already reads any file), defence-in-depth only, left for a scoped consolidation. Model routing:
  orchestration on Claude Fable 5; five parallel hunters, six per-finding refuters, and one of the two
  pre-merge reviewers on Claude Opus 5; the second pre-merge reviewer on Claude Fable 5 (dual-model merge gate).
- 2026-08-20 — Homebrew distribution recorded and parked. Viable now that the
  repo is public, and it is what the reference class (gh, lazygit, k9s) leans
  on; a personal tap is the solo-maintainer shape and homebrew-core needs
  notability. Deferred until after a binary-update verb exists, because the
  two interact: an updater must gate on install-channel detection and print
  `brew upgrade abcd` instead of self-replacing when the binary's RESOLVED
  path sits under a brew prefix (/opt/homebrew, /usr/local,
  /home/linuxbrew/.linuxbrew — a prefix test, never a "Cellar" substring
  match). The named incident is flyctl's autoupdate postmortem, "Flyctl
  Versions, Autoupdating, and the CLI Apocalypse"
  (https://community.fly.io/t/flyctl-versions-autoupdating-and-the-cli-apocalypse/13794):
  brew-path update loops, Nix read-only breakage, Windows UAC. Note
  for the revisit: goreleaser's `homebrew_casks` pipe is the current SOTA
  route, but abcd's release pipeline is bespoke (semantic-gate attestations,
  no goreleaser), so the tap push would be a hand-rolled release-workflow step.
- 2026-08-20 — the spc-33 mint mechanics are ruled and the native mint ships
  for captures: random-suffix width is 4 digits (16-digit ids, the top of the
  range itd-114 priced — entropy alone carries the cross-branch same-second
  case), and the same-instant tiebreak keeps the O_EXCL reservation but
  REDRAWS a fresh id on clash rather than bumping (a bump is a miniature
  max+1). The capture family now mints `iss-<yymmddHHMMSS><rrrr>` through the
  family-generic recordid seam; itd/spc stay on the legacy refs-union
  allocator until they adopt the seam as configuration. Capture's refs scan
  and `mint_warning` retire with the maximum it warned about; the uniqueness
  detectors stay armed as the scheme's fail-safe. Once this lands on main,
  the per-task defensive minting protocols (fetch-and-verify twice, scanning
  unmerged hunt ledgers) retire for captures, per adr-45.
- 2026-08-20 — Bug-hunt round 3 (hunt B), branch bughunt-b/round-3. Baseline
  (make preflight + gofmt -l .) green before any change. Captured iss-382..iss-392
  (11 records), each adversarially refuted before capture. Fixed and resolved this
  round: iss-382 disembark plan's RenderManifest printed source-repo file paths raw
  while the same view sanitised its neighbours (terminal-escape injection over an
  untrusted source repo); iss-383 the history transcript store's read path used raw
  os.ReadFile while its write path was fully hardened, so a planted FIFO wedged
  history capture under the store flock and a symlink was followed (both reads now
  fsutil.ReadGuarded); iss-384 Baseline.validate sanitised final_url alone while
  echoing the entry key and quoted fields raw to the terminal through the CLI error
  surface (incomplete-fix residue of iss-359); iss-386 launch scaffold stripped the
  go.mod patch and refused one back, shipping every adopter a floating go-version
  (the adopter-side residue of iss-289); iss-385 commands/version.md still named only
  two network verbs after two rewrites (docs cite refresh and memory ingest also
  fetch); iss-387 the release-gate manifest pinned 17 briefDocs while four shipped
  surface chapters (guard/ideate/identity/banlist) sat outside it, so every full-tier
  crosscheck attested coverage it never ran; iss-388 the phases README called the
  shipped itd-88 planned; iss-389 the examples-use-reserved-identifiers principle
  described its shipped iss-154 lint as not-yet-built; iss-391 research/notes/README
  routed to the adr-30-rejected research/phase and research/adr layouts; iss-392 the
  confirmed nitpick batch (canonical id-uniqueness key closing the zero-padded-filename
  hole, .yaml coverage in the go-version lockstep sweep, scripts/*.sh eol=lf, the
  lifeboat-reviewer 0.1.1 reconciliation, and the commands/abcd/ path repoint across
  five live records). Recorded, not fixed: iss-390 — the pre-decided promotion of the
  reserved-identifiers principle to a discipline-kind intent is a maintainer ceremony,
  so it stays open as the tracker. Refuted this round (kept out of the ledger): the
  scanner genericHomeRe case-fold (iss-308 already adjudicated against folding the
  POSIX /users/ arm — real API-route false positives, and home_path_other is warn-tier
  anyway); the ahoy refounding-candidate case compare (documented mutable labels; the
  miss lands on the designed decline path; iss-221 owns the real repair); govulncheck
  non-gating (documented deliberate advisory, DECISIONS 2026-08-19 and commit 61a15b4);
  the CODEOWNERS publish-surface omissions (the .abcd/* paths reach no installed user
  and the Go-tree reading proves too much; sole-maintainer bypass makes the scenario
  moot — a golden-file test for the bare scaffold render is the real follow-up if a
  write rung is ever granted); isHexSHA 40-hex (display-only, sha256 repos unreachable,
  iss-206 owns the sibling); the coverage_index.json raw read (already named by open
  #365); commands/docs.md's scoped "on behalf of documentation" heading; the agents/
  host-agnosticism claim (docs-lint bans harness names, not the model id, which is the
  sanctioned disclosure token); the make smoke vacuous-gate and runbook-parity claims.
  Model routing: orchestration on Claude Fable 5; five parallel hunters, seven
  per-finding adversarial refuters on Claude Opus 5; the dual pre-merge review runs one
  reviewer as Claude Fable 5 and one as Claude Opus 5.
- 2026-08-20 — ideate: deterministic-delivery-pipeline — verdict reframed. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-08-20-ideate-deterministic-delivery-pipeline.md
- 2026-08-21 — Bug-hunt round 4, branch bughunt-b/round-4. Baseline (make preflight
  + gofmt -l .) green before any change. Five parallel Opus 5 hunters swept the four
  dimensions; each candidate was adversarially refuted by an independent Opus 5
  subagent before capture. Findings: 8 substantive, 7 nitpick, 2 refuted. Fixed and
  resolved this round (code, each with a watched-fail test): iss-2608211132061930
  (MAJOR) lint.LoadConfig read the docs-lint/record-lint config with a raw os.ReadFile
  while its .abcd/*.json siblings guard.Load/rules.Load/positioning.LoadConfig all
  guard it — hook-reachable (ahoy.Detect) and cross-repo-clonable, so a committed
  config symlink to a FIFO wedged docs lint/lint/ahoy/hook/cite refresh, /dev/zero
  OOMed the CLI, and an out-of-repo target ran the whole ruleset from a file the repo
  does not own; now fsutil.ReadGuarded + an Lstat dir-symlink refusal. iss-2608211134281912
  four stdin operand readers (ideate/lessons/synthesis/readSource) read exactly cap
  and silently truncated an over-cap payload — worst on history capture (stored an
  8 MiB prefix under a sha256 idempotency key over the prefix, breaking spc-4's
  refuse-whole invariant); added readCappedStdin (cap+1 probe). iss-2608211139076040
  the iss-392 canonRecordID fix left four sibling lint keyspaces raw (delivery_state
  bucket + citation, spec intent-existence, superseded_by, spec_id_unique), so a
  zero-padded id slipped a gate open or false-blocked a link; routed every key and
  lookup through canonRecordID. iss-2608211140410761 abcd adr-N confirmed the
  frontmatter id byte-exact while record-lint and the citation resolver treat
  padded/quoted/cased ids as one handle, reporting a present ADR absent (and
  abcd adr-0003 routing then refusing); now a parsed-handle compare rendering the
  canonical id. Docs/record fixes: iss-2608211142146585 + iss-2608211142142469 the
  release-gate runbook claimed the verify job runs macOS+Linux (it is ubuntu-only;
  ci.yml's check job is the matrix and the merge gate) and framed the semantic-receipt
  gate as dormant-until-public-flip though it is public and has fail-closed two releases
  (v0.3.0/iss-108, v0.6.0/iss-326). iss-2608211142496517 cmd/scaffold-sync named a
  "scaffold-sync workflow" that was built and rejected (iss-209). iss-2608211143184945
  + iss-2608211143185943 the README Go badge (1.25→1.26) and the retired "next free id"
  capture description. iss-2608211144265046 commands/lint.md documented a line:0 sentinel
  the omitempty JSON never emits. iss-2608211144555085 the ADR index omitted the accepted
  adr-45 row (third recurrence of the class; iss-38 wants such indexes gated). iss-2608211145448208
  the DCO deferral text adr-43 retired survived in AGENTS.md and scripts/check-attribution.sh.
  iss-2608211146222734 the banned pre-adr-30 "roadmap/intents" literal survived in AGENTS.md
  and .abcd/README.md, outside the lint roots. iss-2608211146562243 RD001 was documented
  unconditional though check-reviews.sh exempts 7 of 10 live dirs (sha-keyed receipts).
  iss-2608211147403129 the ADR-RFC frontmatter pairing contract disagreed across three
  surfaces; reconciled rfcs/README.md and rfc-2. Refuted (kept out of the ledger): the
  CHANGELOG [Unreleased] Fixed-vs-additive sectioning (the cut discards Unreleased prose
  and re-derives sections from records; owned by open iss-256); the .abcd/work tier map
  "omission" (highlight enumerations, not inventories; iss-333 already blessed the table);
  and the launch DenyNamespaces case-fold symlink slip (inert — the hardlink alias map
  rejects via aliasDenied). Model routing: orchestration on Claude Fable 5; five parallel
  hunters and ten per-finding adversarial refuters on Claude Opus 5; the dual pre-merge
  review runs one reviewer as Claude Fable 5 and one as Claude Opus 5.
- 2026-08-21 — Visual identity roles (itd-133, maintainer-ruled at the planning
  interview): the block-pixel duckling is the mascot; the a-b-c-d signal-flag
  hoist (true ICS geometry at full size) is the official logo of the terminal
  surfaces (CLI and plugin); a small lifeboat marks the lifeboat verbs; the
  existing `docs/assets/img/logo.png` remains the forge/web logo for now. One
  pixel-grid source of truth in the Go tree; all terminal rendering behaviour
  stays with itd-112. This forecloses itd-112's object-vs-text-logo open
  question — the object is the flag hoist.
- 2026-08-21 — Bug-hunt round 5, branch bughunt-b/round-5. Baseline (make preflight
  + gofmt -l .) green before any change. Five parallel Opus 5 hunters swept the four
  dimensions; each candidate was adversarially refuted by an independent Opus 5
  subagent before capture. Findings: 4 substantive, 9 nitpick, 9 refuted. The C1
  ScrubbedEnv/GIT_CONFIG_GLOBAL candidate split the refuters (one CONFIRMED, one
  REFUTED); adjudicated REFUTED — keeping the developer's global git config readable
  is deliberate and test-pinned (TestScrubbedEnvStripsHijackKeepsGlobalConfig), the
  attack needs control of abcd's process env (out of scope under the trusted-env
  model), and the obvious fix (scrub to /dev/null) would blind the redaction probe
  for every developer whose identity lives in global config. Fixed and resolved this
  round (code, each with a watched-fail test): iss-2608211432258689 launch --dry-run
  printed WouldRefuseOn unsanitised so a committed control-char filename injected raw
  terminal escapes into the preview and CI log (now termsafe); iss-2608211432257954
  the same dry-run leaked a raw absolute path in lockstep.detail (a success envelope
  that never passes the error-surface scrub) — stripped at loadJSON; iss-2608211432258975
  abcd lint mapped a rule-engine fault to exit 1 (the tri-state's warnings-only code)
  so a CI gate keying on >=2 read a lint that never ran as an advisory pass — now
  exit 2; iss-2608211432254405 launch scaffold deriveBranch assumed .git is a
  directory, stamping the fallback branch main into the generated release workflows
  from a linked worktree — now resolves the gitfile (worktree gitdir HEAD + shared
  commondir origin/HEAD). Nitpicks fixed: iss-2608211432389181 the intent verdict
  reader was the last CLI operand on Lstat-then-os.ReadFile (false benign-TOCTOU
  comment) — routed through fsutil.ReadGuarded; iss-2608211432384430 the history
  rootSHA diagnostic named only 40 chars though the regex accepts 64 (SHA-256);
  iss-2608211432384091/389477/389791 three user-facing doc claims (README SessionEnd
  self-bootstrap overclaim, memory.md source.class vs source_class, ahoy.md missing
  the shadowed-on-PATH install_mode suffix); iss-2608211432489481/489288/482363/483805
  record hygiene (seven display-text/href link mismatches, two bare paths to absent
  dirs, the rfc-1/itd-26 one-sided pair, the work-tier roster omitting the issue
  ledger/reviews/rulesets). Refuted / prior art (kept out of the ledger): C1 above;
  C2 folding local_username to case-insensitive (net-negative — broadens a documented
  hard_fail false-positive class and reopens iss-31); S3 widening hasControlChar (the
  render sanitiser, not the integrity gate, is the fix); S4 launch --no-index dropping
  a force-added tracked file (dry-run preview writes no artefact, exclusion is visible
  and over-exclusion is the safe direction); D2 guard check exit-2 enumeration (the
  governing "could not be evaluated at all" definition already covers a disabled
  registry); R2 itd-3 spc-1 (a documented, minting-honored reservation). Model routing:
  orchestration on Claude Fable 5; five parallel hunters and nine per-finding
  adversarial refuters on Claude Opus 5; the dual pre-merge review runs one reviewer
  as Claude Opus 5 and one as Claude Fable 5.
- 2026-08-21 — Bug-hunt round 6, branch bughunt-b/round-6. Baseline (make preflight
  + gofmt -l .) green before any change and on the final tree. Five parallel Opus 5
  hunters swept the four dimensions; each candidate was adversarially refuted by an
  independent Opus 5 subagent before capture. Findings: 8 substantive + 2 confirmed
  doc/CLI, 5 refuted, 1 (I2) reverted on implementation. Fixed and resolved this
  round (each behaviour-changing code/script fix carries a test watched fail
  before the change, the lifeboat probe size-TOCTOU excepted — that race is closed
  by construction and its boundary test guards the cap, not the race):
  iss-2608211849467013 the
  guard tokenizer did not know shell redirection operators, so a glued redirection
  (git push --force>/dev/null) mutated the flag token and the blocker missed — a
  silent allow — and a leading redirection degraded a Tier-1 block to a warn; the
  tokenizer now drops a redirection and its target. iss-2608211849463840 the
  record/docs lint reporting walk read cfg.Roots with a raw os.ReadFile and no
  containment while the sibling citation collector guards the same field, so a
  cloned repo's committed lint config could read/lint/follow a file outside the
  tree (reproduced: escaping root, symlinked leaf, /dev/zero OOM) — routed through
  containedRepoPath/resolvedInsideRoot/containedRealPath/ReadGuarded, and the
  sibling glossary_dir walk in loadForbiddenSynonyms (the same cloned-repo-config
  read path, caught by both pre-merge reviews) was swept through the same stack;
  the remaining config-derived record reads in the lint package are tracked as a
  follow-up (iss-2608211914592726).
  iss-2608211849461061 frontmatter.Fields requires --- on line 0, so a comment-led
  record (13 glossary term files) slipped the no_git_metadata blocker; the
  lint-local frontmatterFields now slices past the comment, preserving absolute
  line numbers. iss-2608211849466878 citation DaysBetween took each date's local
  calendar day and restamped UTC, making staleness and the release-gate overdue
  blocker timezone-dependent; both ends now convert to UTC first. iss-2608211849468791
  the attribution gate's trailer/None presence checks are line-end-anchored over a
  class excluding CR, so a CRLF web-UI pull-request body false-red a correct
  trailer on the required check; check_text now normalises CRLF before every rule,
  with CRLF corpus cases. iss-2608211850070541 identity init faults exit 2 (was 1);
  iss-2608211850074600 the lifeboat probe read cap+1 and refuses a file grown past
  the cap (size TOCTOU); iss-2608211850070318 history list --json emits [] not null.
  Doc/record: iss-2608211849581263 + iss-2608211849583757 the persona brief said
  the picker chooses at random and omitted Nia (a residual of resolved iss-300 —
  selection is by role, never by name; roster is fourteen); iss-2608211849583208
  the bundled INTENTS default persona rule reached this repo's own intent authors
  because .abcd/rules.json never overrode INTENTS, so an INTENTS override now points
  authors at the registry (the default stays correct for a registry-less adopter);
  iss-2608211849580624 the README described download-on-every-update and a no-op
  .binary-meta remedy, rewritten for the spc-35 persistent cache; iss-2608211849582190
  CONTRIBUTING granted the fenced-quotation carve-out for commit messages too, now
  scoped to the pull-request body. Refuted / considered-and-rejected (kept out of
  the ledger): R3 the release-gate manifest prompt naming ten of twenty commands
  (an illustrative grounding string, not a gated inventory, and editing it ripples
  into receipt hashes); R4/R5 related_adrs and prose citations of superseded ADRs
  (resolve via the retired-set mechanism, deliberate historical provenance); R6
  "11 adapters" vs "five capability seams" (implementations vs categories, not a
  contradiction); R7 the adr-13 bare path to itd-36's old bucket (a benign ungated
  breadcrumb; the authoritative related_intents resolves by id); the update.go
  redirect-host case-fold (fails closed, availability only, impractical to test
  through a loopback redirect). I2 (release.yml "dormant until the public flip"
  comments) was implemented then reverted: release.yml is a scaffold template
  rendered for adopters who may be private, so the visibility-flip framing is
  generic-correct there, unlike the abcd-specific runbook round 4 corrected. Model
  routing: orchestration on Claude Fable 5; five parallel hunters and eight
  per-finding adversarial refuters on Claude Opus 5; the dual pre-merge review runs
  one reviewer as Claude Opus 5 and one as Claude Fable 5.
- 2026-08-22 — Bug-hunt round 7, branch bughunt-b/round-7. Baseline (make preflight
  + gofmt -l .) green before any change and on the final tree; started from main at
  9db1316. Five parallel Opus 5 hunters swept the four dimensions; each candidate was
  adversarially refuted by an independent Opus 5 subagent before capture. Findings:
  8 substantive (2 major, 6 minor) + 8 nitpick confirmed, 7 refuted/prior-art;
  nitpicks-only: no. Fixed and resolved this round, each behaviour-changing code fix
  carrying a test watched fail before the change and pass after (the repolint size
  TOCTOU excepted — closed by construction, its boundary test guards the cap not the
  race): iss-2608220131352917 the guard tokenizer read a leading & as a
  background/&& operator, so a glued or spaced &>/&>> both-streams redirection split
  the simple command and dropped its dangerous flag out of command position — a
  silent allow on every blocker-tier entry; now recognised as a redirection before
  the list split, target dropped, fd digit kept. iss-2608220134344680 the frontmatter
  scanners anchored on line 0 and TrimSpace does not strip a UTF-8 BOM, so a BOM- or
  multi-line-comment-led record slipped the no_git_metadata blocker and the
  record_schema gates; shared frontmatter.TrimBOM plus a stateful multi-line-comment
  skip in the lint and glossary scanners. iss-2608220142158516 abcd update and abcd
  history leaked the absolute home root into their success/refusal envelopes (text and
  --json), including the plugin-session refusal the plugin relays into agent chat;
  redacted to ~ via a new fsutil.RedactHome at the render boundary. iss-2608220136593438
  ahoy tested .git for dir-ness, so a linked worktree or submodule (gitfile .git) was
  misclassified as an unmanaged folder and ahoy install exited 0 aborted with a wrong
  reason; now tests existence, the last isDir(.git) holdout after iss-72.
  iss-2608220136597127 ahoy --json serialised a never-computed all-false guard object
  for an unmanaged folder; the field is now a pointer omitted for a folder, matching
  the Banlist sibling. iss-2608220144233519 the repolint privacy scanner read exactly
  the cap, so a file grown past it during the read was scanned as a truncated prefix
  and reported clean; now cap+1 with the grown file routed to the not-scanned path.
  iss-2608220147106835 spec/intent/memory/capture emitted bare null for an empty --json
  collection; the constructors seed them non-nil so an empty store marshals [].
  iss-2608220150151397 the mental-model brief claimed zero live bundles while four
  planned intents declare a bundle-member of spc-83 (residual of iss-123's README-only
  fix). Nitpicks: iss-2608220142154022 update.Plan had no default so an unknown target
  kind fell through to swap (now fail-closed); iss-2608220145356167 docs lint returned
  an engine fault as exit 1 not 2; iss-2608220148289898 abcd adr-N routed by a fixed
  %04d filename prefix, missing a differently-padded ADR file (spc-26 amended);
  iss-2608220149008905 the pull_request_target checkout guard external-review.yml states
  in prose was unarmed (now a workflow-walking test); plus the doc corrections
  iss-2608220150157497 (adr-22/26 links), iss-2608220150154972 (version.md install_mode),
  iss-2608220150152332 (CI classifier standdown), iss-2608220150152535 (docs
  requirements transitives). Refuted / prior art (kept out of the ledger): the abcd-cli
  name in commands/launch.md (the live internal project name, record-lint-prescribed);
  the spc-3 predecessor-store collision in intents/README (a new instance of open
  iss-239, carried there); the .abcd/work roster intake.md/attribution-rewrite omission
  (round 4's highlight-not-inventory doctrine still governs); the CI gitleaks same-origin
  checksum (adr-46's accepted trust bar, residual tracked as open iss-379); the go.mod
  vs setup-go patch coupling (GOTOOLCHAIN=auto self-corrects); the Makefile CRLF eol
  pin (GNU make strips the CR on POSIX); and the wrangler/mkdocs docs-site deploy pin
  (dashboard-side, not an in-repo mechanism). Model routing: orchestration on Claude
  Fable 5; five parallel hunters and twenty-two per-finding adversarial refuters on
  Claude Opus 5; the dual pre-merge review runs one reviewer as Claude Opus 5 and one
  as Claude Fable 5.
- 2026-08-22 — ideate: abcdev-site — verdict survives. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-08-22-ideate-abcdev-site.md
- 2026-08-22 — ideate: record-explorer-generalisation — verdict reframed. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-08-22-ideate-record-explorer-generalisation.md
- 2026-08-22 — The abcdev.app website enters the record: adr-47 (rendered from
  this repository alone; single-source rule; the adr-30 amendment "never
  bundled, rendered read-only"; the generic/specific boundary carrying the
  reframed generalisation verdict) and adr-48 (deploys per release from the
  tag via the release chain — `release: published` never fires for
  GITHUB_TOKEN-created releases — with a source-built labelled preview on
  main and dispatch-from-latest-tag emergencies). Intents itd-135 (landing
  page, umbrella), itd-136 (record explorer pages), itd-137 (relationship
  chart + genealogy), itd-138 (install.sh); itd-139 (generic explorer on a
  second instance) held in drafts pending its fixture demonstration; itd-140
  (generic/specific boundary discipline). README migration and the site verb
  family recorded as plumbing in brief 05-internals/10-site.md. The
  migration bundle stays zipped in research/abcdev-site/ because record-lint
  links_resolve rightly refuses an unpacked copy whose links resolve only at
  their Phase-1 destinations. A same-day adversarial review reframed the
  "standardised README for every managed repo" idea to offered-never-imposed
  (2026-08-22 research note; future-work seed in the ledger).
- 2026-08-22 — Website decision interview (nine questions, all ruled): retired
  ADRs render as baseline stubs, no tombstone files (revisit labels on real
  use); /record/ pages carry full bodies, the issue ledger is opted in for
  this repo, and every condensed view is countable-only — counts, dates, ids,
  titles — never prose written for the site; the featured intent is derived
  (newest entered-shipped/, id descending tie-break, no pin); the README
  migration lands as drafted, every heading entering the repo before the site
  may select it; the contributors page publishes per-model Assisted-by
  tallies under the attribution escape with a separate labelled bots-and-
  tools row (.mailmap canonicalises the pre-policy commit); analytics are
  declined outright — no scripts, no trackers, nothing that could require a
  consent banner (edge request counts suffice); the prototype's artifact URL
  stays out of the committed record; the prototype itself stays in local
  scratch until the private-banlist sources-sync anchors its short patterns
  (iss-2608220150157507); token usage becomes a local-only history-store
  datapoint with explicit-ask pricing (iss-2608220150157508). Amendment
  traceability added to itd-137 (last-touched links the record's git
  history); a foundations page (principles and disciplines, lists-and-links)
  added to itd-136; itd-140 cross-references script-first-mvp as its general
  form.
- 2026-08-22 — Bug-hunt round 7 merge gate (PR #415, resumed after the round's
  session ended between PR-open and gate). origin/main (the site-build and
  itd-112 merges) merged into bughunt-b/round-7 cleanly; baseline green on the
  merged tree; all 12 CI checks green. Dual pre-merge review: the Opus 5
  reviewer returned NO-SHIP with three remediable blockers, the Fable 5
  reviewer SHIP with the same frontmatter residual flagged non-blocking. All
  three remediated with tests watched to fail under mutation of each fix: the
  update dispatch-refusal receipt now redacts target_path at construction
  (refusalReport seam; it had shipped the raw home root beside the redacted
  detail), capture list --json now emits [] not null for empty issues/skipped
  (the CHANGELOG claim is now true), and the repolint cap+1 grown-file branch
  is pinned through a capRead seam (the prior boundary test passed against the
  pre-fix read, so nothing guarded the +1 — the round-7 "closed by
  construction" exception is retired). Residuals captured open, not widened
  into the PR: iss-2608221126066379 the frontmatter BOM tolerance now diverges
  from the sibling parsers whose comments promise byte-exact parity
  (fail-closed, comment-invariant falsified); iss-2608221126066631 the guard
  process-substitution redirection family still allows a glued blocker flag
  (pre-existing, same family as the fixed &>). The reviewer's probe-horizon
  concern was refuted in-session: isBinary self-caps at 8 KiB, so both
  oversize branches agree.
- 2026-08-22 — Interview-workstream records filed (the 2026-08-22 handover's
  items 5-9, three itd-84 hand-runs graded into the calibration note):
  itd-142 (the brief-creation interview) filed as a DRAFT with its spec
  explicitly waiting until the collaborating prototype has run once — no
  speccing ahead of that run; adr-50 + brief invariant 14 (framing traces
  never enter the record, automated reviewers never read them; local side
  lives in .abcd/.work.local/); principle widen-options-never-recommend (at
  most three construals, ceiling not target, null always available, no
  recommendation markers); itd-143 (framing chapter under 01-product/, with
  its brief-lifeboat mapping row) — itd-142 `refines` itd-90, surfaced by
  the candidate pass before minting; adr-51 + intent-template additions
  (optional Mechanism and Scope Conditions sections, enforcement an
  explicitly deferred discipline question); hold-route seed
  iss-2608220750029991 stays OPEN (RFC-vs-intent undecided, tied to the
  hold-register home question now recorded in the evidence chapter);
  glossary README names the glossary a deliberate frame surface;
  ACKNOWLEDGEMENTS credits mattpocock/skills for the glossary format and
  the frontier-questioning pattern in the change that adopts it. Open
  sign-offs carried in itd-142's Open Questions: the escalation rule,
  one-vs-three intents at the final round, a held working-principle at the
  final round.
- 2026-08-22 — Writing style guide: One canonical reference page
  (docs/reference/writing-style.md) consolidates the scattered prose rules
  (British/US split, present tense, Diátaxis) and adds the punctuation rules
  (no em dash in list items — use a colon; capital after a colon; lower case
  after a semicolon), every rule labelled machine-enforced or review
  (enforcement-claims-are-facts). The DOCUMENTATION domain in .abcd/rules.json
  points at the guide (the OPINIONS pattern — point, don't copy); the
  machine-checkable punctuation subset is staged as itd-141 (native docs-lint
  now; Vale recorded in the intent as Related/SOTA and the named preferred
  future upgrade path; adopting an open-licensed guide — Google / Microsoft /
  GitLab / errata-ai styles — explored at the intent's SOTA pass). No ADR
  unless a new rule family emerges.
- 2026-08-22 — All-dimensions bug-hunt round 7 (branch `bughunt-b/round-7`).
  Fixed 11 substantive and 4 nitpick findings across five hunt dimensions, each
  adversarially refuted before fixing and each behaviour change pinned by a
  watched-fail test. Security/code: the guard tokenizer now recognises `$'...'`
  ANSI-C and `$"..."` locale quoting (a Tier-1 blocker bypass); the hard-fail
  `local_username` redaction matcher folds case; five site-generator fixes
  (SVG prolog/comment/CDATA build-vs-check disagreement, unparsed-page gate
  masking, unscreened repository `href` scheme, record-id collision, non-ASCII
  path dates); the launch scaffold's symlink path leak and its `./internal/...`
  race leg that wedged a bare adopter; a capture `skipped[].error` absolute-path
  leak. Infra/docs/record: CODEOWNERS now covers `site-src/` and
  `docs/requirements.txt`; the shipped `abcd site check` is called shipped in the
  brief and pinned in the release-gate manifest; the site-build read set, the
  writing-style escape scope, the invariants and evidence-chapter banners, and
  several doc/string nitpicks corrected. The guard brace-expansion bypass
  (`{--force,}`) is recorded open (iss-2608221457227161) as a distinct
  expansion-not-quoting follow-up. Refuted/out of scope this round: the site.md
  "runs in CI" claim (in-flight PR #436), the iss-315 guard-warn citation, the
  install-surface transport-pin and `-q` deltas (defence-in-depth, no reachable
  attack), the record-lint job-name and prepare-this-repo template path (prior
  art iss-304/iss-87).
- 2026-08-22 — Website build rulings, from the slice reviews (the interview's
  nine rulings stand; these refine them): the featured-quote derivation skips
  a press release that is one of the two minted placeholder templates —
  mechanical template match, no other filtering, order still entered-shipped
  descending then id descending. `abcd site check` holds `/references/` to
  the composed rules even though the manifest selects no span there — a page
  must not escape the single-source rule by being unnamed — and implements
  the attribution escape as verification against the trailers git carries,
  never as an exemption. The deploy stamp contract: production passes
  `--version` bare (the renderer prepends `v`; `vv0.6.1` fails the gate) and
  previews pass `--preview` (renders `unreleased · <commit>`); `--date` is
  never injected — the CHANGELOG heading is the release date per adr-37.
  `install.md`'s lead lands with the first successful deploy, reading
  spc-40's "same change that first serves the script" as the deploy, so the
  docs never name an endpoint that 404s. Build order is fixed: `site build`
  first, `mkdocs -d site/docs` second — the purge makes the reverse loud.

- 2026-08-23 — `abcd site check` excludes `docs/` at the page walk, rather than
  teaching the parser to tolerate HTML comments on SSG-authored pages
  (iss-2608230838592910's own candidate fix, rejected). Tolerating comments
  would admit mkdocs-material output into gates whose rules were written for
  generator output — the provenance walk and banned-token gate already exempt
  the tree via `isComposedSurface`, so only the mobile and figure-label gates
  would newly run over markup this repo cannot act on. Those pages never
  entered the page list on any prior revision either (the parser has always
  refused comments), so the exclusion removes no coverage that existed; it
  states the boundary the overflow audit already draws. The cost is recorded
  honestly in the gate's own scope comment: a green report describes the pages
  abcd renders and no others, and `docs/` layout stays the SSG's own concern.
- 2026-08-22 — SOTA research protocol captured, no /abcd:research verb minted
  (session assessment; adoption is the maintainer's gate). Three research runs
  (context-window management ×2 passes, local MLX models) followed the same
  shape — parallel research pass, dated *-sota.md synthesis with per-technique
  fit challenges, lint gates, fresh-context adversarial reviewers with disjoint
  lenses (source fidelity / repo fidelity), every finding applied. Assessment:
  /abcd:ideate does NOT cover SOTA research (idea-with-verdict vs
  question-with-survey; a survey is ideate's leg-1 feeder, not its overlap);
  /abcd:consult + /abcd:ingest cover the sources-corpus side; execution stays
  host-delegated. Rejected alternative: minting a research-record validator verb
  now — fails script-first-mvp (thin genre corpus, tier vocabulary still
  moving). Protocol + revisit condition recorded in
  .abcd/development/research/notes/2026-08-22-sota-research-protocol.md.
- 2026-08-23 — SOTA-protocol proposal adversarially reviewed (two fresh-context
  reviewers, disjoint lenses, authorship stripped): verdict REFRAME — the
  no-verb deferral stands, but on script-first-mvp's actual trigger (contract
  uncertainty) alone. Withdrawn: the thin-corpus argument (the genre corpus is
  ~13 notes, and ideate's own validator was minted after three manual runs, so
  corpus size is not the criterion). Repaired: reviewer yields reclassified as
  session testimony; feeder claim downgraded to expectation (no ideate run has
  consumed a *-sota.md note yet); protocol gains step 6 — every survey note
  records its review verdicts inline, without which the revisit condition is
  undetectable. Repairs applied to the protocol note and both survey notes.
- 2026-08-23 — ideate: abcd-research-verb — verdict killed. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-08-23-ideate-abcd-research-verb.md

- 2026-08-23 — Manual-test triage rulings (maintainer): (1) spc-38's "every
  visual has a table twin" is amended — a twin is dropped wherever the visual
  already carries the same labels and numbers as text; the relationship
  chart's list twin stays (spc-39 AC 4). (2) spc-39's reduced-motion
  behaviour (list in place of the chart) is re-confirmed as ruled — not a
  defect. (3) Record sub-nav iconography uses text glyphs as ui.json
  interface strings, never image assets (keeps adr-47's picture rule
  untouched). (4) The Genealogy release-cadence list is dropped now; a
  commits-per-release-window ridgeline is its candidate replacement, designed
  within the IA-restructure intent seed (iss-2608230752354909).
- 2026-08-22 — ideate: cli-verb-taxonomy-restructure — verdict reframed. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md
- 2026-08-24 — bughunt round 5 (state #368): corrected the shipped-intents
  bucket README drift — the false "directory is empty" claim (18 shipped
  intents present; the parent index already reads populated) and the retired
  `intent-fidelity-reviewer` name (→ `intent-auditor`, spc-28; the sibling
  front-door the iss-334 sweep of the parent index missed).
  iss-2608240815458212, iss-2608240815453131. A concurrent second firing of the
  hunt-a routine took the `bughunt-a/round-5` branch and PR #480 (planned/
  bucket README) at the same time; these disjoint shipped/ findings ship on
  `bughunt-a/round-5-2` rather than clobber a live peer's branch. The persona
  nitpick in the abcdev-site prompt was dropped (that firing logged it
  considered-and-rejected as a frozen historical artefact); the one code
  candidate — `frontmatter.Fields` duplicate-key first-wins — was refuted
  (every lint status read shares the one first-wins map, first-wins is the
  documented contract, the exemption was already audited in iss-39).
- 2026-08-24 — Bug-hunt round 5 (bughunt-a): baseline green; five parallel
  hunters (core code, newer code, infrastructure, docs-vs-functionality,
  internal record consistency) on this v0.6.3 tree returned four genuine nulls
  and one confirmed substantive record finding, adversarially refuter-survived.
  Fixed iss-2608240806090544 — planned/README.md categorically claimed every
  intent carries a spec_id: spc-N link, but 20 of 43 planned intents legally
  hold spec_id: null (the lint-enforced intent_lifecycle state, the surface
  contract 05-intent.md, and adr-34's orthogonality); the subheading silently
  reintroduced the over-narrow framing adr-34 already corrected in the parent
  intents/README (resolved iss-3). Rewritten to the authoritative wording.
  drafts/README refuter-cleared (describes the promotion transition, not
  current contents) — kept out of scope. Persona-pronoun drift in a frozen
  dated implementation-prompt transcript logged as considered-and-rejected
  (historical artefact; the shipped intent itd-135 already reads they/them).
- 2026-08-26 — bughunt round 7 (state #368): hardened the new issue-resolution
  gate scripts. check-issue-resolution.sh now cd's to the repo root (RS003 no
  longer passes vacuously from a subdirectory), extracts iss-ids non-fatally (a
  non-record file last in the diff no longer silently aborts RS001/RS002 under
  set -e), and RS002 reads each changed record's frontmatter via a helper shared
  with RS003 instead of the raw diff (a body-level commit: example is no longer
  reachability-checked); check-issue-resolution-cases.sh gained the
  check-attribution-cases hermetic git-env scrub (iss-28/iss-313 sibling authored
  four days after that fix). iss-2608261040378346, iss-2608261041020040. Doc
  corrections: the 05-internals and 04-surfaces indexes no longer call the shipped
  site check gates / explorer / deploy design targets (10-site.md is the correct
  side; adr-48), brief/README nine→ten internals chapters, capture.md --shipped-in
  named as derivation-time not resolve-time validation, CONTEXT surface_coverage
  bullet names the sub-verb CLI-tree pass. iss-2608261041027043,
  iss-2608261041029596, iss-2608261041027268, iss-2608261041020419. Recorded not
  fixed (autonomous-scope limits): release.yml feeds an unvalidated tag into make
  build's -ldflags shell recipe — a shape-check-parity gap vs site.yml,
  insider-bounded, deferred because CI cannot exercise the release pipeline
  (iss-2608261041218890); the principle spec-moves-with-the-surface sits unpromoted
  while its named surface_coverage gate is armed at blocker — promotion is a
  governance act for the maintainer (iss-2608261041210476). Refuted: a resolved_by
  bare-null "regression" (the shape is unwritable and the new rejection removes a
  phantom empty-pointer); the release.yml verify job "not mirroring ci" (a
  deliberate, gate_lockstep-checked subset).
- 2026-08-26 — "Less, but better" (Rams) adopted as a guiding principle. Design
  for the STEADY state, not the migration in front of you; name a migration
  accommodation in the mechanism itself or it becomes permanent by default.
  Recorded as a principle.
- 2026-08-26 — The changelog is an INDEX of record transitions, not a curated
  release note: every record reaching a terminal folder earns its line, with no
  inclusion judgement. Rationale lives in the record; the changelog is the trace
  from record to version, which is why a record does not carry its own release.
  Consequences: `impact` keeps its version arithmetic and LOSES its silencing
  job; two families the cut ignores today (principles, ADRs) come into scope;
  bundling stays, because fewer lines carrying the same information is the
  "better" half, while dropping a record for dullness is not. `shipped_in`
  (iss-2608241612087533) is reclassified as MIGRATION-ONLY: abcd was built before
  RS001 made resolution ride the fixing commit, and a repo managed from its first
  commit should never set it. Not yet implemented — captured, so the change lands
  behind a record like everything else.

- 2026-08-26 — Bug-hunt round 8 (bughunt-a, state #368): baseline green; five
  parallel Opus 4.8 hunters across the four dimensions, each candidate
  adversarially refuted by a fresh independent subagent. 9 substantive confirmed
  and fixed, 3 refuted as prior art/design. Code: iss-2608261206490430 — the
  disembark scan fell open to a wide read on an all-ignored repository (an empty
  ls-files listing was conflated with a git failure; now parted, with a
  watched-fail test). Infra: iss-2608261206505385 — site check ran in no gate
  before a release published, so a check-only trip failed after the binaries
  shipped (now run in the Makefile site-render recipe and the ci.yml step);
  iss-2608261206507510 — RS001 unioned its leaves-open and enters-closed id sets,
  so a bare `git rm` of an open record satisfied a Resolves trailer while the
  record vanished from the ledger — corrected to require the id to ENTER a
  terminal folder (the refuter's intersect recommendation would have broken
  fresh-capture-and-resolve in the same PR; caught and verified before shipping).
  Docs/record: iss-2608261206491486 — disembark's .gitignore guarantee scoped to
  the free-text tree scan, since pack copies declared records verbatim regardless;
  iss-2608261206506035 / iss-2608261206505136 / iss-2608261206508851 — the
  shipped/ intent-auditor entry precondition, the roadmap Planned spec-link claim,
  and the surface contract's "shipped/ is empty" all corrected to the close-only
  lifecycle (20/43 planned intents legally hold spec_id null; shipped/ holds 18);
  iss-2608261206504574 — CONTRIBUTING repointed from the hand-written CHANGELOG
  flow to the derived one; iss-2608261206504710 — eleven live brief citations
  moved from the retired adr-18 to its successor adr-28. Refuted: the
  ScrubbedEnv/GIT_CONFIG_GLOBAL candidate (already adjudicated round 5), install.md
  SessionStart stderr (deferred iss-2608251011427187 / open iss-253), and the
  submodule-walk drop (aligned with the vendor-skip design; iss-133 wording
  family). nitpicks-only: no.
- 2026-08-26 — Bug-hunt round 8 (bughunt-b): baseline green after unshallowing
  the environment's clone (the shallow false-red itself became a finding). Five
  parallel hunters returned 16 substantive candidates and 7 nitpicks across the
  four dimensions; per-finding adversarial refutation confirmed 9 substantive
  and 4 fixable nitpicks and refuted 5 — the merge-queue RS001/RS002 skip
  (RS003 at the batch head plus the push-to-main range cover it, the
  attribution/external-review exemption shape), the ungated mkdocs half
  (adr-48's deploy-fail-closed design plus the preview job on every main push),
  the CODEOWNERS baseline gap (round-3 prior adjudication; external-review is
  strictly stronger), the site verb-roster counts (inside
  iss-2608231346137587's do-not-hand-fix corpus), and the site-render depth-1
  gate (shallow history changes bytes, never the exit). Fixed and resolved with
  provenance: gitutil.RepoShaped three-state fail-closed (privacy-hygiene scan,
  private banlist write gate), capture/record-lint quoted-null parity at the
  gate, the RS001 deleted-record hole, shallow-checkout refusal in both history
  gates, the pre-push roster plus its detector blind spot, scanner config
  containment via ReadGuardedInRoot, phase-audit design-target corrections,
  invariant 12's SessionEnd clause assertion, the install guide's installer
  subject, verify-as-subset framing, skew-tripwire messages, and persona
  pronouns in two planned intents. Recorded open: the memory store-lock S_IFMT
  mask (rides iss-129), the quoted-enum impact split, and a deferral-currency
  detector seed.
- 2026-08-26 — Bug-hunt round 9 (bughunt-a): baseline green after unshallowing
  the environment's clone. Five parallel hunters returned 21 candidates across
  the four dimensions; per-finding adversarial refutation confirmed 5
  substantive and 4 fixable nitpicks and refuted or deferred 12. Fixed and
  resolved with provenance: the memory-store guarded-read sweep (every reader
  outside ingest followed a committed symlink unbounded — five reproduced
  hangs), the issue-resolution gate's swallowed git probes (a dubious-ownership
  refusal read as an empty ledger, exit 0), lint-config severity validation
  with strict decoding (off-enum severities counted toward no exit code),
  the docs-lint renderer's unsanitised config-derived fields, runBounded's
  whole-buffer trim corrupting the first NUL-list entry, privacy-hygiene's
  silent skip of unreadable tracked files, the terminology page's corpus
  "ships" overclaim, the intake S4 container pin below the module toolchain,
  and the site command description's missing check-write disclosure. Recorded
  open: the scanIntentTree ENOENT-parting alignment (all claimed triggers
  proved closed upstream). Refuted with prior art: the spc-28 planned/drafts
  sweep (iss-94 convention), the verification-matrix capture-promote row
  (iss-2608231346137587 embargo), the attribution fence guard (iss-270
  wontfix; zero verdict flips across all historical PR bodies), isHexSHA
  40-hex (prior round's refutation stands), frontmatterOpen's comment latch,
  the skew-meta precedence claim, guard check's exit-2-on-disabled (specified),
  the prepare-this-repo bucket gloss, the MADR label, the itd-27 pronoun
  candidate (a cited real person is outside the persona rule), and the
  record-lint job-name mismatch (mirror is correct by construction).
  nitpicks-only: no.
- 2026-08-26 — Bug-hunt round 9 (bughunt-b): baseline green on the unshallowed
  clone. Five parallel hunters, per-finding adversarial refutation: 12
  substantive and 5 nitpick findings confirmed and fixed, 4 refuted or prior
  art — the renderReviewMD provenance gap (open GitHub #325), the receipt
  sha-width split (two different jobs, machine-derived 40-hex everywhere),
  the principles-README universal claim (iss-390's scope), and the DECISIONS
  ordering inversions (order-independent by declared union-merge design).
  Fixed with provenance: capture --blocked-by existence probe, intent verbs
  resolving spec_id by number, ScanSpecLinks failing closed, cleanLessonProse
  delegating to CleanProse, firstRootSHA bounded, RD002's one-pass DMRT
  history scan with a cases harness, record_schema's required-frontmatter
  invariant with the issueschema leaf plus the repaired live record, the
  RS001 terminal-folder wording at three sites, the plugin-command count,
  the derived-changelog Breaking clause with adr-37 amended, the data-theme
  claim, CI-account rosters, the disciplines intent-auditor sweep with
  spc-28's late-sweep amendment, itd-5's task_classes source of truth,
  itd-43's glossary reconcile, itd-24's core/epic entry, and the
  install-template eol pin. Recorded open: the unknown-property half of the
  ledger-visibility gap and itd-5's struck-tiebreak residue.

- 2026-08-28 — Punctuation enforcement stays mechanical-only (adr-54): the
  em-dash core shipped as the punctuation/em-dash-in-list-item banned token,
  corpus evidence (0 TP, unclosable FP classes) rules the colon/semicolon
  casing lints unbuildable under the warn-then-promote policy, and itd-141
  is superseded at its planning interview. Same session: audience-by-
  placement ratified (adr-53) with the Audience section landing in the
  writing-style guide; both decisions rest on the 2026-08-28 docs-audience
  research note.
- 2026-08-27: `.abcd/work/attribution-rewrite-2026-08-06/` promoted to `.abcd/development/research/data/` (iss-2608271707587825) — the tables are historical and complete, durable-record shape; this supersedes the highlight-not-inventory adjudication for that path (intake.md's clearance stands unchanged).
- 2026-08-28 — Transcript deep-secret coverage lands as an OPT-IN gitleaks adapter, off by default (iss-96). The native scanner stays the always-on default; a repo arms deeper coverage by dropping an enabled `.abcd/config/gitleaks.json`, whereupon `internal/adapter/gitleaks` shells out to a gitleaks binary (PATH or configured path) over the transcript and its findings AUGMENT native redaction through the same `scanner.Redact` discipline. This realises option (c) of the iss-96 grill (labelled+delimited+high-entropy reach the prefix-anchored native set misses) as the host-delegated/opt-in-adapter pattern (adr: LLM/CLI oracles are opt-in adapters), NOT an always-on entropy detector (option (a), rejected for its redaction false-positive cost on prose) and NOT a hard gitleaks dependency in the default build. Fail-closed, never a silent no-op: an armed-but-absent binary makes capture refuse the write with `gitleaks configured but not found`. Default-off invokes nothing — no lookup, no process, no cost — so the unopted path is byte-for-byte unchanged. iss-96's residue beyond (c)'s reach (bare no-keyname values, sub-entropy-floor tokens) stays open.
- 2026-08-28 — Release-pipeline gates: two decisions land together. (a) The
  semantic `receipt_gate` moves to the safe side of the tag (adr-52,
  Alternative 1 accepted): its arming — content-commit resolution plus the
  fail-closed `record-lint --release-gate` — now runs in release.yml's `verify`
  job, alongside the deterministic gates, gated off the rehearsal path
  (`github.event_name != 'workflow_dispatch'`) so a receipt-less rehearsal never
  refuses. The `release` job keeps only provenance signing (re-derive + attest),
  and the tag itself is untouched — only the gate moved (iss-2608231226347380).
  Caveat recorded for a real-release run: auto-release.yml still mints the tag in
  its `tag` job before it calls release.yml, so fully eliminating a consumed
  version on the auto-release path also needs auto-release's `tag` gated on
  verify — out of this change's scope, maintainer-verified separately. (b) The
  reviewed content commit is now derived from the RECEIPTS DIRECTORY
  (`record-lint --derive-content-sha`, reading `.abcd/work/reviews/<sha>/` of the
  released tree) rather than `HEAD^2^`/`HEAD^` ancestry, because a batched
  merge-queue push makes `github.sha` the batch tip and `HEAD^2^` can resolve an
  unrelated PR's commit past the ancestry guard (iss-355). New Go derivation
  (`lint.DeriveReleaseContentSha` + `gitutil.IsAncestor`) with tests covering the
  merge, batched-queue, nearest-ancestor, stray/off-lineage, and fail-closed
  cases; release.yml and the scaffold template (self-scaffold parity held) and
  the rehearsal (now a batched-queue regression) all use it.
- 2026-08-29 — v0.6.8 semantic gates: two maintainer judgements recorded so they are not re-litigated. (a) The iss35 cross-check was RE-RUN against the final content commit (00f1509d) rather than accepting an earlier full-tier run whose tree was amended in flight: a receipt attests one sha and the earlier run could not honestly name one; at ~13 minutes wall and ~1.7M tokens a re-run is cheaper than the ambiguity. (b) PROMOTE with a disposition on every finding (60, all independently re-verified, 0 refuted) rather than HOLD: the findings are the design brief lagging the shipped surface (iss-2608231346137587, whose corpus rule forbids hand-fixing ahead of the mining intent), not something the release introduced or ships, and a HOLD would block a correct changelog behind debt the record already owns. Also noted: the local_username payload rule collides with product vocabulary on a machine whose account name is a common word (iss-2608291444328326); the release push used a HOME symlink alias so the pre-push preflight could run in full, never --no-verify, and CI re-ran every gate under its own account.
- 2026-08-29 — The gitleaks adapter admits a binary under an allow-shape, not a deny-list (GHSA-fg9r-3f8g-89m6, iss-2608291807456485): absolute path, resolved OUTSIDE the repository both lexically and after symlink resolution, regular, executable — applied identically to a committed `path` and to the PATH lookup result. Refusal is `ErrConfiguredPathRefused`, as loud as not-found, and a refused configured path never falls back to PATH. PATH itself stays trusted as the operator's environment (abcd already resolves git, gh and grep from it; a hostile checkout cannot set it); what the admission rule closes is a PATH entry or config that reaches INTO the checkout. Rejected: a fixed safe search path for the fallback (would break Homebrew/asdf/custom installs for no gain against an operator-level PATH compromise) and executing with an isolated env (the env is the operator's, like PATH — gitleaks does honour GITLEAKS_CONFIG, but a checkout cannot set it; cwd is pinned to the private temp dir instead). Containment routes through `fsutil.PathWithin` with the case-folding predicate AND an `os.SameFile` ancestor walk, because a byte-exact compare admitted a case-variant or NFD respelling of the root on APFS.
- 2026-08-29 — v0.6.9 security pass, three process decisions recorded so they are not re-litigated. (a) The four GitHub bug-hunt issues (#485–#488) were verified before any capture and found already fixed on main by the 2026-08-27 security cut; they were closed against their fixing commits rather than re-captured, because a duplicate record beside a resolved one is noise the ledger's status signal cannot carry. (b) The installer's proxy/CA-bundle lockdown keeps no environment-variable escape hatch: the lockdown is the GHSA-x4v8 fix and an env opt-out reopens the vector; the availability cost (NixOS, corporate proxies) is captured as an open record for a product decision, and only the failure message was improved. (c) Every fix branch received two independent adversarial reviews (security + code) from reviewers that did not author the change, repeated until neither returned FIX-FIRST, and the assembled multi-branch diff received the same two again before the PR; the fix PR merges with a merge commit, never squash or rebase, so every resolved_by.commit stamp stays reachable (RS003).
- 2026-08-30 — Every release object older than v0.6.9 is deleted from the forge (v0.1.0, v0.2.0, v0.3.0, v0.4.2, v0.5.1, v0.6.7, v0.6.8; the v0.6.0–v0.6.6 tags never had one), and every tag is kept. The three v0.6.9 advisories mark `<= 0.6.8` vulnerable, and a release object is the only thing that makes a vulnerable binary downloadable by version: the installer, the bootstrap hook and `abcd update` all resolve latest, so removing old assets affects only a deliberate pin, which is the point. Tags are the immutable audit trail the release gate relies on (adr-52), and the build-provenance and receipt attestations that named those releases stay in the attestation store even though the artefacts they attest are gone; the deletion is recorded here because no earlier release removed a predecessor and the decision log carried no practice for it. Reversing it means re-cutting a release from the surviving tag, never re-uploading old assets by hand.
- 2026-08-28 — Cold-reading workstream rulings adopted (facilitator; committed in design discussion, disclosed as a pre-tooling lapse): (1) the construal-admissibility extension of adr-50 files first — the construal as it presently stands is committed record and readable by automated readers, its revision history is not (adr-55 carries the full rules); the construal is sited as a brief section under 01-product/. (2) The reading record (type name ruled at (17)) is a distinct record type in the issue tier with a separate disposition record and its own disposition vocabulary — the five-state slate with availability by position, per (19); the existing issue states are not reused for it; status is the presence of the keyed disposition, never folder membership. (3) One closing reading run is scheduled over the same object set after dispositions, with interpretations fixed in advance: silence is weak evidence of settlement; an accepted-and-acted detection returning is a finding; a rejected-with-purpose detection returning bears on whether the purpose resolved the tension. Amnesia is proven by a repository eval, never by a case run. A clean opening run is recorded as a run with an empty item set; the cycle proceeds and no further reading is commissioned. (4) The supply-regime check is built now: a regime field on the output contract, validated at ingest; enforcement is the default, and degradation to record-and-flag happens only on observed noise, recorded. (5) Reading invocation carries no free text: position and target state only. (6) The lapse log is a value in capture's validated category list, not the source enum. (7) Elicitation: the articulation round produces one intent, the exit noting candidates; the escalation rule is adopted (a defaults question that turns out conjectural escalates to the options regime); a held working principle permits articulation to proceed with the mechanism recorded as an explicit nullity, flagged in the readiness summary. (8) From the readings design, agreed as proposed: the reading record carries a position-typed item body (one record type, four bodies — registrative, generative, explicative, evaluative); the selection criteria are a recorded discipline with the committed six-criterion slate, never supplied at invocation; the comparative reading's candidate set is the widening reading's pre-admission output, with admission following the comparative characterisation; the no-input-is-authoritative condition enters the blindness core (no document passed to a reading is the fixed side of any comparison), and the widening/entailment asymmetry over draft and planned intents is stated explicitly in the assembler's include list — both confirmed at the facilitator's review, 2026-08-28; the disposition-side recurs citation is adopted as the recorded form of warm recurrence recognition. (9) Ruled 2026-08-28 (revised the same day): the manifest is committed to the durable tier at `.abcd/development/readings/<run-id>/`, alongside the run record — a new record family; lifecycle selects the tier, and commit reference plus per-item hashes make the assembly re-runnable and diffable. (10) Adopted 2026-08-28: brief invariant 15 — reading contexts and the ledger never meet, the scribe is not a transcript consumer, and the session-transcript store is reached through one door by an enumerated allow list (custodian writes and reads; lifeboat session-hunting consult-only; session-separation metadata; opt-in memory curation); a new consumer is an invariant change, never merely a code path, and only the history core package touches the store's path, held by a boundary test. (11) Adopted 2026-08-28: the escalation-logging addition — every escalation in the elicitation interview is logged as data, which question and on what grounds. (12) Ruled 2026-08-28: run metadata's instrument identity comprises the model identity, the definition's content hash, and the assembler version. (13) Ruled 2026-08-28: kinds bind as stamped — the instrument trio as one bundle with a shared spec, the rest standalone, the two disciplines as disciplines. (14) Ratified 2026-08-28: the 2026-08-27 grill-pass resolutions, as implemented in the drafts. (15) Ruled 2026-08-28: warm transcript consultation proceeds with the redaction residue disclosed; transcript content is not quoted into committed records until the secret-scan work closes the residue. (16) Ruled 2026-08-28: the construal extension refines adr-50 — adr-50 stays accepted, the extension states the refinement, and adr-50's related_adrs gains the minted id at filing (adr-55). (17) Ruled 2026-08-28: the record type is the reading record (folder `issues/readings/<run-id>/`); "detection" stays the name of the Step-6 instrument and its registrative body. (18) Ruled 2026-08-28: pattern-named is an envelope field; the two assembler rules (no include names a directory containing a record family; a reading's object excludes what it exists to change) are adopted; the read-block eval also asserts the instrument's own prior outputs never reach a reading; where the widening reading returns fewer than two configurations, the comparative reading is not exercised and the outcome is recorded as such. (19) Ruled 2026-08-28 (R7): five disposition states with availability by position — admission is `accepted` (position lives on the envelope); declining a widening proposal is `declined`, never `rejected`; the grounds field is `disposition_grounds`, required on every state except `held`; the disposition record validates its state against the envelope's position. (20) Authorised 2026-08-28 (facilitator): the build session runs the intent → spec → ship ceremony autonomously for this workstream's filings — adoption, spec creation and approval, readiness, kind binding, shipped moves, and audit ingest — with every act logged; PR merges, reversal flags, new dependencies, and records outside the workstream remain human-gated. The autonomy is deliberate and is itself part of what the cycle observes; ownership of every act rests with the committing identity, per the provenance rules.
- 2026-08-28 — Ledger mechanism and ledger content are distinct: the mechanism is engineering (schemas, commands, gates, evals), reviewable and reworkable; the content is the recorded reasoning, populated only by actual use and never reconstructed. A defect in the mechanism is fixed; a defect in the content is disclosed.
- 2026-08-28 — Grounds recording extends the ADR family rather than duplicating it: the family holds decision-granularity grounds (Alternatives Considered); the ledger adds conjecture granularity as a grounds argument on the selection surfaces, one canonical primitive, no parallel store.
- 2026-08-29 — Roles for the cold-reading cycle (facilitator): the session user is the facilitator; the co-author is the product thinker, who dictates the construal. Both are named by role only, never by name, in every record and commit. The construal is the ledger's construal for this cycle, not abcd-wide. Frame origination is reserved to the warm side: the orchestrator formats and may widen options, never recommends a construal or rules whether one fits.
- 2026-08-30 — Cycle 1 restarts on v0.6.9 (facilitator): baseline v0.6.9-4-g976575f9; the orchestrator is Fable, research and implementation sub-agents are Opus 5 in fresh contexts; the merge target is the integration branch `experiment/cold-reading` in a worktree outside the checkout, never main — the final PR to main is the facilitator's to test and merge. Widening decision (20): the orchestrator also runs the two adversarial reviews and the dependency-ordered merges onto the integration branch autonomously; the only stops are gate weakening, a security BLOCK genuinely unresolvable, and a new Go dependency (allowed only if absolutely necessary, logged). Improvisation inside the rules is allowed and logged, never quiet.
- 2026-08-30 — Ratified (facilitator): the not-yet-real marker is a passage-level token — a blockquote line holding exactly `**Status: NOT YET REAL.**`, a blank line, then the statement as the first paragraph — so a bold opening sentence can never be mistaken for the token. The ledger's construal passage carries it.
- 2026-08-30 — The ledger's construal is filed (pre-tooling entry 01 resolves to this pointer): `.abcd/development/brief/01-product/06-framing.md`, section "Construal". Production mode: dictated by the product thinker, formatted by the orchestrator against docs/reference/writing-style.md with zero text operations (the one rule it meets, capital after a colon, is waived for the lowercase-by-design name), confirmed per item by the facilitator before filing. The 150-word outward-facing form is held on the local ledger side, unfiled, until a home is assigned. Ambiguity flag from the two adversarial reviews carried, not ruled: "ledger" also names the issue ledger; the section heading names whose construal it is, and a glossary term remains an open option.
- 2026-08-30 — Confirmed (facilitator): the six-criterion selection slate (plausibility, generativity, cost, risk, learning value, practical importance) files as the selection-criteria discipline as committed 2026-08-28.
- 2026-08-30 — Deferred (facilitator): whether `held` is available at the widening position or a deferred configuration belongs to the selection vocabulary's `deferred`; revisit point is the first widening run's dispositions, a later cycle. Non-gating: the per-position availability row stays unfilled until then.
- 2026-08-30 — Framing section sited as `01-product/06-framing.md`, appended rather than inserted so existing chapter links hold (facilitator). The brief↔lifeboat mapping row and the `00-meta.md` regeneration are deferred to itd-143's build, carried from session 1: the filing PR carries no code beyond the lapse enum line.
- 2026-08-30 — Deferred with revisit point at Phase 2 planning: the workstream's roadmap phase home (options presented: Phase 3; Phase 2; a new Phase 7; a split) and the [HAND] owners for the scribe-protocol rehearsal and the step-2 admission records (facilitator, product thinker, or either-first).
- 2026-08-30 — Ruled at the checkpoint (facilitator): the workstream's roadmap phase home is a new Phase 7, the next free number; the phase document files with the Phase 1 records. This supersedes the deferral recorded above.
- 2026-08-30 — The brief-lifeboat mapping row for the framing section lands in the Phase 1 filing after all (orchestrator's improvisation, logged for the facilitator): `TestEveryBriefSectionHasARow` refuses a brief section without a row, an armed test cannot stay red, and the construal must file first; the row is its own commit, droppable if the facilitator prefers to hold it for itd-143's build. This supersedes the deferral recorded above.
- 2026-08-30 — Phase 1 adversarial review (ruthless-reviewer, fresh context) findings captured as iss-2608300114352100 (a first name disclosed through a homonym in the session-1 lapse record; the wording is replaced, and the pushed branch's history still carries it until the PR is squash-merged), iss-2608300114357633 (this log contradicting the branch, remedied by the two lines above), and iss-2608300114359543 (record consistency: one role name for the 2026-08-28 rulings, the ratified marker form in itd-143, the framing section statement-first, no adoption-date marker on invariant 15, the phase count, and "Phase 2 planning" reworded so it cannot read as roadmap Phase 2). All three resolved in the same change.
- 2026-08-30 — itd-179 round-3: the grounds substance floor is measured in letter-runs (`MinTextWords = 3`) alongside the character count, not instead of it. Rejected: replacing the character floor, which would admit "a b c"; raising the word count further, which the corpus (shortest real entry, 32 words) does not need and which would start refusing terse honest reasoning.
- 2026-08-30 — itd-179 round-3: the double-quoted frontmatter scalar decoder's canonical home is `internal/core/frontmatter` (`Unquote`), joining `IsNull` there, rather than exporting `capture.unquote`. Rejected: lint importing capture, which is an import cycle — capture's own tests import lint.
- 2026-08-30 — itd-179 round-4: the grounds floor's unit is a WORD where the script separates words and a LETTER where it does not (`textUnits` + a named, deliberately partial `scriptioContinua` set: Han, Hiragana, Katakana, Thai, Lao, Khmer, Myanmar, Tibetan, Javanese), and `MinTextWords = 3` is unchanged. The letter test is load-bearing, not tidy: a Unicode SCRIPT table carries its script's digits, punctuation and combining marks too, so counting table members would have made twenty Thai digits — or twenty Tibetan tsheg, which are twenty dots — a twenty-unit text and reopened iss-2608301206034359 once per script. Rejected: the review-converged shape of "contains a scriptio-continua rune → skip the run count and apply the rune floor", because `supercalifragilistic 中` then clears a 20-rune floor on one word; the shipped counting refuses it at two units, though the closure is one ideograph deep (`supercalifragilistic 中中` passes, as `supercalifragilistic 中 中` already did before this change). Rejected: Hangul in the set — Korean is spaced, and counting its syllables would let two Korean words clear a three-word floor. Limit stated in the doc comment: a scriptio-continua script not named in the set still counts as one word and is still refused, and the set is where that is fixed, one script at a time.
- 2026-08-30 — itd-179 round-4: the grounds floor's character half counts LETTERS (`MinTextLetters = 20`, `isTextLetter`) and is asked AFTER the unit half. Both halves stay, as the round-3 entry above requires — what changes is the unit each is measured in and the order they are asked in. A rune count is answered by whatever occupies a rune: `a b c` plus seventeen dots cleared twenty "characters" carrying three letters of reasoning, and a text of Hangul fillers cleared the whole floor while rendering as nothing end to end. Rejected: asking the letter count first, which makes the scriptio-continua refusal unreachable — every unit that refusal speaks for is a single letter, so a text with twenty letters has twenty units and can never be below the three-unit floor, and one with fewer would be refused for its letters instead. Rejected: filtering only the zero-width format characters, which leaves the filler class open because U+115F, U+1160, U+3164 and U+FFA0 are category Lo; `unicode.Other_Default_Ignorable_Code_Point` names exactly those four letters and nothing else. Corpus: the 25 recorded grounds texts in the tree are refused by neither the old floor nor the new one.
- 2026-08-30 — itd-179 round-5: the ISSUE ledger's grounds become APPEND-ONLY, as a `## Grounds` body section mirroring the intent half, and the frontmatter `grounds:` scalar is retired (16 committed records migrated in the same change). A scalar is SET, so a resolve or wontfix silently destroyed the conjecture a promote recorded — on the ledger's mainline sequence, since fourteen resolved records carry `promoted_to` — and the result still reported success (iss-2608301657354776). Rejected: refusing a transition that would overwrite a non-empty grounds, as the record proposed; promote, resolve AND wontfix all REQUIRE grounds, so the refusal would make a promoted issue impossible to resolve. Rejected: last-write-wins declared as a chosen tradeoff, because `intent/grounds.go` already argues the opposite for the same data — the earlier conjecture is precisely what a later reader checks the outcome against. One-canonical-primitive: the record form (heading, section reader, append, readability check) moved to `core/grounds`, and the generic markdown-body machinery under it to a new leaf `core/mdrecord`, so both record families share ONE definition rather than holding two. Rejected: leaving the form in `core/intent` and having capture call it, which is one implementation but puts an issue-record body parser behind the intent package's door. The retired key is TOLERATED in `issueschema.Known`, not refused: a refusal makes capture SKIP the record, and a record invisible to every surface while it still sits in the ledger is worse than a value nothing reads — the record lint's `record_schema` blocks the misplacement and names the section instead.
- 2026-08-30 — itd-179 round-5: a RELOCATION is not a BACKFILL, and the forward-only rule forbids only the second. Moving a pre-tooling `## Grounds (pursued)` section from a spec onto its intent (commit 09f7d91a, thirteen specs) put text authored at the moment of pursuit where it belongs; nothing was reconstructed, which is the whole of what "never backfilled" refuses. Three shipped intents therefore carry a `## Grounds` section — itd-177, itd-182, itd-188 — and the surface claim now says so instead of asserting the corpus carries none (iss-2608301657357989). The enforcement is unchanged and stays maximally strict: `RecordGrounds` refuses a shipped or superseded record whatever the provenance of the text, because nothing in it can tell relocated text from invented text, so the relocated state is deliberately not reachable through `abcd intent ready --grounds`. Rejected: weakening the refusal to admit a `--relocate` escape, which would make the distinction a caller's assertion rather than a reviewable fact about a commit.
- 2026-08-30 — itd-179 round-5, correction to the round-4 letter-floor entry above: `unicode.Other_Default_Ignorable_Code_Point` does NOT "name exactly those four letters and nothing else", and the remainder is not "format characters and variation selectors". Measured over the whole code space: the table holds 3776 code points, of which exactly seven are assigned — the four Hangul filler letters (Lo) and three non-spacing marks (Mn) — and the other 3769 are unassigned; it carries zero format characters and zero variation selectors, which reach `Default_Ignorable_Code_Point` through the other contributors to that derived property. The conclusion the round-4 entry drew is unchanged and was verified exhaustively — the only LETTERS the table can exclude are those four — but the description of the rest of it was wrong, so it is corrected here rather than left to be re-derived (iss-2608301657350399).
- 2026-08-30 — itd-179 delta close: the grounds APPEND and its read-back run over the record BODY, spliced back onto the frontmatter, rather than over the whole file. `frontmatter.Split` is the one lossless answer to where the block stops and `grounds.Body` is what both halves ask, so the writer judges the bytes the reader consults and the writer/reader disagreement is unrepresentable rather than banned one spelling at a time (iss-2608301805069999). The intent half's reader moves to the same scope, where the whole-file reading would have found an empty pseudo-section. Rejected: refusing a frontmatter `# Grounds` comment, which bans the one spelling that was reported and leaves the scope split that produced it.
- 2026-08-30 — itd-179 delta close: body scope does NOT close consequence A, and the record says so. An unclosed opener in the BODY still masks the body, so an appended entry still cannot read back and every triage route still refuses — verified by a test that stayed red at the body-scope commit and only passed at the next one. What makes that survivable is the other two halves: the refusal names the opener's line and construct instead of the grounds operand, and promote dry-runs the append against the pre-flight bytes so a deterministic refusal mints nothing (iss-2608301803423101). Rejected: refusing every record carrying an unclosed opener, which would refuse writes that succeed — a `## Grounds` section ABOVE the opener takes the entry perfectly well. Rejected: a file-relative line number in the refusal, because the triage verbs append after setting their note field and the number would name a line the unwritten record on disk does not have; the number is body-relative and the opener's text is quoted beside it.
- 2026-08-30 — itd-179 delta close: `mdrecord.findBacktickRun`'s superlinear shape stays, on a measurement rather than a shrug. A line of 120 distinct-length backtick runs costs ~200us and an ordinary record line ~76ns; precomputing the runs into a slice takes the bad line to ~7us and the ordinary line to ~115ns with one allocation per line, and stepping between runs with `strings.IndexByte` takes the bad line to ~243us. Both trade the case that always happens for a case no record body has. Rejected: landing the precomputed-run version, which reads as an optimisation and is a regression on this repo's workload. The two benchmarks and the numbers live in the package so the trade is not re-litigated.
- 2026-08-30 — Correction to the itd-179 delta-close entry above, which is
  substantively right and wrong in two words. It says "consequence A" — a label
  that exists nowhere in the repository, the orchestrator's briefing shorthand
  leaked into the durable record; the record it points at enumerates its
  consequences as 1, 2, 3. And it says "the record says so", which the record
  does not: its resolution field names the diagnosis and mint-ordering halves
  only, so consequence 1 reads as fixed to anyone arriving fresh. The residual —
  an unclosed comment or fence in an issue BODY still locking the record out of
  all three triage verbs — now has its own open marker at iss-2608301908270888,
  which is where a later session should find it rather than by following a
  pointer to a record that does not mention it. Everything else in that entry
  stands, including both rejected alternatives.
- 2026-08-30 — The scribe-protocol rehearsal is owned by the facilitator and runs in this cycle, scoped to the process alone: whether the protocol works, not whether it matches the product thinker's intentions. The product thinker is unavailable, so the fidelity half cannot be exercised and is deliberately out of scope for this rehearsal rather than silently skipped; a rehearsal that tested only what one participant could judge would otherwise be reported as if it had tested both. This is the remaining half of the Phase 0 item (d) whose phase home was ruled earlier; the step-2 admission-record owner stays unnamed, and the roadmap now records why nothing is owed there before a widening reading runs.
- 2026-08-30 — The disposition vocabulary ships four states and the fifth stays unnamed, which is spc-58's decision restated rather than a new one: the shipped enum is `accepted`, `rejected`, `declined`, `held`, the schema is data so a fifth is one line the day it is named, and naming it is a vocabulary judgement belonging to the researcher rather than to a build. `admitted` is not the missing fifth — ruling (19) already folds admission into `accepted` because position lives on the envelope, so introducing it would reverse that ruling rather than complete it. The `recurs` citation is likewise ruled out by spc-58 as never a state, and nothing meaning "already covered" exists at any position because an undispositioned item is reported as outstanding instead. The open candidate the record leaves is the selection vocabulary's `deferred`, which makes this the same question as whether `held` is available at the widening position — already deferred to the first widening run's dispositions, and settled there or not at all.
- 2026-08-30 — The two cold-reading evals get their own always-run CI job rather than riding the `smoke` job that stands down on record-only pull requests. The stand-down is anti-correlated with the risk these evals exist to catch: they read the live record tree, and `docs/`, `.abcd/development/`, `.abcd/work/` and the root prose files are both the classifier's inert allowlist and the material the assembler includes, so the diff most able to introduce warm content is exactly the diff that skips the check. A stood-down job still reports its context green, so the current shape manufactures a green for work that did not happen, which the loud-staging principle refuses. The remedy follows the reasoning `ci.yml` already documents for the ubuntu unit lane, which never stands down precisely because its tests read the live tree: a small job with no `inert` condition running these two evals alone, the rest of the smoke harness unchanged, the workflow edit landing with itd-186/187 so it is reviewed beside the evals it serves. Ruled before the build rather than after, because spc-64 and spc-65 both promised no workflow edit would be needed and that promise is what changes.
- 2026-08-30 — The in-place mutation hazard graduates from a session habit to the committed record as itd-193, a discipline: a verifier works on a copy, and whoever acts on a tree proves it clean immediately before the act rather than inheriting an earlier check. Occasioned by a reviewer running its mutation matrix against a live build worktree while a peer review, correctly, reported the modification as untouchable peer work it could not distinguish from real work. Routed by the four-piece table: the stance and its gate to a discipline record; the operational line to the AGENTS.md concurrent-sessions section beside the peer-work rule it complements; no capability, because the documented protocol is the MVP at this rung (script-first-mvp); no ADR, because it governs method rather than architecture. Adjacent to the one-writer-per-file principle and deliberately not folded into it: that principle governs committed records across merge windows and puts per-worktree files out of scope, while this governs the working tree inside a single window. REACH IS THIS REPOSITORY ONLY and the record says so — a rule reaches every abcd-managed repo through a bundled default domain compiled into the binary, which is a code change and its own intent; rung 2 (a recall-injected domain rule) and rung 3 (a mechanical clean-tree assertion) are named in the record and neither is claimed.
- 2026-08-31 — Three rulings from the two fidelity audits. (1) The grounds
  obligation gets a FORWARD-ONLY gate: a terminal-folder record must carry an
  entry if it was created after the gate is armed, and the cutover is the ARMING
  COMMIT rather than a date, because a date in prose is the fact itd-195 says
  not to state. Precedent is this cycle's own forward-only intent-side gate, and
  the reason is that the ~689 records that would otherwise be refused were made
  by the repo's ordinary working practice. Rejected: no gate at all (honest but
  leaves three-quarters of new records escaping a claim the press release makes
  universal) and forcing the verb to be the only door (universal, but 689
  migrations and a fight with daily practice). (2) The four remaining intents'
  ACCEPTANCE CRITERIA are reviewed BEFORE building, and any criterion conjoining
  a structural half a machine can enforce with a judged half it cannot is split
  — the remedy itd-183's audit proposed for its own ac-5. Five criteria have now
  been audited and none reached MET; two are unmeetable by nature, and MWC stops
  carrying information once it is the default. (3) The grounds body-lockout is
  lowered to minor and the hand edit accepted as the repair: the guard is
  correct, the exit exists in a text editor, exposure is latent, and validating
  at capture time would contradict the adopted rule that capture stays
  frictionless.
- 2026-08-31 — Phase 0 of the last four intents executed: the ruled criteria
  split, applied to itd-184, itd-185, itd-186 and itd-187 before any builder
  started, with the four specs' mapping tables rewritten in the same change so
  intent and spec cannot drift. Splits: itd-184's regime criterion separated
  the value stated in the definition from the absence of any operator surface
  that sets it, the latter given its own observable and an executable
  enumeration rather than a written list; itd-185's malformed-output criterion
  separated refusal-with-no-partial-write from the crash window between staging
  and the commit marker, and its registrative and explicative criteria each
  separated the reserved-name half the schema holds absolutely from the
  semantic half bounded by the signature registry; itd-187's second criterion
  was replaced by three, because its Given — a nondeterminism introduced into
  the shipped assembler — is unestablishable by any artefact, an eval being
  forbidden to patch the code under test, and the remainder is disclosed as a
  recorded hand-run. Every residue is stated in the record beside the criterion
  it qualifies, so a later audit reads the bound rather than rediscovering it.
  BEYOND THE LETTER OF THE RULING, and cheap to reverse because each sits
  contiguously at the end of its list: three criteria added to itd-185 (named
  provenance at every regime, the payload regime that disagrees with the
  definition, and verb-minted item ids) and two to itd-186 (the oracle's
  structural independence from the assembler, and the anti-vacuity guard that
  every declared sentinel is planted). Each covers a promise already in the
  intent's scope and already carrying a named test in its spec, and leaving a
  load-bearing promise unmeasured is the same failure as an unmeetable
  criterion, taken from the other side. Item-level refusal granularity in
  itd-185 is the one such gap left unclosed, and is queued rather than added.
  Rejected: splitting at build time (the criterion is rewritten in minutes
  before the work and re-litigated in an audit after it) and leaving the
  semantic halves conjoined under MET_WITH_CONCERNS (which stops carrying
  information once it is the default outcome, and on the present trajectory it
  is).
- 2026-08-31 — The wave plan is re-cut, because itd-184 and itd-185 are not
  independent: spc-63's regime gate resolves a run's position to
  agents/cold-reading-<position>.md and reads its regime key, and recomputes
  that file's hash for instrument identity, so itd-185 cannot be built until
  itd-184's four definition files and their locator exist. The real graph is
  two file-disjoint lanes — itd-184 then itd-185 over internal/core/reading and
  agents/, itd-186 then itd-187 over evals/, the Makefile and ci.yml — so the
  ceiling of two concurrent agents is held by running one lane in each rather
  than by pairing the two deepest. itd-186 and itd-187 need no assembler change:
  spc-64's four demands on spc-61's interface (dry-run, an operator-named output
  directory, separate bundle and manifest artefacts, position and target flags,
  non-zero exit) are all already shipped. Rejected: building itd-185 first
  against fixture definitions, which would ship the keystone verb unwired to the
  definitions that are its source of truth.
- 2026-08-31 — itd-185's payload item is FLAT (`pattern` plus the position's own
  body fields, read from `issueschema.ReadingBodyFields`), not spc-63's drafted
  `{pattern_named, body}` with shortened names. Ground: the four shipped
  definitions instruct the flat shape and the record schema's field names, so
  the drafted table would have refused every output the shipped instrument can
  produce and would have put a rename between the payload and the record for no
  gain. Rejected: keeping the spec's table and translating at ingest.
- 2026-08-31 — The ingest stage holds a write-aside MARKER, not the rendered
  reading records; `capture.IngestReading` writes them against the real ledger.
  Ground: that writer mints every item id under the ledger lock precisely so its
  collision probe sees the tree it is about to write into, and staging into a
  second issues root would move the probe off the real ledger. The orphan sweep
  rolls the run back instead, bounded by the `rdi-N.md` filename grammar.
  Rejected: staging rendered records per spc-63's first draft.
- 2026-08-31 — The position/regime disagreement is refused in
  `reading.LoadDefinition`, not cross-checked in the ingest verb. Ground: the
  locator is the one thing that claims to resolve a position to its regime, and
  a resolver returning a confidently wrong answer would hand the supply-regime
  gate the wrong licence silently. Rejected: a cross-check inside the ingest
  path, which would be a second table of one fact and leave the primitive broken
  for the status render.
- 2026-08-31 — itd-185's record-size limit is DECIDED in capture.IngestReading on
  the assembled bytes, not estimated from the payload; reading's recordBytes stays
  as a cheap early filter that buys item-level granularity. Ground: two attempts
  to decide it upstream each modelled one lengthening step (the escaper) and
  missed the next (the redactor, which exceeds 2x and scales with body length),
  and a record past the limit is durable and permanently undispositionable.
  Rejected: a third coefficient.
- 2026-08-31 — itd-185's ingest containment is layered and the layers are
  mutually redundant: writeJSONIn's contained write is currently unreachable
  because refuseARerun probes the same path through the root first, and nothing
  asserts that ordering. Recorded rather than fixed, because the property is
  proved by mutating containment as a whole; a future edit that moves or drops
  the rerun probe must re-check it. Rejected: a per-layer test, which would pin
  an ordering the design does not promise.
- 2026-08-31 — itd-185 folds invisible and compatibility-equivalent runes
  (Unicode spaces, Cf + Other_Default_Ignorable_Code_Point + Variation_Selector,
  NFKC) before signature matching and before every blankness rule, and DISCLOSES
  the script-confusable class as open. Ground: the registry's own phrasing with a
  byte substituted is an evasion of the gate, not the calibration residue; a
  confusables table is a new dependency and the maintainer's call. Rejected:
  filing the invisible-rune class under the disclosed residue.
- 2026-08-31 — The cold-reading evals' CI job is NOT added to the branch
  ruleset's required-check list yet, and the sequencing is the decision rather
  than the delay. The job and its make target exist only on
  `experiment/cold-reading`; neither is on `main`, which was checked rather than
  assumed. A required status check blocks a pull request until that context
  reports, and a context no workflow on the base branch produces never reports —
  so arming `cold-reading-evals` on the active `main protection` ruleset before
  the workflow reaches `main` would wedge every merge to `main`, including the
  very pull request that would deliver the workflow. The order is therefore:
  merge the workstream to `main` first, confirm the job runs and reports its
  context on a real pull request, and only then add it to the live ruleset and
  to the committed mirror at `.abcd/work/rulesets/main-protection.json` in one
  change. Rejected: updating the mirror now to record the intent, because a
  mirror asserting a required check that is not required is a false record of
  exactly the kind this phase spent the day removing, and it would read as
  done. Until the arming lands, itd-186's always-run lane runs on every pull
  request and gates none of them, which is a green for work that did happen but
  binds nothing — the weaker half of the failure spc-64 was written against.
  Tracked at iss-2608311051046981.
- 2026-08-31 — A capability is rehearsed end to end before it ships, at three
  rungs, adopted as itd-196 at the documented-protocol rung on the maintainer's
  instruction. Per intent, run what that intent delivers; per intent
  cumulatively, rerun what the PHASE can deliver so far; at the phase's close,
  one full run against the corpus the phase is meant to serve. Always over a
  real repository state, always in a throwaway snapshot, and the result recorded
  whether or not it is good news. Occasioned by the cold-reading workstream,
  where thirteen intents, six adversarial delta reviews, five fidelity audits
  and three review rounds on one eval all passed, and the first end-to-end run
  found in about fifteen minutes that the artefact a reading is handed is around
  9.8 MB, roughly 2.45 million tokens, so no reading can be given one, and that
  three of the four positions receive a byte-identical item set although their
  definitions state four distinct objects. Routed by the four-piece table: the
  stance and its gate to a discipline record; no capability, because the
  rehearsal is a hand-run over verbs that already exist and script-first-mvp puts
  the documented protocol at this rung; no ADR, because it governs method rather
  than architecture; no trust rule, because nothing here is a trust boundary. The
  cumulative rung is the load-bearing one and it is the addition the maintainer
  made: the size defect belongs to an assembler shipped in an EARLIER phase which
  passed its own tests, review and audit, and none of the three intents rehearsed
  at the end delivers anything that hands a reading an artefact on its own, so
  only the question "what can this phase deliver right now" reaches a defect
  living in the composition. A phase that carries a cumulative rehearsal is
  runnable at every point rather than being a list of merged intents, which makes
  the demonstration a deliverable rather than a by-product. Rejected: a single
  rehearsal at the phase close, which is what was actually done here and which
  found the defect one whole phase after the intent that caused it. REACH IS THIS
  REPOSITORY ONLY; a rule reaching every abcd-managed repo is a bundled default
  domain compiled into the binary, which is a code change and its own intent.
- 2026-08-31 — A fact a source can settle is read, not recalled, adopted as
  itd-197 at the documented-protocol rung on the maintainer's instruction.
  Recall locates a source and never quotes one; the rule binds any assertion
  about to be written into a record, a commit message or a brief, or used as the
  grounds for a decision, and leaves conversation free to be provisional so long
  as it says which it is. Occasioned by four errors in one session, each about a
  document its author had read carefully hours earlier: the grounds put to the
  maintainer for degrading the supply-regime gate cited a licence the design does
  not give, since its condition is "degradation only on observed noise" and the
  evidence was a constructed corpus over an instrument that has never run; a
  captured issue asserted that no non-test file referenced the agent definitions
  when one already declared the directory constant; a residue paragraph claimed
  an enumeration could not fall behind while two written lists survived inside
  it; and a synthetic battery's result was reported as measured. The pattern is
  the finding: confidence tracked FAMILIARITY rather than accuracy, and the
  material nobody had read that day produced no false claims because nobody felt
  able to assert anything about it. Routed by the four-piece table: the stance
  and its gate to a discipline record; no capability, because the remedy is to
  open the file; no ADR, because it governs method rather than architecture; no
  trust rule. Filed as a discipline rather than left as agent memory BECAUSE it
  was already agent memory: a note from a previous session saying to check a
  principle's letter before citing it was in context from the first turn and did
  not bind, which is memory-graduates-to-record's promotion signal in its
  stronger form, a lesson recalled and then not followed, showing the home rather
  than the lesson to be at fault. Rejected: leaving it as a memory item, which is
  what had already failed, and writing a detector, because whether an assertion
  was read or recalled is not visible in the text. REACH IS THIS REPOSITORY ONLY.
- 2026-08-31 — The abcd lab convention: experiments run in throwaway snapshots at the operator level, never inside or committed to the repo. Home: `~/.abcd/lab/<timestamp>-<source-sha7>/`, keyed by mint-time timestamp plus 7-char source-root sha — several labs may share one baseline, and the directory must exist before the first mutation; this diverges from the voyage log's root-sha keying (voyage keys one event per source state; a lab keys a session whose identity is its intention), and the divergence is stated rather than claimed as fidelity. Layout: `INTENTION.md` (question, hypothesis, measures, STOP conditions — written before any mutation), `snapshot/` (a throwaway clone mutated freely; records mint freely inside it and never promote — harvests re-file into the real corpus), `transcripts/`, `harvest/`, `amendments.md`, plus an append-only `~/.abcd/lab/index.jsonl` registry (id, source sha, status, harvest pointer). DISCARD ends the lab session, not the lab home: the snapshot's live tree is archived to `snapshot.bundle` and deleted, and every mutation session inside a snapshot ends with a commit inside the snapshot before archiving, so minted records and the snapshot's own `.abcd/` tier survive in the bundle; `INTENTION.md`, `transcripts/`, `harvest/`, and `amendments.md` survive discard, and a later lab clones its base world from an earlier lab's bundle. The real repo is touched only when a harvest files records through normal ceremony, each carrying `found_during: lab-<id>` provenance and the lab's base sha. Harness-isolation preflight re-runs per lab against that lab's own snapshot world. Hand-maintained until a lab verb exists; the verb is a capstone candidate.
- 2026-09-01 — Reading positions share one pile by default, and a position can be given its own (maintainer, ruling on iss-2608311501240566). Three of the four positions receive a byte-identical item set today although each is defined over a different object. The default stays one shared assembly, because the readers' definitions already tell each position what to attend to and a shared pile keeps the four readings comparable; a per-position include table is added as configuration so a position can be handed only its own object (the comparative position only the widening output, for instance) when a run needs it. Rejected: making per-position piles the default (four assemblies per run, and the comparability the closing run relies on is lost by default); waiting for a real run to show harm (the record already shows the definitions and the delivery disagree).
- 2026-09-01 — The changelog composer writes only Added and Fixed until it can see the previous release (maintainer, ruling on iss-2609011207114761). The composer's inputs are the records that shipped and their bodies, never the base tag's surface, yet Changed, Deprecated and Removed are claims about exactly that surface, and in v0.7.0 it wrote three such lines about a verb that did not exist in v0.6.9. From here the composer emits no Changed, Deprecated or Removed section, the ingest refuses a composed changelog that carries one, and the release notes say so once so the absence reads as a rule rather than an oversight. Rejected: handing the composer the base tag's command reference (the right long-term input, deferred until the release surface is exported in a form the composer can diff); leaving it to the pre-tag review (a review is not a gate).
- 2026-09-01 — The supply-regime gate refuses only a real decision field (maintainer, ruling on iss-2608311518056854). Fourteen of thirty-four realistic readings were refused for quoting the document they read, because the disposition detector fires on the bare token followed by a colon or equals anywhere in prose, and a reading that REPORTS a disposition is most of what a reading legitimately does. From here the detector matches the structural shape only: a top-level field in the reader's own output carrying the reserved name, never the name inside a sentence or a quotation. Rejected: accepting and flagging for a human (the gate exists so a human is not the filter); banning the words in the readers' prompts (a detection reading must be able to say that one section says merged and another says pending).
- 2026-09-01 — The design branch stays as it is (maintainer). `design/roles-loopback-workstream` is 471 commits behind main and carries two decision records numbered 0055 and 0056 that main has since used; with decisions moving to timestamp ids, the next design session merges main and takes the ids then, when the mint verb may exist, rather than renumbering by hand tonight.
- 2026-09-01 — The brief is the shipped state, consolidated without a new intent (maintainer). The request "a product thinker must be able to understand the current shipped state of the repository by looking only at the brief" is already adr-5's decision and is delivered by two existing drafts, itd-60 (the doc-fidelity pass that gates spec close on the brief reflecting delivery) and itd-147 (generated surface blocks so a shape claim cannot drift), so no umbrella intent is filed; the drift records from the v0.7.0 receipts link to them and the two exact duplicate pairs close as duplicates. Added to the decision: brief, record and release stay in sync, with one legitimate lead — a brief edited after a release was cut is ahead of that release until the next cut, and the gap closes at the cut rather than between cuts. The work is sequenced as Phase 8, "the brief is the shipped state"; the design branch's unmerged "closed loop" phase takes 9 when it merges. itd-61, the reverse direction, stays adjacent as an open question of the phase.
- 2026-09-02 — The 2026-09-01 supply-regime ruling reverses the 2026-08-31 degrade-to-flag decision (recorded at the maintainer's ruling, flagged as a reversal for the record). On 2026-08-31 the four prose signatures were degraded from refusing to flagging, so an honest report that tripped one landed with a review flag; the 2026-09-01 ruling names accept-and-flag as rejected and keeps only the structural rule, a reserved name as a key of the reader's own output. The implementation withdraws the signature registry and the review-flag plumbing it alone fed, since a detector that produces nothing is dead scaffolding, and re-stamps iss-2608311518056854, which the 31 August change had already resolved, under the new ruling with no `Resolves:` trailer, because RS001 refuses a trailer for an id that entered resolved/ before the branch diverged.
- 2026-09-01 — The strict up-to-date policy stays in front of the merge queue, and the branch update is automated instead (maintainer, ruling on iss-2609012202237613 against the two directions it names). Every auto-merge pull request opened tonight sat `BEHIND` outside the queue until a hand-run update, because auto-merge enqueues only a CLEAN pull request and the strict policy makes a behind pull request never CLEAN; the queue that exists to test a pull request against the current base therefore never received it. The queue's group commit does gate the merged result, so dropping strict would preserve the duplicate-id gate iss-172 records as an invariant, but the maintainer keeps strict and ships iss-172's first rung: `scripts/pr-keep-current.sh` walks every armed pull request and asks the forge to merge the base into any that is `BEHIND` and not queued, `--watch` repeating until none is armed. Rejected for now: relaxing strict (belt and braces are kept on purpose); a scaffolded update-branch workflow (rung 2, still needing a token whose pushes trigger checks, carried by itd-107). CONTRIBUTING no longer claims branches never need updating.
- 2026-09-01 — ADRs take timestamp ids through the same seam as issues (maintainer, the turn adr-45 ruling 3 deferred). Two branches minted `0055` and `0056` on the same day for different decisions, so the hand-numbered ordinal has the add/add collision the timestamp mint removed for captures. From here an ADR id is minted by the binary as `adr-<timestamp>` with a filename ordered by that stamp; `0001`–`0058` keep their ids and filenames, and nothing is renumbered. Rejected: minting ordinals from one checkout only (a convention, not a gate), and renumbering at merge (the collision tonight was found by reading, and a reader is not a gate). The mint verb and the lint rule that admits both shapes are implementation, tracked in the ledger.
- 2026-08-31 — Degrade itd-185's four semantic supply-regime signatures
  (RG-EVAL-ORDERING, RG-EVAL-RECOMMENDATION, RG-REG-FIXPROPOSAL,
  RG-EXPL-DISPOSITION) from enforce to flag: a hit raises a review flag on the
  run record and the item lands. This is the reserved degradation path spc-63
  names, taken deliberately, and it weakens the claimed property from ENFORCED to
  OBSERVED. The structural halves are untouched and stay absolute — the reserved
  names and the strict per-position schema still refuse. THE EVIDENCE IS
  SYNTHETIC, and saying so is the point: the itd-185 fidelity audit constructed a
  corpus of 34 realistic reading outputs and 14 were caught, every one of them for
  REPORTING what the read document said rather than for proposing anything (the
  disposition detector fires on any claim quoting the token this repository's
  records carry everywhere; "section 3 says the fix is already merged while
  section 8 says it is pending" is the canonical shape of a detection finding and
  fires too). No reading has ever been run through this verb — every payload the
  delivery has validated is synthetic — so the ruled condition, "degradation only
  on observed noise" (BUILD-PLAN.md:70, spc-63), is NOT met and this departs from
  the ruled design. Taken anyway because the alternative departs further: the gate
  is currently enforced over a calibration that has never been taken, which is the
  standing tension itd-185 already records against widen-options' "calibrated
  before it gates", and waiting for observed noise means enforcing indefinitely on
  no calibration, because the assembled input is about 9.8 MB and cannot be handed
  to a reading at all (iss-2608311501186646). Of the two departures this one
  cannot produce a false refusal of a real reading, and it is reversible by the
  same one-line mode change. REVISIT POINT: the first real reading. When the
  instrument can be run, the flags it raises are the observed calibration, and the
  mode is reconsidered on that evidence rather than left at flag by default.
  Rejected: keep enforcing — defensible under the letter of "degradation only on
  observed noise" and indefensible under "calibrated before it gates", and on the
  synthetic evidence it would refuse roughly two in five legitimate outputs the
  first time a reading ran. Consequences recorded in the same change:
  TestEverySignatureShipsEnforced is replaced by a test pinning each entry's mode
  by name so a silent flip in either direction fails; itd-185's ac-5 and ac-9 are
  rewritten to say flag rather than refuse; the ingested audit verdict
  rcp-fe3450ca55ff records those two criteria MET on the strength of a refusal and
  therefore describes superseded behaviour needing re-issue at the next audit; and
  the disclosed residue on itd-185 and spc-63 is widened to name over-catching
  beside under-catching, with the propose-versus-report distinction as the reason.
- 2026-09-02 — Source and tests are opt-in to a reading, never admitted by default, and an opted-in item travels whole and marked unscanned (maintainer, at the Iteration 2 planning interview, checked against the design framework v4 section 7.2 and the readings companion v4 section 4.5, which both name code, tests, documentation and configuration as the shipped tree a reading may see). The `source` and `test` kinds stay in the include table so the object stays as the documents state it; no committed cold preset names either; a preset or a scope that opts one in gets it whole, and the manifest marks each such item as unscanned so the exclusion floor's assertion is made only for items it parsed (invariant 16). The detection definition's Object is unchanged. itd-194 is planned on this ruling: the floor refuses markdown it cannot resolve and marks what it does not parse; the strict alternative, removing the test kind from the table, was declined because it contradicts both documents, and the alternative of widening the floor to scan every type stays declined (2026-08-30). Cost of the narrowing under the committed cold presets: none, since none admits source or tests; the live fixture leak at itm-0736 is absent from every committed preset and disclosed as unscanned wherever a preset opts tests in.
- 2026-09-02 — adr-2609021016272867, adr-2609021016275803, adr-2609021016270132 and adr-2609021016288378 are adopted (maintainer, at the same interview, each checked against the framework and the companion first). adr-2609021016272867: the comparative reading receives two body fields of one named widening run's items as the single exception to the prior-run exhaust, through a fourth closed operand naming the run; the maintainer chose the named operand over a derived selection because two widening runs coexist after the closing run, on the condition that an absent operand lists the runs the operator may choose from. adr-2609021016275803: per-run context stamps and a session-separation check that reports held-for-what-was-seen or unobserved, never clean; `scribe assemble` and `scribe ingest` with a ledger-only allow list derived from the ledger's directory list. adr-2609021016270132: the principles family is a declared record store with four typed keys, forward-only, read cold by statement and never by citation, with the inheritance check. adr-2609021016288378: in its three-surface form, because the single-section draft contradicted the design's two landings and adr-55's definition of the construal.
- 2026-09-02 — The widening position does not receive the shipped intents (maintainer, from the documents: the framework's W12 object and the companion's section 5.2 both list the widening object without them). The shipped-intent projection row's positions exclude widening; iss-2609012259587904 is settled and its code change lands with itd-194's include-table work.
- 2026-09-02 — The workstream's own fifteen shipped intents keep their missing mechanism claims and their `None stated.` conditions as the Iteration 1 baseline (maintainer, from the framework's section 10: existing records are untouched, sparseness is information, and an absent stamp on an older record is never backfilled). iss-2609012259581057 is settled as will-not-fix on that rule; the entailment reading's yield bound is reported beside its findings instead (the companion's section 6.6, delivered by the size report as iss-2609012259585189 asks, folded into the preset-windows intent).
- 2026-09-02 — The Iteration 2 interpretations are fixed in advance, as the framework's section 13 and the companion's section 7.6 state them, and are recorded here before any reading runs: the opening readings are the widening reading and the detection pass over Iteration 1's shipped state; one closing run is scheduled over the same object set after the dispositions and whatever revisions they occasion; amnesia is a repository property held by the eval and not evidenced by a case run; a detection rejected with a named purpose over a state deliberately unchanged that returns is the purpose-durability finding; a detection accepted and acted upon whose tension returns is the convergence finding; silence at the closing run is weak evidence of settlement and is reported as such; where the opening readings surface nothing, the iteration proceeds on the researcher's own conjectures, the null result is recorded as a run with an empty item set, and no further reading is commissioned to manufacture a finding; where the widening reading returns fewer than two configurations, the comparative reading is not exercised and that outcome is a committed comparative run with an empty item set; recurrence matching is warm work and the recognition is itself a disposition judgement.
- 2026-09-02 — The supply-regime signatures stay in flag mode, code-pinned by a test, and no configuration seam is built (maintainer; the 2026-09-01 ruling that degraded the four semantic signatures on the measured over-catching already records the weakening the framework's section 8.9 requires, and the mode is reconsidered on the evidence of the first real reading rather than flipped by a setting). iss-2609012259571458 is settled as will-not-fix.
- 2026-09-02 — The NOT YET REAL marker on the construal lifts only once what the construal describes has been implemented (maintainer: "lift it once it's true"). Iteration 2's eight intents are planned and not built, so the marker stays; the maintainer lifts it at the readiness statement, and the widening reading does not run before then.
- 2026-09-02 — Two hundred thousand estimated tokens is a target for a committed preset, not a limit (maintainer): any figure inside the reader's window is acceptable, a figure over the target is stated to the operator by the size report in one line, and nothing is refused for size. The preset-windows intent carries the criterion.
- 2026-09-02 — The invocation is a position and a target state and nothing else, as the design specifies (maintainer: "all I want is to run Iteration 2 exactly as designed in v4"). adr-2609021016286571 supersedes adr-58: the scope operand and the override stamp go, the committed preset for the position supplies what the reading is handed, and changing it is a commit. adr-2609021016272867 is corrected the same day from a named candidate-run operand to a derived selection: the comparative assembly takes the one committed widening run at the target whose items carry no disposition and no admission, and refuses naming the runs when none or more than one qualifies. Both corrections close the only two points at which the record contradicted the framework's and the companion's letter on the invocation; the earlier entry today that recorded the named operand is superseded by this one.
- 2026-09-02 — adr-56's third rule is refined, not contradicted, by the source-and-tests ruling (maintainer): a control that admits an input it cannot examine may pass it only with a per-item unscanned mark in its attestation, so the second rule (an attestation never says more than the examination) carries what the third rule carried; refusal stays the rule for an input the control claims to have examined and could not. Invariant 16 amended to say so. Every ADR is now checked against the framework v4 and the companion v4: adr-58 is superseded by adr-2609021016286571, adr-2609021016272867 is corrected to the derived run, and none of adr-55, adr-57, adr-2609021016275803, adr-2609021016270132, adr-2609021016288378 or adr-2609021016286571 contradicts either document. The two-hundred-thousand-token target is a clarification the documents do not speak to, kept as such.
- 2026-09-02 — What a reading is handed has two committed dimensions, both the documents' own (maintainer): the OBJECT SET (the framework's term, section 13: which records and delivered paths the run is about; for Iteration 2, Iteration 1's shipped state, which here is the ledger: itd-177 to itd-189, itd-198, itd-199, spc-55 to spc-69, and the packages and pages they delivered) and the KINDS within it (the companion's four, section 4.5). A committed preset entry per position carries both plus the window it measures at; nothing at the invocation chooses either. A reading of the object set includes all kinds when the object set fits the reader's window, with the size report stating any figure over the two-hundred-thousand target; measured on 2026-09-02, the ledger at the detection position with all kinds is about 611,000 estimated tokens, which fits a million-token reader and is stated over target. A kind that proves useless leaves the default entry by a commit, which is how the record learns; a larger object set in future narrows kinds the same way. The word "object" alone stays the companion's, for what a position reads; "object set" is the run's, and "object of interest" is its plain-language gloss.
- 2026-09-02 — The readings open Iteration 2 and the build follows them, as the framework's section 13 states (maintainer, correcting the marker entry of the same day). Four intents complete Iteration 1's instrument and land before any reading runs: the two-operand invocation, itd-194, the preset windows and the comparative channel, together with the eval gate and the ledger hygiene. The opening widening reading and detection pass then run over Iteration 1's shipped state and claim record with the construal as it stands, marker included (framework section 12: a construal is readable while marked), and their outputs are dispositioned. The six remaining intents (admission and surprise verbs, reframe record, principles as a read object, condition disposition, reading-occasioned origin, scribe verbs) are the framework's own section 13 build list, planned in advance because the documents commit to them, and built at Step 5 after the dispositions, with the first admissions, surprises, reframes and condition dispositions hand-authored in the target format until each verb lands (framework 13, "schema before commands"; companion 5.6). The marker's lift is not a precondition of the widening run; it lifts when the maintainer judges the ledger the construal describes to be real. The sentence "the widening reading does not run before then" in the earlier marker entry is superseded by this one.
- 2026-09-02 — The sentence "no committed cold preset names either" in the source-and-tests entry of this day is superseded by the object-set entry of the same day: the default entries at the widening and detection positions name source and test within the object set, every such item marked unscanned; the entailment entry names neither, because its object (companion section 6.2) holds no tree.
- 2026-09-02 — On the supply-regime check, the 2026-09-01 maintainer ruling governs and the "stay in flag mode" entry of this day is superseded by it: the framework's section 8.9 names two structural signatures (aggregation at evaluative, fix-proposing at registrative) and a record-shape check at explicative, and those refuse, enforced; the four prose detectors were an addition the framework never asked for, proved noisy on synthetic evidence, and are withdrawn rather than left flagging. The property the case report claims is therefore enforced for the structural check and unclaimed for prose. No configuration seam is built; the wontfix on iss-2609012259571458 stands on this ground.
- 2026-09-02 — The interpretations the Iteration 2 materials take where the framework and the companion are genuinely open are registered in the dated research note `research/notes/2026-09-02-iteration-2-divergence-register.md`, one entry per point with the document section, the choice and its ground (maintainer's rule: divergence only where the specification is open, and always recorded). The register is the place a later reader checks before calling the instrument non-conformant.
- 2026-09-02 — Corrections after the final compliance review (maintainer's rule: the documents govern). (1) Phase A of the implementation is the two-operand invocation, itd-194, the preset windows, the comparative channel and the reading-occasioned origin, the last because the framework's section 6.3 has the origin key written only by commands and no promotion of an accepted reading item may precede its writer; Phase B is the condition disposition, the admission and surprise verbs, the reframe record, the scribe verbs and the principles read object, built at Step 5 after the readings, with the first entries hand-authored where the framework permits. (2) Between the phases the maintainer runs all four readings in the cycle's order, not only the two that open it: the widening reading opens (Step 2), the entailment reading runs on the claim record (Step 3), the comparative reading characterises the widening run's candidates (Step 4) and the detection pass reads the shipped state against the claim record (Step 6), as the framework's section 14 places them; admission follows characterisation. (3) The opening target is the commit at which Phase A is complete, and "Iteration 1's shipped state" is read at that commit: the object set names the ledger's records and delivered paths, and a tension the detection pass surfaces between a Phase A change and a superseded claim in a shipped instrument intent is dispositioned like any other, with the supersession named as the purpose. (4) A run that returns no items is committed at every position as a run with an empty item set, never refused, per the framework's section 13; refusal is for a malformed payload. (5) "Brief current text" is the whole brief bar the evidence chapter, which the include table excludes as verdict material (the same ground as the Audit Notes exclusion), and bar the glossary, which is its own row; the product, constraints, meta, surfaces, internals and delivery chapters are admitted as brief sections at every position that reads the brief, and the preset windows are re-measured at landing. (6) The supply-regime check as this tree stands: the reserved-name refusals are the framework's structural signatures and refuse, enforced; the four prose detectors flag, observed; their withdrawal is the 2026-09-01 ruling carried by an unmerged change, and whichever state the opening target carries is what the case report describes, "enforced" for the reserved-name check only. The entry of this day that described the detectors as already withdrawn is corrected by this one. (7) The blindness core's fourth condition takes the companion's sentence, "items are returned in the order they arise in the object"; the four definitions move together with a patch version.
- 2026-09-02 — The per-position configuration the 2026-09-01 ruling asked for ("one shared pile by default, and a per-position include table as configuration when a run needs a position handed only its own object", recorded against iss-2608311501240566) is delivered by the per-position preset entry of the object-set ruling of this day, which carries each position's object set, kinds and window, applied with no operand. One configuration surface, not two: a per-position include-table section beside the preset entry would be a second answer to one question, and the change carrying it is reconciled to the preset entry or waits for the maintainer.
- 2026-09-02 — The claim vocabulary is the documents' word everywhere (maintainer's rule that the documents govern): criterion, causal and context on intents, on principles and on the entailment output contract, with the section heading `## Mechanism` unchanged as the causal claim's home (framework 7.1). The shipped intent token `mechanism` (itd-177, itd-190) moves to `causal` in its own change, filed as an intent by the implementer and planned by the maintainer; the principles record store uses `causal` from its first entry. The detection figure of about 611,000 estimated tokens in the object-set entry was measured without the glossary; the presets spec's figure of about 624,000 includes it, and both predate the ledger glossary context and the brief-chapter rows, so the figures the window eval measures at landing are the ones the record keeps.
- 2026-09-02 — The entailment reading is handed the OBJECT SET's drafts and planned intents, and none by default (maintainer, at the Phase A review). The framework's section 13 fixes the object set as "Iteration 1's shipped state and claim record" — the fifteen workstream intents, which the record extends by the ten Iteration 2 intents — and the readings companion's section 6.2 makes drafts and planned intents ADMISSIBLE at that position: admissible is a permission, the object set is the scope, and the assembly was handing every draft and planned intent in the repository (147 projected intents on the Phase A tree) because a record row is admitted whole when the object set names none of its records. The drafts and planned rows therefore narrow by the entry's record list exactly as the shipped row does, so a draft or planned intent travels only when the committed entry names it; the companion's admissibility becomes a switch on the entailment entry — `admit_drafts_and_planned`, yes or no for drafts and planned intents beyond the object set, default no, refused at load on any other position because those two rows are admitted at entailment alone. The committed entailment entry names the ten Iteration 2 intents (itd-194, itd-2609021003095168, itd-2609020625400169, itd-2609020625400194, itd-2609020625400445, itd-2609020625402518, itd-2609020625402599, itd-2609020625405170, itd-2609020625405251, itd-2609020625407419) beside the fifteen, and declares no switch; several of the ten are shipped by the time Phase A completes and travel through the shipped row, which is the same object set either way. Issues, ADRs and every other warm family stay excluded by construction and gain no switch. Divergence register entry 1 is corrected accordingly: it recorded the ten as reaching the position through the companion's admission, which is the reading this ruling replaces.
- 2026-09-09 — The ledger's human gates on the ordinary flow are parked, reported and never refusing, until the reading work is rethought (maintainer; iss-2609091009111294). The built-in machinery of the provenance ledger and the cold reading stays as shipped, and so does every refusal that guards a reader's blindness or what a reading may return; what changes is every refusal that stops a person in the ordinary intent-to-ship flow for a sentence the gate cannot judge. `intent ready` keeps all seven rows and gates on four: the mechanism, scope-condition and grounds rows are advisory, named with their remedies, and the verdict ignores them. `capture promote` and `capture resolve` record `--grounds` when it is given and write nothing for it when it is not; a value that is given is held to the vocabulary and the floor as before. A lapse capture without `--lapsed-at` records no instant rather than being refused, and the committed-ledger gate no longer raises the absence finding, so gate and reader still agree. This reverses two rulings of 2026-08-30, promoting the grounds readiness check to a refusal and refusing a triage that records no grounds, flagged as a reversal and confirmed by the maintainer rather than filed on the tool's judgement. Rejected: deleting the checks, which would lose the report and the remedy; keeping them armed, which is manual work the product thinker's language does not yet justify; and demoting the reading instrument's own refusals, which stop no human. The armed form returns, if it returns, through the legible-surface phase rather than by flipping this back.
- 2026-09-08 — The hook shims' PATH rung is OWNED-ONLY, and the 2026-08-01 itd-105 / spc-21 entry above ("Rejected: a PATH fallback in the hook commands (spc-21 forbids it)" and "A PATH fallback is forbidden by the intent's Decisions and is NOT added") is superseded by this one (maintainer, ruling option A on iss-2609012039107700 / GHSA-gx3m-3224-qqcv, CWE-426). iss-254 added the rung in 20968bdf (v0.5.1) with no entry reversing spc-21, so the ledger and the tree have disagreed since; what is true from here is that a rung EXISTS and is narrow. A hook takes an `abcd` from PATH only when the lookup yields an absolute path, out of a directory that is neither under the shim's working directory nor world-writable (iss-2609012039117381), AND `~/.abcd/path-entry` records that exact path as this machine's installed binary — a `path=` string comparison, no hashing, because adr-46 rejects hashing ~11 MB on every hook event and this rung fires on every prompt and every tool call. What spc-21 was protecting is what this restores: before the ownership check, a plausible `abcd` earlier on PATH than the operator's became the session's rules loader and, through PreToolUse's pass-through of the guard's 0/1/2 verdict, an approver of every shell command it was asked about. A refused binary now degrades to the shim's existing loud line naming it and the reason; for PreToolUse that is UNGUARDED and exit 1, never the exit 0 the harness reads as approval. SessionStart is unchanged — it has no PATH rung and fails closed. THE CONTRACT CHANGES WITH IT: the install one-liners in README.md and docs/how-to/install.md write `path=` and `binary_sha256=` into `~/.abcd/path-entry` (they know both values; `plugin_root` is optional per readPathEntry), because the documented rescue — "the install one-liner restores it, after which the hooks resolve the PATH binary" — would otherwise stop working the moment the rung became owned-only, and a one-liner run after this change re-points the record at `~/.local/bin` even where `ahoy install --bin-dir` had put the binary elsewhere. Rejected: option B, dropping the rung from PreToolUse alone (smallest change, but it leaves the prompt channel and the transcript path exposed to the same foreign binary); option C, accepting the residual as the operator's own PATH (the same shims do resolve sh, find and printf from PATH, but none of those is handed the guard's verdict, and the advisory is a CWE-426 with a working reproduction). Residual, named rather than papered over: the record is home-scoped and equally writable by any same-UID process, so ownership is provenance against a hijacked PATH, not against a compromised account — the same bound adr-46 decision 4 already states, and the reason the check is called ownership and not verification. Pinned by the four inverted cases in internal/surface/cli/hooks_selfprovision_test.go (owned-runs, unrecorded-refused, record-names-another-file-refused, SessionStart-unchanged), with the PreToolUse exit code asserted as exactly 1.
- 2026-09-09 — A tool never creates directories in user-owned project space, and agent and session scratch is machine-scoped (maintainer, on finding twenty-two agent worktrees created as siblings of the checkout beside their other projects: "I don't want a user to be surprised that a folder is all of a sudden full of stuff"). The isolation stands — the checkout is the unit of isolation — and the location moves: a session's worktree goes under `~/.abcd/worktrees/<root-sha>/<name>/`, keyed the way the history, transcript and voyage stores are keyed, and a store nobody can list is refused as the same pile somewhere less visible, so listing and reclaiming are part of the capability. Decomposed on the maintainer's ruling into three records: adr-2609091014087993 (the trust rule, accepted), itd-2609091014076309 (the store, its list and prune verbs and the status-board line, in drafts pending adoption) and the principle `the-users-directory-is-theirs` (the stance, with its write-side enforcement stated as the intent's future work). Rejected: the in-tree `.abcd/.work.local/worktrees/` placement, because every tree scan walks it; a per-user configured location, because its default would still have to be something. The existing sibling worktrees are a hand cleanup, never a migration: the store moves nothing it did not create.
- 2026-09-09 — adr-2609091014087993 is superseded by adr-2609091248200336, which states the split its Consequences left unstated (maintainer, ruling option (b) on iss-2609091129426411 against accepting the bullet as a prediction fulfilled in two steps or reverting the `AGENTS.md` edit): the location half of the worktree rule binds now — § Concurrent sessions names `~/.abcd/worktrees/<root-sha>/<name>/` and refuses the sibling and in-checkout forms, as of `805bb023` — and the verb half binds when the store ships, because the original's deferral was about naming a VERB that does not exist, never a location plain git already reaches. The successor carries the trust rule, its five declarations and its rejected alternatives whole, and folds in the helper correction iss-2609091155525689 owed the same original, since an original's `superseded_by` names one record. The original's decision text is untouched; only its status and its forward link change.
- 2026-09-09 — adr-2609090717039680 is superseded by adr-2609091248201071, one record for that original alone (maintainer, ruling on iss-2609091155525689 that each record describing the real-dir helper at its pre-consolidation home gets its own successor rather than one record covering both). The mechanism is unchanged and its home moved: `24c2f2e3` consolidated the three `ensureRealDir` copies into `fsutil.EnsureRealDir` / `EnsureRealDirAll`, so `internal/core/history` owns the store's layout and mode and calls the primitive for the create-then-prove step. Rejected: describing the mechanism and naming no symbol, which is the reading under which a fourth copy is a faithful implementation of the record.
- 2026-09-09 — itd-2609091034175565, the claim intent, is split into three records (maintainer, ruling after two adversarial reviews of the widened draft, design/feasibility and record-discipline). The reviews found the expensive parts unsound as drafted: a `claimed_by` stamp on an issue record is invisible to any peer on an older binary, because `issueschema.Known` is a closed allow-list whose reader refuses and skips a record carrying an unknown key and `record_schema` mirrors the refusal into the gate, and version skew is the steady state here (five plugin-cache vintages beside a `go run` checkout); the write-verb refusals fire after the fix is written, because `AGENTS.md` puts `capture resolve` in the same change as the fix; no staleness threshold is safe in both directions for two sessions in one worktree, since the worktree-exists and branch-merged tests are identical for both and a host may fire session-end on a context clear with the human still present; and the pushed half costs a merge-queue pass per claim, measured at fifteen to sixteen minutes on this repository's merge-group `ci` leg. Every collision on record is one of two claim-free shapes. The ruling: itd-2609091416295622 carries the read-only sibling-worktree ledger diff over `git worktree list --porcelain` (no claim, no lease, no hook, no threshold; the piece that ships soonest); itd-2609091416304128 carries `capture resolve` and `capture wontfix` refusing a record terminal at the local `origin/main` ref as last fetched, stating the ref's age and fetching nothing, with the same judgement rendered read-only on `abcd <record-id>`; itd-2609091034175565 keeps the claim, the lease and the refusals as a draft marked not ready, carrying the refusal-surface, liveness and pushed-price questions as open questions and the two-release stamp migration as a hard constraint. The existing record stays the claim so the `promoted_from` trail from iss-2609020716570699 — whose remedy is the claim and the pushed half — stays true rather than being retitled onto a listing the issue never proposed; the two new records reach the issue through `related_intents` on the issue and on themselves, and every relation the schema cannot type is stated in prose as a prose cross-reference (iss-2609091256264547). All three sit in `drafts/`; adoption is the interview's. Rejected: re-scoping the existing record to the listing, which keeps the trail mechanically and breaks it semantically; a fifth relation word for "split from", which is the vague form the decomposition principle forbids.
- 2026-09-15 — Identity redaction masks exactly the byte spans the detector flagged, and nothing else on the line (product thinker, ruling on iss-2609120446083912 against the whole-string rewrite that `redact.go` had recorded as deliberate). The detector's refusals — a component of a reverse-DNS identifier, a collision inside a longer word, an occurrence inside a URL span — are only real if the rewrite honours them; the whole-string rewrite overrode every one of them whenever a genuine mention shared the line, corrupting the technical content a record exists to hold. Spans are validated against the line before the secret seal and applied after it (the seal preserves byte length), overlapping spans merge into one cluster masked with the widest member's placeholder, and the line is rebuilt from original bytes so no offset is ever applied against a shifted line. The cost is accepted and named: an occurrence the detector deliberately clears — a login inside a URL, including a credentials URL's userinfo — is no longer masked by accident, and the stage-two re-scan does not flag it either; the caller's own home path is not in that residue because the detector flags every occurrence by the same anchor the literal sweep uses. Rejected: keep masking every copy but exempt the shapes the detector exempts (a second copy of the detector's judgement, which drifts); leave it as it is.
- 2026-09-15 — An intent owns one or more specs, and it ships when its last spec closes (product thinker, ruling on iss-2609100508566552, where `spec close` shipped the intent unconditionally and a session meeting a half-delivered intent could only stop). An intent that has been thought through stands as written: a spec that delivers part of it is closed on its own terms, a new spec is minted for the remainder and attached to the same intent (spec closed X, spec open Y, intent still `planned/`), and the intent moves to `shipped/` on the close after which no open spec names it, with `--impact` demanded at that transition and no earlier. The rule is adr-2609151513118583 and invariant 17 of the brief; the build is owed and the issue stays open until it lands. Rejected: close still means ship and narrow the spec (falsifies a thought-through intent); a spec may close without shipping and the intent is shipped by hand (a step after the merge is the one that gets forgotten); split the intent at close.
- 2026-09-15 — Banned names get both halves (product thinker, ruling on iss-2609100506269348, choosing "both" once told a home list is invisible to CI): a committed declaration lifts the public visibility fence so a fresh public repository can create its committed banned-names layer on its first commit, and a machine-global private list in the user-level home bans a name in every repository on the machine. CI never reads the home list and nothing from it reaches a committed file. Filed as itd-2609151516525843 (`builds_on` itd-74, `refines` adr-56); whether the home is one macOS user's or shared across users stays the product thinker's separate open question. Rejected: move the banlist config out of the hidden directory; make the public list private-visibility only.
- 2026-09-15 — The decisions log becomes a folder of individually minted records with a derived index, and `DECISIONS.md` a symlink to that index for now (product thinker, reframing iss-2609100507439414 from "how does adoption propagate abcd's merge-attribute workaround into a managed repository" to "why do the two single append-to-the-bottom files conflict when the five one-file-per-record families did not once across 27 merges", the measurement in iss-2609100508570803). The shape applies to abcd and to every managed repository; the decisions-append gate (DA001–DA004) is retired by the shape, in the same change the shape lands. The rule is adr-2609151138420062, the capability itd-2609151138388536 in `drafts/`; the changelog's conflict class is itd-2609150819432059's and closes separately. The adoption question the product thinker was first asked — point at the conflict, fix it silently, or fix it and take over the changelog — is dissolved by the shape rather than answered. Rejected: keep the file and propagate the union attribute; the folder for abcd only; keep the file and accept the conflicts.
- 2026-09-15 — An id written in a record's prose must resolve unless the author marks it illustrative or forward-looking (product thinker, ruling on iss-2609100518527863, where a spec composed in an autonomous run carried two invented ids in ordinary sentences and only a human reading it back noticed). `record_schema` already resolves the typed frontmatter references; the new `prose_citation_resolves` rule reads record bodies and free-text frontmatter fields through the one canonical resolver, folding case and padding, with a line-scoped `<!-- record-lint: illustrative -->` or `<!-- record-lint: forward-looking -->` marker as the only escape. A slug does not stop an id being an id, so a filename-shaped handle is a citation of its id. The corpus that predates the gate is carried by id, never by file, in `.abcd/prose-citations-baseline.json`, each entry declaring its class and reason, and the baseline ratchets: a new unresolvable id fails even in a file the baseline names, and a spent entry is reported so the list only shrinks. One entry is a probable defect, iss-2608231243286557, cited five times as settled fact with no record ever minted; it is carried as `suspect` for a human to correct. Rejected: only structured references count (would not have caught the case that prompted the rule); check prose but only warn (a warning nobody reads is the silent exemption the rule closes).
- 2026-09-15 — A commit message or pull-request title or body that names an iss-N declares its relation to it, and a bare mention is refused in CI before the merge (product thinker, ruling on iss-2609100507421759, first asking whether a formal resolution trailer can be enforced and then choosing "enforce it, and also hint on the default branch"). `Resolves: iss-N` says the change fixes it and RS001 then requires the record to move in the same diff; `Refs: iss-N` says touched but not fixed and demands nothing of the ledger; the vocabulary is closed at those two words, a comma-separated list of ids is admitted on either, and a near-miss (`Ref:`, `See:`, `Related:`) reads as what it is. RS004 covers `iss-N` only, the family the trailer vocabulary owns; an id inside a record body is `prose_citation_resolves`'s question. The hint is `abcd capture mentions`, a read-only listing over the default branch of open records whose ids appear in commit history without a resolution, ranked resolves over tree over record, silent on the commit that added the record's own file; it lists and never moves a record. Rejected: any mention of the record's id as proof of a fix (a mention proves the author's attention, not a fix); a detector with no enforcement (a silent fix cannot be seen at all, so the gate has to force the mention into a declaration).
- 2026-09-15 — The 1:n intent–spec rule of adr-2609151513118583 is built, and the spec's `intent:` back-link is the single source of truth for which specs realise an intent (implementing session, ruling on iss-2609100508566552's build, reversible by the product thinker): the intent's `spec_id` stays a scalar meaning the spec it was planned with, the set is derived by one canonical comparison shared by the spec store, the intent corpus and the record-lint index (a zero-padded back-link once let one reader ship an intent the other saw as still open), and the bidirectional check becomes membership. `spec close <spc> --remainder <slug>` mints the follow-on spec before any move and reuses an open one with that slug on retry; `--remainder` is refused on a shipped intent or a closed spec; `--impact` at a close that does not ship is refused rather than ignored, because the flag's only effect is the judgement `shipped/` requires. The fidelity audit request names every spec that realised the intent and the receipt stays owed once per intent. Noted for the gate itd-2609111003026787 plans on the peer's branch, a commit declaring delivery of an intent must move it to shipped/: under this rule a change may deliver one spec and leave the intent planned with a remainder open, so that gate's condition should read "no open spec remains" when it is built. Rejected: an intent-side list of spec ids (a second writable carrier that can disagree with the store, and eighty-five records to edit); a new verb for the remainder (the close is the moment the remainder is known).
- 2026-09-15 — itd-200 / spc-70 ship with five implementation rulings the spec left open, taken by the implementing session and reversible by the maintainer (recorded here so none is silent). (1) The status-line offer is its own consent category, `status-line`, advisory and never approved by `--yes`, exactly as the git-identity pin is: it rewrites a harness-wide user setting and takes element choices, so only an answered prompt writes it, and `yes |` answers it. A decline is never persisted; the offer returns on the next install until the user-level setting exists. (2) Harness detection is positive evidence only: the harness's user settings file exists and parses (its directory from `$CLAUDE_CONFIG_DIR`, else the harness home); no file means nothing offered and nothing written. Ownership of a status command is decided by shape (`<absolute entry with leaf abcd> statusline`), which is what makes a dangling entry decidable when the binary is gone. (3) The two record counts on the row are FOLDER counts (files named `iss-*.md` in the open ledger; `itd-*.md` in drafts plus planned), not the board's parsing readers, because the verb runs on every status refresh and a parse over a clone-controlled record measured 0.96 s at twenty thousand records; folder membership is the record's own status signal, so the number differs only where a parser would skip an unreadable record. The spec's Scope text is corrected to say so. (4) Two guards the reviews required and the spec did not name: the previous-command fallback refuses to run when its own marker is already in the environment and marks its child, so a recorded command that reaches `abcd statusline` in any spelling terminates instead of forking without bound, and the install refuses to record such a command at all; and every write to the harness settings re-reads the file immediately before the act and merges into the fresh document, refusing when the status-line key changed under the prompts (itd-193's rule applied to a user file a live harness also writes). (5) `abcd mode <state>` prints the owed-answer line only when the user-level setting is absent or disabled — the setting is the only evidence of a status surface abcd has — and `managed` prints nothing; the bare board renders the presence line even when the line is disabled, because the switch silences the line and the board is the fallback. Rejected: making the offer a config-change gap (one consent for the PATH entry and the status line would let the reason paragraph be skipped); counting through the board's readers with a cache (a second mechanism to keep correct for the cheapest surface in the tree). The fidelity audit rcp-636b39d5541a records five criteria met with concerns, all disclosed on the shipped record; the one the record could close is closed: the GRILL rule domain now tells an agent to set the mode before it stops for a verdict.
- 2026-09-15 — The roles and loopback design workstream lands on main, executing the 2026-09-01 ruling that the branch waits for the mint verb and takes its ids at the merge. `abcd decide` now exists, so the two decisions the branch numbered 0055 and 0056 are re-minted as adr-2609151528057260 (three roles, who each artefact addresses, and when the loop stops) and adr-2609151528057131 (abcd owns the product thinker's surface), content, status and date unchanged; every citation that meant the roles decisions is re-pointed (rfc-3, the phase-8 page, the roles page, the out-of-scope list, three intent drafts, and iss-168, whose 29 August extension cited them by the colliding numbers), and main's own 0055 and 0056 keep their ids. The branch's twelve hand-numbered intent drafts (itd-165 to itd-176) collide with nothing and keep their ids, as adr-45's grandfathering allows. Occasion: itd-200 shipped today refining the two roles decisions, so their citations on main pointed at the wrong records until this landed.
- 2026-09-15 — GHSA-4q78-ccfv-f374 (iss-2609012039102770) is closed by OPTION B, ruled by the maintainer at an interactive question: bind the cache to a record the environment does not choose alone. The owned PATH-copy promotion re-verified the cache only against the `binary-meta` beside it, and `CLAUDE_PLUGIN_DATA` is taken from the environment as given, so whoever chose the directory wrote both the bytes and the record that "verified" them — reproduced at v0.7.0 as a one-byte file installed 0755 as `~/.local/bin/abcd` with provenance recorded. Now the bootstrap, the one process holding the harness's real data dir that has just established manifest trust for the cache (an authenticated cache hit, or a fresh download verified against the same-origin manifest), writes `~/.abcd/cache-attestation` — `data_dir`, the manifest-authenticated `binary_sha256`, `cache_trust=manifest`, `attested_at`; 0600, temp-and-rename, beside `path-entry` — and `ahoy install` promotes a cache only when that record names the directory, the co-located record carries the attested hash, and the artefact hashes to it, whichever route (environment or the root's `.data-dir` stamp) named the directory; detection offers the heal on the same predicate. An offline run neither writes nor rewrites the attestation, so a cache provisioned offline waits for a networked session before it reaches PATH, said out loud. Why B: the trust floor moves from a value the environment supplies to a write into the caller's own home, which adr-46 decision 4 already treats as the ownership root, so the attestation grants nothing that authority did not hold and costs `ahoy install` no network (adr-38 stands). Rejected: A (a manifest GET on a disk-only verb, re-fetching what the session already proved); C (a documented residual — weaker than it reads, since a harness honouring a committed settings file's environment block lets a hostile checkout set the variable, and the owned-copy claim is what the hook shims trust); the mechanical partial of cross-checking against the plugin-root binary (breaks dogfood installs whose root binary is a local build); and a terminal rung through the attestation alone (it would heal a dogfood checkout's stable symlink into a release copy). adr-46 is superseded by adr-2609151706587280 (never amended, always superseded; retained because its numbered decisions are cited), spc-35 Design 2 step 1 and Design 3 are revised in place and dated, and brief invariant 12 gains the clause.
- 2026-09-09 — The headline product is settled (maintainer, closing the press release's "Product framing after adr-35" open question, both halves). abcd helps a product thinker realise an intent as a high-fidelity prototype or demonstrator, carrying the why from idea to shipped reality: the identity block's story, the one the README strapline and the roles page already tell and the one the roles and product-thinker-surface decisions on the design branch (numbered 0055 and 0056 there, re-minted at merge) are built on. The lifeboat is a key capability of that product rather than its headline, and its widening from whole repositories to a single feature, a lab session or an abandoned worktree enters the press release under the not-yet-real marker, as intention rather than commitment. `disembark probe` is a user-facing command and keeps its place in the press release's scope list. Rejected: the rescue story as headline (it would re-pin the identity block and every surface held to it); adr-35's "read any repository for its theory" as headline (it is the probe's own promise, now one capability among the surfaces); deferring (the audit of the press release against delivered reality had been blocked on this question since adr-35).
- 2026-09-21 — Phases and milestones are retired, and the word roadmap with them (product thinker, at an interactive interview after an independent research pass; adr-2609212115255771 supersedes adr-9). Sequencing is dependencies plus the lifecycle shelves, rendered as a Now / Next / Later status block on the `abcd` board and the site (itd-2609212103568351), never stored; the checkpoint is the derived release plus each intent's acceptance criteria, with an optional `target_release` the cut reports and moves forward (itd-2609212103572513); the unit below a spec is the step, a section not a family (itd-2609212103565953); an issue carries no spec by design; the batch is the run's internal order; one page maps the families (itd-2609211913453478). itd-24 becomes release retrospectives; itd-34 drops its phase rule. Same interview: `build` is the verb a person types and `implement` the loop (itd-2609201916151817 decision 8), the loop takes an issue key (decision 10), only the loop writes a verdict (decision 9); the person's command list is grouped with an agents block behind `--agent` (itd-146), every verb opens with a does/writes/refuses sentence (itd-2609212113220149), modes become flags and five checks one lint (itd-2609212130136102, breaking); the status-line badge is guarded and reset (itd-2609212130146198); a new capture or draft is matched and linked at filing, never refused (itd-2609212137116617); abcd's text names the product thinker or the technical facilitator and never the maintainer (itd-2609212137129937, captured for the run, no sweep today); the lab conventions become `abcd lab` (itd-2609212137128014); a managed repository's site goes live by one verb behind a provider seam (itd-2609061543533170).
- 2026-09-22 — Model routing, after an independent research pass (product thinker, at an interactive interview). Jev (TypeSafe AI, on OpenRouter) is a typed-decision model, not a router: it fits abcd's closed-option judgements and nothing generative. Rulings: a provider adapter serves only the models it lists under a vendor denylist no listing overrides, everything else on the host (adr-2609221009491186); the OpenAI-compatible API adapter is planned with OpenRouter as configuration, a one-time walkthrough at ahoy and the key kept out of the harness (itd-2609081951381895); a decision adapter for the typed judgements runs in shadow as an abcd lab and a research note is the evidence a ruling turns a judgement type on with (itd-2609221009495079); escalation inside a lane is a rule on a failed fix round, recorded on the model tier (itd-2609170822093401); model per role, the pick and consistency are not routing and stand as ruled; itd-17's learned router is superseded. Every external credential abcd holds goes through one store with three homes the person chooses (adr-2609221017021499, itd-2609221017023290); the API adapter and the site setup read through it. The research is filed as research/notes/2026-09-22-jev-and-model-routing-sota.md.
- 2026-09-22 — ideate: abcd-operator-console — verdict reframed. The idea, the three legs, and the rejected alternatives: .abcd/development/research/notes/2026-09-22-ideate-abcd-operator-console.md
- 2026-09-22 — The control-programme idea is REFRAMED, not adopted as put (product thinker, after the ideate gauntlet; research/notes/2026-09-22-ideate-abcd-operator-console.md). Two kill attempts were fatal to the idea as stated: it named the product thinker as the configurer of operator-register material, which adr-2609151528057260 bars, and it made harness-less operation the destination, inverting the host-delegated boundary whose reversal itd-2609201916151817 records as opt-in only. The reframing, adopted: an operator console SERVED BY THE BINARY over local web (no new dependency), which is also the resident process that hosts the opt-in process driver, with a separate product-thinker pane in their own register (open stops, the plain-language verdict, the pace) — the web page itd-167 already decided on; configuration stays operator-side; many repositories waits for a registered-checkout store; a localhost surface that accepts secrets needs a per-launch token; the whole sequenced after the runner and routing intents; and it must carry a falsifier, since no evidence exists that a GUI control plane beats a CLI plus a status page. Ruled with it: where abcd runs as the binary with no host session, the fallback is a HOST THE OPERATOR CONFIGURED (their main connector), never nothing; and EVERY fall back to the host is recorded — a receipt per event and a count per runner and per role in the run record — as intel on what to improve (itd-2609201916056194, planned; itd-22 widened so that "any harness" includes none).
- 2026-09-22 — Two intents the product thinker added for the autonomous run of 2026-09-23, interviewed and planned the night before (both READY, both in the run's batch 1b). A managed repository reports back to abcd (itd-2609221656361680): one verb files a written account against an abcd-issued template with a machine-readable block beside the prose, into an inbox in the user account's machine store, never in either repository's tree; abcd says at its next start how many wait and from how many repositories; a read-only verb renders them naming the sender; nothing is filed until a person or a session acts, and what is filed carries the sender's root-commit fingerprint and a generic description, never its name. Two sessions share one run (itd-2609221656373558): the second joins without a word from the first and the first never waits on it; all three division modes are tried, one per window in order (a claim per record, whole batches, build against review-and-land), because the product thinker wants each measured rather than one chosen in advance; a claim is an atomic create with a lease that lapses; the second session holds at most one lane, may never cut a release or take a lane that recalibrates the reading corpus, backs off on contention with a logged reason, and its stop conditions stop only itself; the run's report compares the three modes on lanes landed, wall clock, collisions and agent minutes. Cross-account and cross-machine agent communication stays out of the run: the four drafts (itd-33, itd-2609151838312703, itd-2609151838327688, itd-2609150819440345) get planning briefs only, with the trust question a shared mailbox raises named in the brief.
- 2026-09-23 — The product thinker ruled on the owed items of the autonomous run of 2026-09-23 in one interview, one question at a time (07:51Z to 09:32Z, while the run was paused): two release blockers, two intents, the 35 major captures routed to the product thinker, and two older owed rulings. The release waits for two things: version-stamped plugin payloads (iss-2609230733497536, built by a run lane, with no criterion amended and no deferral) and a release press release composed at the cut from the shipped intents' press releases, which is a new capability and gets an intent and its planning interview before any lane builds it. The run may approve the release environment itself once every gate is green; that agenda line is written in the release commit, not here. itd-4's AC3 is built as written, the `--issue-drift` check and the rename of the back-links to `related_issues` / `related_intents` together, and spc-6 closes only when both land (the 2026-07-17 ruling stands). itd-43's third criterion is moot and dropped, so spc-8 closes. The guard contract is extended to read a host-supplied per-call working directory (iss-2609212142557657, option A), after a probe of what the host does when none is supplied. Four majors are ruled to a build (iss-2608231120121681 write-path smoke, iss-2608290810036869 preflight before the push connects, iss-2609090828371674 per-agent staging locks, iss-2609120505141653 the provenance comment) and do not hold the tag. The rulings that set a rule each have their own entry below; every other routed major is deferred past v0.9.0 with its ruling as the `deferral_reason`, and the 168 routed minor and nitpick captures take the run's default deferral. The product thinker also asked that the interview be kept as autonomous-run experiment data, towards making a run transparent to the person it serves; the observations feed the run's end-of-run research note. Rejected, as the frame: the run taking any of these rulings itself.
- 2026-09-23 — An autonomous run keeps at most five sub-agents alive at once, in any mix of roles (product thinker, directly to the run A session at 09:09Z; supersedes the run's ceiling of 2026-09-22). A fork of an agent is an agent, so a lane does not fork itself to get under it.
- 2026-09-23 — An autonomous run works in three-hour windows with a two-hour pause after each (product thinker, directly to the run A session at 09:28Z; supersedes two hours on and five off).
- 2026-09-23 — The proxy-gate class joins enforcement-claims-are-facts as one paragraph, not as a sibling principle (product thinker, ruling on iss-2608230847432286; one-canonical-primitive): a gate that runs and reports green is evidence only for the property it measures over the subjects it measures, so a gate that measures a proxy, or exempts the cases that matter, withdraws vigilance just as a phantom gate does, and an exemption from a gate carries the gate's own burden of proof. Detectors for the class are commissioned and need planning; itd-147 already cites the class. The adjacent sampling case (iss-2608230817034768), an assurance nobody issued but a reader inferred, is NOT admitted, because it would remove the class's boundary.
- 2026-09-23 — `abcd lint` becomes a merge-blocking gate on a narrow, high-precision subset of its findings, and every other finding stays advisory in the same run (product thinker, ruling on iss-2608231000561060; the example given is a real personal home-folder path). The facilitator proposes the blocking set, judged against the current privacy-hygiene errors, and the product thinker confirms it. Until the gate runs, the record's claim that the lint gates CI is not a fact.
- 2026-09-23 — The resolution field of a resolved issue gets two detectors, in order (product thinker, ruling on iss-2608250844259345): first a co-edit rule, under which a body change on a resolved record requires a resolution change in the same commit; later, as a second layer, a detector for a resolution that contradicts its body. Both belong to one intent, planned next cycle, for the three unwatched edges of issue resolution, with the inverse open-record detector (iss-2608241612007530) and audit verdicts that never reach resolved issues (iss-2608261635558358).
- 2026-09-23 — The changelog-as-index ruling of 2026-08-26 includes stale closures (product thinker, ruling on iss-2608260941298050): every terminal transition gets a changelog line, principles and ADRs included and `internal` no longer silenced, and so does an issue closed with no fixing commit. Planned next cycle as its own intent.
- 2026-09-23 — Delivered work states the verification rung it was checked at, and a project whose blast radius reaches strangers declares that radius before it may lower the rung (product thinker, ruling on iss-2608290944122400). This answers the question rfc-3 names as the one it most needs answered, and rfc-3 records it; the RFC stays open on its other questions, so there is no ADR yet. The carriers are itd-173 and itd-176, both drafts.
- 2026-09-23 — The cold-reading research workstream lives in a separate fork, not in this repository, so its open rulings are taken there (product thinker, ruling on iss-2609021857343626, the reading of "at the target" in adr-2609021016272867's comparative derivation). That record is closed here as out of scope for this repository, naming the fork as its home. The code that ships here is unaffected by the move.
- 2026-09-23 — The four typed relations decompose-before-filing mandates are all supported (product thinker, ruling on iss-2609091256264547, choosing to build over narrowing the rule or sanctioning prose): `reverses`, `duplicates` and `refines` become real typed record fields in the schema and in every reader, beside `supersedes`. Until that lands, those relations are written in prose, the 2026-09-09 practice, because a typed field the readers do not know hides the record from them. Needs planning.
- 2026-09-23 — Agent guidance that walks a person through a third party's interface carries a verification tag, observed on screen in this session or taken unverified from the vendor's documentation, and asks for a screenshot before a second guess, or reads the page itself where it can; and every redaction rule states its purpose (product thinker, ruling on iss-2609100506256173, adopting both halves). Its home is the managed-repository agent conventions or a principle, still to be written; this entry is the ruling until then.
- 2026-09-23 — "At every refusal, emit what the tool knows" is a principle (product thinker, ruling on iss-2609100509531349, grading the claim that record from the 2026-09-09/10 field run made): a refusal names the allowed values, the value it had, and the condition it checked. It lives at principles/at-every-refusal-emit-what-the-tool-knows.md. Five open records are breaches of it (iss-2608290810037524, iss-2609100506265392, iss-2608270559313719, iss-2609100508570527, iss-2609100508566033), and the record that proposed it is resolved by the promotion.
- 2026-09-23 — A load experiment is trusted only under four conditions (product thinker, ruling on iss-2609210828122412, after orphaned load processes from another repository starved the machine into a crash): background load processes start as one owned process group and are killed together; "clean" is proven by checking what is actually running, never by reading a list of what was started; an experiment on a live development machine needs explicit consent and a cap below the core count; and abcd's own test lanes check machine load before they start and refuse when it is too high, naming what is running. The first three hold as rules now; the last is an intent to plan.
- 2026-09-23 — Four corrections to the entries of 2026-09-23 above that record the product thinker's run A interview, appended because the ledger is append-only and its gate refuses an in-place edit even of an entry that has not reached the default branch. (1) The interview entry leaves out a clause of ruling B4: itd-43 ships in the next release, so the next cut announces it. That stands against AGENTS.md's rule that a record closed for work an earlier release already carried is stamped `shipped_in:` with that release, since v0.2.0's changelog already named the GL002 gate (itd-43); the product thinker's ruling resolves the tension, so itd-43 carries no `shipped_in:` stamp and the next release's changelog names it. (2) In the ceiling entry, the sentence "A fork of an agent is an agent, so a lane does not fork itself to get under it" is the run's own rule for its lanes, not part of the product thinker's ruling. (3) In the proxy-gate entry, a gate that measures a proxy, or exempts the cases that matter, withdraws vigilance more thoroughly than a phantom gate does, as enforcement-claims-are-facts and iss-2608230847432286 say, not "just as" a phantom gate does. (4) In the cold-reading fork entry, the closing sentence "The code that ships here is unaffected by the move" is not part of the ruling and is withdrawn.
- 2026-09-23 — How a release reaches `/plugin update` (product thinker, rulings E1 to E3 of the run A interview, 09:50Z to 10:13Z). This supersedes the interview entry's "no criterion amended" for the version-stamped plugin payloads. E1, the form, 09:50Z: a pinned archive. release.yml renders the plugin payload reproducibly as `abcd-plugin-vX.zip`, the ship pull request commits the marketplace entry with that release's URL and sha256, and release verify refuses a digest mismatch. The product thinker's stated want, in their words: "the user should get the latest cut release from github; verified as such and fingerprinted through a hash". The window between the ship merge and the asset upload, when the listing may name a zip not yet published, is to be confirmed and then closed or documented. E2, the amendments, 09:55Z: itd-67's first acceptance criterion is amended from source `./` to a pinned archive source, and an ADR minted with `abcd decide` amends adr-19 and adr-20 so that the default branch's marketplace.json carries the latest release's URL and sha256 while the plugin.json version stays out of the working tree; both land with the build, each saying why, with no review before the build. E3, the harness floor, 10:13Z: Claude Code v2.1.224, the minimum the archive source needs, is accepted and stated in the install instructions and the release notes; no install-time detection is asked for. E4 owed: whether contributors keep a source-`./` development entry once installs stop following the default branch, and under what name.
- 2026-09-23 — itd-4 AC3 built as ruled (B3): the promote join's back-links are `related_issues` on the intent and `related_intents` on the ledger record, and `abcd intent audit --issue-drift` checks them. Seven implementation decisions the ruling left open, taken by the implementing lane. (1) No reader tolerates the retired `promoted_to` / `promoted_from`: the ledger reader refuses and skips a record carrying `promoted_to`, the intent reader ignores `promoted_from`, and a promote write meeting either refuses. The refusal, the record gate's finding and the drift finding each name the successor and the remedy, `abcd capture migrate --apply`, a verb that reports by default, writes only with `--apply`, and is idempotent. It is a verb and not a one-off script because managed repositories hold the same retired names and need the same repair. (2) "Promoted" is the PAIR, an intent in the record's `related_intents` that names the record back in its `related_issues`, because an issue's `related_intents` already carried loose relations before the rename and still does; the double-promote refusal and the record page's next move read the pair. So the migration completes a join an older abcd wrote from one end only (iss-327 and itd-93, iss-2609091642508005 and itd-2609111003026787), or those records would stop reading as promoted. (3) Link mode on the issue route writes both halves; before the rename it stamped the issue alone, which left a hand-filed draft with no edge back. (4) `related_issues` holds `rdi-N` as well as `iss-N`: a reading item lives in the issue ledger's tree and graduates the same way, and one key per side keeps the join one shape. The first entry is the record the intent was promoted from; a later link appends, so `back_edge_kept` still reports the first. (5) The drift check reports an intent naming a record that does not name it back, but not an ISSUE naming an intent that does not name it back, since that is a loose relation; a reading item carries no loose relation, so from its end a one-way stamp is drift. A shipped intent naming an issue in `wontfix/` is drift, like one in `open/`, because the itd-4 scope asks whether the issue "actually moved to resolved/". The check is not wired into preflight or CI; `--strict` is there for whoever wires it. (6) The historical passages in shipped intents, closed specs and adr-0057 carry a `(historical)` marker beside the retired names, the precedent `development/activity` set, so the new `retired-promote-back-links` ban can gate every current record. (7) The impact is `breaking`: itd-4 declares none, and the rename removes `promoted_to` from `capture list --json`, `promoted_from` from the intent JSON and both link keys from the record page, and makes an unmigrated ledger unreadable until the migration runs.
- 2026-09-23 — Two corrections to the entry above recording itd-4 AC3 built as ruled, appended because the ledger is append-only. (1) Decision (5) says the drift check is not wired into preflight or CI. That contradicts the criterion it delivers, which says drift detection is ENFORCED, so it is wired: `make issue-drift` runs `abcd intent audit --issue-drift --strict` as a `make preflight` prerequisite, and the CI `check` job runs the same line on its Linux leg. A promote join that stops reading the same from both ends now fails the pre-push gate and CI. (2) Decision (6) is narrowed: the `(historical)` marker goes on the authored passages, never inside an INGESTED Audit Notes block, because an audit receipt is written only by the ingest. The `retired-promote-back-links` ban exempts those receipt lines by the line shapes the ingest renders (a per-criterion verdict, a scope-condition disposition, an evidence pointer), and it exempts the two gap-audit claims that name a retired key by their own text, since a gap-audit claim is a bare nested bullet and exempting that shape would also exempt live prose.
- 2026-09-23 — The two rulings that close how a release reaches `/plugin update` (product thinker, rulings E4 and E5 of the run A interview, 10:23Z and 10:27Z), recorded beside the build they govern; the E1 to E3 entry left E4 owed. E4, the contributor entry, 10:23Z: there is no second, development entry in the marketplace catalog. The public listing carries the pinned release archive alone, and contributors load the plugin from their own checkout, a step documented for contributors only (`CONTRIBUTING.md`), never in the user docs. E5, binary upgrade integrity, raised by the product thinker in E1, 10:27Z: `abcd update` verifies a downloaded binary only against the same release's `checksums.txt`, which catches corruption but not a release page whose binary and checksums were both replaced. A check independent of that release, such as a digest committed in the repository's history or a signed build attestation (a new verification dependency needs sign-off), is next-cycle work; this release keeps the same-page check. It is captured as iss-2609231050273096, deferred past v0.9.0 with this ruling as its reason.
- 2026-09-23 — The one sanctioned mention of abcd in an adopted repository (product thinker, ruling F of the run A interview, 14:38Z, on iss-2609231103413459). The 2026-09-11 ruling that `ahoy install` must not write abcd by name into a repository it adopts (carried by iss-2609110944498549) covers the name-guard hooks (`.githooks/pre-commit`, `.githooks/pre-merge-commit`) and the `.gitignore` fence, and they are its sanctioned exception: both keep their markers and their naming, documented as the one allowed mention of abcd in an adopted repository, because the hooks run the binary and the fence tells people not to hand-edit it. No rename and no migration: detection classifies an adopted repository by exactly that hook marker and fence, so a rename would have to migrate every repository already adopted. The product thinker ruled the markers, not abcd's record ids: the hook templates' citations of abcd's own records (itd-74, spc-20, itd-150, iss-370, and a path into the design record) resolve to nothing in an adopter's repository, so they are dropped from the templates and a test keeps them out. iss-2609231103413459 closes as wontfix on this ruling.
- 2026-09-23 — Three rulings from the product thinker's run A interview, 14:38Z to 14:41Z, on items F and G. F, abcd's name in an adopted repository's two name-guard hooks and its `.gitignore` fence (iss-2609231103413459): the 2026-09-11 ruling that the install does not write abcd by name into a target repository (iss-2609110944498549) covers them, but they are its sanctioned exception. The hooks run the binary and the fence tells people not to hand-edit it, so both keep their markers and naming as the one allowed mention of abcd in an adopted repository, with no rename and no migration; whether the record ids in the hook prose count as part of that mention is left to check, since the ruling named the markers, not the ids. G1, the em-dash-in-list-item house-style token in a prepared repository's docs lint: it is offered at install and the adopter chooses whether it blocks or warns, in the product thinker's words "It's offered at abcd install. the user chosen: Either blocking or warning"; not "off". An unattended (`--yes`) install seeds it as a warning (the 14:41Z follow-up). The run settled what the ruling left open: a bare Enter or end of input takes the displayed default, the warning; an answer naming neither choice, such as the `y` a piped `yes` sends, seeds the warning and the install result says what was heard rather than guessing a gate; the chosen severity is recorded as the token's severity in the seeded config itself; and the question is asked only when that config is being created, so a repository with its own config keeps its severity. G2, `docs lint` with nothing checked: warn, do not fail. The exit status stays 0, since an exit 2 would turn red the CI of every repository prepared before its config carried rules, and the run prints a loud warning that nothing was checked and why (no rule armed, or roots that hold no markdown document), on stderr and in the JSON envelope.
- 2026-09-23 — The load check before abcd's own test lanes warns and never refuses (product thinker, planning interview for itd-2609231434459890; spc-2609231542463113, Decision 2). This changes the fourth condition of the run A interview ruling on iss-2609210828122412, which had the check refusing. The check names foreign sustained CPU (a near-full core for longer than 30 minutes) and an extreme load (four times the online cores), both limits overridable in the caller's `~/.abcd/load-limits`; it names the caller's own strays with a remedy and counts other accounts' processes without naming them; it writes a run-log event; it runs once at the start of `make preflight` and of the eval harness; CI skips it with a stated reason; it always exits 0, and its findings are minor.
- 2026-09-23 — An autonomous run keeps at most six sub-agents alive at once, in any mix of roles, from the run's session 3 on (product thinker, directly to the run A session at 14:10Z; supersedes the ceiling of five recorded above at 09:09Z).
- 2026-09-23 — The payload example in spc-2609231435545473 (closed) is stale on one point: its itd-121 quote opens `"... what I'd do next,"`, a truncation the release-page ingest refuses by design, since a quote is a whole quoted sentence as the press release has it. The example's verb is right: the ingest accepts an attribution after `says` as well as after `said`, because six intents of that release (itd-119, itd-120, itd-121, itd-123, itd-124, itd-125) quote as `says <Name>, <role>.`. The closed spec stays as written; `TestQuoteSaysFormFromARealRecord` carries a working `says` quote, taken from itd-121 as its record has it (lane implementer, on review 2 of the release-page build).
- 2026-09-24 — An autonomous run works continuously, with no pauses, and keeps at most four sub-agents alive at once, in any mix of roles (product thinker, directly at 02:02Z, relayed to the run A lanes by the run's first orchestrator session). There are no sleep windows: this supersedes the three-hour windows with a two-hour pause recorded above at 09:28Z on 2026-09-23. The ceiling of four supersedes the ceiling of six recorded above at 14:10Z on 2026-09-23; a fork of an agent is still an agent. The orchestrator still rotates to a fresh session at about 60% of its context.
- 2026-09-24 — Correction to the entry above on the 02:02Z ruling: the clause "a fork of an agent is still an agent" is not part of the product thinker's ruling. It is the run's own lane rule for autonomous run A, which counts a fork toward the ceiling because a fork is an agent alive. The ruling itself is the rest of that entry: no pauses, a ceiling of four sub-agents in any mix of roles, and rotation at about 60% of context unchanged (lane implementer, on review of the followup lane; the ledger is append-only, so the entry above stands as written).
- 2026-09-24 — Release v0.10.0 is cut by autonomous run A, and the run's agenda line is: approve the publish step. Under ruling A2 of the product thinker's run A interview (2026-09-23 07:52Z, "approve the publish step": the product thinker authorises the run to approve the release environment itself once every gate is green), the run approves the `release` environment's deployment of v0.10.0 only after the merge queue, the verify job and every other gate on the tagged commit report green, and stops with a handover instead if any does not. The cut: v0.10.0, impact breaking, 43 records since v0.9.0 (nine shipped intents, thirty-four resolved or declined issues, four of them breaking), content commit 64ea8f62, the composer's payload accepted on the first ingest and recomposed once for two docs-currency findings. Both semantic gates ran at tier full: docs-currency-reviewer (Fable 5.1) with three findings, two fixed and one deferred because the load check's intent stays planned pending the product thinker's ruling on the stray definition (iss-2609231947544298); the brief-surface cross-check (40 pinned checkers, Opus 5.5, four at a time under the run's ceiling) with 154 findings, all deferred to their records: four user-facing ones captured as iss-2609240519413467, iss-2609240519418856, iss-2609240519471816 and iss-2609240519427388, one inside an appendix chapter captured as iss-2609240519422232, which records itd-147's ac-6 as not met, and the design-record drift to the systematic brief pass iss-2609091956001547.
- 2026-09-24 — v0.10.0 is published. PR #693 merged as 1ac8b3a0, and auto-release run 35963282477 tagged it. The `release` environment was approved under ruling A2 once all 30 checks on the tagged commit had settled (25 success, 5 skipped by design). The release was published at 2026-09-24T06:33:10Z with four binaries, `checksums.txt`, the plugin archive `abcd-plugin-v0.10.0.zip` (its sha256 equals the marketplace pin, 1ab2acd1…) and the rendered site, which was deployed. Verified locally afterwards: the darwin-arm64 binary's checksum and its build attestation, and the binary reports v0.10.0.
- 2026-09-25 — A push is gated before it opens its connection, by receipt (iss-2608290810036869, iss-2608210738378295; product thinker's ruling M16 of 2026-09-23, "check before connect"). The committed `.githooks/pre-push` no longer runs `make preflight`: git opens the connection before the hook, and a preflight inside it outlasted the transport's idle timeout, so a push reported success and moved nothing. `make preflight` ends by minting a receipt for HEAD under the checkout's local tier (`scripts/preflight-receipt.sh`), and only when the working tree matched HEAD (nothing staged, unstaged or untracked) both when the run began, read while the Makefile is parsed, and when it ended, with HEAD unmoved, so the gates read the tree CI checks out. The hook refuses a push whose commit is new to the remote and carries no receipt from any worktree of the repository; a commit the remote already holds (a tag on a merged commit) passes. Alternatives not taken: a push wrapper that runs the preflight and then `git push --no-verify` (it normalises `--no-verify`, and a plain `git push` would go ungated); keepalive settings on the transport (the hook would still hold the connection for ten minutes); a preflight on a clean export of HEAD (a second full tree and build per push, where refusing a divergent tree costs nothing). `git push --no-verify` skips the hook exactly as before, and CI stays the authority (lane hooks, autonomous run A).
- 2026-09-25 — Reach of the push-receipt entry above, stated after review (iss-2608210738378295). The receipt's clean-tree test is `git status` read when the preflight begins and when it ends, and a tree `git status` cannot see is not vouched for. So `scripts/preflight-receipt.sh` also refuses to mint while any tracked file is flagged skip-worktree or assume-unchanged (`git ls-files -v` tags it `S` or in lower case), because either flag hides that file's edits from the status read; a sparse checkout sets skip-worktree and so never mints. What the receipt still cannot see, stated in the script's header: files git ignores, which are outside the commit yet can be read by a gate (a `go.work`, which `.gitignore` lists, changes every Go gate's module resolution); HEAD moved and moved back, or the tree changed and restored, between the two reads; and anything a gate reads from outside the checkout. The receipt stays a local convenience gate and CI the authority; closing those would mean running the gates on a clean export of HEAD, the alternative the entry above did not take (lane hooks fix round, autonomous run A).
- 2026-09-25 — The layered configuration resolver has one home, `internal/core/layered`, and two file families; every later consumer reads through it rather than opening a file of its own (lane implementer, autonomous run A, on spc-2609180535002478 part 1). Precedence is flag, then repository, then machine, then bundled, and every value comes back with the layer and the origin that supplied it. The families are `layered.Config`, which is `.abcd/config.json` in the checkout the session resolved and `~/.abcd/config.json` on the machine, and `layered.OracleRouting`, which is `.abcd/config/oracle-routing.json` and `~/.abcd/oracle-routing.json`, the files itd-2609170822093401 names. Scalar keys go in `Config` under a namespace: `pace.*` (itd-2609201925079472), `oracle.review` (itd-6), `roles.<role>.runner` (itd-2609201916056194) and `match.threshold` (itd-2609212137116617). The per-agent routing table keeps its own file, as its intent names, and no new file family is added without an entry here. Loudness: an absent file is an absent layer; a present file that is malformed, symlinked, carries a key twice, trails content, declares the wrong `schema_version`, or (machine layer) is not the caller's own owner-only-writable file is an error naming it; a consumer claims its namespace and every key in it (`Stack.Claim`, with `*` for an open segment such as the role name), and an unknown key under a claimed namespace is refused. Because ahoy writes `oracle.backend` into the shared file, whoever claims `oracle` lists `backend` too. A winning value that does not decode or fails its check is refused, and nothing falls through to a lower layer or to the default.
- 2026-09-25 — Two rulings for the model-tier routing table, which the spec leaves open (lane implementer, autonomous run A, on spc-2609180535002478 part 1). First, a table is accepted when a routing file exists at the repository or the machine layer. Only then does the bundled proposal fill in an agent the table has no row for; with neither file, every agent resolves to `none`, the harness at `host-decides`, as AC 1 and the spec's criteria section say. A `--route` alone accepts nothing: it overrides the one agent it names for one run. Second, no agent contract under `agents/` declares a fan-out ceiling, and every agent in the roster is a single prompt that spawns no sub-agent, so each ceiling is 1. The proposal carries it (`oracle.Ceiling`), and a row's `fan_out` above it is reported and clamped. The roster test holds the proposal to `agents/`, so an agent that gains a ceiling field moves the number there.
- 2026-09-25 — The load check's stray rule is "busy for its share" (ruling H1, the product thinker via the interview session, 07:57Z, on iss-2609231947544298). A long-running process outside abcd's lanes is a stray when it uses nearly all the CPU it could get on the machine as loaded: its lifetime CPU share is measured against its fair share, the online cores divided by the runnable demand, not against a fixed 0.9 of one core, so forty busy loops each at a fortieth of the machine all count. The share test applies to the caller's own processes and to other accounts' alike, and other accounts' strays stay counted only. It is not a second sample and not a summed-cores trigger. The build reads the runnable demand as the snapshot's one-minute load average and caps the fair share at one core, so on a machine loaded no higher than its cores the rule is the near-full core it was (`machineload.FairShare`, spc-2609232027132755). This closes the band between 1.125 and 4 times the cores in which the check said nothing (pinned by `TestStrayRuleSilentBand`, succeeded by `TestStrayRuleCoversTheOversubscribedBand`), and lets the load check's remainder spec close and itd-2609231434459890 ship. The check still warns and never refuses (the 2026-09-23 entry above on that intent).
- 2026-09-25 — Autonomous run A defers every open capture routed to the product thinker out loud to v0.10.0, under the product thinker's directive of 2026-09-25 ("I want the ledger drained": a capture ends fixed, wontfix with its reason, closed as a duplicate, or deferred out loud where it needs a product-thinker ruling or a planning interview). The product thinker is away, so no one in the run can give those rulings. 190 records each carry `deferred_after: "v0.10.0"` and a `deferral_reason` that quotes the ruling owed verbatim. 184 of them renew a v0.9.0 grant that lapsed when v0.10.0 re-anchored, and 6 carried none. Every question is asked once in the run's rulings-owed list, grouped under the routing pass's eleven themes: A, planning interviews already ruled "plan next cycle" (27); B, confirmations owed on rulings already given (8); C, dependency and publish sign-offs (6); D, narrowing a shipped promise (4); E, principles and conventions to adopt (23); F, record schema and lint rules (34); G, security and trust design forks (16); H, autonomous runs, implement and multi-agent planning (22); I, site, docs voice and product story (17); J, future capabilities to plan or close (27); K, parked on a trigger, or a human act outside the tree (6). Within each theme the questions covering a major record come first, and the list is the agenda for the next interview. The same pass closes 9 duplicates and 44 captures on their recorded merits, so none of those is deferred (implementer of lane records1).
- 2026-09-25 — On the auto-release path the tag is still made before the `release` environment's approval, and that residual is recorded for the next cycle rather than built now (finding F2 of the workflows-lane review, LOW). release.yml's `tag` job needs `verify`, so a refused gate leaves no tag (iss-2608231226347380), but the job runs before the publish job waits on its human approval and before the four steps between the tag and `gh release create` (the vcs stamp, the archive re-verify, the two attestations). A rejected approval or a red step there leaves a tag with no Release, which the heal path then rebuilds on the next push. The closure shape: delete the `tag` job and create the tag as a step inside the `release` job, immediately before `gh release create`, through `gh api -X POST repos/<owner>/<repo>/git/refs -f ref=refs/tags/<tag> -f sha=<commit>` with the job's own `GH_TOKEN`, so it needs neither `persist-credentials` nor an extra job; a rejection or a red publish step then leaves no tag either. It also removes the auto-path half of iss-2609251125599536: once no auto-path failure can leave a tag without a Release, the only such tag is a hand-pushed one, which `detect` already refuses to rebuild when its own verify failed. Not built in this lane because the chain is unverifiable without a live release, and the order the lane shipped is the one the review walked (implementer of the workflows fix round).
