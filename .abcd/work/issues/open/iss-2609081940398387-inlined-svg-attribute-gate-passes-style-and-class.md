---
schema_version: 1
id: "iss-2609081940398387"
slug: "inlined-svg-attribute-gate-passes-style-and-class"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/assets.go"
---

checkSVGAttr claims an allowlist of drawing attributes but behaves as a denylist: it refuses on* names and scheme-checks href / xlink:href / src, then returns nil for every other attribute, including style and class. Inlined SVG bytes are concatenated verbatim into the published page by assetPipe.render, so those attributes become live markup: inline CSS on the root svg element beats the .svgasset svg and .card .icon svg rules (neither is !important), and a child can position:fixed to the viewport. The same gate already refuses a style ELEMENT (TestAssetsRefuseExecutableSVG); the attribute is untested. No script execution follows: the CSP is script-src self. This is a missed control on the inlining gate, not markup injection in the Markdown renderer. Fix: hold inlined SVG attributes to a closed set covering what the committed drawings actually carry (geometry, paint, id, clip-path, marker-*, refX/refY, aria-*), refusing style and class, with the existing href/src fragment and raster checks kept first; do not sketch an allowlist that omits an attribute process-loop.svg already uses. Detector: extend TestAssetsRefuseExecutableSVG with style= and class= payloads that render must refuse, naming the file, while a committed drawing still inlines. Reported as GitHub issue 621 against ec7f40d6.
