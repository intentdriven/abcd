---
schema_version: 1
id: "iss-2608300349493306"
slug: "itd-180-fourth-round-residue"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-180 fourth-round security review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, internal/core/capture/reading.go, internal/core/capture/promote.go, internal/core/lint/schema.go"
resolution: "An unreadable item directory or record file is reported unsafe rather than unanswered, a sole standing record nobody can read is reported illegible rather than vanishing, and every reading and disposition read in the three files goes through fsutil.ReadGuarded, retiring the Lstat-then-ReadFile pair. record_schema's skip of a symlinked status bucket is pre-existing and left alone."
impact: internal
---

itd-180 fourth-round residue, all Low: a symlinked item directory or disposition file is reported as outstanding rather than unsafe; a sole standing record that is not well-formed has an empty state and matches no case so the item vanishes from the board; every disposition and reading file read is Lstat then ReadFile rather than fsutil.ReadGuarded, leaving a stat-then-read window a racing local writer could use to swap a symlink under the read that licenses a stamp; pre-existing and not this branch's: record_schema skips a status bucket that is itself a symlink without a finding.
