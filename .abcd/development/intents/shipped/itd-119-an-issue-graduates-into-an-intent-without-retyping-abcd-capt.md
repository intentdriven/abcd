---
id: itd-119
shipped_in: v0.6.0
spec_id: spc-24
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: major
impact: additive
slug: an-issue-graduates-into-an-intent-without-retyping-abcd-capt
---

# An Issue Graduates Into An Intent Without Retyping

## Press Release

Capture stops being a dead end. When a one-line issue turns out to be a
capability, `abcd capture promote <iss-N>` graduates it into an intent draft
without retyping — the minted draft carries the issue's text, the issue is
stamped with the `itd-N` it became, and the trail survives in both directions.
"I used to hesitate at capture time — issue or intent? Now I capture everything
as one line and promote the ones that grow up," says Iris, product lead.

## Why This Matters

Step 2 of the twelve-step record walk — *decide it is a capability* — has no
verb. The schema already models the graduation: `promoted_to` is validated
against `^itd-[0-9]+$` and documented as "the itd-N this issue graduated into",
yet no verb writes it and no issue in the ledger carries it (refines the
`promoted_to` half of iss-245; the `resolved_by` half is a sibling intent). The
current `commands/capture.md` promote path is skill-orchestrated retyping with
no back-link. A native verb closes the forced intent-vs-issue choice the
"Which ledger?" note imposes at the moment of lowest information: capture now,
decide later, promote the ones that grow up.

The press release of the *promoted* intent is stated at its planning interview,
not at promotion time — promotion mints a draft with the standard placeholder,
so the promote moment stays one cheap command.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** an issue in the ledger, **when** `abcd capture promote <iss-N>`
  runs, **then** a new intent draft is filed under `drafts/` — its slug reused
  from the issue's slug, its body carrying the standard placeholder Press
  Release section plus a by-id pointer ("Graduated from iss-N") and the issue's
  one-line summary, never a copy of the issue body (SSOT) — and the issue's
  `promoted_to` is stamped with the minted `itd-N` in the same invocation.
- **Given** an issue in *any* status (`open/`, `resolved/`, `wontfix/`),
  **when** promote runs, **then** the graduation succeeds and the issue keeps
  its folder — promotion is orthogonal to fix-status and is not resolution.
- **Given** an issue already carrying `promoted_to`, **when** promote runs
  again, **then** the verb refuses and reports the existing `itd-N` — no
  duplicate drafts.
- **Given** a malformed or unknown `iss-N`, **when** promote runs, **then** it
  fails structurally with a diagnostic and nothing is written.
