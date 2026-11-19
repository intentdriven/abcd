---
schema_version: 1
id: "iss-2610262037311796"
slug: "the-build-intent-puts-an-issue-lane-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 729 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:76"
remedy: "In decision 10 of the build intent replace 'the lane report carries a handback: field' with 'the lane receipt', which is where internal/core/implement/loop/receipt.go carries it and where the drain chapter puts it."
related_intents: [itd-2609201916151817]
finding_ref: cfk-8958d586530edc12
resolution: "Decision 10 of the build intent (and its matching acceptance criterion) now says the lane receipt carries the handback: field, where LaneReceipt.HandBack and the drain chapter put it; TestBuildIntentPutsHandBackInTheReceipt holds the intent to it."
impact: internal
resolved_by:
  commit: "e49e3e2aff4c0ea20ead4de0689c1a6580db6cf3"
---

The build intent puts an issue lane's hand-back in the lane report; the drain brief puts handback in the receipt

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:76` — "the lane report carries a `handback:` field the loop reads as a first-class outcome"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:572` — "A lane that finds a decision in its issue writes `handback` (a kind, a reason and, where the kind needs one, a home) in its receipt instead of resolving it."

Decision 10 of the build intent says the lane report carries a `handback:` field that the loop reads. The drain chapter says the lane writes `handback` (kind, reason, home) in its receipt instead. The brief treats the receipt and the implementer's report as two separate things, so the two documents put the field in different places.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 729.

## Grounds

- pursued: every line of itd-2609201916151817 naming `handback:` places it in the lane receipt, matching receipt.go's json tag and 35-drain.md; a line placing it in the lane report again, or the receipt field being renamed, would show it wrong, and the test fails on either.
