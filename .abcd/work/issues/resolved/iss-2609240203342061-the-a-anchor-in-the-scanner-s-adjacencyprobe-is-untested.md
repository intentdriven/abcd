---
schema_version: 1
id: "iss-2609240203342061"
slug: "the-a-anchor-in-the-scanner-s-adjacencyprobe-is-untested"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
deferred_after: "v0.9.0"
deferral_reason: "The anchor is a constant-factor cost with no wrong result, and it duplicates a record on the parked security-sweep branches (named in this record's body); the sweep that carries that record lands the anchor test, so one of the pair closes as a duplicate then."
resolution: "TestAdjacencyProbeMatchesOnlyAtOffsetZero pins the anchor directly: a probe for each of four bundled pattern shapes matches its token at offset 0 and nothing after filler; it fails with the anchor removed. The parked twin iss-2609021205449871 is not in main's ledger; whichever branch carries it closes it as a duplicate of this record when it lands."
impact: internal
resolved_by:
  commit: "94c0e382e58676262736fa33dc5a00e58e419f37"
---

The \A anchor in the scanner's adjacencyProbe is untested: with the anchor removed and the fixed probe window kept, the whole scanner package passes (confirmed by the gallop lane's second implementer and by its review, item 4). scanAllPatterns rejects any probe match whose start is not 0, so an unanchored probe returns no wrong result; the window still bounds each attempt, so the loss is a constant-factor cost (each attempt searches its whole window for a leftmost match instead of failing at byte 0, up to 512x per attempt), not a wrong finding. No cost guard sees it, because the count guards assert growth ratios and a constant factor leaves every ratio unchanged. This duplicates the parked iss-2609021205449871, which is not on main: whoever lands the parked sweep closes one of the pair as a duplicate of the other. <!-- record-lint: forward-looking -->

## Grounds

- pursued: removing the start anchor now fails a test; a mutant with the anchor dropped that passes the package would show it wrong
