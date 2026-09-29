---
schema_version: 1
id: "iss-2608210932052003"
slug: "abcd-launches-autonomous-routines"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "itd-131 decomposition; user vision"
deferred_after: v0.11.1
deferral_reason: "The product thinker deferred this ruling on purpose at the 2026-09-23 interview (M6): external security audits are being explored instead of abcd-launched bug-hunt routines. Owed: one ruling: build abcd-launched routines, or close this record in favour of the external audits?"
remedy: "Waits on ruling M6 (build abcd-launched routines, or close in favour of external audits): if built, render the routine from itd-107's one versioned template as a scheduled workflow on the forge that sets the human identity, the gates and the merge rule at launch, proven by a scaffold test that renders the workflow and a gate test that refuses its commit under a tool identity; if closed, resolve this record as wontfix naming the external audit that replaces it, and point iss-2608210738367948's establishment at the documented environment step itd-131 names."
---

abcd launches autonomous bug hunts (and other routines) for the user, opt-in — beyond handing the user a prompt to paste into a cloud routine. Today the bughunt is a Desktop prompt the user wires into an external cloud routine by hand; the direction is abcd owning the launch: the user opts in, and abcd assembles and starts the routine, applying the run contract it already governs (the human git identity per itd-131, the gates, the merge decision, the state issue). This is the realisation of the run seam — adr-27 (run is a pluggable seam, not a bespoke engine), itd-29 (the run operator surface: start/status/pause/resume/ship), itd-107 (routines assemble from one versioned template; the bughunt and a delivery-pipeline archetype), and the reframed iss-381 (the deterministic delivery pipeline survives as an itd-107 archetype, not an engine). When abcd launches the routine it sets the human git identity at launch, which is the clean mechanism the itd-131 identity gate points at for routine commits (vs today's prompt/env workaround). Big, cross-record capability — needs decomposition and likely ideate before it is filable; recorded so the direction is durable.

## Deferral 2026-09-29

Deferred past v0.11.1: The product thinker deferred this ruling on purpose at the 2026-09-23 interview (M6): external security audits are being explored instead of abcd-launched bug-hunt routines. Owed: one ruling: build abcd-launched routines, or close this record in favour of the external audits?

## Remedy grounds (2026-09-29)

- OpenSSF Scorecard rewards analysis wired into the repository's own workflows on every merged change (SAST, fuzzing) rather than a point-in-time review (https://github.com/ossf/scorecard/blob/main/docs/checks.md, consulted 2026-09-29): scheduled hunts and an external audit are complementary, so the ruling is about who launches the hunt, not either-or.
- Rejected: an abcd-owned scheduler or daemon, which adr-27 (run is a pluggable seam, not a bespoke engine) rules out.
