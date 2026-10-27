---
schema_version: 1
id: "iss-2610262032520382"
slug: "the-phase-glossary-names-phase-1"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 329 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:64"
remedy: "none (filed automatically)"
finding_ref: cfk-825ea187a21cedbf
---

the phase glossary names Phase 1 'Substrate'; launch links Phase 1 as the ahoy phase document

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:64` — "ships in [Phase 1] (../../roadmap/phases/phase-1-ahoy.md)"
- End B: `.abcd/development/brief/glossary/core/phase.md:72` — "Phase 1 — Substrate ends when the oracle backend ships and project state is reconciled."

core/phase's example gives Phase 1 the name Substrate. 04-launch's ownership note says the curated-release cut ships in Phase 1 and links phase-1-ahoy.md for it. One phase number carries two names, so a reader cannot tell which history document Phase 1 is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 329.
