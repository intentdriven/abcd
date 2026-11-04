---
schema_version: 1
id: "iss-2611040555083048"
slug: "06-delivery-02-verification-matrix-md-26"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Remove the clause from the row, or mark it (staged)."
---

06-delivery/02-verification-matrix.md:26 says a second install at the same version gives a transparent re-confirm of visibility. A second abcd ahoy install prints nothing about visibility and no non-test source under internal/core/ahoy contains such a step; the already_up_to_date headline (install_summary.go:425) does not mention it either. The matrix's preamble says rows without (staged) describe the binary today.
