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
resolution: "installMarkerFile refuses a file whose block would be appended inside a fence or HTML comment nothing closes (appendsInsideOpenSpan, over mdrecord.Unclosed) and leaves it untouched; classifyMarker reports it unplaceable, detection raises the non-resolvable marker.unplaceable gap naming the hand fix, and EnsureMarker's probe refuses it."
impact: fix
resolved_by:
  commit: "29231438"
---

ahoy's marker install appends the canonical block inside an unclosed fence or comment. composeMarkerInsertion (internal/core/ahoy/marker.go) places the block after frontmatter, else after the first live H1, else at end of file; firstOutOfFenceH1 reads liveness through mdrecord.Mask, so a fence or HTML comment nothing closes masks every line below it, no H1 is found, and the block is appended at end of file INSIDE the open span, where no reader sees it as markdown. classifyMarker's byte regex then finds the block and reports it current, so nothing ever says so. mdrecord.Unclosed is available and unconsulted.

## Grounds

- pursued: an unclosed backtick fence, an unclosed tilde fence and an unclosed comment each leave the file byte-identical, classify unplaceable and fail the embark probe, while an H1 above an unclosed fence still takes the block (TestMarkerRefusesToAppendInsideAnUnclosedSpan), and the ahoy package passes; a block written inside the open span, or a refused file that has a live insertion point, would show it wrong
