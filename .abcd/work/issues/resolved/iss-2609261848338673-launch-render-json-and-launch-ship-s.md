---
schema_version: 1
id: "iss-2609261848338673"
slug: "launch-render-json-and-launch-ship-s"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/render.go"
resolution: "PayloadRenderResult keeps the resolved staging directory as its working Dest (json:\"-\"), which the archive step packs from, and reports DisplayDest under the unchanged key dest through fsutil.DisplayPath: the home redacted to ~, a destination always being outside the repository. The text payload line prints the display field."
impact: fix
resolved_by:
  commit: "ae58e9260"
---

launch ship --json reports the release payload's destination as an absolute path in payload.dest, and the text report prints it on its payload line: PayloadRenderResult.Dest (internal/core/launch/render.go) is the symlink-resolved destination, so a destination under the home names the developer in machine output, against the iss-81 rule the site and lifeboat verbs are held to.

## Grounds

- pursued: a render staged under HOME reports payload.dest as ~/staging and its JSON carries neither spelling of the home, while the payload is still written to and packed from the real directory (TestTheRenderAndTheArchiveReportTheirPathsWithoutTheHome); an absolute dest, or an archive packed from the wrong directory, would show it wrong
