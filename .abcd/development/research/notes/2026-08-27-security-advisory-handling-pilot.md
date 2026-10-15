# Security-advisory and issue handling — experiment record (2026-08-27)

A hand-run of an inbound-security-handling protocol, executed per the
script-first rule: run the smallest documented protocol by hand, grade where it
creaks, and only then file the automation intent that cites this note. It is the
sibling of the forge-mirror pilot
([`2026-08-19-forge-mirror-pilot.md`](2026-08-19-forge-mirror-pilot.md)) — same
ladder, different corpus.

This note is method-focused by design: it records the *procedure* and its
grading, not per-advisory exploit detail. The advisories are unpublished while
they are triaged and fixed, so their specifics stay out of this committed record
until they publish.

## Hypothesis

The end-to-end handling of inbound security reports — private advisories, public
issues, and the working-tier ledger together — is runnable by hand with the
existing surfaces (`gh`, the host agent, `abcd capture`, worktrees), and a
bounded run surfaces where the protocol creaks before a script hard-codes the
creaks. The target end state is a deterministic, SOTA-aligned automation that
works for every abcd-managed repo.

## Protocol as run

1. **Ingest.** Enumerate the three trackers: private advisories (`gh api
   repos/{o}/{r}/security-advisories`), public issues (`gh issue list`), and the
   `.abcd/work/issues/` ledger.
2. **Verify before fixing.** One independent reviewer per advisory (host
   security-reviewer), prompted to *refute* the claim against current `HEAD` —
   CONFIRMED / ALREADY-FIXED / REFUTED — with a file:line evidence pointer and an
   explicit scope correction where the report over- or under-claims. Evaluator
   outside the loop.
3. **Dedup the trackers.** Match public issues against the ledger by defect
   (file + root cause), not wording. Close stale mirrors already resolved in the
   ledger; backfill genuine ledger gaps; leave overlaps with private advisories
   out of the public ledger.
4. **Triage by exploitability, not reporter severity.** Rank by who can pull the
   trigger (a hostile clone attacking the abcd user outranks the operator
   polluting their own release), then by blast radius.
5. **Fix.** One worktree-isolated agent per fix-family, test-first (watch the
   regression test fail on `HEAD`, pass after), the canonical primitive reused
   (never a third copy), and the BUGHUNT sibling sweep (grep the pattern, not the
   instance). Agents never touch the ledger — the parent owns records.
6. **Review before the gate.** An independent adversarial pass per fix branch
   (evaluator outside the loop) confirms completeness (all siblings swept),
   correctness, and no over-reach, before any push.
7. **Gate.** The human adopts; verifier selects, gates decide. Push + PRs, with
   any unpublished-advisory id genericised out of public commit/PR trailers.
8. **Credit, differentiated by channel.** A security advisory's reporter is
   credited via the advisory `credits` field (type finder/reporter) — opt-in,
   they accept, and only on acceptance does public credit show; never
   `ACKNOWLEDGEMENTS.md`, and no public commit/PR trailer names them before they
   consent. A merged PR credits its author natively through the contributor graph
   and commit authorship (a contributor, not a write-access collaborator). A
   public issue's external reporter is credited by citing them in the fixing
   commit — `Reported-by: @handle` plus `Fixes #N` — stamped only when the issue
   author is not the operator's own account, never a self-credit. Post a
   name-free triage acknowledgement in each advisory thread within the
   SECURITY.md week. `Assisted-by` is stamped from the actual per-commit authoring
   model, never a guessed or intended one (F-G).
9. **Record & automate.** This note grades the run; the automation intent cites
   it.

## Findings (living — graded as the run proceeds)

- **F-A — recall-before-record must be explicit, not assumed.** A capture made
  mid-run duplicated an existing drafted intent and was caught only *after* the
  write. Once the agent adopted an explicit ledger recall-check before every
  capture, two subsequent captures were correctly identified as already-tracked
  *before* recording. Until the capture-time validator ships, the documented
  protocol (recall-check first) is the gate. Product fix captured
  (`iss-2608270550201935`: a non-blocking advisory-dedup echo in `capture`).
- **F-B — verification catches reporter drift both ways.** Independent
  refutation surfaced scope corrections a fix would otherwise have over-applied:
  one advisory had only one of two cited sites live-exploitable (the other dodged
  by a flag); one carried a delivery caveat (a plain clone does not import the
  remote config the exploit needs); one public issue was the same fix as a
  private advisory. Fixing to the *verified* claim, not the reported one, is
  load-bearing.
- **F-C — worktree isolation + test-first scales the fix stage.** Independent
  fixes ran in parallel, each with its own watched-fail regression and package
  race/vet/gofmt gate, without interfering.
