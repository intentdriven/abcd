---
schema_version: 1
id: "iss-2609081951416843"
slug: "setup-discovery-ahoy-could-probe-openrouter-reachability-and"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruled 2026-09-29: offer the outside AI service at setup, skippable, once the API adapter ships — waits on the API adapter"
---

Setup discovery (ahoy) could probe OpenRouter reachability and, when reachable, offer it to the user as an opt-in code-review oracle — once the OpenAI-compatible api oracle adapter ships, OpenRouter is provider config (baseURL https://openrouter.ai/api/v1 + key), and the offer is an ahoy-time question, never an auto-selected backend.