- **Given** the two-store write, **when** promote executes, **then** it mints
  the draft first and stamps the issue second; a failure after the mint reports
  the orphan draft's path and the remedy — `capture promote <iss-N> --intent
  <itd-N>`, a stamp-only mode that links an *existing* draft instead of
  minting (which also serves "I already filed the intent by hand; link them").
  Concurrent ledger transitions serialize under the existing ledger lock.
- **Given** the minted draft, **then** its frontmatter records the source issue
  (field named in the spec), so the edge is two-sided.
- **Given** `--json`, **then** the result reports the issue id, the minted (or
  linked) intent id, and both repo-relative paths.
- **Given** the sweep, **then** `commands/capture.md` documents the native verb
  (replacing the skill-orchestrated-retyping paragraph),
  `02-constraints/04-naming.md` drops the design-target marker, and the
  `04-surfaces` registry reflects `promote` as shipped — coordinating with the
  sub-verb-table intent if its rows land first.

## SOTA

Promote-with-link is a ubiquitous tracker pattern (Linear convert issue →
project, Jira "convert to `epic`", GitHub issue → discussion); nothing
importable —
abcd's record stores are native. **Chosen path: bespoke**, thin Go over
existing primitives, reusing the intent-create core function
(one-canonical-primitive) and the capture ledger's transition lock. No new
dependency.

## Open Questions

_None gating. Grill findings (two-store ordering, seed shape) were confirmed
and folded into the criteria, 2026-08-16._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-1e7e37b98414 -->
Fidelity review — receipt rcp-1e7e37b98414 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:708063e97662066d79a5984ff11edbbe2675de36fb27723003d684c4e05a742c
Input attestations: diff:tree at da7b7cf409b41758b3502d3873687dc8b817d8c4 (worktree HEAD; the host supplied no commit range, so the whole tree at that commit is the delivered reality)@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Mint mode reuses the issue slug, seeds a by-id pointer plus the first body line (never the body), writes promoted_from, then stamps promoted_to in the same call; the test asserts every one of those facts on the minted draft and the stamped issue.
  evidence: internal/core/capture/promote.go:198 — "slug := asString(fm["slug"])"
  evidence: internal/core/capture/promote.go:200 — "seed := "Graduated from `" + req.ID + "`: " + title +"
  evidence: internal/core/capture/promote.go:243 — "newContent, err := setScalarField(content, "promoted_to", rawScalar(itdID))"
  evidence: internal/core/capture/promote_test.go:107 — "if !strings.Contains(draft, "## Press Release") {"
  evidence: internal/core/capture/promote_test.go:120 — "if iss.PromotedTo != res.IntentID {"
- ac-2 — MET: The stamp is an in-place atomic write to the file's current status directory and a dedicated test graduates issues from resolved and wontfix without moving them.
  evidence: internal/core/capture/promote.go:263 — "// In place, atomic — the file keeps its status directory (promotion is"
  evidence: internal/core/capture/promote_test.go:129 — "func TestPromoteWorksInAnyStatusAndKeepsFolder(t *testing.T) {"
- ac-3 — MET: A promoted_to already present refuses before the mint and again under the lock, naming the existing intent id; the test and the CLI surface test both exercise the second run.
  evidence: internal/core/capture/promote.go:157 — "return PromoteResult{}, fmt.Errorf("%s is already promoted to %s; refusing to promote twice", req.ID, existing)"
  evidence: internal/core/capture/promote_test.go:166 — "func TestPromoteRefusesAlreadyPromoted(t *testing.T) {"
  evidence: internal/surface/cli/capture_surface_test.go:303 — "// Second promote refuses (exit non-zero) and names the existing intent."
- ac-4 — MET: findIssue refuses a malformed id by shape and an unknown id by absence before anything is minted; the test proves zero drafts after each bad id and no stamp after a bad link target.
  evidence: internal/core/capture/alloc.go:410 — "return "", "", fmt.Errorf("invalid iss-N identifier: %q", issID)"
  evidence: internal/core/capture/promote_test.go:186 — "func TestPromoteUnknownOrMalformedIDWritesNothing(t *testing.T) {"
  evidence: internal/core/capture/promote_test.go:193 — "t.Fatalf("Promote(%q) minted a draft on a structural fault", bad)"
- ac-5 — MET: Mint precedes the stamp; a stamp failure returns the orphan draft path with the stamp-only remedy, --intent links an existing draft after verifying it exists, and the stamp runs inside withLedgerLock with a serialisation test.
  evidence: internal/core/capture/promote.go:284 — ""%w — the minted draft %s (%s) is orphaned; complete the link with `abcd capture promote %s --intent %s --grounds %s`","
  evidence: internal/core/capture/promote.go:189 — "rel, ok := findRecordFile(repoRoot, intentStoreRelDirs(), req.LinkIntent)"
  evidence: internal/core/capture/promote.go:227 — "stampErr := withLedgerLock(repoRoot, issuesRoot, func() error {"
  evidence: internal/core/capture/promote_test.go:210 — "func TestPromoteStampFailureReportsOrphanAndLinkRepairs(t *testing.T) {"
  evidence: internal/core/capture/promote_test.go:276 — "func TestPromoteSerializesOnLedgerLock(t *testing.T) {"
- ac-6 — MET: The draft skeleton writes a bare promoted_from line when the draft graduated from a record, the intent reader parses it, and the mint test asserts the line on the minted draft.
  evidence: internal/core/intent/create.go:424 — "b.WriteString("promoted_from: " + opts.PromotedFrom + "\n")"
  evidence: internal/core/intent/intent.go:80 — "PromotedFrom string `json:"promoted_from,omitempty"`"
  evidence: internal/core/capture/promote_test.go:113 — "if !strings.Contains(draft, "promoted_from: "+issID) {"
- ac-7 — MET: PromoteResult carries issue id, intent id and both repo-relative paths under json tags, the CLI renders it through the shared marshaller under --json, and the surface test decodes the envelope and stats both reported paths.
  evidence: internal/core/capture/promote.go:44 — "IssueID string `json:"issue_id"`"
  evidence: internal/core/capture/promote.go:293 — "IssuePath: fsutil.RepoRel(repoRoot, stamped.path),"
  evidence: internal/surface/cli/capture_surface_test.go:296 — "t.Fatalf("promote paths must be repo-relative and non-empty: %+v", r)"
- ac-8 — MET_WITH_CONCERNS: The plugin page documents the native verb with no retyping paragraph and the capture surface's sub-verb table lists promote as shipped; concern: the naming register no longer carries a capture promote row at all (the only promote row left is the predecessor promote-check mode), so the design-target marker is gone by removal of the row rather than by editing it.
  evidence: commands/capture.md:384 — "## Promote an issue into an intent"
  evidence: commands/capture.md:394 — "One invocation mints a new intent draft under"
  evidence: .abcd/development/brief/04-surfaces/06-capture.md:31 — "| `promote` | — | shipped |"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:186 — "| `promote-check mode` | **(predecessor vocabulary; superseded by the native spec store"

Gap audit:
- honoured:
  - one invocation mints the draft and stamps the issue, with the trail in both directions
    evidence: internal/core/capture/promote.go:84 — "// Promote graduates an issue into an intent without retyping"
    evidence: internal/core/capture/promote_test.go:76 — "func TestPromoteMintsDraftAndStampsIssue(t *testing.T) {"
  - the promote moment stays one cheap command; the press release is a placeholder filled at planning
    evidence: internal/core/intent/create.go:447 — "b.WriteString("> " + seedNote(opts) + "\n\n")"
  - a post-mint failure names the orphan and the stamp-only remedy
    evidence: internal/core/capture/promote.go:283 — "return PromoteResult{}, fmt.Errorf("
- diverged:
  - the naming register drops the design-target marker on capture promote
    evidence: .abcd/development/brief/02-constraints/04-naming.md:186 — "| `promote-check mode` |"
    evidence: .abcd/development/brief/02-constraints/04-naming.md:154 — "Where the machinery a row describes is a design target rather than shipped behaviour, the row says **(staged)**"
- missing: (none)
