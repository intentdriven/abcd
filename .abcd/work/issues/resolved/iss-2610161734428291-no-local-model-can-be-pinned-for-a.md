---
schema_version: 1
id: "iss-2610161734428291"
slug: "no-local-model-can-be-pinned-for-a"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/runner/opencode.go"
remedy: "Give a local route a pin abcd records with every call: runtime and version, weight file sha256 or model digest, quantisation, served context length and KV-cache type, written into the sealed per-run opencode configuration for a tool-using role and onto the connection for a self-contained call. Probe the server once before use for tool-call support and its served context, and refuse a build-role route to a server that fails the probe, naming the runtime and the setting that would fix it."
related_intents: [itd-2610161733316518]
resolution: "A block on this machine takes a local pin per model (runtime, version, weights digest, quantisation, context length, KV-cache type, and on llama.cpp the GGUF file's name, since llama-server reports no digest); the server is probed before a pinned route is used (before every runner run; a self-contained call reuses a report the pin held for a minute at most, never a refused one) and a mismatch, or a tool-using route the server cannot hold, is refused naming the fix; the record carries the pin and the server's report (the KV-cache type, which no runtime reports, listed not verified), and opencode runs against a sealed config naming the probed server. Tested against httptest fakes of Ollama, llama-server and mlx_lm.server only; a live probe against a real local server is owed."
impact: additive
resolved_by:
  commit: "46c81d7c4"
---

No local model can be pinned for a tool-using build role. Nothing records the weight file's sha256 or Hugging Face revision, the quantisation, the runtime and its version, or the served context length, and the known traps go unchecked:

- llama-server (b9860 here) calls tools through the chat template, and a q4 KV cache can degrade tool calling.
- Ollama (0.14.2 here) sets context length server-wide (`OLLAMA_CONTEXT_LENGTH`, default 4096) or per model in a Modelfile, never per request, so a long prompt is truncated silently at the default.
- mlx_lm.server (0.31.3 here) accepts tools and a seed, but its tool parsing is per model family and fragile.

As with OpenRouter, a tool-using role reaches a local server through the opencode runner, whose provider configuration is the person's own. A model-matrix run (itd-2610161733316518) needs local cells it can name exactly. Versions checked locally on 2026-10-16; see research/notes/2026-10-16-model-matrix-sota.md.
