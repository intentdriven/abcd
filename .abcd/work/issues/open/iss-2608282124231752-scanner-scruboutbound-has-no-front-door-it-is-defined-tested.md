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
remedy: "Waits on the owed planning ruling; if a scrub verb, add abcd scrub --label <kind>, reading stdin and writing ScrubOutbound's text to stdout with findings on stderr, wired on the CLI and a commands page, and have the routine prompts pipe every pull-request body, issue and comment through it; if itd-107's posting path, call ScrubOutbound inside that path so an assembled routine scrubs by construction. Either way the re-read-and-strip step stays until a test proves a session URL and a tool footer on the input come out stripped."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Which front door does scanner.ScrubOutbound get: a scrub verb, or itd-107's posting path?"
---

scanner.ScrubOutbound has no front door: it is defined, tested and documented as the outbound-artefact scrub, but no CLI command, plugin verb or routine prompt calls it, and internal/ cannot be imported from outside the module, so the posting-time half of the harness-leak class is unmitigated in practice. This is not an oversight in spc-45, which deliberately scopes a forge client out; the gap is that nothing hands the primitive to the party that does post. Two candidate shapes for a future change: a small stdin-reading verb (abcd scrub --label pr-body) that the autonomous routine prompts pipe every PR body, issue and comment through before posting; or wiring the primitive into whatever posting path lands with itd-107, so an assembled routine scrubs by construction. Until one lands, the operative control is the re-read-and-strip policy in scanner.OutboundPolicy, as AGENTS.md, iss-178 and spc-45 state.

## Remedy grounds (2026-09-29)

- Why: the record's two shapes, each with its wiring and proof; the ruling is unanswered and none is picked. The check direction already has a front door (abcd lint outbound), so either answer adds only the rewrite direction.
- Rejected: retiring the re-read-and-strip policy before a front door is proven, which would drop the only control at posting time.
