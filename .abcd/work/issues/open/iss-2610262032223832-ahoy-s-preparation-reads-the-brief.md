---
schema_version: 1
id: "iss-2610262032223832"
slug: "ahoy-s-preparation-reads-the-brief"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 260 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:359"
remedy: "none (filed automatically)"
finding_ref: cfk-596532dfaa74cce5
---

Ahoy's preparation reads the brief, principles, ADRs and intents from the plugin root; launch says the plugin payload never ships the .abcd/ record

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:359` — "Read the abcd record from the plugin root: the three-tier README, the brief, principles, ADRs, intents"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:12` — "And it never ships the design record: the payload is default-deny with the whole `.abcd/` namespace excluded structurally, so no include line can put it back."

The install's preparation workflow opens by reading the abcd record (the brief, principles, ADRs and intents) from the plugin root, and its self-contained boundary says every applied asset resolves from that record or from the binary. Launch says the release payload is default-deny with the whole .abcd/ namespace excluded structurally, and that the catalog pins the cut archive so an install receives the release payload rather than the working tree. A plugin root installed that way holds no design record, so the orient phase depends on files the artefact it reads from cannot carry.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 260.
