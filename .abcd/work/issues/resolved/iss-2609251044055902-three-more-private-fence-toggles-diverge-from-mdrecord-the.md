---
schema_version: 1
id: "iss-2609251044055902"
slug: "three-more-private-fence-toggles-diverge-from-mdrecord-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "parseCLIReference, markerLines and the memory quoted-span fenceMask read fences through mdrecord.Read; the private toggles are gone, and TestNoSecondFenceRule in mdrecord names no file and holds an empty allowlist."
impact: fix
resolved_by:
  commit: "ac0f03ee"
---

Three more private fence toggles diverge from mdrecord, the tree's CommonMark fence rule, and each gives a wrong result on a probe. site parseCLIReference (check.go) flips on any line starting with three backticks, so a flag in a tilde fence or in a four-backtick fence quoting a three-backtick line is never credited and the reference check reports drift that is not there. surface markerLines (appendix.go) flips on either a backtick or a tilde line, so a backtick line inside a tilde fence leaves the toggle on and a live appendix marker after the fence is refused as ErrMarkerInFence. memory fenceMask (coverage.go) closes a fence on a run indented four or more columns, which CommonMark reads as content, so a blockquote example inside the fence becomes a quoted span and the live quote after it is masked when the real closer reopens the toggle. Found by the TestNoSecondFenceRule detector ported onto mdrecord.

## Grounds

- pursued: each probe from the capture now gives the CommonMark answer (TestParseCLIReferenceReadsFencesByTheCommonMarkRule, TestMarkerLinesReadFencesByTheCommonMarkRule, TestExtractQuotedSpansReadsFencesByTheCommonMarkRule) and the detector, which names eight files at 6034ae70, names none; a new toggle the detector does not name, or a probe answering the old way, would show it wrong
