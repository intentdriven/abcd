---
schema_version: 1
id: "iss-2608221456462677"
slug: "checkinlinablesvg-accepts-an-svg-xml"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/site/assets.go"
resolution: "checkInlinableSVG refuses prolog/comment/CDATA at the asset, matching the emitted-page reader."
impact: fix
resolved_by:
  commit: "de19186"
---

checkInlinableSVG accepts an SVG XML prolog, comment and CDATA section (whitelisting the xml declaration, no case for comments/CDATA) but the emitted-page reader htmlscan refuses all three, so a normal exporter SVG makes abcd site build succeed and abcd site check refuse the page with a message naming no asset.