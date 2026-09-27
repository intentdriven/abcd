---
schema_version: 1
id: "iss-2609170709035405"
slug: "the-status-line-paints-everything-after-the-badge-in-the-ter"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first day on the v0.9.0 status line, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/render.go"
resolution: "Resolved as itd-2609212130146198 scopes it: the badge's colour ends at the badge. Its closing sequence restores only the default foreground and background (SGR 39;49), not a full reset, so the rest of the row keeps the host's own styling instead of being forced to the terminal's plain foreground. The two-tone split this capture proposed for the later elements is not built; the intent ruled the closure."
impact: fix
resolved_by:
  intent: "itd-2609212130146198"
  spec: "spc-2609212139593041"
  commit: "f31c564a07c9e78db4c44bd0efcaada88597d8b5"
---

The status line paints everything after the badge in the terminal's plain foreground, so the repository name, the branch, the model, the context percentage and the intent and issue counts all carry the same weight and the row reads as one undifferentiated run. Give the elements two tones: a softer grey for the repository and branch names and for the itd and iss counts (the standing facts, glanced at rarely), and a darker colour for the model and the context figure (the two that change and get read). The badge keeps its own pair; the split is the one the previous hand-written status line already drew, and it must keep the rest of the row legible on the status surface's dim background.

## Grounds

- pursued: ending the badge with 39;49 rather than a full reset leaves the host's styling of the rest of the row in place; shown wrong if the elements after the badge still render in the plain foreground on a host that dims its status surface
