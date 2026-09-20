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
---

The status line paints everything after the badge in the terminal's plain foreground, so the repository name, the branch, the model, the context percentage and the intent and issue counts all carry the same weight and the row reads as one undifferentiated run. Give the elements two tones: a softer grey for the repository and branch names and for the itd and iss counts (the standing facts, glanced at rarely), and a darker colour for the model and the context figure (the two that change and get read). The badge keeps its own pair; the split is the one the previous hand-written status line already drew, and it must keep the rest of the row legible on the status surface's dim background.
