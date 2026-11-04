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
---

The reserve chapter's Exits section enumerates what exits 3 and puts every other refusal and failure at exit 2 (.abcd/development/brief/04-surfaces/37-reserve.md:945). The one-release-at-a-time back-off described at line 560 is not listed, and reserving a release while another release is being cut exits 3 (internal/core/claim/withdrawact.go:1351, internal/surface/cli/reserve.go:538-539), including when the same person and copy hold the other release.
