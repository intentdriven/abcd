---
schema_version: 1
id: "iss-2610262042130595"
slug: "itd-6-keeps-a-re-review-in-the-same-chat"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1055 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-cf2d5a09bdf350f1
---

itd-6 keeps a re-review in the same chat for every backend, the in-session subagent included; the verification matrix says consecutive host-delegated calls are independent dispatches.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:45` — "Same rule applies whether the backend is RP, Codex, or in-session subagent (in-session uses `Task` with continuation prompts)."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:34` — "Two consecutive calls are independent dispatches, so the audit-fix loop re-primes context every iteration"

itd-6 makes it a rule that a re-review after fixes MUST stay in the same chat, and applies the rule to the in-session subagent through continuation prompts. The matrix's default host-delegated oracle row says two consecutive calls are independent dispatches, so the audit-fix loop re-primes its context on every iteration. The default backend cannot both carry on the earlier conversation and start fresh each time.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1055.
