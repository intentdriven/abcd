---
id: itd-120
shipped_in: v0.6.0
spec_id: spc-25
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: major
impact: additive
slug: a-resolved-issue-points-at-what-fixed-it-abcd-capture-resolv
---

# A Resolved Issue Points At What Fixed It

## Press Release

A resolved issue now points at what fixed it. `abcd capture resolve <iss-N>
"<note>" --impact fix --intent itd-N --spec spc-N --commit <sha>` stamps the
structured provenance the schema has modelled all along — the `resolved_by`
pointer that was parsed on read and written by nothing. "When I close an issue
I name the intent, spec, or commit that fixed it, and six months later the
trail is still there," says Nia, facilitator.

## Why This Matters

Step 12 of the twelve-step record walk closes the issue but loses the trail:
`ResolvedBy{Intent, Spec, Commit}` is parsed on read while `Resolve` writes
only `resolution` and `impact`, so a resolved issue asserts it was fixed in
prose but cannot point at what fixed it — while the intent side of the same
record store binds its verdicts to a SHA-256 receipt. One record store, two
evidence standards. This intent closes the `resolved_by` half of iss-245 (the
`promoted_to` (historical) half is itd-119); when both ship, iss-245 itself resolves *with*
provenance — the first entry in the ledger to carry the trail.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** an open issue, **when** resolve runs with any of `--intent itd-N`,
  `--spec spc-N`, `--commit <sha>`, **then** `resolved_by` is written with
  exactly the supplied members, alongside `resolution` and `impact`, in the
  same atomic transition to `resolved/`.
- **Given** `--intent` or `--spec`, **then** the id must *exist* in its record
  store (any bucket, open or closed); `--commit` is shape-checked only (7–64
  hex characters). An unknown id or malformed value refuses the whole
  transition — nothing written, the issue stays open.
- **Given** no provenance flags, **then** resolve behaves exactly as today: no
  `resolved_by` key is written at all — provenance is optional, never
  defaulted, never guessed.
- **Given** `--json`, **then** the result reports the transition and the
  written `resolved_by` members.
- **Given** the change, **then** `wontfix` is untouched — a non-action ships
  nothing and points at nothing.
- **Given** the sweep, **then** `commands/capture.md` and the issues README
  document the provenance flags.

## SOTA

Close-with-reference is standard tracker practice (GitHub "fixes #N" commit
links, GitLab closing patterns, Jira issue links); nothing importable — the
provenance lands in abcd's native frontmatter schema, already modelled.
**Chosen path: bespoke**, extending the existing `Resolve` transition and its
atomic-write machinery. No new dependency.

## Open Questions

_None gating. Backfilling `resolved_by` onto already-resolved issues was
ruled out of scope at the grill, 2026-08-16 — the verb closes the gap going
forward._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-b741cdb3204d -->
Fidelity review — receipt rcp-b741cdb3204d (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:1b78d748d8fbc6654013fa3563875ad19302a5986ba546eba3bf3b6e8bc3ee6c
Input attestations: diff:tree at da7b7cf409b41758b3502d3873687dc8b817d8c4 (worktree HEAD; the host supplied no commit range, so the whole tree at that commit is the delivered reality)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Resolve appends only the supplied members as a nested resolved_by object to the extras that ride the single transition call beside impact and the resolution note; the core test writes all three members and the partial test writes a subset.
  evidence: internal/core/capture/workflow.go:266 — "if rb != nil {"
  evidence: internal/core/capture/workflow.go:279 — "extras = append(extras, kv{"resolved_by", members})"
  evidence: internal/core/capture/workflow.go:281 — "res, err := transition(req.RepoRoot, req.IssuesRoot, req.ID, "resolve", "resolution", req.Resolution,"
  evidence: internal/core/capture/resolve_provenance_test.go:33 — "func TestResolveWritesResolvedBy(t *testing.T) {"
  evidence: internal/core/capture/resolve_provenance_test.go:78 — "func TestResolvePartialProvenance(t *testing.T) {"
- ac-2 — MET_WITH_CONCERNS: resolveProvenance runs before the move: intent and spec ids are shape-checked then probed for existence in any bucket, the commit is shape-checked only, and the refusal test proves the issue stays open with its bytes unchanged for every bad value; concern: the sha regex admits lowercase hex only, narrower than the criterion's 7-64 hex characters (git itself only emits lowercase, so the gap is theoretical).
  evidence: internal/core/capture/workflow.go:298 — "func resolveProvenance(req ResolveRequest) (*ResolvedBy, error) {"
  evidence: internal/core/capture/workflow.go:308 — "if _, ok := findRecordFile(repoRoot, intentStoreRelDirs(), req.ByIntent); !ok {"
  evidence: internal/core/capture/capture.go:327 — "reCommitSha = regexp.MustCompile(`^[0-9a-f]{7,64}$`)"
  evidence: internal/core/capture/resolve_provenance_test.go:128 — "t.Fatalf("%s: issue must stay open, got status=%q err=%v", name, status, ferr)"
  evidence: internal/core/capture/resolve_provenance_test.go:135 — "t.Fatalf("%s: refused resolve mutated the issue", name)"
- ac-3 — MET: With no member supplied resolveProvenance returns nil and the extras never gain a resolved_by key; the flagless regression test guards it.
  evidence: internal/core/capture/workflow.go:299 — "if req.ByIntent == "" && req.BySpec == "" && req.ByCommit == "" {"
  evidence: internal/core/capture/resolve_provenance_test.go:143 — "func TestResolveFlaglessWritesNoResolvedBy(t *testing.T) {"
- ac-4 — MET: TransitionResult carries the written members under a resolved_by json tag and the CLI surface test decodes id, to_status and the exact members from the --json envelope.
  evidence: internal/core/capture/capture.go:245 — "ResolvedBy *ResolvedBy `json:"resolved_by,omitempty"`"
  evidence: internal/core/capture/workflow.go:286 — "res.ResolvedBy = rb"
  evidence: internal/surface/cli/capture_surface_test.go:312 — "func TestCaptureResolveProvenanceJSON(t *testing.T) {"
  evidence: internal/surface/cli/capture_surface_test.go:355 — "t.Fatalf("resolved_by members wrong: %+v", r.ResolvedBy)"
- ac-5 — MET: WontfixRequest has no provenance members, so nothing on that path can write resolved_by, and the plugin page states it.
  evidence: internal/core/capture/capture.go:220 — "type WontfixRequest struct {"
  evidence: commands/capture.md:311 — "back in the JSON as `resolved_by`. `wontfix` takes no provenance — a non-action"
- ac-6 — MET: The plugin page documents the three flags with the existence and shape rules, and the issues README documents the field and the flags that write it.
  evidence: commands/capture.md:289 — "`resolve` also takes optional provenance — the structured `resolved_by` pointer"
  evidence: .abcd/work/issues/README.md:127 — "- `resolved_by` — optional pointer object (`intent`, `spec`, `commit`) naming"

Gap audit:
- honoured:
  - a resolved issue points at what fixed it through a structured pointer written in the same transition
    evidence: internal/core/capture/workflow.go:222 — "// resolved_by object (spc-25) in the same atomic transition. Ids are validated"
  - provenance is optional, never defaulted, never guessed
    evidence: internal/core/capture/workflow.go:226 — "// resolved_by key at all: provenance is optional, never defaulted."
- diverged:
  - the commit sha accepts 7-64 hex characters
    evidence: internal/core/capture/capture.go:327 — "reCommitSha = regexp.MustCompile(`^[0-9a-f]{7,64}$`)"
- missing: (none)
