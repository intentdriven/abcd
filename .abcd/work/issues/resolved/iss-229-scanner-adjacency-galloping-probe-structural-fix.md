---
schema_version: 1
id: "iss-229"
slug: "scanner-adjacency-galloping-probe-structural-fix"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "2026-08-15 forward-plans grill: maintainer settled iss-189/190 retirement as supersession into the recorded structural design"
found_at: "internal/adapter/scanner/scanner.go:scanAllPatterns"
related_intents: [itd-155]
resolution: "the galloping (exponential-doubling) trueMatchEnd probe replaces the fixed 512-byte adjacency window, so a match end is never a truncation artefact of the window edge; iss-189 and iss-190 repro shapes land as regression tests with a deterministic cost-class guard, and neither a boundary classifier nor a clipped-so-skip branch was needed"
impact: fix
resolved_by:
  intent: "itd-155"
  spec: "spc-48"
  commit: "8d97f686"
---

implement the galloping/exponential-doubling probe (trueMatchEnd) that replaces the fixed 512-byte adjacency window in scanAllPatterns/probeAt: the window grows only while a match keeps running into its edge and stops the moment the match ends short of the edge or reaches the real end of line, so a match end is never a truncation artifact. This is the structural fix designed in the 2026-08-08 DECISIONS entry after bug-hunt rounds 6-8 each BLOCKed on local patches for the same root cause (a truncated window-edge view driving a discard/skip decision reaching further than the ambiguity). It dissolves iss-189 (trailing boundary satisfied by the artificial window edge, spurious over-redaction) and iss-190 (window-capped recovery truncating a token and breaking the recovery chain) outright, without reintroducing the round-6 cost regression: the window grows only for matches genuinely still growing, at the amortized cost class the top-level unbounded match already pays. iss-189 and iss-190 (both superseded by this capture) are its acceptance corpus: their repro shapes must pass without a boundary classifier or a clipped-so-skip special case.