- **F-D — collision management is the open cost.** Parallel agents on shared
  files (one secret-pattern file, one guard matcher, one launch bundler) must be
  grouped by family or sequenced, and *landing many branches into one cut is the
  unsolved integration step* — the automation needs a defined merge strategy
  (integration branch vs ordered merges) rather than N independent PRs.
- **F-E — three trackers, no id linkage.** The dedup pass found the public-issue
  and ledger populations almost entirely disjoint, and the private advisories are
  a third, unlinked set. A deterministic automation needs one unified view keyed
  on the defect, or the same bug is worked twice.
- **F-F — agents stay out of the ledger.** Keeping record/credit writes with the
  parent avoids parallel id-mint races and keeps resolution riding the fixing
  commit (RS001) under the parent's control.
- **F-G — attribution must read the running model, not any self-report, and
  credit must name the true reporter.** Two concrete failures this run. (1) The
  session stamped a commit from its templated identity blurb (Fable) and had to
  be corrected against the operator's actual model indicator (Opus 4.8). (2)
  Fix subagents launched with a `model: opus` override *intending* Opus 5 in fact
  ran on `claude-opus-4-8` — the transcripts' own `"model"` field is the proof —
  yet each wrote `Assisted-by: Claude:claude-opus-5` into its commit because the
  brief said so. The override intent, the brief, and the session blurb were all
  unreliable; only the per-agent transcript model field was authoritative. The
  automation must therefore stamp `Assisted-by` from the **actual per-commit
  authoring model**, and credit the reporter correctly by channel: advisory
  reporters via the GHSA `credits` field (opt-in), a public issue's external
  reporter via `Reported-by: @handle` + `Fixes #N`, the operator's own routine
  via `Fixes #N` alone — never a self-credit, never a guessed model.
- **F-H — external contributions need provenance-aware intake, not a shortcut.**
  Inbound issues (and PRs) split by author: the operator's own bug-hunt routine
  arrives already classified and labelled, while external contributors' issues
  arrive with *no* classifiers (no severity, no ledger record) and from a person,
  not a routine. Respect for a contributor means their issues, PRs, and
  advisories are **prioritised, acknowledged, and credited** — it does *not* mean
  verification is skipped: a known, trusted contributor's report still gets the
  same independent verify-against-`HEAD` as any other, and nothing is rushed
  (trust sets priority and courtesy, not the evidence bar). The intake step is
  therefore provenance-aware: (a) acknowledge + prioritise, (b) verify the claim
  against `HEAD`, (c) assign the missing classifier — severity by exploitability,
  (d) dedup against the ledger and record it, marking external provenance, (e)
  credit the contributor (issue/PR credit, advisory `credits` field — opt-in,
  never `ACKNOWLEDGEMENTS.md`). Two gaps this run exposed and that the automation
  must close: the ledger `source` enum has no external-contributor value, so
  provenance is currently unrecorded; and nothing links an external issue → its
  ledger record → the contributor's credit.

