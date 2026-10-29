---
schema_version: 1
id: "iss-2610271115000748"
slug: "filing-every-minor-editing-inconsistency"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "interview on the 2026-10-25 run's outcomes, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/reviews/2026-10-26-consistency/00-summary.md"
remedy: "Build the four rungs of the discipline itd-2610271130017437, in order: teach the rule in the bundled ISSUES rules domain every managed repository loads; make capture refuse a minor inconsistency or documentation finding that names no reason it cannot be fixed in place (needs-ruling, outside-boundary or too-large); give the consistency ingest a safe-or-ruling mark on each finding, applying or proposing safe fixes and filing rulings alone (the automatic mode itd-2609291923559186 plans); and have each pass report its not-real rate. Grounds: .abcd/development/research/notes/2026-10-27-fix-in-place-or-file-sota.md (Infer at review time over 70% fixed against almost none in batch; FindBugs Fixit 640 of 3,954 fixed; the safe-fix tiers of Ruff and ESLint)."
---

Filing every minor editing inconsistency as an issue costs far more than fixing it where it is found. The 2026-10-26 consistency pass filed 1301 records from 1307 findings. Its triage then found 584 of them to be one-passage edits with a known fix, 648 to need no change at all (duplicates, conflicts a later record already settles, or no real conflict), and only 69 to need a person's ruling. Every one was filed, read by a triage agent and approved, and the edits still have to be made: work an in-place fix at discovery would have done once, with none of the noise. The product thinker's lesson (2026-10-27): a minor editing inconsistency should become an issue only when it cannot be fixed in place, because it needs a ruling, crosses a boundary the finder may not change, or is too large for the change at hand. Otherwise it is fixed in place as it is discovered.
