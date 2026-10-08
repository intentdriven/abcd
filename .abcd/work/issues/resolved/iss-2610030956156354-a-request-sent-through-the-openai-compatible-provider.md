---
schema_version: 1
id: "iss-2610030956156354"
slug: "a-request-sent-through-the-openai-compatible-provider"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "guided-connect record review, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
remedy: "Before sending, compare the request's estimated size with the model's served size where the server publishes one labelled as served, and refuse with a message naming both numbers and a listed model that fits; where no served figure is published, send as today and keep the server's own refusal text intact. Grounds: the 2026-10-03 guided-connect state-of-the-art report (only the server can judge exactly; listed figures differ in meaning), so the check must name its source and never guess."
resolution: "Before each chat call the OpenAI-compatible adapter now reads the service's model list. This GET is paid on every call, on every provider, including those whose list publishes no served size. Where the list labels the asked model's served size (vLLM's max_model_len, the only such field), the adapter asks the same server's /tokenize for an exact count of the brief. It refuses before the chat call only when that count plus max_tokens is over the served size. The refusal names both numbers, where each came from, and a listed model the server's own count says fits, or says that none was found. No refusal rests on an estimate. Without a count (no /tokenize, an error, or no answer in time) or without a served figure, the request is sent as before and the server's own refusal text is kept. The checks wait no longer than the listing did and are spent inside the call's total cap. llama.cpp, LM Studio and Ollama publish their served size outside the model list and are not checked, so the issue's own case may be untouched: the record does not name the reporter's server."
impact: fix
resolved_by:
  commit: "955c85c601c883008fa3d29de5f73224cb8f6617"
---

A request sent through the OpenAI-compatible provider adapter is never judged against the chosen model's size before it is sent. In a user test on 2026-10-03, an intent consistency check sent to a keyless local service carried about 29,300 tokens against a model the service runs at about 29,200, and the service refused it with HTTP 400, about 0.5% over. Some servers publish each model's served size in their model list; the standard list has no such field, and a published figure may be the trained size rather than the served one. Routed out of the guided-connect draft itd-2610030821294016 by its record review; a follow-up of the adapter's streaming fix captured the same day.

## Grounds

- pursued: a vLLM request whose /tokenize count plus max_tokens exceeds max_model_len is refused before the chat call; every other request, including one only a bytes estimate puts over, is sent unchanged and the server's own refusal is kept, as size_test.go's fake-server tests prove. What would show it wrong: a vLLM version whose /tokenize count differs from the prompt its chat endpoint judges, or which refuses on a different sum.
