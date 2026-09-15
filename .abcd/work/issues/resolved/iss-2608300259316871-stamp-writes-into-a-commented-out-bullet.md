---
schema_version: 1
id: "iss-2608300259316871"
slug: "stamp-writes-into-a-commented-out-bullet"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-177 second-round reviews, 2026-08-30"
found_at: "internal/core/intent/claims.go (fenceMask, stampScopeConditions, conditionBlocks)"
resolution: "The mask now spans HTML comments as well as fenced code, and a masked line is neither matched as a heading or bullet nor written to, so a bullet parked inside a comment is no longer counted as a live condition. The stamp refuses a section carrying a comment outright — writing a marker there would close the outer comment with the marker's own arrow and start rendering the parked block — and the gate names the fault with a remedy, so the refusal is never reachable only as a dead end. The corpus receipt-stability test stays green over 172 records."
impact: fix
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

The claim parser and stamper are blind to a multi-line HTML comment, so a commented-out bullet under Scope Conditions is counted as a live condition, the stamp writes a marker inside the comment, the marker's closing sequence terminates the outer comment early, a stray closing sequence renders as prose, the parked condition carries a live identity, and ready reports READY — a rewrite that corrupts rendered content. The resolution of iss-2608300235388164 does not cover the comment clause its body names. Extend the mask to a comment span (an unclosed opener masks until the line carrying the closer), let the gate name it and the stamp refuse it exactly as for a fence.
