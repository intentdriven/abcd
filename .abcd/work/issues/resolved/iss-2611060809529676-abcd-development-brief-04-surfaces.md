---
schema_version: 1
id: "iss-2611060809529676"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Say the exemption comes before the blanks in both reports, after the probe's heading line."
refines: [iss-2609091956001547]
resolution: "The embark chapter now says the pass-B exemption prints before the blanks in both reports: after the probe's heading line, and as the first line of the unpack's report. A test holds the chapter and both renders to this."
impact: internal
resolved_by:
  commit: "e4827e515a3a9397384ea1e7f92f1619c09e758a"
---

`.abcd/development/brief/04-surfaces/03-embark.md:213-214` says embark carries the pass-B exemption through as the first line of both reports. The first line of the `embark probe` report is its heading, `embark probe: <name>  (lifeboat verified)` (`internal/core/lifeboat/embark_render.go:25`), and the exemption follows it. Only `embark from` opens with the exemption.

## Grounds

- pursued: the chapter should match embark_render.go, where the probe opens on its heading and the exemption follows. A render whose first line was not the heading, or a chapter that still says 'first line of both reports', would show it wrong, and TestBriefEmbarkChapterPlacesTheExemptionAfterTheProbeHeading checks both.
