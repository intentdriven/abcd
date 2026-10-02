---
schema_version: 1
id: "iss-2609302306003610"
slug: "the-surface-appendix-generator-labels-abcd-version-a"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.12.0 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/appendix.go"
remedy: "Emit HostDelegatedSentence only for a row the host_delegated list names, and give a root-flag surface such as version its own sentence naming the flag; regenerate the 12-version.md appendix and pin both sentences with a generator test."
resolution: "The appendix generator reads record-lint's surface_coverage host_delegated list: a verbless shipped row is host-delegated only when the list names it, a root-flag surface such as version gets a sentence naming the flag, and any other verbless shipped row is refused by name; 12-version.md is regenerated and TestRegenerateChaptersLabelsAShippedRowWithNoVerbByWhatItIs pins the three sentences and the refusal."
impact: internal
resolved_by:
  commit: "94ac9a9a1"
---

The surface appendix generator labels /abcd:version a host-delegated command page: HostDelegatedSentence (internal/core/surface/appendix.go:84) is emitted for every shipped register row with no registered verb, so the generated appendix of 12-version.md:147 says the command page is host-delegated and has no flags, while commands/version.md runs the binary's root --version flag and record-lint.json's host_delegated list holds only consult, ingest and prepare-this-repo. The generator, not the prose, is wrong, and it re-emits the error on every regeneration. Found by the v0.12.0 release-gate brief-surface cross-check (x-039, checker a11, and its duplicate x-118, checker b00); the generator is new in the v0.12.0 cycle.

## Grounds

- pursued: the generated appendix states what each verbless surface actually is; a regenerated chapter that calls a non-listed surface host-delegated, or a drift-test pass over a misdescribed chapter, would show it wrong