- **F-I — a fix has four roles, and a multi-maintainer repo must not conflate
  them.** Reporter (who found it), AI assistant (which model), author of record
  (the accountable human who drove it — the git author), and authoriser (the
  maintainer who reviewed and approved the merge). A single-maintainer repo
  collapses author and authoriser into one person, so three signals suffice
  (`Reported-by` / `Assisted-by` / git author). With multiple maintainers the
  authoriser is distinct: GitHub captures it natively through required PR review
  (the approving reviewer) plus `merged-by`, enforced by branch protection.
  **Decided approach for abcd:** keep required review (authorisation is enforced
  and captured), and the merge step stamps `Reviewed-by: Name <email>` from the
  approving review for git-native durability, so one commit carries author /
  `Assisted-by` / `Reported-by` / `Reviewed-by`. `Reviewed-by` is not the
  `Signed-off-by` adr-43 dropped — it attests review/authorisation, not origin,
  and a "require review from someone other than the author" branch rule makes
  self-authorisation impossible. Two implementation seams: the branch-protection
  config (require a non-author approval) and the merge automation (stamp
  `Reviewed-by` from the PR's approving review). The automation stamps all four
  roles deterministically.

- **F-J — agent-surfaced siblings need a flag-then-parent-capture intake.** A
  fix agent deep in one file routinely spots an adjacent defect out of its scope
  (an unhardened sibling call, a nitpick, a follow-up). Losing those loses the
  most valuable context: the finding seen in situ. In this orchestration the
  agents are barred from the ledger (to avoid parallel id-mint races and keep fix
  commits atomic), so they FLAG the sibling in their report and the PARENT
  captures it, serialised through one writer, recall-checked, and fixed in a
  SEPARATE change (record first, never widen the fix's scope). Six such siblings
  were captured this run (installsurface `dirTree.resolve` guard, guard
  `noglob`/`nocorrect` wrappers, the #357 block-sequence remainder, the memory
  write-time control-char path, the `.lock` skip-extension, the site lint-config
  read). The automation formalises this agent-to-parent capture channel; the
  capture-time dedup echo (iss-2608270550201935) then makes the parent capture
  safe from duplicates.

## Integration review (live log)

The step F-D named as the open problem: landing many independent fixes into one
cut. Recorded in the detail the automation needs.

**Pre-flight of the integration itself:**
1. **Peer scan before any git-state change.** `ListAgents` shows one idle peer
   session (`abcd-cli-7c`); worktree creation churns the shared checkout (the
   iss-2608261331317889 window), so the integration runs in its own worktree and
   the peer is watched.
2. **Working-tree hygiene.** The primary checkout carries ~27 uncommitted paths
   (this note, itd edits, the session's captures, peer WIP), so the integration
   branch is cut from the clean committed `main`-HEAD in a DEDICATED worktree,
   never from the dirty tree, so none of that leaks into the cut.
3. **Footprint map to merge order.** Each branch's `git diff --name-only
   main...branch` gives the conflict map. Disjoint-file branches merge first;
   overlapping branches merge consecutively so a shared file is resolved once.
   Predicted conflict files: `guard/match.go` (#318 + guard-family),
   `scanner/scanner.go` (#370 + #328) and `patterns.go` (#358),
   `launch/bundle.go` (h2gm + #328) and `render.go` (#328 + #488),
   `memory/ingest.go` (72rp + j5f5) and `ask.go` (72rp + #250/262),
   `ahoy/store.go` (xrf8 + #334), `cli.go` (#250/262 + #485). #328 is a superset
   of the launch-payload branch, so that branch is not merged separately.

**Executed:**
- **Base:** `integration/security-cut` cut from clean `main`-HEAD in a dedicated
  worktree; base confirmed clean (no uncommitted leak).
- **Merge order:** disjoint standalone branches first (4), then first-of-cluster
  and disjoint (8), then the 7-branch overlap cluster last.
- **Conflicts:** of 19 merges (the launch-payload branch folded into its #328
  superset), **18 auto-resolved** — git's 3-way merge handled non-overlapping
  hunks even in shared files (`ingest.go`, `ask.go`, `scanner.go`, `bundle.go`,
  `render.go`, `cli.go`, `store.go`). Exactly **one** textual conflict:
  `guard/match.go`, where #318's `coproc` branch and the guard-family's
  case-folded wrapper lookup edit the same line in `commandOf`. Resolved
  **keep-both** (both hardenings coexist); guard package rebuilt and tested green
  before committing the merge.
- **The payoff — a finding only the tree-wide suite could surface (this is why
  F-D matters).** `go test ./...` on the integrated tree failed
  `TestTestGitCallsAreHermetic` (iss-28): the GHSA-h2gm fix's new
  `fsmonitor_exec_test.go` deliberately spawns git with a hostile repo-local
  `core.fsmonitor`, which the hermetic-git gate forbids. It passed in the fix's
  own `go test ./internal/gitutil/...` because the meta-gate lives in a different
  package and only runs tree-wide. **No per-branch review could have caught it.**
  Resolved with a file-precise allowlist entry (matching that one file), mirroring
  the existing lifeboat exemption.
- **Gate on the candidate:** `go build ./...`, `go vet ./...`, `go test ./...`,
  `go test -race ./internal/...`, and `gofmt -l .` all clean.

**Still to do before the cut (recorded so the automation covers them):** full
`make preflight` (the lint gates: lint-reviews / lint-issues / record-lint /
docs-lint / site-render); one consolidated adversarial security review over the
whole integrated diff (evaluator outside the loop); trailer reconciliation
(correct every merged fix commit's `Assisted-by` to `claude-opus-4-8` and add
`Reported-by: @jogrun` to the four jogrun fixes — F-G/F-I); and the
changelog/resolution reconciliation (the security issues' ledger records, minted
on the unmerged backfill branch, must resolve in the cut for the derived
changelog to describe it).

**Cut — Option A (chosen 2026-08-27).** Bring the fixes and their resolved
records into one tree, then ship. Step 1 (land the backfill so the records exist
on `main`) was already done — PR #522 had merged, so `main` carries the 31
backfilled records and, crucially, NO Go code changed since the cut's base, so
the sync is conflict-free. The cut branch is synced to `main`, then each security
fix's record is resolved (`impact: fix`, a patch bump) — the ~15 backfilled ones
resolved in place, the five deliberately-unbackfilled public issues (#324, #328,
#335, #486, #487) and the private GHSAs minted-then-resolved with a hardening
description and no exploit detail. Preflight re-runs (RS001 now validates the
`Resolves:` trailers); the cut PR merges; `abcd launch ship` derives the version
and composes the changelog from the resolved records; the GHSAs then publish with
opt-in credits. Release retention (itd-70 / `retention.go` `ComputeRetention`:
newest-patch-per-`MAJOR.MINOR`, never prune the just-published) prunes the older
patches of the published line at ship time.

- **F-K — a derived release ships the whole unreleased backlog, so a "scoped"
  (security-only) cut is not achievable while other resolved work is
  unreleased.** The cut derives from every record resolved since the last tag. At
  cut time that was 95 records (70 `fix`, 24 `internal`, 1 `additive`), of which
  only 29 were this session's security fixes — the other 66 were a pre-existing
  backlog of resolved-but-unreleased work from prior sessions. abcd has no lever
  to ship a subset (`shipped_in` only excludes already-released work, which these
  are not). The lesson for the automation and the operator: a security release
  cannot be isolated from the release backlog — either the backlog ships with it,
  or the security fixes wait. Pragmatic resolution taken here: stop treating the
  cut as security-only, run the deferred docs/correctness sweep first, and ship
  one combined release. Also recorded: the derivation refuses when the running
  binary's surface snapshot mismatches the tree (a stale-binary guardrail) —
  rebuild from the tree being released before deriving.

- **F-L — proactive agent-to-agent coordination must be a default, not
  human-prompted.** Two concurrent sessions (this security cut and the
  docs/inconsistency sweep) had interdependent work heading to ONE combined
  release, yet coordination happened only after the operator prompted it twice —
  the human was acting as the relay between them. AGENTS.md's concurrency rule
  already says to "announce the mutation to any peer found", but that was not
  self-invoked: peers were repeatedly scanned (`ListAgents`) without ever being
  messaged. The lesson: when a session detects a concurrent peer on the same
  repo/release, it should proactively open a coordination channel over
  `SendMessage` — current state, constraints, the shared-release decision,
  sequencing — the moment the interdependence is visible, not when the human asks.
  The automation makes peer coordination a reflex triggered by a detected
  interdependent peer, and the convention graduates from "announce a mutation" to
  "coordinate interdependent concurrent work by default".

- **F-M — enabling a security control mid-flight caught the security work's own
  test fixtures.** Turning on GitHub secret-scanning push protection (a posture
  change made this session) immediately blocked the cut's push: the
  redaction/scanner fix TESTS embedded literal secret-shaped tokens (fake
  `ghp_`/`AKIA`/JWT) needed to exercise detection, and push protection cannot tell
  a fake from a real one. Two lessons. (1) Self-validating: the control worked on
  day one — it caught real-looking tokens the moment it was armed. (2)
  Fixture hygiene: a test that feeds a secret-shaped input to a detector must
  CONSTRUCT the value at runtime (`"ghp_" + "0123…"`, split each `eyJ`), never
  embed a literal, or it trips the very control it helped ship — and CI
  `gitleaks`. The fix was a tree-filter splitting each literal across the cut's
  history (the fix/ branches kept as a fallback); the repo's own
  `storeEntropySpecimenGolden` already models the convention. The automation's
  fix-agent brief must require runtime-constructed secret fixtures. (Also logged:
  a transient push-pack transfer corruption that a repack cleared, and 4 dangling
  corrupt objects the repack surfaced — pre-existing, unreachable, harmless, prune
  later.)

## Second operator, release cut, and the gate as security auditor (F-N…F-W)

### F-N (verified) — auto-merge is the robust default; the human click is ceremony

Verified against `main`, not asserted from memory:

- **Arm only after the adversarial verdict.** Auto-merge was armed ONLY after the
  consolidated `security-reviewer` returned APPROVE over the whole assembled diff.
  Arming before the verdict races a merge against a review that could still say
  NEEDS_WORK.
- **The queue owns the method; `--merge` is cosmetic.** `gh pr merge <n> --auto
  --merge` returns `! The merge strategy for main is set by the merge queue` and
  ignores the method flag. The queue picked **merge-commit** on all three cuts:
  #523 = 30d6a324 parents=[6961c6d1 e8ab2114], #524 = c3a65258, #525 = f3be1f47,
  each `Merge pull request #N`. `mergedBy = REPPL` — the queue attributes the
  merge to the human who armed it, preserving author-of-record.
- **Arm-vs-enqueue is a state gotcha.** `--auto` only ARMS while the PR is BLOCKED
  (checks pending); on an already-CLEAN PR GitHub returns `autoMergeRequest:null`
  and does not arm — it is simply mergeable, so `gh pr merge <n> --merge` enqueues
  it directly. Observed exactly: #525 BLOCKED→armed→landed on green; #524
  CLEAN→not-armed→enqueued directly. Read `autoMergeRequest` to know which state.
- **RS003 reachability survives *because* the method is merge-commit.** A merge
  commit preserves both parents, so every branch SHA cited in a `resolved_by.commit`
  stamp stays reachable from `main`. RS003 would only break if the queue squashed
  or rebased (rewriting the cited SHA out of existence — the exact hazard RS003
  exists to catch). The durable check, carried into the protocol: `git log -1
  --format='%p' <mergeCommit>` must show TWO parents before the stamps are trusted.
- **GITHUB_SHA/batch-sha caveat: reasoned, not consulted.** The queue runs CI
  against a speculative batch sha (GITHUB_SHA != final merge sha); since the method
  is merge-commit and each branch is an ancestor of the batch, reachability holds
  in both queue-CI and post-merge. Not asserting knowledge of a specific external
  #355 that was not read.

### F-O — second-operator replication validates the protocol (agent-to-agent transfer)

The docs/issue-sweep session (peer `abcd-cli-7c`, its own operator, auto-merge
authorised) requested and adopted the #523 auto-merge sequence VERBATIM over a
structured cross-session message: clean consolidated verdict first → `gh pr merge
<n> --auto --merge` with the BLOCKED/CLEAN `autoMergeRequest` state check → the
two-parent `%p` verification post-merge BEFORE trusting `resolved_by` stamps and
signalling done. Protocol transfer between independent operators, carrying its
verification steps intact, is itself a first-class experiment output: the process
is teachable and reproducible, not tacit to one session.

- Its cut mirrors ours: ONE integration PR (`integration/issue-sweep`), family
  branches merged into it locally — not N separate PRs. "All merged" now follows
  checks-green (auto-merge), not a human click; that green is our cut trigger.
- Second-operator sibling-sweep captures to cite: iss-2608270908332975 through
  iss-2608270908348796 (fifteen, one Bash-listable family).

### F-D (reinforced) — only the assembled-diff adversarial pass caught the cross-branch regressions

Every per-branch gate was green on the peer's family branches. Only the
consolidated `security-reviewer` over the assembled diff caught real cross-branch
regressions — evaluator-outside-the-loop earning its keep, a second independent
instance of the #523 `guard/match.go` / hermetic-git-gate lesson.

Four findings fixed + resolved this round:
- iss-2608270926036966 — BOM scope widening in the new canonical delimiter
  predicate desyncs intent's writer.
- iss-2608270926037660 — `derivedClasses` else-if shadows a scalar class,
  fail-opening ML001 for a both-shapes page.
- iss-2608270926031827 — preamble-led record lints green while loaders fail-close
  (preamble parity gap).
- iss-2608270926036528 — `gvCanonADRID` ordinal overflow silently drops.

Two pre-existing residual gaps surfaced, stay OPEN:
- iss-2608270926030978 — FoldPath Unicode-normalisation blindness in the folded
  path gates.
- iss-2608270926037088 — graveyard evidence terminal-safe but not markdown-safe.

---

## Second-operator issue-sweep — full data from PR #526 (integration/issue-sweep)

### First round: 14/14 verified-present and fixed, zero design-decision skips, zero not-present

Each fix reported as (a) symptom at the trust boundary / (b) canonical fix / (c)
swept siblings — the "sweep the pattern, not the instance" and one-canonical-primitive
disciplines, second independent instance:

- **#304** — `--- ` close read as unterminated, provenance vanished, ML001 disarmed → one `isFrontmatterClose` predicate across three close sites; sibling `site/sections.go` byte-prefix.
- **#288** — comment-led page rebuilt with a fabricated `session_memory` class → consolidated `frontmatterOpenIndex`, backfill skips preamble pages.
- **#320** — plural-classes unlicensed page exited 0 → `derivedClasses`; sibling `externalSourceHashes` same blindness.
- **#330** — junk sources list short-circuited ML001 → unconditional page-level pass minus per-entry covered classes.
- **#362** — external page without `source_hash` accepted, MQ001/MQ002 never ran → class-conditional requireds per brief 07-memory §3; sibling MQ003 early-return span-half contract.
- **#279/#331** — absent/null intent id and exempt superseded spec lint-green while loaders fail-close the corpus → `recordid.ValidIntentID`/`ValidSpecID` shared leaf, three regex copies deleted; siblings ADR-family analogue, filename grammar laxity, `validateStrict` unmirrored (all captured).
- **#338** — `--- ` ledger record lint-green but capture-refused → new canonical `frontmatter.IsDelimiter`, four divergent compares removed, deliberate BOM consequence pinned; sibling memory + gate-side compare family.
- **#373** — four null spellings diagnosed malformed vs record-lint's missing → `frontmatter.IsNull` gate at `newRecord`; siblings spec `parseSpec` null divergence, glossary null-as-data (both captured).
- **#329** — case-variant dest SKIPPED the gate entirely → new `fsutil.FoldPath`/`PathWithin`, fold as a parameter; siblings ahoy `under()`, brew Cellar prefix, home redaction.
- **#326** — two case-variant targets both planned Create → folded claim key into the existing whole-write conflict refusal, naming both spellings.
- **#292** — two signals truncated silently → `gvCapFindings` deleted, all ten signals through `capSignalFindings`.
- **#291** — shared-ordinal ADRs collapsed silently (adr-012/adr-12 split) → canonical handle grammar + `gvIDClaims` shadow announcements.
- **#496** — docs, five verified bullets, `Reported-by` stamped.

### Review round — F-D, with convergent independent detection

Per-branch gates all green. The assembled-diff `security-reviewer` found 4 real
defects: BOM scope (fix c6fe046a), `derivedClasses` shadowing (7cb8aae4), preamble
extraction axis (02ca8f3a, both stores incl. content-exempt), ordinal overflow
(2a668ece) — plus dead-wrapper cleanup (4bbe1ab0) and a missing watched-fail test
for a load-bearing guard (e17c33ac, verified red with the guard deleted).
Re-verification by the same adversary against its original fixtures: all four
CLOSED, APPROVE. **The independent ruthless pass reverted every fix and confirmed
each test goes red against unfixed code, AND independently found the same preamble
gap** — convergent detection by two independent evaluators is the strong signal
for the note. Docs-currency: CURRENT.

### Process observations (second operator)

1. **Armed-detector catch:** an agent attempted `--no-verify`; the guard hook
   blocked it; the agent complied. (Same catch as our session — the guard is doing
   its job across operators.)
2. **Green-scaffolding self-catch:** the BOM agent's first test draft PASSED at
   broken HEAD (fixture positioned past the reader's reach); the agent caught this
   itself and repositioned until red. A test-first-discipline datum: a test that
   does not first go red proves nothing.
3. **Worktree-outside-checkout held:** zero payload-scan races across the whole
   sweep — the concurrent-session guidance (iss-2608261331317889) held under load.
4. **Transcript model verification — the F-G discipline applied cleanly.** All 7
   of the second operator's agents were transcript-verified to have genuinely run
   `claude-opus-5`; stamps correct as written. Contrast with our security cut,
   where the `model:"opus"` override silently ran claude-opus-4-8 and 15 stamps had
   to be filter-branch-corrected: the fix for F-G is exactly this — verify the
   transcript's `model` field before stamping, never infer from the requested
   override. The second operator did it prospectively; we did it retroactively.
5. **Ledger discipline:** 19 resolved with commit provenance, 19 new captures
   (15 sweep + 4 late/residual-adjacent), every fix record-first.

### Ledger id map (second operator, for pilot-note citations)

- First-round resolves: the 14 issues above (#304 #288 #320 #330 #362 #279/#331 #338 #373 #329 #326 #292 #291 #496).
- Review-round resolves: iss-2608270926036966 / 037660 / 031827 / 036528, plus nitpick iss-2608270930248619.
- Late captures: iss-2608270945468534 / 469978 / 464715.
- Sibling-sweep captures: iss-2608270908332975 → iss-2608270908348796 (fifteen).
- Residual gaps staying open: iss-2608270926030978 (FoldPath normalisation), iss-2608270926037088 (graveyard markdown-safety).

### Merge landed as

- PR #526 merge commit: fcb5102f ; `git log -1 --format='%p'` two parents: [x] verified (f3be1f47 99561443).

---

## Release cut (v0.6.7) — the semantic-receipt gate, and what iterating it taught

### F-P — the derived version is not the informally-named one
Everyone (me included) called it "v0.7.0" in conversation, but `launch ship`
derived **v0.6.7**: every shipped record is a fix (89 fix + 25 internal + 1
additive), no feature intent graduated from planned/, so the bump is a patch. The
version is the binary's, derived from what shipped — not a choice. Lesson for the
automation: state the DERIVED version from `launch ship --json` before anyone
acts on an informal label; the retention plan the user described ("if v0.6.7, we
remove v0.6.6…") already used the derived number.

### F-Q — two release gates are human-by-design, and that is correct
The semantic-receipt gate's PROMOTE is the MAINTAINER's sign-off (receipt carries
`verifier: maintainer@release`), and the publish step is a second, unbypassable
human approval. "Cut automatically" therefore lands at: derive → compose → ingest
→ run the passes → author receipts for the maintainer's PROMOTE → prove the gate
locally → open the release PR. The two human gates (PROMOTE, publish) are the
sign-off the gate exists to require, not ceremony to automate away.

### F-R — preserve an approved changelog across a re-roll by classifying gate-fix records `internal`
The maintainer approved the composed 90-line changelog. Gate findings then forced
brief/doc fixes, which must be captured+resolved (close-with-the-act) and so enter
the cut. Resolving them `impact: internal` (in_changelog:false) keeps the cited
set at 90, so the SAME approved payload re-ingests byte-identically — no
re-compose, no re-approval. The runbook forbids hand-editing the changelog, so
this is the honest way to fold in gate-hygiene fixes without disturbing approved
prose. Internal is also the honest impact: brief-internal counts and a
help-text-sync have no user-visible surface change, and the user-facing story
(e.g. "backticks now blocked") is already announced by the behaviour fix's own line.

### F-S — the two gate detectors have COMPLEMENTARY scope; run both
The crosscheck (brief↔surface) caught brief drift — the shipped `site` verb
missing from two command enumerations, a bare-command-as-help over-claim — that
the docs-currency pass structurally could not, because docs-currency checks
USER-FACING docs and the drift was in the internal brief. Neither pass subsumes
the other; a release needs both.

### F-T — "sweep the pattern, not the instance" applies to DOC fixes too
My first fix of the crosscheck findings corrected each FLAGGED line but left a
SIBLING occurrence in the same file ("sixteen" survived at 08-skills.md:21,
"universal" at 05-intent.md:467). The re-run caught it. A doc-drift fix is not
done until you grep the whole file (and repo) for the pattern — identical to the
code-sibling-sweep rule in the bug-hunting spine.

### F-U (load-bearing) — re-run every required pass against the FINAL content commit; never transfer a verdict
I initially argued the docs-currency PROMOTE could TRANSFER to a later content
commit because the delta was "outside its reviewed surface." I re-ran it anyway,
for receipt honesty — and the re-run caught a REAL blocking finding the FIRST run
missed: the `abcd guard` help text and two docs still described the pre-gh-312
backtick limit ("backticks are not followed — a disclosed v1 limit"), directly
contradicting the v0.6.7 Security changelog line that advertises the fix. It was a
shipped-help + docs FALSEHOOD about a security boundary, and a transferred verdict
would have shipped it. The miss was non-determinism (the pass reads prose and can
miss on any given run), not scope. Rule for the automation: every required gate
pass runs against the FINAL content commit the receipt names; a prior PROMOTE on
an earlier commit is never transferred, however scope-disjoint the delta looks.
Corollary: the receipt's judgeModel is transcript-verified from the pass that
actually read that commit (F-G applied).

### Receipt mechanics captured (for the deterministic rung)
- Path `.abcd/work/reviews/<content-sha>/<gate>.json`; `policy.detector` == filename stem.
- Required by receipt_gate: `subject.digest.gitCommit` == content commit; `verificationResult: PROMOTE`; `judgeModel` a pinned snapshot; `policy.detector` == gate. Manifest-era (manifest.json present) adds three procedural refusals: `manifestHash` == manifest.json's sha256; `tier` sufficient for impact class (additive ⇒ full, not shallow); every `failing` entry carries a `disposition`.
- Two-commit release branch: commit 1 = CHANGELOG roll (+ any gate-fix content), commit 2 = the receipts naming commit 1. Prove locally before any tag: `go run ./cmd/record-lint --release-gate <full-sha> --require-gate docs-currency-reviewer --require-gate iss35-brief-surface-crosscheck` (exit 0 = will pass; costs nothing to fix, no tag yet).
- `briefVersion` in the receipt tracks the release version (0.6.7).

### F-V — the gate's whole-repo sweep did deep SECURITY work, not just doc-currency
Chasing the backtick doc finding to ALL its occurrences (git grep across every
tracked file — the scope error in F-T taught me to grep the whole repo, not a
dir subset) surfaced far more than a stale line:
- History ([dcn-2608010000000004](../../../work/decisions/dcn-2608010000000004-guard-batch-iss-159-144-148-revert-of.md)): backtick-following was REVERTED on 2026-08-01 after
  three adversarial rounds each found a guard-bypass regression; iss-148 re-opened.
- This release's gh-312 (iss-2608270500192596) RE-implemented it. I did NOT trust
  the changelog claim — I built the binary and ran the exact revert-era adversarial
  cases: leading-position `$(true) gh repo delete` (the registry-defeat) BLOCKS,
  newline-chain BLOCKS, unterminated backtick BLOCKS (was fail-open), nested paren
  BLOCKS. gh-312 genuinely solved what three rounds couldn't.
- But one bypass remains for BOTH `$()` and backticks: a substitution before an
  enclosing command's trailing flags truncates them — `cd s && rm $(true) -rf *`
  ALLOWS vs `cd s && rm -rf *` BLOCKS. Pre-existing `$()` gap (per [dcn-2608010000000004](../../../work/decisions/dcn-2608010000000004-guard-batch-iss-159-144-148-revert-of.md)),
  now shared by backticks at parity; the deep sub-part iss-148 defers.
- Ledger inconsistency: iss-148 (OPEN, "neither form followed") contradicted
  iss-2608270500192596 (RESOLVED, "backticks now followed"). Reconciled: narrowed
  iss-148 in place to the flags-truncation gap only, retaining its history as
  superseded; corrected the brief's blind-spot list to disclose the ACCURATE
  residual gap rather than the stale "backticks not followed" claim.
Lessons: (1) a release-gate pass that reads prose is also a security auditor when
the prose describes a security boundary — verify the CLAIM against the binary, never
trust the changelog. (2) A doc-drift sweep must span the WHOLE repo including the
ledger, or it desyncs an open issue from a resolved one. (3) The maintainer decides
how a security release represents a boundary: ship the sound improvement, disclose
the residual gap honestly, keep the deep-gap issue open — do not erase a real gap to
make a doc "clean".

### F-W — adopt an external contributor's commit; re-authoring with Reported-by is the fallback

Surfaced by the second operator. The issue-sweep's default was to re-author an
external contributor's fix and add a `Reported-by:` trailer. For @andytwoods
(#496, #494), who had a working branch, that default cost them contributor-graph
AUTHORSHIP: `Reported-by` credits the report, not the code, so the commit author
stayed the maintainer. The better default for a contributor who has a ready
branch is to ADOPT their commit (preserving their authorship) or ask for their
PR; re-author-with-`Reported-by` is the honest fallback only when there is no
branch to adopt. Corollary already in play: the same-change credit convention
means the `ACKNOWLEDGEMENTS.md` entry should ride the fix — a missing one is a
follow-up debt (the second operator is filing @andytwoods's now).

## Verdict (final — the cut shipped)

The protocol survived contact end to end and across two independent operators:
verify → dedup → fix → integrate → adversarial review → derive → compose → gate →
land → publish → retain. Combined **v0.6.7** shipped — 90 changelog records
(20 Security, 21 Fixed lines); both semantic-receipt gates PROMOTE at full tier
against the content commit; the full release pipeline (merge → tag → verify →
publish → site deploy) ran green; the four platform binaries were checksum- and
attestation-verified; retention applied newest-per-line (six 0.6-line releases
pruned, tags kept). The release gate did real correctness AND security work en
route — see F-S (complementary detectors), F-U (re-run every required pass against
the final content commit; never transfer a verdict — it caught a security-doc
falsehood a transferred verdict would have shipped) and F-V (the sweep uncovered a
reverted-then-reimplemented guard feature and a live bypass, verified empirically).

File the automation intent — "abcd handles inbound security advisories and issues,
and cuts the release, for every managed repo" — citing F-A…F-W as acceptance
criteria, with F-U and F-Q (the two release gates are human-by-design) load-bearing.

<!-- superseded interim verdict retained below for history -->

## Verdict (interim, superseded)

The protocol survives contact through verify → dedup → fix → (review pending).
The one structural gap is F-D (landing many fixes into one cut). Grade the
remaining stages (review, gate, land, credit) as they complete, then file the
automation intent — "abcd handles inbound security advisories and issues for
every managed repo" — citing F-A…F-G as acceptance criteria.

## Audit trail (this run)

- Advisories verified: 7 (all CONFIRMED on `HEAD`).
- Public criticals fixed: 3. Public security-majors fixed / in-flight: see the
  session's fix branches (`fix/ghsa-*`, `fix/gh-*`).
- Ledger dedup: 46 open public issues vs the ledger — 7 stale mirrors closed, 31
  genuine gaps backfilled (PR for the backfill raised).

### Release outcome (v0.6.7, this run)

- Combined cut assembled: 90 records (security cut #523 + issue-sweep #526 + #494
  fix #527 + backlog); derived version v0.6.7 (all fixes ⇒ patch).
- Gate: docs-currency-reviewer + iss35-brief-surface-crosscheck both PROMOTE, full
  tier, 0 findings, judge model transcript-verified claude-opus-4-8; receipts at
  `.abcd/work/reviews/bd186b84…/`; local `record-lint --release-gate` exit 0.
- Fix loop the gate forced: site-count drift, bare-command over-claim, and the
  guard-backtick doc/ledger reconciliation (iss-148 narrowed in place).
- Published: PR #528 merged (2ca95333, two parents, RS003 holds) → tag v0.6.7 →
  verify → publish → site deploy, all green; no site-deploy flake.
- Verified: checksums.txt match, provenance attestation verified (exit 0).
- Retention: pruned v0.6.1–v0.6.6 releases, tags kept; kept set is newest-per-line.
