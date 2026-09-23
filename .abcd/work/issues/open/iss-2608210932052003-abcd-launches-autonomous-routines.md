---
schema_version: 1
id: "iss-2608210932052003"
slug: "abcd-launches-autonomous-routines"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "itd-131 decomposition; user vision"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M6: product thinker deferred the ruling in the 2026-09-23 interview: external security audits are being explored instead of abcd-launched bug-hunt routines)."
---

abcd launches autonomous bug hunts (and other routines) for the user, opt-in — beyond handing the user a prompt to paste into a cloud routine. Today the bughunt is a Desktop prompt the user wires into an external cloud routine by hand; the direction is abcd owning the launch: the user opts in, and abcd assembles and starts the routine, applying the run contract it already governs (the human git identity per itd-131, the gates, the merge decision, the state issue). This is the realisation of the run seam — adr-27 (run is a pluggable seam, not a bespoke engine), itd-29 (the run operator surface: start/status/pause/resume/ship), itd-107 (routines assemble from one versioned template; the bughunt and a delivery-pipeline archetype), and the reframed iss-381 (the deterministic delivery pipeline survives as an itd-107 archetype, not an engine). When abcd launches the routine it sets the human git identity at launch, which is the clean mechanism the itd-131 identity gate points at for routine commits (vs today's prompt/env workaround). Big, cross-record capability — needs decomposition and likely ideate before it is filable; recorded so the direction is durable.