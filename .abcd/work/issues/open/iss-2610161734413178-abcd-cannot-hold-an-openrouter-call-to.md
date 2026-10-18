---
schema_version: 1
id: "iss-2610161734413178"
slug: "abcd-cannot-hold-an-openrouter-call-to"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/opencode.go"
remedy: "Two seams, one for each path a call takes. For a tool-using role run through the opencode runner, abcd writes a sealed per-run opencode provider configuration carrying the pin (provider order, fallbacks off, required parameters, quantisations, data collection denied, the canonical model slug), records its hash and opencode's version with the route, and reads OpenRouter's generation record for each call where the generation id is available. For a self-contained call through the OpenAI-compatible adapter, the listed route carries the same pin and the adapter reads the same record into the route receipt. Waits on the planning of itd-2610161733316518, which decides the fallback policy."
related_intents: [itd-2610161733316518]
---

abcd cannot hold an OpenRouter call to one named upstream provider, and it records only part of what served it.

- **Tool-using roles go through opencode, which abcd does not configure.** The opencode runner passes only `--model` (internal/core/runner/opencode.go, `args`); the provider, its routing and any fallback come from the person's own opencode configuration. So a builder or reviewer copy routed to OpenRouter can be served by another upstream provider, or a differently quantised endpoint, without abcd knowing.
- **The adapter path pins nothing either.** The OpenAI-compatible adapter passes seven sampling settings and no provider routing object; provider routing was out of scope for itd-2609081951381895. It does record the model the provider reported (internal/core/oracle/call.go, `ModelReported`), but not the upstream provider, fallback attempts, token counts or cost.
- **Reading the served side needs the key.** OpenRouter's generation record is fetched with the account's key, which for the opencode path is held in the person's opencode configuration, not in abcd's credential store; the remedy must read it through the one credential reader or not at all.

A model-matrix run (itd-2610161733316518) cannot pin a cell or show drift without both. Providing a pin refines the out-of-scope line of itd-2609081951381895; the product thinker's request for reproducible OpenRouter cells is the ground.
