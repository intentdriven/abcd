---
schema_version: 1
id: "iss-2608282124231752"
slug: "scanner-scruboutbound-has-no-front-door-it-is-defined-tested"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "intent-implementation-run"
found_at: "internal/adapter/scanner/outbound.go"
related_intents: [itd-107, itd-152]
related_issues: [iss-178]
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: which front door scanner.ScrubOutbound gets (scrub verb or itd-107 posting path); spc-45 scoped it out). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

scanner.ScrubOutbound has no front door: it is defined, tested and documented as the outbound-artefact scrub, but no CLI command, plugin verb or routine prompt calls it, and internal/ cannot be imported from outside the module, so the posting-time half of the harness-leak class is unmitigated in practice. This is not an oversight in spc-45, which deliberately scopes a forge client out; the gap is that nothing hands the primitive to the party that does post. Two candidate shapes for a future change: a small stdin-reading verb (abcd scrub --label pr-body) that the autonomous routine prompts pipe every PR body, issue and comment through before posting; or wiring the primitive into whatever posting path lands with itd-107, so an assembled routine scrubs by construction. Until one lands, the operative control is the re-read-and-strip policy in scanner.OutboundPolicy, as AGENTS.md, iss-178 and spc-45 state.