---
schema_version: 1
id: "iss-2610140739284100"
slug: "four-longest-run-counters-want-one"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "cut 2, installer branch's code-span canonical test, 2026-10-14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/termsafe/prose.go"
remedy: "Export one termsafe helper that returns the longest run of a set of fence characters (backtick, tilde) and move termsafe's longestBacktickRun, history's reconstruct_render counter, capture's promote codeSpan counter and the rules override fence writer onto it; the code-span canonical test then drops their allowlist entries."
---

The tree holds four hand-written longest-run counters for backtick or tilde runs: termsafe's private longestBacktickRun (internal/termsafe/prose.go), internal/core/history/reconstruct_render.go, internal/core/capture/promote.go's codeSpan, and the override fence writer in internal/core/rules (landing with the cut-2 installer branch). The one-canonical-primitive principle forbids a third copy; each is correct today, and each is allowlisted in the code-span canonical test rather than routed through one helper.
