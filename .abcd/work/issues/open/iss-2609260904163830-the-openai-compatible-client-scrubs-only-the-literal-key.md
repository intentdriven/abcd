---
schema_version: 1
id: "iss-2609260904163830"
slug: "the-openai-compatible-client-scrubs-only-the-literal-key"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
---

The OpenAI-compatible client scrubs only the literal key from a provider's error text: a 4xx body that echoes the key JSON-escaped in a field other than error.message (for example {"detail":"bad key sk-abc\/def+ghi"}) falls to the raw body, the scrub misses the escaped form, and the key reaches the error and stderr. The provider-reported model is recorded and quoted in the denylist refusal unscrubbed as well.
