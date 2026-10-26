---
schema_version: 1
id: "iss-2610262031188637"
slug: "04-scope-defines-what-has-shipped-by-the"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 113 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:42"
remedy: "none (filed automatically)"
finding_ref: cfk-986a248f630c6bce
---

04-scope defines what has shipped by the intents in shipped/ and the derived release, phases being history; 03-out-of-scope still frames the brief around eight planned phases and defines what has shipped by which phases are complete.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:42` — "What has shipped is defined by which intents are in `shipped/`, and the checkpoint is the derived release"
- End B: `.abcd/development/brief/06-delivery/03-out-of-scope.md:174` — "What has shipped is defined by which phases are complete and which intents are in `shipped/`"

04-scope says no stored unit sits above the intent, the phase documents are history, and 'What has shipped is defined by which intents are in `shipped/`, and the checkpoint is the derived release'. 03-out-of-scope opens with the work bundled into the eight planned phases, schedules a planned intent when a phase doc's ## Scope names it, has a later-phase intent enter work by being scoped into a phase, and closes with 'What has shipped is defined by which phases are complete and which intents are in `shipped/`'. 04-scope names 03-out-of-scope as the canonical out-of-scope list, so a reader following that pointer is handed the retired sequencing model as current.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 113.
