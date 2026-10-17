---
schema_version: 1
id: "iss-2610161734413178"
slug: "abcd-cannot-hold-an-openrouter-call-to"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/runner/opencode.go"
remedy: "Two seams, one for each path a call takes. For a tool-using role run through the opencode runner, abcd writes a sealed per-run opencode provider configuration carrying the pin (provider order, fallbacks off, required parameters, quantisations, data collection denied, the canonical model slug), records its hash and opencode's version with the route, and reads OpenRouter's generation record for each call where the generation id is available. For a self-contained call through the OpenAI-compatible adapter, the listed route carries the same pin and the adapter reads the same record into the route receipt. Waits on the planning of itd-2610161733316518, which decides the fallback policy."
related_intents: [itd-2610161733316518]
resolution: "Both seams built. Adapter: a machine provider block pins a listed model (oracle.api.<provider>.pins: exactly one provider in order, exactly one quantization, the canonical_slug; allow_fallbacks false, require_parameters true and data_collection deny fixed), read only on an OpenRouter base URL; a call sends OpenRouter's documented provider routing object (openrouter.ai/docs/features/provider-routing) and records it as provider_pin in place of the not_configured pin. Readback: GET /api/v1/generation?id= (openrouter.ai/docs/api-reference/get-a-generation) only on a machine provider block that sets read_generation true (OpenRouter base URL only, refused elsewhere and from any repository layer, one extra request per call), off by default, with the key the call resolved through the credential reader, bounded to 20s with a doubling wait on not-found; it fills served provider, attempts (provider_responses), cost, and the native token counts only all together when the record carries all four (otherwise the response's counts stand, never a mix of two tokenizers; a null count or cost is missing, never zero), recording whose counts it holds as usage_source native or response, cleaned and bounded, and a failure records generation_readback failed and never fails the call; the ahoy connect verification call never makes it. opencode: a pinned run writes a sealed per-run config (0700 dir, 0600 file, refused inside the repository, removed after) and points opencode 1.18.31 at it with OPENCODE_CONFIG (verified in the binary and with opencode debug config against a scratch home: merged over the global config, which was not written), refuses the pinned run before launch when the person's environment sets OPENCODE_CONFIG, OPENCODE_CONFIG_DIR or OPENCODE_CONFIG_CONTENT so their own configuration is never set aside or made to override the pin (~/.opencode/opencode.json, merged after the sealed file, is deliberately not refused, so a routing entry there can override the pin), and records the config sha256 and opencode --version; the readback is not made there since opencode holds the key, so the pin is listed not_verified (configured, not verified on the wire). Tested against httptest fixtures and a fake harness only: a live test against OpenRouter, and a live opencode run honouring the pin, are owed."
impact: additive
resolved_by:
  commit: "d0ab8898e"
---

abcd cannot hold an OpenRouter call to one named upstream provider, and it records only part of what served it.

- **Tool-using roles go through opencode, which abcd does not configure.** The opencode runner passes only `--model` (internal/core/runner/opencode.go, `args`); the provider, its routing and any fallback come from the person's own opencode configuration. So a builder or reviewer copy routed to OpenRouter can be served by another upstream provider, or a differently quantised endpoint, without abcd knowing.
- **The adapter path pins nothing either.** The OpenAI-compatible adapter passes seven sampling settings and no provider routing object; provider routing was out of scope for itd-2609081951381895. It does record the model the provider reported (internal/core/oracle/call.go, `ModelReported`), but not the upstream provider, fallback attempts, token counts or cost.
- **Reading the served side needs the key.** OpenRouter's generation record is fetched with the account's key, which for the opencode path is held in the person's opencode configuration, not in abcd's credential store; the remedy must read it through the one credential reader or not at all.

A model-matrix run (itd-2610161733316518) cannot pin a cell or show drift without both. Providing a pin refines the out-of-scope line of itd-2609081951381895; the product thinker's request for reproducible OpenRouter cells is the ground.
