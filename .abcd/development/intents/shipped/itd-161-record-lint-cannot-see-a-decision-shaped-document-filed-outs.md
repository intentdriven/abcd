---
id: itd-161
shipped_in: v0.6.8
slug: record-lint-cannot-see-a-decision-shaped-document-filed-outs
spec_id: spc-53
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2608230752354926]
impact: additive
---

# record-lint cannot see a decision-shaped document filed outside the record stores

## Press Release

> _Seeded by promotion from iss-2608230752354926. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608230752354926`: record-lint cannot see a decision-shaped document filed outside the record stores. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a markdown file filed outside the configured record stores that asserts a record id already held by a real record (for example a heading `# ADR-23` with `Status: Accepted` reusing a taken adr id), **when** record-lint runs, **then** the cross-store detector flags the outside-store id claim.
- **Given** the probe file `research/notes/zz-recurrence-probe.md` reusing the taken id ADR-23, **when** record-lint runs, **then** it exits non-zero with a finding, where before the change it exited 0 with zero findings.
- **Given** a legitimate record filed inside its own record store, **when** record-lint runs, **then** the detector does not flag it.
- **Given** a grandfathered undated Phase 0 note, **when** the detector runs, **then** it does not fire on the filename alone, being weighed against the record baseline.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4c60ae841084 -->
Fidelity review — receipt rcp-4c60ae841084 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:874d926e3298d608830044d2af6c4337dd8b043eaa14f84aaedff38c369dc45e
Input attestations: diff:328a6755^1..328a6755 (PR #555) -- internal/core/lint and .abcd/record-lint.json, judged against the tree at 0ab2ad02@sha256:3384730886b02e997754fc4e8958d79349a288c601f0a5c0a345ded1346b7f72;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: checkCrossStoreIDClaim builds the taken-id set from LoadRecordGraph, walks the tracked markdown outside every record store, and crossStoreClaim fires when the H1 opens with a taken handle and the body carries a lifecycle Status; the rule is registered as a blocker in .abcd/record-lint.json and run once outside the per-root loop; TestCrossStoreIDClaimFlagsTheProbe asserts the finding names adr-23 on the claiming heading
  evidence: internal/core/lint/crossstore.go:79 — "func checkCrossStoreIDClaim(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {"
  evidence: internal/core/lint/crossstore.go:215 — "func crossStoreClaim(rel string, lines []string, taken map[string]bool, severity string) (Finding, bool) {"
  evidence: .abcd/record-lint.json:294 — ""cross_store_id_claim": {"
  evidence: internal/core/lint/lint.go:466 — "if csCfg, ok := cfg.Rules[ruleCrossStoreIDClaim]; ok && csCfg.Enabled {"
  evidence: internal/core/lint/crossstore_test.go:48 — "func TestCrossStoreIDClaimFlagsTheProbe"
- ac-2 — MET: the probe is a test fixture written at research/notes/zz-recurrence-probe.md in a temp root (not a committed file), and the test asserts exactly one blocker finding; the exit flip was demonstrated directly on a scratch copy of 0ab2ad02: go run ./cmd/record-lint exits 1 with '[BLOCKER cross_store_id_claim]' on the staged probe and exits 0 once it is removed
  evidence: internal/core/lint/crossstore_test.go:40 — "const probeNote = "# ADR-23: Transport Agnostic Core (probe)\n" +"
  evidence: internal/core/lint/crossstore_test.go:52 — "writeFile(t, root, filepath.Join("research", "notes", "zz-recurrence-probe.md"), probeNote)"
  evidence: internal/core/lint/crossstore_test.go:69 — "if f.RuleID == ruleCrossStoreIDClaim && f.Severity != severityBlocker {"
  evidence: internal/core/lint/crossstore.go:47 — "const ruleCrossStoreIDClaim = "cross_store_id_claim""
- ac-3 — MET: crossStoreCandidates lists only markdown outside the configured store directories, so a record in its own store is never a candidate; TestCrossStoreIDClaimLeavesRealRecordsAlone writes an adr with an id-claiming H1 and an accepted Status inside the adr store and asserts zero findings
  evidence: internal/core/lint/crossstore.go:162 — "func crossStoreCandidates(repoRoot string, storeDirs []string) ([]string, error) {"
  evidence: internal/core/lint/crossstore_test.go:76 — "func TestCrossStoreIDClaimLeavesRealRecordsAlone"
- ac-4 — MET: the fire condition is the pair of an id-claiming H1 and a lifecycle Status, weighed against the taken-id set from the record graph, so a filename alone never fires; TestCrossStoreIDClaimNeedsBothSignals asserts an undated 01- Phase 0 note, a meeting note with Status, and an id-mentioning reading note are all silent, and TestCrossStoreIDClaimUntakenIDDoesNotFire pins the baseline half; the rule deliberately does not consult contentExempt (spc-51's approach) and records why at crossstore.go:29
  evidence: internal/core/lint/crossstore.go:21 — "// The pair is what grandfathers the undated Phase 0 notes without an allowlist of"
  evidence: internal/core/lint/crossstore_test.go:95 — "func TestCrossStoreIDClaimNeedsBothSignals"
  evidence: internal/core/lint/crossstore_test.go:128 — "func TestCrossStoreIDClaimUntakenIDDoesNotFire"

Gap audit:
- honoured:
  - a decision-shaped document outside the stores that claims a taken record id is a blocking record-lint finding
    evidence: internal/core/lint/crossstore.go:215 — "func crossStoreClaim(rel string, lines []string, taken map[string]bool, severity string) (Finding, bool) {"
    evidence: .abcd/record-lint.json:294 — ""cross_store_id_claim": {"
  - the probe that once passed clean now exits record-lint non-zero
    evidence: internal/core/lint/crossstore_test.go:48 — "func TestCrossStoreIDClaimFlagsTheProbe"
  - records in their own store and grandfathered Phase 0 notes are never flagged, by the shape of the fire condition rather than by an allowlist
    evidence: internal/core/lint/crossstore_test.go:76 — "func TestCrossStoreIDClaimLeavesRealRecordsAlone"
    evidence: internal/core/lint/crossstore_test.go:95 — "func TestCrossStoreIDClaimNeedsBothSignals"
  - the rule reads only tracked files inside the repository and masks fenced and frontmatter Status
    evidence: internal/core/lint/crossstore_test.go:207 — "func TestCrossStoreIDClaimSkipsUntrackedFiles"
    evidence: internal/core/lint/crossstore_test.go:167 — "func TestCrossStoreIDClaimIgnoresFencedStatus"
- diverged: (none)
- missing: (none)