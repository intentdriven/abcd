---
schema_version: 1
id: "iss-2609300805371434"
slug: "a-provider-that-reports-model-is-accepted-openaiapi-s-decode"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
remedy: "Treat an empty (or blank) reported model as a contract failure in openaiapi's decode: refuse the answer with an error naming that the provider reported no model, so what answered is always on the record and the denylist is never bypassed by silence; proven by a client failure-table case for model \"\" and for a blank model. Grounds: review-denyRetire INFO 3; the chat-completions response object carries the model that answered, which the adapter records as ModelReported."
---

A provider that reports model "" is accepted: openaiapi's decode records ModelReported empty and returns the answer, so the record cannot show what answered and no oracle.denylist entry can ever match it (internal/adapter/openaiapi/client.go decode).
