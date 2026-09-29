---
schema_version: 1
id: "iss-2609290055092630"
slug: "fsutil-redactroot-and-the-scanner-s-home-sweep-still"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the technical facilitator (rulings-owed BB; lane reanchor, run A 2026-09-29): which leading-boundary rule both redactors follow for a home of two or more segments under a longer root, the scanner sweep's (redact it, iss-324) or fsutil.RedactRoot's (leave it whole, iss-2609230641546141, so '/private~/wt/a' is never produced); the '.' polarity's rendering and the case-folding split in the 2026-09-29 evidence ride on the same ruling"
---

fsutil.RedactRoot and the scanner's home sweep still disagree on the LEADING boundary for a home of two or more segments under a longer root: SweepCallerHome sweeps '/Volumes/T7/srv/qzhome/x' to '/Volumes/T7~/x' (iss-324: a longer root does not make the name someone else's), while RedactRoot, and so RedactHome, the CLI error scrub and the install receipt, leave it whole because the byte before the root is a path-segment byte (iss-2609230641546141 chose that to stop '/private~/wt/a' mangles). The identity survives in CLI output wherever a message names a home under a backup volume or mount. Closing it needs a choice between the two rules for the '~' polarity (the '.' repo polarity has no counterpart); found while unifying the trailing half under iss-2608292037564347.

## Evidence, 2026-09-29

The review of the trailing-boundary unification (iss-2608292037564347) found two further places where the redactors read a path differently. Both belong to the question this record already asks: should RedactRoot and the scanner's home sweep agree.

1. The '.' repo polarity renders a suffixed repo root as a dot pair. Since 5d8bdc084 '.' ends a name, so a message naming the repo root with a trailing dot or a `.git` suffix is redacted with the replacement's own '.' left in front of the suffix: `fsutil.RedactRoot("see /Users/alice/proj.", "/Users/alice/proj", ".")` returns `see ..`, and `fsutil.RedactRoot("clone /Users/alice/proj.git now", "/Users/alice/proj", ".")` returns `clone ..git now`. Both read like a parent-directory reference. The redaction is the safe direction: the same probe at 5d8bdc084's parent returns both strings whole, leaking the full path. A fix is a choice of rendering for the '.' polarity, not of boundary.
2. On a case-folding host (macOS/APFS) the two redactors disagree on a case-variant home: `fsutil.RedactRoot("open /USERS/ALICE/a", "/Users/alice", "~")` returns `open ~/a`, while `scanner.SweepCallerHome("open /USERS/ALICE/a", "/Users/alice")` returns the string whole. RedactRoot folds case by design (iss-2608270908341622); the sweep does not, and whether the store's backstop covers the variant is unverified. This predates 5d8bdc084.

Probes: an external `fsutil_test` probe on a `git archive` copy of fix/drain-drift-2 at b8d69535c, and the same probe on a copy of 5d8bdc084's parent, both on darwin.
