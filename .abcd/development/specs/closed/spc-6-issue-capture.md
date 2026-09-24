---
id: spc-6
slug: issue-capture
intent: itd-4
---
# issue-capture

## Summary

spc-6 is the native record catch-up for itd-4: the capture engine shipped long
before this spec existed and is in daily use (`abcd capture [text] | list |
resolve | wontfix`, the `iss-N` ledger under `.abcd/work/issues/` with
folder-as-status, the `related_issues`/`related_specs` schema). This spec
verifies the intent's five Acceptance Criteria against the built binary,
closes the one genuine coverage gap (AC5's list shape), and records — rather
than papers over — the two places the live system diverges from the AC letter:
the resolve-note storage design (AC2) and, until the 2026-09-23 build, the
promote flow the verbs could not complete (AC3). The product thinker ruled AC3
built as written (ruling B3, DECISIONS.md 2026-09-23): the back-links renamed
to `related_issues` / `related_intents` as the AC names them, every existing
record migrated, and the drift check the AC cites delivered as
`abcd intent audit --issue-drift`. With AC3 delivered the spec closes and itd-4
ships.

## Approach

Verification-first against `bin/abcd-darwin-arm64` in hermetic scratch repos
(temp `HOME`, `t.TempDir()` git repos; the real ledger untouched). Only what
the binary could not demonstrate became work — which, for this milestone, was
a single characterization test; no production code changed. Implementation was
delegated to a sub-agent worker (manual test B of the run protocol,
`.abcd/development/plans/2026-07-16-run-findings.md`, burst 3) with the
orchestrator re-running the gate.

## Milestones as delivered

1. AC-by-AC verification of the shipped capture engine against the binary
   (no engine changes required).
2. AC5 coverage pin: `TestCaptureListOpenRendersIssueFields`
   (`internal/surface/cli/capture_surface_test.go`) — five captures, then
   `capture list --open --json` asserted to return all five with id, slug,
   severity, and summary body.
3. This record: the AC-satisfaction map below, including the AC2 deviation
   and the AC3 blocked analysis.
4. AC3, built as written (2026-09-23, run A lane `itd4`): the promote join
   under the AC's names, link mode writing both halves, the retired names
   refused by every reader and banned by record-lint, `abcd capture migrate`
   to rewrite them (run over this tree: 58 records), and
   `abcd intent audit --issue-drift [--strict]` to check the join.

## Acceptance-criteria satisfaction

AC as ordered in itd-4 → status and evidence:

1. **`capture "<text>"` → `open/iss-N-<slug>.md`, frontmatter populated,
   text in body** — met by the shipped engine. Binary check: the canonical
   AC1 command produced `open/iss-1-review-nitpick-…md` with id, slug,
   severity, category, source, and found_during populated and the captured
   text as body. Covered by `internal/core/capture/workflow_test.go`
   (`TestCaptureAppendsAndReadsBack`).
2. **`capture resolve iss-N "<note>"` → moves to `resolved/`, note
   persisted** — met, with a **recorded deviation from the AC letter**: the
   note is stored as a structured frontmatter scalar
   (`resolution: "<note>"`, via `setScalarField`), not appended to the body
   prose as the AC literally says. The live design is deliberate and
   queryable, in daily use across the 100+-entry ledger; the note is captured
   and durable. Adjudicated as an intentional design evolution, not a gap
   (DECISIONS.md 2026-07-17). Covered by `TestResolveTransition`.
3. **`capture promote iss-N` → intent draft + bidirectional links** —
   **met (2026-09-23).** `abcd capture promote <iss-N>` mints the draft (the
   quoted-text create itd-46 shipped and `capture.Promote` both route through
   `intent.CreateDraft`) with `related_issues: [iss-N]`, and appends the minted
   `itd-M` to the issue's `related_intents`; `--intent <itd-M>` writes both
   halves onto an existing draft. "Promoted" is the pair, because an issue's
   `related_intents` also carries loose relations. The drift detection the AC
   cites is `abcd intent audit --issue-drift`: warnings on stderr and exit 0,
   `--strict` exit 1, a receipt under `.abcd/.work.local/logs/audit/`. The
   retired names are refused by every reader and banned by record-lint, and
   `abcd capture migrate --apply` rewrites them; the tree was migrated in the
   same change and the drift check reports nothing on it. Covered by
   `TestPromoteMintsDraftAndStampsIssue`,
   `TestPromoteLinkModeWritesBothHalvesOnAnIssue`,
   `TestPromoteKeepsALooseRelatedIntentAndAppends`,
   `TestMigrateRewritesEveryRetiredBackLinkIntoTheTwoSidedJoin`,
   `TestIssueDriftReportsEveryBrokenJoinAndNothingElse` and the surface tests
   in `internal/surface/cli/issue_drift_surface_test.go`. The analysis that
   held it open until then is kept below as the record of why it was open.
   **Held open until 2026-09-23 (historical):** Promote is
   skill-orchestrated by design (never a CLI sub-verb — see the comment above
   `newCaptureCommand` in `internal/surface/cli/cli.go`, brief 04-surfaces/06),
   but the skill surface cannot complete the flow with today's engine:
   (a) no intent-creation verb exists — `abcd intent` exposes only
   plan/link/review/ingest, and the quoted-text create path is itd-46, a
   later item; (b) no verb writes the back-link
   (`related_intents: [itd-M]`) onto an existing open `iss-N` —
   `capture.Capture` sets `related_intents` at creation only, and
   resolve/wontfix write only their own note fields; (c)
   `commands/capture.md` documents promote only as "the intent-new
   interview seeded with the issue body", without the bidirectional
   contract. Hand-editing frontmatter from markdown would violate the
   engine-backed convention (iss-86), so nothing was implemented. AC3
   becomes satisfiable once itd-46 lands **plus** an engine-backed
   back-link step (a capture update/promote verb or equivalent); this spec
   stays open on it.
4. **`.abcd/.work.local/issues.md` migration on upgrade** —
   **satisfied-by-history** (pre-adjudicated in the run plan §M3): the
   migration source no longer exists and the structured ledger is populated
   (103 entries, `iss-1`..`iss-103` with no gaps: 63 open, 40 resolved). The
   historical migration already happened; building migration code for an
   absent source would be dead scaffolding. Record-only; no code.
5. **`capture list --open` renders id, slug, severity, one-line summary for
   all open entries** — met by the shipped engine; exact AC shape was only
   indirectly covered before, now pinned by
   `TestCaptureListOpenRendersIssueFields` (added this run;
   characterization — behaviour already correct, so no watched-fail).

Out-of-scope confirmations: no dredge/synthesis machinery, no auto-capture
hooks, no cross-repo copying, no ledger `schema_version` change. iss-102
(orphan-sweep commit race) untouched — it belongs to its own workstream.
