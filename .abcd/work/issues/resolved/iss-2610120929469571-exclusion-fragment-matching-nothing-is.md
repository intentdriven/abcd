---
schema_version: 1
id: "iss-2610120929469571"
slug: "exclusion-fragment-matching-nothing-is"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane X2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
remedy: "Record every declared exclusion fragment that matched no bundle file in the scan result and surface each as a launch warning naming the fragment (refusing under strict_warnings) in the dry-run, the ship report and the cut precheck; prove with a test that an unmatched fragment is named, watched failing first, while a fragment that matches stays silent"
resolution: "The scan result now names every declared exclusion fragment that matched no payload file (scan.unmatched_exclusions), judged against every file, and the dry-run, the ship report and the cut's precheck each carry a warning naming it, refusing under strict_warnings. Proved by TestUnmatchedExclusionIsNamed (internal/adapter/scanner), TestUnmatchedExclusionWarns and TestPrecheckWarnsOnUnmatchedExclusion (internal/core/launch), each watched failing first; a matching fragment raises no warning or refusal even under strict_warnings, and the existing exclusion tests still pass."
impact: fix
resolved_by:
  commit: "18c841458"
---

The release scan accepts an exclude_path_fragments entry that matches no payload file without a word, so a typo'd exclusion (generatd/ for generated/) reads as working while the files it was meant to leave out are scanned, and a fragment left behind after a rename silently stands ready to exclude whatever later takes that name. Found by the follow-up to lane X's whole-segment exclusion fix (iss-2610111600051340) on 2026-10-12; whole-segment matching makes a near-miss fragment match nothing where a substring match often still hit, so the silent case is now the common one.
