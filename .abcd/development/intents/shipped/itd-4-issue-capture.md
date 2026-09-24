---
id: itd-4
slug: issue-capture
spec_id: spc-6
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
impact: breaking
---

# Nothing You Notice Gets Lost

## Press Release

> **abcd turns the steady drip of nitpicks, review findings, and "huh, that's odd" moments into a queryable ledger.** A new `/abcd:capture` command writes structured `iss-N` entries in seconds; the entries live in a committed `.abcd/work/issues/` ledger with stable IDs (`iss-N`) and folder-as-status (`open/`, `resolved/`, `wontfix/`). The `Mandatory Issue Recording` rule no longer depends on the agent remembering — there's a fast deliberate path with a structured destination, and the ledger is ready for the cross-corpus synthesist (`/abcd:dredge`, see itd-25) to mine once enough data has accumulated.
>
> "I do thorough plan and implementation reviews and they always surface nitpicks the agent files away in the transport output but not on disk," said Maya, autonomous-development practitioner. "I'd write 'remember to log this' a dozen times a session and still lose half of them. With abcd, I just say 'capture: T7 cache_ttl_days dead-config alternative' and it's structured, IDed, and queryable. The synthesis layer (itd-25) depends on the ledger having enough volume to be worth synthesising — but the capture surface is there from day one."

## Why This Matters

The `~/ABCDevelopment/.claude/CLAUDE.md` `Mandatory Issue Recording` rule is correct in spirit but unreliable in practice. Reviews and manual testing surface nitpicks that vanish unless the persona explicitly tells the agent to log them. When they *are* logged, `.abcd/.work.local/issues.md` is gitignored, per-repo, free-form Markdown — there's no way to ask "what categories of finding recur across my repos and over time?".

Two distinct problems compound: **capture is unreliable** (during-work) and **synthesis doesn't exist** (over-time, cross-repo). They have very different value-timing: capture earns its keep on day one; synthesis only earns its keep once a meaningful ledger has accumulated. This intent ships **capture only**. The cross-corpus synthesist that uses the ledger as input is a separate intent ([itd-25](../drafts/itd-25-dredge-cross-corpus-synthesist.md)), deferred until the ledger has sufficient volume.

This split closes a separate loop on `/abcd:capture` as a command name. The maritime-metaphor review for `/abcd:intent` established that "capture" is too neutral for product-framing intent work — but it's *exactly right* for issues, where the verb genuinely shouldn't pre-commit to whether a finding is a bug, nitpick, or systemic pattern. The synthesist (itd-25) decides that later. So abcd lands `/abcd:capture` not as a rename of `/abcd:intent` but as its sibling. Two meta-development surfaces with distinct jobs: `intent` frames product, `capture` ingests signal. `capture` and `intent` are metaphor-exempt; `dredge` (itd-25) rejoins the maritime convention as the cross-corpus counterpart to `lifeboat`.

**Brief-revision dependency:** Promoting this intent invalidates several "5 commands" claims in the brief body. Brief revisions land in the accompanying brief rewrite, not at intent-plan time.

## What's In Scope

- **`/abcd:capture` command (6th plugin command)** — single-entry ingest, with subverbs:
  - `capture <text>` — fast path; appends a structured `iss-N` entry to the ledger with auto-assigned `iss-N`, timestamp, and source provenance (which session / command / file the persona was in)
  - `capture list [--open|--resolved|--wontfix|--all]` — query the ledger
  - `capture promote <iss-N>` — promote an `iss-N` entry to an intent draft (calls `/abcd:intent new` with the entry body + bidirectional link)
  - `capture resolve <iss-N>` — move to `resolved/` with optional resolution note
  - `capture wontfix <iss-N> <reason>` — move to `wontfix/` with explicit decision
- **`iss-N` ledger structure** at `.abcd/work/issues/`:
  - `open/iss-N-<slug>.md` — captured, not yet acted on
  - `resolved/iss-N-<slug>.md` — fixed (with resolution notes)
  - `wontfix/iss-N-<slug>.md` — explicit non-action decision
  - Frontmatter schema: `id`, `slug`, `severity`, `category`, `source` (review / manual-test / drift / nitpick / observation), `found_during`, `found_at` (path), `related_intents` (list of `itd-N`), `related_epics` (list of `spc-N`), `created`, `updated`
  - Folder-as-status (mirrors intent and spec-roadmap conventions)
  - Stable `iss-N` IDs (unpadded; mirrors `itd-N` convention; lexical-vs-numeric sort handled at tool layer)
