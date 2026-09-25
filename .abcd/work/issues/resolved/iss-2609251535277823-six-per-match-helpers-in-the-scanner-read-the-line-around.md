---
schema_version: 1
id: "iss-2609251535277823"
slug: "six-per-match-helpers-in-the-scanner-read-the-line-around"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "inAnySpan searches a sorted merged span list and URL spans carry their path root, found once per span; the footer's line-start test walks back only over its prefix alphabet and its link search stops at maxFooterLinkText; isDottedNamespaceComponent reads at most maxDottedIdentifier bytes either side; colonRunGroups stops counting once the run is longer than an address. Each bound fails toward the finding (TestBoundedContextHelpersKeepTheFinding). Twelve shapes in TestScanLineWorkIsLinear were watched failing at 8-16x per quadrupling on a scratch copy of the parent commit and pass at the fix."
impact: fix
resolved_by:
  commit: "3c5e8f5f"
---

Six per-match helpers in the scanner read the line around every match, so a line dense in matches costs the square of its length, independently of the generic-account check. (1) inAnySpan and atURLPathRoot scan the whole URL span list for every identity match, and atURLPathRoot searches each span from its start for the path root; (2) the footer SkipAt runs its line-prefix regex over the whole line before every footer match, and footerLinkTarget searches the rest of the line for the link; (3) isDottedNamespaceComponent walks and re-splits the whole dotted run for every login occurrence in it; (4) colonRunGroups walks and splits the whole hex/colon run for every address candidate in it. The cost meter's linearity guard (TestScanLineWorkIsLinear) measures 8-16x per quadrupling on each shape where linear work is 4x; wall-clock probes agree (a dotted run of a non-generic login, 8000 units: 35, 127, 479 ms). Pre-existing: none of these came in with the scanner cluster lane, and no guard reached them.

## Grounds

- pursued: the six helpers now do bounded or amortised work per match, so the scan is linear in line length; a TestScanLineWorkIsLinear shape growing past 6x per quadrupling would show it wrong.
