---
schema_version: 1
id: "iss-2610262032368674"
slug: "configuration-says-install-asks-about"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 291 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:33"
remedy: "Edit 03-configuration.md (The keys the binary reads, lines 33-34): drop the oracle backend from the list of values install asks about; say install records oracle.backend as host-delegated with one note and asks only visibility, docs target, deep scan and the README question."
finding_ref: cfk-4c476ce17c61f7f3
refines: [iss-2610031236155833, iss-2609261447395216]
---

configuration says install asks about the oracle backend; ahoy says the oracle backend is not asked

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:33` — "Install asks about visibility, the docs target, the oracle backend"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:893` — "The oracle backend is not asked while host-delegated is the only answer with an adapter"

The configuration chapter lists the oracle backend among the values install asks about. The ahoy chapter says that question is not put to the person while host-delegated is the only answer with an adapter, and that install records host-delegated with a note. Install cannot both ask the question and skip it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 291.
