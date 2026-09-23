---
id: itd-125
shipped_in: v0.6.0
spec_id: spc-30
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-lifeboat-verdict-says-what-it-is-disembark-oracle-become
---

# The Lifeboat Verdict Says What It Is

## Press Release

The lifeboat's verdict says what it is. `abcd disembark review` weighs a
packed lifeboat and returns `SHIP`, `NEEDS_WORK`, or `MAJOR_RETHINK` — a
judgement over the pack, which is a review, and now the verb, the flag, the
artefact, and the prose all say so. "The file used to be called an audit,
produced by an oracle, holding a review verdict — three words for one thing.
Now it's one word," says Kira, maintainer.

## Why This Matters

The 2026-08-16 planning investigation found what adr-40 §5 missed: the binary
verb never invokes the oracle seam. `disembark oracle` is a compute-or-ingest
verdict endpoint — deterministic mode is a mechanical mapping over the
manifest seal and coverage summary, delegated mode validates a verdict the
host's agent produced elsewhere. Naming the verb for the seam claims a seam
this verb doesn't touch, would collide with the `/abcd:oracle ask` design
target (the real seam surface, adr-25), and leaves the artefact
(`audit/oracle-*.json`) contradicting the vocabulary at every level. The
maintainer therefore **reversed adr-40 §5** ("keep the verb, fix the prose")
in favour of the rename — the reversal is recorded as a dated amendment
inside adr-40 §5 and a `DECISIONS.md` line. The oracle seam itself is
untouched: adr-25 stands, and `oracle` remains the seam's name.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `disembark review <lifeboat-dir>
  <source-repo> [--review-json <path>|-]` replaces the `oracle` spellings,
  and the old sub-verb and flag fail as unknown — no alias, no shim.
- **Given** behaviour, **then** both modes are preserved byte-for-byte: the
  deterministic manifest+coverage mapping and the delegated
  validate-and-ingest (enum-membership gate, cite-or-be-dropped findings,
  core-stamped attestation), including the exit contract.
- **Given** the artefacts, **then** the synthesis output moves to
  `review/review-<manifest12>.json` + `.md`, still excluded from
  `manifest_sha256`.
- **Given** an older lifeboat carrying `audit/oracle-<manifest12>.json`,
  **then** a re-run writes the new path and removes the superseded old-name
  file for the same manifest — the clean-replacement guarantee survives the
  rename; two verdicts for one manifest never coexist.
- **Given** the sweep, **then** the `lifeboat-oracle` agent becomes
  `lifeboat-reviewer`, `commands/disembark.md` and the brief's "oracle
  audit" prose move to "review", the historical-records boundary holds as in
  the sibling renames, and the `oracle_review` task-class token is
  re-checked in the same change.
- **Given** the decision record, **then** the adr-40 §5 amendment (dated,
  in-place, recording the compute-or-ingest finding and the reversal) and
  one dated `DECISIONS.md` line land with this intent's planning records.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  `disembark` sub-verb table row flips in the same change — `record-lint`
  exit 0 proves the migration; landing before the check is armed is
  forbidden.
- **Given** the release record, **then** the CHANGELOG entry is breaking.

## SOTA

Same family as itd-123/itd-124: pre-1.0.0 breaking rename, no aliases
(adr-40; iss-171 precedent). The naming pattern follows the session's ruled
convention — target-grain + role (`intent-auditor`, `lifeboat-reviewer`) and
self-describing artefact basenames. **Chosen path: bespoke sweep**, proved
complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The adr-40 §5 reversal was investigated, confirmed, and homed
by the maintainer, 2026-08-16._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-89f97822ba49 -->
Fidelity review — receipt rcp-89f97822ba49 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:6dbb1a43082685877015e8825ee2795557731ff7170eec33db60323b6bab853d
Input attestations: diff:tree at da7b7cf409b41758b3502d3873687dc8b817d8c4 (worktree HEAD; the host supplied no commit range, so the whole tree at that commit is the delivered reality)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The disembark command registers review with the --review-json flag and nothing else under the old names, and the surface test drives both the retired sub-verb and the retired flag to a refusal.
  evidence: internal/surface/cli/cli.go:833 — "Use: "review < lifeboat-dir> < source-repo> [--review-json < file|->]","
  evidence: internal/surface/cli/cli.go:861 — "reviewCmd.Flags().StringVar(&reviewJSON, "review-json", "", "path to the host-produced review verdict JSON (or - for stdin); absent runs deterministic mode")"
  evidence: internal/surface/cli/disembark_synthesis_test.go:323 — "if _, err := runCLIErr(t, "disembark", "oracle", "a", "b"); err == nil ||"
  evidence: internal/surface/cli/disembark_synthesis_test.go:329 — "t.Fatalf("--oracle-json must be an unknown flag, got: %v", err)"
- ac-2 — MET_WITH_CONCERNS: Both modes are present and tested: the deterministic verdict table, the enum-membership gate, cite-or-be-dropped findings, the attestation the core stamps from the manifest, and the exit contract (usage and fault at 2, a verdict at 0); concern: byte-for-byte parity with the pre-rename output is asserted by the changelog and by a bytes-stable test of the current output, not by a test comparing the two vintages.
  evidence: internal/core/lifeboat/synthesis_review.go:84 — "func ReviewLifeboat(lifeboatDir, sourceRepo string, raw []byte) (ReviewResult, error) {"
  evidence: internal/core/lifeboat/synthesis_review_test.go:195 — "func TestDeterministicVerdictTable(t *testing.T) {"
  evidence: internal/core/lifeboat/synthesis_review_test.go:289 — "func TestReviewLifeboatDelegatedVerdictMembership(t *testing.T) {"
  evidence: internal/core/lifeboat/synthesis_review_test.go:314 — "func TestReviewLifeboatDelegatedFindingsCiteOrDropped(t *testing.T) {"
  evidence: internal/core/lifeboat/synthesis_review.go:103 — "// 2. Manifest attestation and packed coverage summary — the trusted inputs the"
  evidence: internal/surface/cli/cli.go:838 — "return &exitError{Code: 2, Msg: "disembark review: < lifeboat-dir> < source-repo> are both required"}"
  evidence: internal/core/lifeboat/synthesis_review_test.go:258 — "func TestReviewLifeboatDeterministicBytesStable(t *testing.T) {"
- ac-3 — MET: The artefact pair is written under review/ with the review- basename keyed on the twelve-hex manifest prefix, and the review/ prefix is in the set the manifest hash excludes.
  evidence: internal/core/lifeboat/synthesis_review.go:163 — "jsonRel := path.Join(reviewArtefactDir, "review-"+manifest12+".json")"
  evidence: internal/core/lifeboat/synthesis_review.go:164 — "mdRel := path.Join(reviewArtefactDir, "review-"+manifest12+".md")"
  evidence: internal/core/lifeboat/embark_types.go:373 — "manifestExcludedPrefixes = []string{"graveyard/low-confidence/", "review/", "audit/"}"
- ac-4 — MET: After writing the new pair the core removes the pre-rename pair for exactly that manifest through the lifeboat's os.Root and prunes the emptied legacy directory; tests cover the replacement, that other manifests are left alone, and that the sweep cannot escape the lifeboat.
  evidence: internal/core/lifeboat/synthesis_review.go:72 — "func removeLegacyReviewArtefact(root *os.Root, manifest12 string) {"
  evidence: internal/core/lifeboat/synthesis_review.go:176 — "removeLegacyReviewArtefact(root, manifest12)"
  evidence: internal/core/lifeboat/synthesis_review_test.go:603 — "func TestReviewReplacesPreRenameArtefact(t *testing.T) {"
  evidence: internal/core/lifeboat/synthesis_review_test.go:646 — "func TestReviewLeavesOtherManifestsAlone(t *testing.T) {"
  evidence: internal/core/lifeboat/synthesis_review_test.go:687 — "func TestReviewLegacySweepCannotEscapeLifeboat(t *testing.T) {"
- ac-5 — MET_WITH_CONCERNS: The agent is registered as lifeboat-reviewer, the plugin page documents disembark review with --review-json, the naming register records why the oracle_review token survived unchanged, and the sibling boundary holds; concern: one brief page still titles a verification-matrix row 'Press-release oracle audit', a residue of the retired vocabulary in current-state prose.
  evidence: agents/lifeboat-reviewer.md:2 — "name: lifeboat-reviewer"
  evidence: commands/disembark.md:208 — "disembark review < lifeboat-dir> < source-repo> --review-json < path> # or - for stdin"
  evidence: .abcd/development/brief/02-constraints/04-naming.md:180 — "the seam keeps its name even where the verb that consumes it does not (`disembark review`, spc-30), so the token survived that rename unchanged"
  evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:59 — "| Press-release oracle audit |"
- ac-6 — MET: The ruling ADR carries a dated in-place amendment recording the compute-or-ingest finding and the reversal, and the decision log carries one dated line for it.
  evidence: .abcd/development/decisions/adrs/0040-review-audit-lint-are-three-verbs.md:156 — "> **Amendment (2026-08-16).** The keep-the-verb ruling above rested on a claim"
  evidence: .abcd/work/DECISIONS.md:1147 — "- 2026-08-16 — adr-40 §5 REVERSED at the process-planning interview (amendment recorded in-place in adr-40): `disembark oracle` renames to `disembark review`"
- ac-7 — MET_WITH_CONCERNS: The disembark sub-verb table carries review as shipped in the review bucket and record-lint exits 0 over the tree at this commit (run by the auditor, warnings only); concern: the landing-order clause (armed before this landed) is a history fact the delivered tree cannot show.
  evidence: .abcd/development/brief/04-surfaces/02-disembark.md:36 — "| `review` | review | shipped |"
  evidence: .abcd/development/brief/04-surfaces/README.md:46 — "The **Status** column is machine-checked: the `surface_coverage` record-lint rule"
  evidence: Makefile:164 — "record-lint:"
- ac-8 — MET: The shipped record declares impact breaking and the 0.6.0 changelog section carries the entry under the breaking heading.
  evidence: .abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is-disembark-oracle-become.md:10 — "impact: breaking"
  evidence: CHANGELOG.md:1110 — "- **The lifeboat verdict says what it is.** `abcd disembark oracle` is now `abcd disembark review`"

Gap audit:
- honoured:
  - the verb, the flag, the artefact and the agent all say review
    evidence: internal/core/lifeboat/synthesis_review.go:6 — "// review/review-< manifest12>.json (+ .md), a post-pack mutable artifact kept out of"
    evidence: agents/lifeboat-reviewer.md:2 — "name: lifeboat-reviewer"
  - the oracle seam itself is untouched and keeps its name
    evidence: .abcd/development/brief/02-constraints/04-naming.md:180 — "so the token survived that rename unchanged"
  - two verdicts for one manifest never coexist across the rename
    evidence: internal/core/lifeboat/synthesis_review_test.go:603 — "func TestReviewReplacesPreRenameArtefact(t *testing.T) {"
- diverged:
  - the brief's oracle audit prose moves to review
    evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:59 — "| Press-release oracle audit |"
- missing: (none)