- **Brief § 5 update** (lands in the accompanying brief rewrite): exemption note for `/abcd:intent` + `/abcd:capture`; reserved-meta-command table covering `/abcd:dredge` (see itd-25), `/abcd:audit` (reserved, see itd-16 hash-chain-merkle-audit), `/abcd:reflect` (reserved, see itd-24).
- **`.abcd/.work.local/issues.md` migration path**: `dev-sync` promotes existing `.abcd/.work.local/issues.md` entries to the structured ledger on first run after install (or first `/abcd:ahoy` upgrade). Idempotent. Old `.abcd/.work.local/issues.md` becomes a staging buffer (still works for ad-hoc scribbles; promoted on next `dev-sync` or `/abcd:capture promote`).
- **`intent-fidelity-reviewer` extension**: when an intent ships that was promoted from an `iss-N` entry, the reviewer cross-references whether the related `iss-N` entry actually moved to `resolved/`. Mismatch = drift finding.

## What's Out of Scope

- **`/abcd:dredge` cross-corpus synthesist** — a separate intent ([itd-25](../drafts/itd-25-dredge-cross-corpus-synthesist.md)). Capture's value is immediate (every captured `iss-N` entry is useful from day one); dredge's value depends on having an accumulated ledger to synthesise. Shipping dredge without a meaningful ledger produces a synthesist with nothing to synthesise.
- **`issue-synthesist` agent** — belongs to itd-25 (`/abcd:dredge`).
- **Auto-capture hooks** (the "B" tier of the design exploration). Stop-hook / PostToolUse heuristic extraction of `iss-N`-shaped statements from session output. Brittle on output-style drift; risks false-positive fatigue. Defer until the ledger has accumulated real usage data showing whether the manual path's volume is sufficient.
- **Cross-repo `iss-N` copying.** Entries stay in the repo where they were captured. No "promote to a global ledger" behaviour.
- **Severity-based prioritisation UI.** This intent surfaces severity as frontmatter; no dashboard, no SLA, no triage workflow.
- **Renaming `/abcd:intent` to `/abcd:capture`.** Explicitly considered and rejected.
- **Real-time `iss-N` tracking integration** (Linear / GitHub Issues sync). The existing `issue-scout` agent already covers GitHub upstream annotation; sync in the other direction is a separate intent.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-installed repo, **when** the persona runs `/abcd:capture "review nitpick: T7 cache_ttl_days dead-config alternative"`, **then** a new file `.abcd/work/issues/open/iss-N-<slug>.md` exists with frontmatter populated (id, severity, category, source, found_during) and the captured text in the body.
- **Given** an existing `iss-N` entry at `.abcd/work/issues/open/iss-3-foo.md`, **when** the persona runs `/abcd:capture resolve iss-3 "fixed in spc-7 task 4"`, **then** the file moves to `.abcd/work/issues/resolved/iss-3-foo.md` with the resolution note appended to the body.
- **Given** an existing `iss-N` entry, **when** the persona runs `/abcd:capture promote iss-N`, **then** `/abcd:intent new` is invoked with the entry's content as the seed; the resulting intent's frontmatter has `related_issues: [iss-N]`; the `iss-N` entry's frontmatter has `related_intents: [itd-M]` (the new intent's ID). Drift detection enforced by spc-23 (intent-fidelity-reviewer `--issue-drift`).
- **Given** a fresh `/abcd:ahoy` upgrade with an existing `.abcd/.work.local/issues.md`, **when** `dev-sync` runs, **then** every entry in `.abcd/.work.local/issues.md` is promoted to a corresponding `.abcd/work/issues/open/iss-N-<slug>.md` with provenance noting "migrated from .abcd/.work.local/issues.md".
- **Given** the persona runs `/abcd:capture list --open`, **when** there are 5 open `iss-N` entries, **then** the output lists all 5 with id, slug, severity, and one-line summary.

## Open Questions

- **Manual-only capture vs hook-assisted.** This intent defers auto-capture (B) to a future intent. Should a minimal hook ship (e.g., a Stop-hook that simply prompts "Any captures from this session?" rather than auto-extracting), or wait for ledger-volume data first?
- **`iss-N` ID space — per repo, or global across the corpus?** Per-repo is simpler and matches `itd-N` (which is per-plugin-repo). Global would require a registry. Default: per-repo.
- **Promotion from `iss-N` → intent: 1:1 only.** N:1 (multiple `iss-N` entries fold into a single intent) pairs naturally with itd-25 (dredge) and is deferred to that intent. This intent supports 1:1 only.
- **Migration semantics for existing `.abcd/.work.local/issues.md`.** The current file has rich free-form structure (categories in `[brackets]`, "Found while", "Location", "Details", "Suggested fix"). The structured ledger should preserve all of this. Do we auto-migrate (risk: lossy) or interactive-migrate (cost: tedious for ~20 existing entries)?

