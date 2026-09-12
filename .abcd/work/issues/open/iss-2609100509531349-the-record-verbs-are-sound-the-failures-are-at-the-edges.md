---
schema_version: 1
id: "iss-2609100509531349"
slug: "the-record-verbs-are-sound-the-failures-are-at-the-edges"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "This is a synthesis rather than a defect: the observation that the record verbs are sound and the failures are at the edges, where the tool knows something and does not say it. It cannot be fixed because it is not broken; it is a claim about where to look, and its value is as a lens over the individual findings that evidence it. Several of those are fixed in this cut, which is the only sense in which this record advances. It stays open deliberately, as the place the pattern is recorded."
found_at: "internal/surface (refusal paths across verbs)"
---

The record verbs are sound; the failures were all at the edges, where the tool knows something and does not say it.

This is the synthesis a session reached after a day of autonomous work in a managed repository, and it is recorded as its own claim rather than as a note on any one finding, because it is a claim about WHERE abcd fails rather than a list of failures.

The centre held. The record store, the folder-as-status model, and the verbs that move records between states all worked, including under 27 parallel branches from separate worktrees, and the ledger's one-file-per-record shape produced no merge conflict in the entire run. Nothing found in that run argues against the core design.

What failed was a repeatable shape at the boundary. In each instance abcd possesses the exact information the operator needs, at the exact moment it refuses, and emits a refusal without it:

- the accepted values it is validating against, when it rejects a `--category` or a `--source` (it holds the enum; the error names only the offending value);
- the closing commit it had in hand at `spec close`, when the fidelity-review request it later emits asks the host to supply the delivered range (it had the commit at mint time and did not write it on the receipt);
- the include file it is looking for, when `launch --dry-run` refuses (it names the path but not the condition, not how to create one, and not that this repo may not need one at all).

Others found in the same run fit the shape: the id that appears in a default-branch commit message while its record stays open; the record it just wrote that is untracked; the sibling lint it does not run and does not mention; the dangling symlink whose target it already read.

The claim is testable, which is what makes it worth a record. If it holds, the class has one remedy rather than many: at every refusal, emit what the tool already knows about the thing it is refusing — the legal set, the value it had, the condition it is checking. That is cheaper than the individual fixes and it closes the ones nobody has hit yet.

The individual findings from this run are its evidence. It should be graded, and either promoted to a principle or falsified by the next run's findings landing somewhere other than the edges.

**Evidence from the same run, by record.** The three instances cited above:
iss-2608290810037524 (the accepted enum values, corroborated twice in this run),
iss-2609100506265392 (the delivered range the receipt could have carried from
`spec close`), iss-2608270559313719 (the include file `launch --dry-run` names
without naming the condition, corroborated twice in this run). The others that
fit the shape: iss-2609100507421759 (an open id cited in a default-branch
commit), iss-2609100508570527 (the record it just wrote is untracked),
iss-2609100508566033 (the sibling capability it never names),
iss-2609100506256636 (the dangling symlink whose target it already read),
iss-2609100505140261 (the provenance hashes one half of a verb requires and the
other half never emits), iss-2609100505142469 (the redaction it performed and
reports only as a count). The positive control is iss-2609100508570803: the
centre, exercised hardest, produced one wording defect and no failures.
