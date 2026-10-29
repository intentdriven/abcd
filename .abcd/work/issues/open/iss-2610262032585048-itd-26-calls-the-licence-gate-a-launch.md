---
schema_version: 1
id: "iss-2610262032585048"
slug: "itd-26-calls-the-licence-gate-a-launch"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 343 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "Edit itd-26 line 70: remove 'launch-gate-licence-gate' from the machinery loot reuses; say loot's refusal is its own `loot.acceptable_licences` check built on the substrate's licence detection, and that the substrate's restrictive-licence publish gate belongs to the lifeboat (adr-28), inert at launch. This edit also covers the sibling record about the unbuilt refusing gate."
related_intents: [itd-26]
finding_ref: cfk-b8e6ef50824453be
---

itd-26 calls the licence gate a launch gate; launch says the restrictive-licence gate is not the release artefact's gate, and that its consumer is the lifeboat

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "launch-gate-licence-gate machinery itd-26 needs is the **same** machinery memory ingest needs"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:477` — "The restrictive-licence gate over kept originals is likewise **not this artefact's gate**. Its real consumer is the lifeboat"

itd-26 builds on a substrate it names as including the 'launch-gate-licence-gate'. The launch chapter says the restrictive-licence gate is not this artefact's gate: its real consumer is the lifeboat, it is future and inert at launch, and it insists on 'two mechanisms, never one name'. Calling it a launch gate attaches the gate to the surface that says it is not there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 343.
