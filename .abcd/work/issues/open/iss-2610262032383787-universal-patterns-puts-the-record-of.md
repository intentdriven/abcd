---
schema_version: 1
id: "iss-2610262032383787"
slug: "universal-patterns-puts-the-record-of"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 296 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:101"
remedy: "Edit 04-universal-patterns.md section 6 layout tree: drop the ahoy prompts.json line (or note that setup answers are recorded in .abcd/.work.local/interviews/ and the machine tier's interviews/)."
finding_ref: cfk-22c3f84ba8c2dfd4
---

universal patterns puts the record of setup questions and answers in logs/ahoy/< timestamp>/prompts.json; ahoy writes it to .abcd/.work.local/interviews/

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:101` — "what was asked, what was answered"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:879` — "The repository's answers go to `.abcd/.work.local/interviews/setup-<stamp>.json`"

The universal-patterns log layout puts the record of what ahoy asked and what was answered in a prompts.json under logs/ahoy/< timestamp>/. The ahoy chapter writes that same record, each question as asked with its answer, to .abcd/.work.local/interviews/setup-< stamp>.json. The same artefact has two homes that do not agree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 296.
