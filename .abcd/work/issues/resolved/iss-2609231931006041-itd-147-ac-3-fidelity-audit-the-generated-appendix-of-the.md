---
schema_version: 1
id: "iss-2609231931006041"
slug: "itd-147-ac-3-fidelity-audit-the-generated-appendix-of-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/13-consult.md"
resolution: "The appendix generator reads the register's Status cell: a command the tree does not register gets HostDelegatedSentence when its row reads shipped and UnbuiltSentence otherwise (internal/core/surface/appendix.go, Chapter.Appendix). 13-consult.md, 14-ingest.md and 15-prepare-this-repo.md are regenerated and say they ship as host-delegated command pages; 09-reflect.md and the staged worktree row keep the unbuilt sentence. TestShippedChapterNeverClaimsNoShippedSurface (internal/surface/cli) fails on any shipped chapter that says there is no shipped surface, and was watched fail on the three chapters before the change. The register prose in 04-surfaces/README.md states both sentences."
impact: internal
resolved_by:
  commit: "7a7ea0130"
---

itd-147 ac-3 (fidelity audit): the generated appendix of the three host-delegated commands says 'There is no shipped surface' while their register rows read shipped. UnbuiltSentence (internal/core/surface/appendix.go) is emitted for any chapter whose command the Go tree does not register, and /abcd:consult, /abcd:ingest and /abcd:prepare-this-repo are shipped host-delegated commands with a command page and no Go verb (04-surfaces/README.md, No skills). The block therefore states a false claim about the surface — the class the intent exists to remove — in 13-consult.md, 14-ingest.md and 15-prepare-this-repo.md. A host-delegated chapter needs its own sentence: shipped as a command page, with no Go verb and so no flags or sub-verbs to list; the generator can tell the cases apart from the register's Status column, which it already parses.

## Grounds

- pursued: every chapter whose register row reads shipped carries no 'no shipped surface' sentence; a new host-delegated command added with a shipped row and the old sentence would show it wrong, and the test fails on exactly that