## Implementing specs

itd-4 was implemented across multiple specs of the superseded pre-Go record
system; those ids are preserved below as history (they do not exist in the
native spec store). The frontmatter `spec_id` records the **native** spec,
**spc-6**, the record catch-up that verifies the shipped engine against the
Acceptance Criteria and carries the open AC3 (promote) gap. Historical index:

- **spc-20** (primary) — `iss-N`-ledger primitives (`iss-N` allocator, schema, capture/resolve/wontfix/update_field workflow, structure under `.abcd/work/issues/`).
- **spc-21** — `/abcd:capture` command surface (flow-text ingest into the ledger).
- **spc-22** — `.abcd/.work.local/issues.md` migration to the structured ledger (`dev-sync work` orchestrator, regex-extracted intent linkage on migrated issues).
- **spc-23** — `intent-fidelity-reviewer --issue-drift` mode (bidirectional cross-reference walk; reader half of the bidirectional contract).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2662745d5344 -->
Fidelity review — receipt rcp-2662745d5344 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:63bbd14cfb62139b464a3b8539b3fefb484da8342a9dfa91bc2373aa0f092da4
Input attestations: diff:e86d4f95..07b41ab5 (PR #689, tree read at 07b41ab5)@sha256:32bb9ea2da11af3bf3bf14944bdd3d23bdd28da6be56765809fae7d0d437d984;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: capture writes open/iss-N-< slug>.md with id, severity, category, source and found_during in frontmatter and the text as body; the round-trip test asserts every field and the filename; the plugin page invokes the same verb
  evidence: internal/core/capture/workflow.go:132 — "{"severity", string(req.Severity)}, {"category", ...}, {"source", ...}, {"found_during", req.FoundDuring}"
  evidence: internal/core/capture/workflow_test.go:89 — "func TestCaptureAppendAndReadBack"
  evidence: internal/core/capture/workflow_test.go:165 — "filepath.Base(got.Path) != tc.want.ID+"-"+tc.want.Slug+".md""
  evidence: commands/capture.md:4 — "argument-hint: "[text] | list --open|--resolved|--wontfix|--all | ..."
- ac-2 — MET_WITH_CONCERNS: resolve moves open/ to resolved/ and persists the note, but as the frontmatter scalar `resolution:` rather than appended to the body as the criterion says; the divergence is signed off (DECISIONS.md 2026-07-17, adjudication 1) and the test asserts the scalar
  evidence: internal/core/capture/workflow.go:289 — "transition(req.RepoRoot, req.IssuesRoot, req.ID, "resolve", "resolution", req.Resolution,"
  evidence: internal/core/capture/workflow_test.go:291 — "tr.FromStatus != StateOpen || tr.ToStatus != StateResolved"
  evidence: internal/core/capture/workflow_test.go:299 — "lr.Issues[0].Resolution !="
  evidence: .abcd/work/DECISIONS.md:579 — "AC2's resolve note lives as the structured frontmatter scalar `resolution:` ... not body-appended prose as the AC letter says"
- ac-3 — MET_WITH_CONCERNS: promote mints the draft with related_issues: [iss-N] and appends itd-M to the issue's related_intents, the drift walker checks the join and is enforced in preflight and CI; concerns: the seed is a by-id pointer, not the entry's content (spc-24 design), the mint goes through intent.CreateDraft rather than a `/abcd:intent new` invocation (that alias is deprecated), and the check is named `abcd intent audit --issue-drift`, not intent-fidelity-reviewer
  evidence: internal/core/capture/promote.go:222 — "RelatedIssue: req.ID,"
  evidence: internal/core/intent/create.go:427 — "b.WriteString(RelatedIssuesKey + ": [" + opts.RelatedIssue + "]\n")"
  evidence: internal/core/capture/promote.go:266 — "setListField(content, "related_intents", appendUnique(related, itdID))"
  evidence: internal/core/intent/lifecycle.go:543 — "func AddRelatedIssue(repoRoot, intentID, source string) (Intent, error)"
  evidence: internal/core/capture/promote_test.go:114 — "draft frontmatter missing related_issues: [%s]"
  evidence: internal/core/capture/drift.go:75 — "func IssueDrift(req IssueDriftRequest) (IssueDriftResult, error)"
  evidence: internal/core/capture/drift_test.go:49 — "func TestIssueDriftReportsEveryBrokenJoinAndNothingElse"
  evidence: internal/surface/cli/cli.go:3609 — "promote < iss-N> [--grounds "< token>: < text>"] | promote < rdi-N>"
  evidence: internal/surface/cli/cli.go:2306 — "Use: "audit [< itd-N>] | audit --issue-drift [--strict]""
  evidence: Makefile:172 — "issue-drift:"
  evidence: .github/workflows/ci.yml:346 — "- name: Issue-drift (promote-join gate)"
  evidence: commands/capture.md:406 — "and appends the minted `itd-N` to the issue's `related_intents`"
  evidence: commands/intent.md:534 — "intent audit --issue-drift # warnings on stderr, exit 0"
  evidence: internal/core/capture/promote.go:216 — "seed := "Graduated from `" + req.ID + "`: " + title +"
  evidence: .abcd/development/specs/closed/spc-24-an-issue-graduates-into-an-intent-without-retyping-abcd-capt.md:50 — "a by-id pointer, **never** a copy of the issue body (SSOT)"
  evidence: internal/surface/cli/cli.go:1996 — "WARNING: `abcd intent new` is deprecated; use `abcd intent "<text>"`"
- ac-4 — NOT_MET: promised: the plugin's sync step promotes every entry of the local-tier issues file into open/ with provenance 'migrated from' that file; delivered: no verb or sync step reads that file (its only mention is a package comment), no ledger record carries that provenance (grep over .abcd/work/issues/ finds none), and the source file is absent; the ruling of 2026-07-17 declares it satisfied-by-history, which records the omission rather than realising the outcome
  evidence: internal/core/capture/capture.go:2 — "a per-repo issue ledger that replaces the free-form"
  evidence: .abcd/work/DECISIONS.md:586 — "AC4 migration recorded satisfied-by-history (source absent, ledger populated iss-1..iss-103); no dead migration code built"
  evidence: .abcd/work/issues/resolved/iss-1-launch-phase-ownership.md:9 — "found_during: "roadmap-consistency-review""
  evidence: .abcd/development/specs/closed/spc-6-issue-capture.md:106 — "satisfied-by-history ... Record-only; no code."
- ac-5 — MET_WITH_CONCERNS: the pin test captures five issues and asserts `capture list --open --json` returns all five with id, slug, severity and the one-line body; concern: the human render prints id, status, severity and slug with no summary, so the criterion holds on the JSON surface only (captured as iss-2609240307549105)
  evidence: internal/surface/cli/capture_surface_test.go:182 — "func TestCaptureListOpenRendersIssueFields"
  evidence: internal/surface/cli/capture_surface_test.go:220 — "if iss.ID == "" || iss.Slug == "" || iss.Severity == "" || iss.Body == """
  evidence: internal/surface/cli/cli.go:3447 — "fmt.Fprintf(w, "%s %s %s %s%s\n", iss.ID, iss.Status, iss.Severity, iss.Slug, blockedNote(iss))"
  evidence: .abcd/work/issues/open/iss-2609240307549105-itd-4-ac5-says-capture-list-open-lists-every-open-issue-with.md:1 — "id: "iss-2609240307549105""

Gap audit:
- honoured:
  - `/abcd:capture <text>` writes a structured iss-N entry into the committed ledger with stable ids and folder-as-status
    evidence: internal/core/capture/workflow.go:132 — "{"severity", string(req.Severity)}"
    evidence: internal/core/capture/capture.go:9 — "status directories (open/, resolved/, wontfix/) whose folder membership IS"
  - resolve and wontfix move the record between status folders
    evidence: internal/core/capture/workflow_test.go:278 — "func TestResolveTransition"
    evidence: internal/core/capture/workflow_test.go:364 — "func TestWontfixTransition"
  - promote writes both halves of the issue-intent join, in mint mode and in link mode
    evidence: internal/core/capture/promote.go:266 — "setListField(content, "related_intents", appendUnique(related, itdID))"
    evidence: internal/core/capture/promote_related_test.go:17 — "func TestPromoteLinkModeWritesBothHalvesOnAnIssue"
  - the drift check reports a shipped intent whose promoted issue did not move to resolved/ (the intent-fidelity-reviewer extension the scope names)
    evidence: internal/core/capture/drift.go:121 — "if bucket == intent.BucketShipped && strings.HasPrefix(src, "iss-")"
    evidence: internal/core/capture/drift_test.go:62 — "DriftShippedUnresolved + " itd-5 iss-5""
  - drift detection is enforced: a make preflight prerequisite and a CI check step, both --strict, and the decision log corrects its own earlier 'not wired' statement
    evidence: Makefile:306 — "preflight: load-check lint-reviews lint-issues lint-decisions record-lint issue-drift"
    evidence: .github/workflows/ci.yml:348 — "run: go run ./cmd/abcd intent audit --issue-drift --strict"
    evidence: .abcd/work/DECISIONS.md:2523 — "Decision (5) says the drift check is not wired into preflight or CI. That contradicts the criterion it delivers"
  - every retired back-link in the tree was migrated to the two-sided join and the gate reports nothing at HEAD
    evidence: internal/core/capture/migrate_test.go:106 — "func TestMigrateRewritesEveryRetiredBackLinkIntoTheTwoSidedJoin"
    evidence: .abcd/work/issues/open/iss-327-managed-repo-pii-config-wrong-and-abcd-lint-missing-privacy-hygiene-error.md:10 — "related_intents: [itd-93]"
    evidence: internal/surface/cli/issue_drift_surface_test.go:67 — "func TestIntentAuditIssueDriftStrictCleanExitsZero"
  - both verbs are reachable from the plugin markdown surface
    evidence: commands/capture.md:399 — "capture promote < iss-N> --grounds "pursued: < conjecture>" --json"
    evidence: commands/intent.md:535 — "intent audit --issue-drift --strict # exit 1 on any finding (CI)"
- diverged:
  - resolve note appended to the body — delivered as the frontmatter scalar `resolution:` (signed off 2026-07-17)
    evidence: internal/core/capture/workflow.go:289 — ""resolve", "resolution", req.Resolution"
    evidence: .abcd/work/DECISIONS.md:579 — "recorded as intentional design evolution, not a gap"
  - promote seeds the draft with the entry's content — delivered as a by-id pointer to the issue, never a copy (spc-24 SSOT design)
    evidence: internal/core/capture/promote.go:216 — "seed := "Graduated from `" + req.ID + "`: " + title +"
    evidence: .abcd/development/specs/closed/spc-24-an-issue-graduates-into-an-intent-without-retyping-abcd-capt.md:50 — "a by-id pointer, **never** a copy of the issue body (SSOT)"
  - `/abcd:intent new` is invoked — delivered as a direct call of intent.CreateDraft, the primitive the quoted-text create shares; the `intent new` alias is deprecated
    evidence: internal/core/intent/create.go:303 — "created.RelatedIssues = []string{opts.RelatedIssue}"
    evidence: internal/surface/cli/cli.go:1996 — "`abcd intent new` is deprecated"
  - drift detection by `intent-fidelity-reviewer --issue-drift` (spc-23 of the retired record system) — delivered as `abcd intent audit --issue-drift [--strict]`
    evidence: internal/surface/cli/cli.go:2375 — "auditCmd.Flags().BoolVar(&issueDrift, "issue-drift", false,"
  - frontmatter field `related_epics` (list of spc-N) — delivered as `related_specs` (the glossary bans 'epic')
    evidence: internal/core/issueschema/issueschema.go:69 — ""related_specs": true, "related_issues": true,"
    evidence: internal/core/capture/workflow_test.go:122 — "RelatedSpecs: []string{"spc-12"}"
  - `capture list --open` output carries a one-line summary — delivered on the --json surface only; the human render omits it (captured as iss-2609240307549105)
    evidence: internal/surface/cli/cli.go:3447 — "iss.ID, iss.Status, iss.Severity, iss.Slug, blockedNote(iss)"
    evidence: internal/surface/cli/capture_surface_test.go:199 — "runCLI(t, "capture", "list", "--open", "--json")"
- missing:
  - the plugin's sync-step migration of the local-tier issues file into the ledger with 'migrated from' provenance (ruled satisfied-by-history 2026-07-17; nothing in the tree performs or evidences it)
    evidence: .abcd/work/DECISIONS.md:586 — "no dead migration code built"
    evidence: internal/core/capture/capture.go:2 — "a per-repo issue ledger that replaces the free-form"
  - brief § 5 reserved-meta-command table covering /abcd:dredge and /abcd:reflect — the brief reserves /abcd:audit alone
    evidence: .abcd/development/brief/04-surfaces/16-lint.md:16 — "`/abcd:audit` stays reserved for itd-16's hash-chain fidelity surface."