---
schema_version: 1
id: "iss-2610171545099084"
slug: "a-runner-pointed-at-a-local-model-leaves"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "intent audit of itd-166 (rcp-f25a69346de1)"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/runner/sides.go"
remedy: "When the local pin (iss-2610161734428291) lands, a runner route whose provider it declares local lists the three local fields as not configured until the pin fills them, as the adapter's loopback route does; tested with the opencode runner on a local provider."
resolution: "A runner route whose provider is a block on this machine records local: true and lists the local fields as the adapter's loopback route does: not configured without a local pin, the pin's values (or not verified) with one; tested with opencode and claude against a local provider."
impact: additive
resolved_by:
  commit: "46c81d7c4"
---

A runner pointed at a local model leaves the weight fingerprint, quantisation and runtime version out of its record entirely, where the provider adapter's loopback route lists them as not configured; the intent audit of itd-166 (ac-4) found it with the opencode runner on local/qwen3-coder. A runner cannot tell from its own output that a model is served locally, so the fix needs the local pin's notion of which providers are local.
