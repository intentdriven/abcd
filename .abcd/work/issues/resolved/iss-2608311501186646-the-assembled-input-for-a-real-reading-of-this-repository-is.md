---
schema_version: 1
id: "iss-2608311501186646"
slug: "the-assembled-input-for-a-real-reading-of-this-repository-is"
severity: "critical"
category: "bug"
source: "agent-finding"
found_during: "end-to-end reading rehearsal, cold-reading cycle 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/assemble.go"
resolution: "The committed preset file moves to schema version 2, where each position's entry states the object set the run is about, the kinds admitted within it, and the estimated-token window it was calibrated for, beside the figure measured and the commit measured on. A new eval in the cold-reading lane dry-run-assembles every assembling position's entry on a clean clone of HEAD and fails past the declaration, naming the position, the measured figure and the declaration; the size report states any total over the two-hundred-thousand-token target."
impact: fix
resolved_by:
  intent: "itd-2609020625400445"
  spec: "spc-2609020626048722"
---

The assembled input for a real reading of this repository is about 9.8 MB at every position, roughly 2.45 million tokens, so no reading can be handed one. Measured by running the shipped assembler over the tree at a real commit: widening 9808456 bytes across 951 items, entailment 10156762 across 1297, comparative and detection the same as widening, with the largest single item at 280242 bytes. Field projection works and is not the problem: 104 of the 951 items are projected to a named field and 847 travel whole, so the whole-file rows dominate by two orders of magnitude. The evals cannot see this because they run over a miniature fixture repository of about thirty files, which is the right corpus for asserting a firewall and the wrong one for asserting that the artefact fits anywhere. Nothing in the record states a size budget for the assembled input, and no criterion measures one, so the instrument can be entirely correct and still undeliverable. The instrument has never been run, which is how this survived thirteen intents.

## Grounds

- pursued: we expect a declared window per entry, re-measured by a dry-run eval on every committed change, to keep the assembled input deliverable, because the size defect reached release only because nobody measured; an entry that passes its eval and still overflows the reader it was calibrated for by more than the divisor's recorded spread would show it wrong.
