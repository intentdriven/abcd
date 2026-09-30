---
schema_version: 1
id: "iss-2609100506265392"
slug: "intent-audit-asks-the-host-for-a-diff-range-it-cannot-know"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (intent audit request, spec close receipt)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Should spec close record the closing commit on its audit receipt, or should the audit request sanction auditing the shipped tree at a named commit and record which form was used? Planning is owed before code (itd-2609201916151817 carries a range only for build lanes)."
remedy: "Waits on ruling CA4 (record the closing commit, sanction a tree audit, or both): if the closing commit is recorded, have spec close write its HEAD sha onto the owed audit receipt and the audit request print it in the delivered line in place of host supplies the range; if tree audits are sanctioned, have the request name the tree at a commit as an accepted form and ingest require a form field (diff or tree) on the verdict; if both, do both; each proven by a request-rendering test in internal/core/intent."
---

The fidelity-review request names its delivered side as "the diff/commit range that realised spc-N (host supplies the range)", and there is no mechanism by which the host can supply it.

abcd holds the spec id and the intent, so it knows WHICH work it means; what it does not do is resolve that to commits, and it offers the auditor no way to ask. In practice the host has to guess a range, and the guess is only as good as the commit messages. In one managed repository it was not good at all: the repository's history had been rewritten during a rename, and only six distinct `spc-` ids survived in commit bodies across thirteen shipped intents. Seven of the thirteen had no derivable range whatsoever.

The workaround was to audit the shipped TREE at HEAD instead of a diff, with `file:line` citations. That is arguably better — it judges what is actually running rather than what one range happened to touch — and every one of the thirteen auditors managed it. But it is a substitution the request does not sanction, made silently by whoever runs the audit, and two audits of the same intent by different hosts can therefore be answering different questions without either saying so.

A second session in a different managed repository hit the same wall from the other side and supplied the sharper remedy. Re-running `abcd intent audit <itd>` on an already-owed request still printed "host supplies the range", so the host dug merge commits out of git by hand for three receipts. The point they make: the receipt was minted at `abcd spec close`, at which moment the closing commit was known and was not written down. The information the auditor needs was in the tool's hand when the receipt was created.

Needed, cheapest first: say in the request that auditing the tree at a named commit is an accepted form, and have the verdict record WHICH form was used, so a reader can tell a diff audit from a tree audit. Better: record the closing commit on the receipt at `spec close`, or let a spec record the commits that realised it — the resolve path already accepts a `--commit` pointer for issues — so the range is a fact the record carries rather than one the host reconstructs from prose that a history rewrite can erase.

Distinct from the sibling finding that the same request carries no provenance hashes while `ingest` requires them: that one is about the hashes, this one is about the range.

## Remedy grounds (2026-09-29)

Why: spec close is the one moment the tool holds the closing commit, and a verdict that names its form lets two audits of one intent be compared honestly. Rejected: deriving the range from spc- ids in commit messages, which the record shows a history rewrite erased for seven of thirteen intents.
