---
schema_version: 1
id: "iss-2609260958588580"
slug: "the-error-echo-leaks-an-escaped-key-from-an-undecodable-body"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-apiadapter"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
---

The OpenAI-compatible client's error echo leaks a key written with ordinary ASCII runes as JSON \u escapes when the body is not decodable JSON: a plain-text body, or a JSON body cut at maxErrorBodyBytes, falls to providerSaid's raw-text fallback, which only resolves HTML references before the scrub, and keyForms carries \u escapes for non-ASCII runes alone, so {"detail":"bad sk-<key> cut at the bound reaches stderr while the same body as valid JSON is scrubbed.
