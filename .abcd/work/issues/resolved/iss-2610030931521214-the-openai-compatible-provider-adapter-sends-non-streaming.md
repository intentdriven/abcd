---
schema_version: 1
id: "iss-2610030931521214"
slug: "the-openai-compatible-provider-adapter-sends-non-streaming"
severity: "major"
category: "bug"
source: "managed-repo"
found_during: "peer report from a sibling session's abcd user test of an adopted repository, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
remedy: "Send stream: true and assemble the server-sent events into the same answer the contract reads (the OpenAI chat-completions streaming format, data: chunks ending in data: [DONE]); replace the 120-second end-to-end deadline with an idle timeout between chunks plus a configurable total cap per route; cancel the request (close the connection) when the deadline or an interrupt fires, so the server stops generating; keep the response-size cap across the assembled stream; test against a fake streaming server for slow first token, steady chunks past the old deadline, a mid-stream stall and a cancelled call. Grounds: the OpenAI API reference's streaming format, which vLLM, llama.cpp, LM Studio and Ollama's OpenAI-compatible endpoints serve (state-of-the-art pass of 2026-10-03, reports/sota-guided-connect.md in the local tier, read the same servers' list endpoints)."
resolution: "The adapter asks for a stream and assembles the server-sent events into the answer the contract and denylist judge, accepting a plain body too; a first-byte limit (5 min), an idle limit (2 min) and a total cap (30 min) replace the 120 s deadline, and any limit or cancel closes the connection so the server sees the client go."
impact: fix
resolved_by:
  commit: "4db9c0d43"
---

The OpenAI-compatible provider adapter sends non-streaming chat requests with one end-to-end deadline, so a long delegated step routed to a reasoning model fails and wastes the provider's work. internal/adapter/openaiapi/client.go:291 renders "stream": false; DefaultTimeout (client.go:63) bounds the whole call at 120 seconds (connecting, sending and reading the entire answer), and only the connect verification passes its own timeout (internal/core/oracle/connect.go:116), so a dispatched step gets the 120-second default; max_tokens is sent only when a route or --route sets it (client.go:81). Evidence from a user test on 2026-10-03 against a keyless local OpenAI-compatible server on loopback: that server's gateway answers HTTP 504 to a non-streaming request whose generation exceeds about 600 seconds, because no headers are sent until the answer is complete, and the model keeps generating up to its token limit after the 504; reproduced twice with two reasoning models on a 44-line classification prompt, while a non-reasoning model finished in 181 seconds. abcd's own 120-second deadline abandons such a call earlier still, and an abandoned non-streaming request is not cancelled at the server. Any proxy or gateway with a header or idle timeout behaves the same way; intent consistency's bundled corpus (about 29k tokens) is the kind of step that reaches it.

## Grounds

- pursued: a long reasoning answer that keeps streaming is read to the end and a stall or cancel is cut off and seen by the server, as the fake-server tests in stream_test.go prove; a real OpenAI-compatible server that streams in a shape these tests do not cover (multi-line data events, a model name changing mid-stream) would show it wrong.
