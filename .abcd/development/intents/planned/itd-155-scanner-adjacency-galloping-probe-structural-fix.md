---
id: itd-155
slug: scanner-adjacency-galloping-probe-structural-fix
spec_id: spc-48
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-229
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

_Empty. Populated by intent-auditor when intent moves to shipped/._
