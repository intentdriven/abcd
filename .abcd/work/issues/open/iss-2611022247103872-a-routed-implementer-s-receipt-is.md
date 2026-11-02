---
schema_version: 1
id: "iss-2611022247103872"
slug: "a-routed-implementer-s-receipt-is"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/receipt.go"
remedy: "The runner stamps the model it launched (it chose it, and the transcript records it) into the receipt it collects, or the implementer's brief names the field and its value; a test drives a routed implement stage to an accepted receipt."
---

A routed implementer's receipt is refused since #1356 for a missing model: on run-2611022037190012 the claude runner launched the implementer, which committed and wrote its receipt without the model field, and the receipt check refused it (the field the landing's Assisted-by trailer names), pausing the lane until the host added the model by hand. Every routed build now stops at its receipt.
