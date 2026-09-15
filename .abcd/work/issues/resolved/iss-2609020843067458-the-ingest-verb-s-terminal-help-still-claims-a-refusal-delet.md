---
schema_version: 1
id: "iss-2609020843067458"
slug: "the-ingest-verb-s-terminal-help-still-claims-a-refusal-delet"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "iss-2608311518199679 follow-up"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reading.go"
resolution: "The ingest verb's cobra Long field — the source docs/reference/cli/commands.md is generated from — describes the code. A refusal after the run is proven writes its refusal record and nothing else, and the one delete it makes is on its own run id (rollbackThisRun); the help says that rather than claiming no delete, and the neighbouring claim is qualified to nothing DURABLE, the stage root and its lock being created first. The orphan sweep is split into its two outcomes: no commit marker and the run's reading records leave the committed ledger, marker present and the run stands with only the stage going. The reserved-name rule points at the table the gate reads — ReservedNames at the run's own regime, one row per regime, the generative regime carrying none — and says a reserved name refuses when the item carries it as a field of its own; the prose-signature registry is left out of the help rather than restated, no signature refusing anything. internal/surface/cli/reading_help_truth_test.go holds each sentence to the behaviour and pins the three; commands/reading.md and the brief chapter 04-surfaces/23-reading.md carried the same claims and are corrected, the brief also having asserted that every signature ships enforced where all four ship in flag mode."
impact: fix
resolved_by:
  commit: "3a4601be"
---

The ingest verb's terminal help still claims a refusal deletes nothing, and gives the orphan sweep one outcome where the code has two. iss-2608311518199679 named three false sentences in the same Long field and its fix reaches two of them; the durability sentence reads "written or deleted" without the qualification that makes it true, and the sweep sentence carries the committed-tier delete without the case in which there is none. A refusal after the run's identity is proven calls rollbackThisRun on every path, which removes the never-committed reading records of an earlier attempt at the same run id from the committed ledger — so "leaves its refusal record and nothing else" describes a verb that makes no delete, and the verb makes one. rollbackRun returns without touching anything when the run's commit marker is present, and sweepOrphanStages clears the stage regardless, so a sweep over a leftover stage rolls nothing back and the help promises a rollback that does not come. The neighbouring "before that point nothing is written anywhere" is false of the stage root and its lock, which are created before the payload is read; the claim is about the durable tier. Because docs/reference/cli/commands.md is generated from the Long field, each of these is wrong at the terminal and on the reference page at once.

## Grounds

- pursued: an operator reading the ingest help at the terminal or in the generated reference finds no sentence the code contradicts, and a later edit that reintroduces one fails TestIngestHelpPinsTheThreeMeasuredSentences; a falsifier is a claim in the Long field that no test ties to observable behaviour, or a change to the refusal or sweep paths that leaves the pinned sentences passing
