# Adopt contributor commits

**The rule.** When an external contributor has a working branch, the default is
to ADOPT their commit — merge or cherry-pick it with their authorship intact —
so the contribution carries their name in the contributor graph. Re-authoring
the fix and crediting them with a `Reported-by:` trailer is the honest fallback
only when there is no branch to adopt: `Reported-by` credits the report, not
the code, and using it where a branch exists takes authorship the contributor
already holds. The `ACKNOWLEDGEMENTS.md` entry rides the adopting change, in
the same diff — a missing entry is a follow-up debt, not a separate decision.

Surfaced by the second operator in the 2026-08-27 security-advisory pilot
(F-W): the issue-sweep's re-author-with-`Reported-by` default cost a
contributor with a ready branch their contributor-graph authorship. The
enabling convention beneath this principle is `.github/CONTRIBUTING.md`'s attribution
section; the discipline rung (a gate that notices an adopted-and-rewritten
external branch) is unfiled.
