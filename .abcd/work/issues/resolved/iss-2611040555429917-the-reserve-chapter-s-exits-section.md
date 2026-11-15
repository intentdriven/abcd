---
schema_version: 1
id: "iss-2611040555429917"
slug: "the-reserve-chapter-s-exits-section"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md"
remedy: "Add another release still being cut to the Exits section's list of exit 3 cases."
resolution: "The reserve chapter's Exits section now lists a release reserved while another release is still being cut among the exit 3 cases, the same person's and copy's included; a test holds the exit 3 list to it."
impact: fix
resolved_by:
  commit: "09fa29c0e27fd1be0aa887d447c68c3289f366be"
---

The reserve chapter's Exits section enumerates what exits 3 and puts every other refusal and failure at exit 2 (.abcd/development/brief/04-surfaces/37-reserve.md:945). The one-release-at-a-time back-off described at line 560 is not listed, and reserving a release while another release is being cut exits 3 (internal/core/claim/withdrawact.go:1351, internal/surface/cli/reserve.go:538-539), including when the same person and copy hold the other release.

## Grounds

- pursued: the Exits section's exit 3 list matches otherLiveRelease's backOff, which reserveRefusal maps to exit 3 with no holder or copy exemption; a release reserved during another's cut exiting 2, or the same person's own copy being exempt, would show it wrong
