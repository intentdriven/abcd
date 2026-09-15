---
schema_version: 1
id: "iss-2608311331229368"
slug: "spc-65-and-itd-187-both-prescribe-watching-testassembledinpu"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Both records now name the mutation that can establish the precondition -- rebuilding the assembler's candidate slice from a map, so the walk order is genuinely unstable across processes -- and say why removing the walk sort does not serve: it yields a different order rather than an unstable one, so the two assemblies still agree. The walk-sort mutation stays where it belongs, against the order oracle, which it does falsify."
impact: internal
resolved_by:
  intent: "itd-187"
  commit: "bde26ff0"
---

spc-65 and itd-187 both prescribe watching TestAssembledInputIsByteIdenticalAcrossRuns red by removing the assembler's walk sort. Run on a copy of the tree, that mutation does not make it red: removing the sort yields a different but still deterministic order, so two assemblies of one commit stay byte-identical and only the order oracle fires. The mutations that do make the identity assertion red are a genuinely nondeterministic walk (candidates rebuilt from a map) and an absolute repository path embedded in the bundle. The record's prescribed hand-run for ac-1 names a mutation that cannot falsify ac-1.

## Grounds

- pursued: the acceptance record prescribed a hand-run that could not verify the criterion it was attached to, so a builder following it faithfully would have logged evidence for a criterion the mutation cannot touch; what would show this wrong is the named mutation failing to redden the byte-identity comparison.
