---
schema_version: 1
id: "iss-2610161709506207"
slug: "the-opencode-runner-stamps-the-model-it"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/runner/opencode.go"
remedy: "Read the served model from opencode's run-mode JSON events (or the provider's response) and stamp that; when the events carry none, leave the reported model empty and say so in the receipt rather than copying the asked model."
related_intents: [itd-2610161733316518, itd-166]
resolution: "The opencode runner no longer copies the asked model into the served side: its run mode names no model, so the served model is left empty and listed as not reported, beside the asked model and the summed step-finish tokens and cost (itd-166)."
impact: fix
resolved_by:
  intent: "itd-166"
  commit: "7a305e062"
---

The opencode runner stamps the model it was asked for as the model that answered: internal/core/runner/opencode.go:85 sets ans.Model = o.model after parsing, so the route receipt's model field can never differ from the request, and a run routed through opencode to OpenRouter or a local server cannot show that a different model, provider or fallback served it.

Found while investigating itd-2610161733316518; the runner is itd-2609201916056194's. The claude runner reads the served model from its events (internal/core/runner/claude.go), so the two runners disagree on what the field means.
