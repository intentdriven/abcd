---
schema_version: 1
id: "iss-2610262033262866"
slug: "18-ideate-says-capture-friction-stays-at"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 405 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/18-ideate.md:14"
remedy: "In 18-ideate.md:14 reword 'capture friction stays at one line' to 'ideate adds no friction to capture'; a capture is one line of text plus one line of remedy (06-capture.md:71)."
finding_ref: cfk-e5fb94e7c0b1c3bd
resolution: "18-ideate.md now says ideate adds no friction to capture instead of saying capture friction stays at one line, which 06-capture contradicts (a capture is one line of text plus one line of remedy); a brief test holds the wording to the capture chapter."
impact: internal
resolved_by:
  commit: "93587cc83da5a66df88f5c5824d11c9e18f92b1f"
---

18-ideate says capture friction stays at one line, but 06-capture makes a capture one line of text plus one line of remedy.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/18-ideate.md:14` — "and capture friction stays at one line"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:71` — "is one line of text and one line of remedy."

18-ideate says ideate costs capture nothing because capture friction stays at one line. 06-capture says the remedy is required and has no default, and that this tightens the one-line capture on purpose: a capture is one line of text and one line of remedy, and a capture without the remedy is refused with nothing written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 405.

## Grounds

- pursued: the ideate chapter no longer contradicts the capture chapter's two-line capture; TestBriefIdeateClaimsNoCaptureFrictionNotOneLine would fail if the old clause returned or if 06-capture dropped its one-line-of-remedy sentence.
