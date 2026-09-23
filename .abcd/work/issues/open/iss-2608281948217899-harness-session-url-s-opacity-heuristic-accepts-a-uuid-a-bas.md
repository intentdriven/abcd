---
schema_version: 1
id: "iss-2608281948217899"
slug: "harness-session-url-s-opacity-heuristic-accepts-a-uuid-a-bas"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-152 security review"
found_at: "internal/adapter/scanner/harnessleak.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (fact owed: enumerate the session-URL id spellings covered harnesses emit, then widen the opacity test or record the boundary; not settleable from the repo). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

harness_session_url's opacity heuristic accepts a UUID, a base62 token (digit plus uppercase) and a long lowercase-hex run, and rejects an all-lowercase non-hex id such as session_zk3n9q2mvx8rtq. The rejection is deliberate: without it every documentation slug that follows the word session (there is one in .abcd/development/research/notes/2026-08-22-context-window-management-sota.md) becomes a finding. Whether any harness in scope mints an all-lowercase id is a fact about external services that cannot be settled from the repository. Settle it by enumerating the session-URL spellings the covered harnesses actually emit, then either widen the opacity test or record the coverage boundary in the pattern's own comment. Found by adversarial security review of itd-152.