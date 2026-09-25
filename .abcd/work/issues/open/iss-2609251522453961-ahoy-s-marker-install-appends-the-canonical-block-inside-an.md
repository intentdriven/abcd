---
schema_version: 1
id: "iss-2609251522453961"
slug: "ahoy-s-marker-install-appends-the-canonical-block-inside-an"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/marker.go"
---

ahoy's marker install appends the canonical block inside an unclosed fence or comment. composeMarkerInsertion (internal/core/ahoy/marker.go) places the block after frontmatter, else after the first live H1, else at end of file; firstOutOfFenceH1 reads liveness through mdrecord.Mask, so a fence or HTML comment nothing closes masks every line below it, no H1 is found, and the block is appended at end of file INSIDE the open span, where no reader sees it as markdown. classifyMarker's byte regex then finds the block and reports it current, so nothing ever says so. mdrecord.Unclosed is available and unconsulted.
