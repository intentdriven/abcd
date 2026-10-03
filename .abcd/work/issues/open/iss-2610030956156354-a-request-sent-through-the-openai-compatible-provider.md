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
---

A request sent through the OpenAI-compatible provider adapter is never judged against the chosen model's size before it is sent. In a user test on 2026-10-03, an intent consistency check sent to a keyless local service carried about 29,300 tokens against a model the service runs at about 29,200, and the service refused it with HTTP 400, about 0.5% over. Some servers publish each model's served size in their model list; the standard list has no such field, and a published figure may be the trained size rather than the served one. Routed out of the guided-connect draft itd-2610030821294016 by its record review; a follow-up of the adapter's streaming fix captured the same day.
