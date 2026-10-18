---
schema_version: 1
id: "iss-2610171626389087"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Rewrite the passage at .abcd/development/brief/06-delivery/02-verification-matrix.md:100 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "the matrix's fork-origin row names a run's renewal of its own reservation in place of a --renew flag that does not exist"
impact: internal
---

The v0.15.1 release review found the brief at .abcd/development/brief/06-delivery/02-verification-matrix.md:100 out of step with the shipped code. It says: Reservation on a fork origin row: 'a plain reserve that renews, `--renew` and a release never ask the forge', naming `--renew` as a reservation flag. In the code: No `--renew` flag exists: `abcd reserve --help` lists only --footprint, --holder, --lease, --release, --take-over (a renewal is the same command run again from the same copy), and no non-test Go source registers or names `--renew`.
