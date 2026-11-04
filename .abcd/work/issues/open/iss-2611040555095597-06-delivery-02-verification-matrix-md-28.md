---
schema_version: 1
id: "iss-2611040555095597"
slug: "06-delivery-02-verification-matrix-md-28"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Reduce the row to what is shown, or mark the list (staged)."
---

06-delivery/02-verification-matrix.md:28 says that on a private-to-public flip the user is shown the resulting tracked-versus-ignored list. The gitignore does change, but abcd ahoy install --visibility public prints only that it kept abcd's private files out of git and the visibility change; --details adds the files written, and the --json result has no field for such a list.
