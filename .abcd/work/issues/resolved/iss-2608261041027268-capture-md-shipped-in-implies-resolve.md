---
schema_version: 1
id: "iss-2608261041027268"
slug: "capture-md-shipped-in-implies-resolve"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: "commands/capture.md"
resolution: "capture.md: --shipped-in is shape-checked at resolve; existence/ancestry are enforced at derivation, which keeps a wrong-version record in the cut with a stated reason."
impact: internal
resolved_by:
  commit: "2811321ba4b6adcd06bf05d14101c4bd76d26cdc"
---

commands/capture.md implies --shipped-in is validated for existence at resolve time; the code shape-checks only, and existence/ancestry run at release derivation. The flag doc says the value must name a tag that exists and that the release can reach, a shape-valid but wrong version is refused — read as a resolve-time precondition. In fact resolve only matches vMAJOR.MINOR.PATCH (workflow.go reShippedIn); a nonexistent shape-valid tag (v9.9.9) is accepted and written, and existence/ancestry are enforced only at derivation (changelog/shipped.go), which keeps a wrong-version record IN the cut with a stated reason rather than dropping it. The system never silently drops the record, but the doc gives false confidence at the point of use that resolve validated the tag. Fix: move the guarantee to derivation-time wording.