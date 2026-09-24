---
schema_version: 1
id: "iss-2609231931064874"
slug: "itd-147-ac-6-and-ac-8-fidelity-audit-the-intent-s-outcome"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release-gate/README.md"
---

itd-147 ac-6 and ac-8 (fidelity audit): the intent's outcome check is owed and has no ledger record. The intent shipped by PR #671 with ac-6 (a full-tier brief-to-surface crosscheck after the seam lands classifies to no false-claim or stale-count about a covered flag or sub-verb) and ac-8 (progress assessed on which chapters are implicated) deferred to the next release gate; the only carrier is a paragraph in .abcd/development/release-gate/README.md that asks to be deleted when the result is recorded. Nothing in CI or the release cut asserts it, so a cut that skips the classification leaves the intent's headline outcome unverified with nothing to show it. The fidelity audit recorded ac-6 INCONCLUSIVE and ac-8 MET_WITH_CONCERNS for this reason. Resolve by running the full-tier crosscheck at the next release gate, classifying its findings chapter by chapter against the baseline table in research/data/2026-08-23-brief-surface-crosscheck/README.md, and recording the result with the run's commit in itd-147's Audit Notes.
