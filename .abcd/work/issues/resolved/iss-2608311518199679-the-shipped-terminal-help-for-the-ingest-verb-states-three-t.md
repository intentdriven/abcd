---
schema_version: 1
id: "iss-2608311518199679"
slug: "the-shipped-terminal-help-for-the-ingest-verb-states-three-t"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reading.go"
resolution: "The cobra long description for the ingest verb is corrected, and the reference page regenerated from it. The durability sentence now says what is true — a refusal becomes durable once the run's identity is proven, and nothing is written before that point — the orphan paragraph says in terms that the sweep rolls the run's reading records out of the committed ledger and that the ids are reported on every exit, and the reserved-name sentence says the names a regime declares, naming the generative position as declaring none. The plugin page carries the same corrections, and a wiring test now holds the page's reported keys against the fields the result actually emits."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The shipped terminal help for the ingest verb states three things the code does not do, and because the CLI reference page is generated from the command's own long description, the false text is what an operator reads at the terminal and in the docs alike. It says nothing durable is written until the whole payload validates, which the orphan sweep contradicts by deleting a committed record during a refused run. It describes the orphan sweep in wording that omits the committed-tier delete, which is the exact sentence the code's own comment warns must stay true. And it says each regime has reserved names, which is false at the generative position, where there are none and no signature refusal either. A generated page inherits whatever the source string claims, so one wrong sentence in a Long field becomes a wrong sentence in two shipped surfaces at once.

## Grounds

- pursued: a generated reference page inherits whatever its source string claims, so a wrong sentence in a long description is wrong in two shipped surfaces at once and the fix belongs in the string; a help sentence contradicted by the verb's own behaviour would show this wrong
