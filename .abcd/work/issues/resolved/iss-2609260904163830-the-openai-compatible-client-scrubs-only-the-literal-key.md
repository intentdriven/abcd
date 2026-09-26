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
resolution: "Provider text is decoded (JSON re-rendered from decoded values, HTML references resolved) before the scrub, and the scrub removes the key's JSON-, HTML-, URL-escaped and quoted forms; the contract error is scrubbed before it is bounded and the reported model is scrubbed; ahoy connect's last scrub uses the same Scrub."
impact: fix
resolved_by:
  commit: "8bc48995"
---

The OpenAI-compatible client scrubs only the literal key from a provider's error text: a 4xx body that echoes the key JSON-escaped in a field other than error.message (for example {"detail":"bad key sk-abc\/def+ghi"}) falls to the raw body, the scrub misses the escaped form, and the key reaches the error and stderr. The provider-reported model is recorded and quoted in the denylist refusal unscrubbed as well.

## Grounds

- pursued: no representation of the key reaches an error or a result whatever encoding a provider echoes it in; shown wrong by a provider body that carries the key in an encoding neither decoded nor listed in keyForms, which TestNoRepresentationOfTheKeySurvivesInAnError would then need adding and would fail on
