---
id: itd-155
shipped_in: v0.6.8
slug: scanner-adjacency-galloping-probe-structural-fix
spec_id: spc-48
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-229
impact: fix
---

# implement the galloping/exponential-doubling probe (trueMatchEnd) that replaces the fixed 512-byte adjacency window in scanAllPatterns/probeAt: the window grows only while a match keeps running into its edge and stops the moment the match ends short of the edge or reaches the real end of line, so a match end is never a truncation artifact. This is the structural fix designed in the 2026-08-08 DECISIONS entry after bug-hunt rounds 6-8 each BLOCKed on local patches for the same root cause (a truncated window-edge view driving a discard/skip decision reaching further than the ambiguity). It dissolves iss-189 (trailing boundary satisfied by the artificial window edge, spurious over-redaction) and iss-190 (window-capped recovery truncating a token and breaking the recovery chain) outright, without reintroducing the round-6 cost regression: the window grows only for matches genuinely still growing, at the amortized cost class the top-level unbounded match already pays. iss-189 and iss-190 (both superseded by this capture) are its acceptance corpus: their repro shapes must pass without a boundary classifier or a clipped-so-skip special case.

## Press Release

> _Seeded by promotion from iss-229. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-229`: implement the galloping/exponential-doubling probe (trueMatchEnd) that replaces the fixed 512-byte adjacency window in scanAllPatterns/probeAt: the window grows only while a match keeps running into its edge and stops the moment the match ends short of the edge or reaches the real end of line, so a match end is never a truncation artifact. This is the structural fix designed in the 2026-08-08 DECISIONS entry after bug-hunt rounds 6-8 each BLOCKed on local patches for the same root cause (a truncated window-edge view driving a discard/skip decision reaching further than the ambiguity). It dissolves iss-189 (trailing boundary satisfied by the artificial window edge, spurious over-redaction) and iss-190 (window-capped recovery truncating a token and breaking the recovery chain) outright, without reintroducing the round-6 cost regression: the window grows only for matches genuinely still growing, at the amortized cost class the top-level unbounded match already pays. iss-189 and iss-190 (both superseded by this capture) are its acceptance corpus: their repro shapes must pass without a boundary classifier or a clipped-so-skip special case.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a redactable match whose extent runs past the old fixed 512-byte adjacency window, **when** the scanner probes `trueMatchEnd`, **then** the whole match is captured and its end is never a truncation artifact of the window edge.
- **Given** a short match that ends well before 512 bytes, **when** the scanner runs, **then** its result and its cost are unchanged from the fixed-window behaviour.
- **Given** the iss-189 repro shape (a trailing boundary satisfied only by the artificial window edge), **when** the scanner runs, **then** it produces no spurious over-redaction, and does so without a boundary classifier.
- **Given** the iss-190 repro shape (a window-capped recovery that truncated a token), **when** the scanner runs, **then** the recovery chain stays intact, and does so without a clipped-so-skip special case.
- **Given** a match that is genuinely still growing at the window edge, **when** the probe extends, **then** the window doubles only while the match keeps running into its edge, staying within the amortized cost class the top-level unbounded match already pays.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-f317a716a8e4 -->
Fidelity review — receipt rcp-f317a716a8e4 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:3a57d743083c8f2f18760747e08a2430559e49e4bd0cc334bb055e983c625285
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: gallopingFind doubles the window while the match runs into its edge and a 4096-byte recovered token is captured whole in the test; concern: a per-line growth budget (4*len(line)+4096) exists, and once it is spent the probe keeps the fixed window it has, so a truncation artefact remains possible on a line engineered to grow the window at many junctions — a single long token always fits
  evidence: internal/adapter/scanner/scanner.go:572 — "func gallopingFind(re matcher, line string, at, base int, budget *int) []int {"
  evidence: internal/adapter/scanner/scanner.go:584 — "keeps the fixed window it already has. See gallopBudget."
  evidence: internal/adapter/scanner/scanner.go:611 — "return 4*len(line) + 8*maxAdjacencyProbeWindow"
  evidence: internal/adapter/scanner/adjacency_test.go:458 — "func TestAdjacencyRecoveryCapturesALongTokenWhole"
- ac-2 — MET: the first attempt uses exactly the old 512-byte window, so a match ending before it returns on the first probe with the same result and cost; the pre-existing short-match and linearity tests pass at BASE
  evidence: internal/adapter/scanner/scanner.go:573 — "for w := maxAdjacencyProbeWindow; ; w *= 2 {"
  evidence: internal/adapter/scanner/scanner.go:524 — "const maxAdjacencyProbeWindow = 512"
  evidence: internal/adapter/scanner/adjacency_test.go:18 — "func TestConcatenatedSecretsBothDetected"
  evidence: internal/adapter/scanner/adjacency_test.go:260 — "func TestAdjacencyProbeStaysLinearOnLongLines"
- ac-3 — MET: the iss-189 repro places `.local` exactly at the old window edge with the token continuing, and asserts no LAN-host finding; the stop conditions evaluate the boundary against the real line end and no classifier symbol exists in the package
  evidence: internal/adapter/scanner/adjacency_test.go:405 — "func TestAdjacencyProbeWindowEdgeIsNotAWordBoundary"
  evidence: internal/adapter/scanner/scanner.go:578 — "if hi == len(line) || loc == nil || at+loc[1] < hi {"
- ac-4 — MET: the iss-190 repro (three abutting PATs with a 600-byte middle) asserts all three are found and redacted; stolenJunctions' forward reach is the same galloping probe with no clipped-so-skip branch
  evidence: internal/adapter/scanner/adjacency_test.go:425 — "func TestAdjacencyRecoveryChainSurvivesALongToken"
  evidence: internal/adapter/scanner/scanner.go:725 — "loc := gallopingFind(junctions, line, off, m.end, budget)"
- ac-5 — MET: the loop doubles only when the match reaches the edge with line remaining, a failing attempt never grows, and the cost test asserts the schedule is logarithmic in match length and the scan linear in line length
  evidence: internal/adapter/scanner/scanner.go:562 — "a FAILING attempt is what two bundled patterns'"
  evidence: internal/adapter/scanner/adjacency_test.go:532 — "t.Run("long_match_doubles_logarithmically", func(t *testing.T) {"
  evidence: internal/adapter/scanner/adjacency_test.go:631 — "func TestGallopingProbeCostIsLinearInLineLength"

Gap audit:
- honoured:
  - one mechanism replaces the fixed window in both probeAt and stolenJunctions; no boundary classifier and no clipped-so-skip branch were introduced
    evidence: internal/adapter/scanner/scanner.go:652 — "m := gallopingFind(probes[j], line, at, at, &budget)"
    evidence: internal/adapter/scanner/scanner.go:725 — "loc := gallopingFind(junctions, line, off, m.end, budget)"
  - the round-6 cost regression is guarded by tests that assert the doubling schedule and linearity directly
    evidence: internal/adapter/scanner/adjacency_test.go:503 — "func TestGallopingProbeCostClass"
    evidence: internal/adapter/scanner/adjacency_test.go:570 — "func TestGallopingProbeStaysBoundedOnLongLines"
- diverged:
  - the promise 'a match end is never a truncation artefact' holds under a per-line growth budget; when it is exhausted the probe reverts to the fixed 512-byte window, so on an adversarial many-junction line the old truncation shape can recur (a deliberate resource-exhaustion trade-off recorded in code, not in the intent)
    evidence: internal/adapter/scanner/scanner.go:606 — "reverts to exactly the fixed-window behaviour, which is bounded and was never"
    evidence: internal/adapter/scanner/scanner.go:583 — "if *budget < hi-at {"
- missing: (none)